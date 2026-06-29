package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"bank-service/internal/config"
	"bank-service/internal/infrastructure/firebase"
	jwtProvider "bank-service/internal/infrastructure/jwt"
	"bank-service/internal/infrastructure/totp"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/user"

	"golang.org/x/crypto/bcrypt"
)

// Service cung cấp các phương thức xử lý logic liên quan đến xác thực và quản lý người dùng
type Service struct {
	repo           *Repository
	accountService *account.Service
	userService    *user.Service
	cfg            *config.Config
	firebaseClient *firebase.Client
}

// NewService khởi tạo một instance của Service với các dependency cần thiết
func NewService(
	repo *Repository,
	accountService *account.Service,
	userService *user.Service,
	cfg *config.Config,
	firebaseClient *firebase.Client,
) *Service {
	return &Service{
		repo:           repo,
		accountService: accountService,
		userService:    userService,
		cfg:            cfg,
		firebaseClient: firebaseClient,
	}
}

// Register xử lý đăng ký tài khoản
func (s *Service) Register(req RegisterRequest) error {
	if err := validatePassword(req.Password); err != nil {
		return err
	}

	phoneToValidate := req.Phone
	if len(req.Phone) >= 11 && req.Phone[:3] == "+84" {
		phoneToValidate = "0" + req.Phone[3:]
	} else if len(req.Phone) >= 10 && req.Phone[:2] == "84" {
		phoneToValidate = "0" + req.Phone[2:]
	}

	if err := validateVietnamPhone(phoneToValidate); err != nil {
		return err
	}

	formattedPhone := req.Phone
	if phoneToValidate[:1] == "0" {
		formattedPhone = "+84" + phoneToValidate[1:]
	}

	verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
	if err != nil {
		return err
	}
	if normalizePhone(verifiedPhone) != normalizePhone(formattedPhone) {
		return errors.New("số điện thoại xác thực không khớp với số đăng ký")
	}

	existingPhone, err := s.repo.FindUserByPhone(formattedPhone)
	if err != nil {
		return err
	}
	if existingPhone != nil {
		return errors.New("số điện thoại đã được sử dụng")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	// Cột email được giữ tạm để tương thích migration dữ liệu cũ, nhưng không
	// còn được dùng làm định danh hay hiển thị cho khách hàng.
	internalIdentity := fmt.Sprintf("%s@phone.identity", normalizePhone(formattedPhone))
	user := &User{
		Email:        internalIdentity,
		FullName:     strings.TrimSpace(req.FullName),
		Phone:        formattedPhone,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		IsVerified:   true,
		IsLocked:     false,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return err
	}
	if err := s.accountService.CreateDefaultPaymentAccount(user.ID); err != nil {
		return err
	}
	return s.userService.CreateEmptyProfile(user.ID)
}

// Login xử lý đăng nhập: trả về AuthResponse chứa cờ yêu cầu OTP SMS với user thường hoặc token trực tiếp với admin/super admin
func (s *Service) Login(req LoginRequest, userAgent string, ipAddress string, deviceID string) (*AuthResponse, error) {
	phoneFormatted := "+84" + normalizePhone(req.Phone)
	user, err := s.repo.FindUserByPhone(phoneFormatted)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("tài khoản hoặc mật khẩu không đúng")
	}

	if user.IsLocked {
		return nil, errors.New("tài khoản đã bị khóa")
	}

	if !user.IsVerified {
		return nil, errors.New("tài khoản chưa được xác thực")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		return nil, errors.New("tài khoản hoặc mật khẩu không đúng")
	}

	// Phân loại xử lý dựa trên vai trò
	if user.Role == "admin" || user.Role == "super_admin" {
		// Yêu cầu TOTP code
		if req.TOTPCode == "" {
			return &AuthResponse{TOTPRequired: true}, nil
		}

		if !totp.ValidateCode(user.TOTPSecret, req.TOTPCode) {
			return nil, errors.New("mã xác thực TOTP không đúng hoặc đã hết hạn")
		}

		// Tạo Access Token và Refresh Token ngay lập tức cho Admin/Super Admin
		accessToken, err := jwtProvider.GenerateAccessToken(
			user.ID,
			user.Phone,
			user.Role,
			user.SessionVersion,
			s.cfg.AccessTokenSecret,
		)
		if err != nil {
			return nil, err
		}

		refreshToken, err := jwtProvider.GenerateRefreshToken(
			user.ID,
			user.Phone,
			user.Role,
			user.SessionVersion,
			s.cfg.RefreshTokenSecret,
		)
		if err != nil {
			return nil, err
		}

		refreshTokenHash := sha256.Sum256([]byte(refreshToken))

		savedRefreshToken := &RefreshToken{
			UserID:    user.ID,
			TokenHash: hex.EncodeToString(refreshTokenHash[:]),
			IsRevoked: false,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		if err := s.repo.CreateRefreshToken(savedRefreshToken); err != nil {
			return nil, err
		}

		if deviceID == "" {
			deviceID = generateUUID()
		}

		return &AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			DeviceID:     deviceID,
			User: UserResponse{
				ID:         user.ID,
				FullName:   user.FullName,
				Phone:      user.Phone,
				Role:       user.Role,
				IsVerified: user.IsVerified,
			},
		}, nil
	}

	return &AuthResponse{
		SMSAuthRequired: true,
		Phone:           user.Phone,
	}, nil
}

// validatePassword kiểm tra độ mạnh mật khẩu
func validatePassword(password string) error {
	// Tối thiểu 8 ký tự
	if len(password) < 8 {
		return errors.New("mật khẩu phải có ít nhất 8 ký tự")
	}

	// Ít nhất 1 chữ hoa
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)

	// Ít nhất 1 ký tự đặc biệt
	hasSpecialChar := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	if !hasUppercase || !hasSpecialChar {
		return errors.New(
			"mật khẩu phải có ít nhất 1 chữ hoa và 1 ký tự đặc biệt",
		)
	}
	return nil
}

// Logout xử lý đăng xuất
func (s *Service) Logout(refreshToken string) error {

	refreshTokenHash := sha256.Sum256(
		[]byte(refreshToken),
	)

	return s.repo.RevokeRefreshToken(
		hex.EncodeToString(refreshTokenHash[:]),
	)
}

// RefreshAccessToken tạo access token mới từ refresh token
func (s *Service) RefreshAccessToken(refreshToken string) (*AuthResponse, error) {
	claims, err := jwtProvider.ValidateToken(
		refreshToken,
		s.cfg.RefreshTokenSecret,
	)
	if err != nil {
		return nil, errors.New("refresh token không hợp lệ hoặc đã hết hạn")
	}

	refreshTokenHash := sha256.Sum256([]byte(refreshToken))
	storedToken, err := s.repo.FindActiveRefreshToken(hex.EncodeToString(refreshTokenHash[:]))
	if err != nil {
		return nil, err
	}
	if storedToken == nil || storedToken.UserID != claims.UserID {
		return nil, errors.New("phiên đăng nhập không tồn tại hoặc đã bị thu hồi")
	}

	user, err := s.repo.FindUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("người dùng không tồn tại")
	}

	if user.IsLocked {
		return nil, errors.New("tài khoản đã bị khóa")
	}
	if claims.SessionVersion != user.SessionVersion {
		return nil, errors.New("phiên đăng nhập đã hết hiệu lực")
	}

	accessToken, err := jwtProvider.GenerateAccessToken(
		user.ID,
		user.Phone,
		user.Role,
		user.SessionVersion,
		s.cfg.AccessTokenSecret,
	)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken: accessToken,
		User: UserResponse{
			ID:         user.ID,
			FullName:   user.FullName,
			Phone:      user.Phone,
			Role:       user.Role,
			IsVerified: user.IsVerified,
		},
	}, nil
}

// ChangePassword xử lý đổi mật khẩu
func (s *Service) ChangePassword(userID uint, req ChangePasswordRequest) error {
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("người dùng không tồn tại")
	}

	if user.IsLocked {
		return errors.New("tài khoản đã bị khóa")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.OldPassword),
	)
	if err != nil {
		return errors.New("mật khẩu cũ không đúng")
	}

	if req.OldPassword == req.NewPassword {
		return errors.New("mật khẩu mới không được trùng mật khẩu cũ")
	}

	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.NewPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(userID, string(hashedPassword)); err != nil {
		return err
	}
	_ = s.repo.RevokeAllUserRefreshTokens(userID)
	return s.repo.IncreaseSessionVersion(userID)
}

// ResetPassword xử lý yêu cầu đặt lại mật khẩu
func (s *Service) ResetPassword(req ResetPasswordRequest) error {
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}

	phone := "+84" + normalizePhone(req.Phone)
	user, err := s.repo.FindUserByPhone(phone)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("thông tin xác thực không hợp lệ")
	}

	verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
	if err != nil {
		return err
	}
	if normalizePhone(verifiedPhone) != normalizePhone(user.Phone) {
		return errors.New("số điện thoại xác thực không khớp với tài khoản")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.NewPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(user.ID, string(hashedPassword)); err != nil {
		return err
	}
	_ = s.repo.RevokeAllUserRefreshTokens(user.ID)
	return s.repo.IncreaseSessionVersion(user.ID)
}

// ConfirmLogin xác thực OTP đăng nhập và cấp token (đã nâng cấp hỗ trợ Firebase SMS OTP và nhận diện thiết bị đầu tiên)
func (s *Service) ConfirmLogin(req ConfirmLoginRequest, userAgent string, ipAddress string, deviceID string) (*AuthResponse, error) {
	phoneFormatted := "+84" + normalizePhone(req.Phone)
	user, err := s.repo.FindUserByPhone(phoneFormatted)

	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("người dùng không tồn tại")
	}

	if user.IsLocked {
		return nil, errors.New("tài khoản đã bị khóa")
	}

	if !user.IsVerified {
		return nil, errors.New("tài khoản chưa được xác thực")
	}

	verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
	if err != nil {
		return nil, err
	}
	if normalizePhone(verifiedPhone) != normalizePhone(user.Phone) {
		return nil, errors.New("số điện thoại xác thực không khớp với tài khoản")
	}

	// 1. Tự động dọn dẹp các thiết bị tin cậy cũ quá 30 ngày của user này
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	_ = s.repo.DeleteStaleUserDevices(user.ID, thirtyDaysAgo)

	// 2. Định vị địa lý của IP đăng nhập
	location := resolveLocation(ipAddress)

	// 3. Kiểm tra thiết bị đăng nhập hiện tại
	var isTrusted bool = false

	// Kiểm tra xem đây có phải là thiết bị đăng nhập đầu tiên không
	deviceCount, err := s.repo.CountUserDevices(user.ID)
	if err == nil && deviceCount == 0 {
		// Tự động nhận diện đây là thiết bị quen đầu tiên!
		newDeviceID := deviceID
		if newDeviceID == "" {
			newDeviceID = generateUUID()
		}
		device := &UserDevice{
			UserID:         user.ID,
			DeviceID:       newDeviceID,
			UserAgent:      userAgent,
			LastActiveIP:   ipAddress,
			LastLocation:   location,
			LastLoggedInAt: time.Now(),
		}
		if err := s.repo.CreateUserDevice(device); err == nil {
			deviceID = newDeviceID
			isTrusted = true
		}
	}

	if !isTrusted && deviceID != "" {
		device, err := s.repo.FindUserDevice(user.ID, deviceID)
		if err == nil && device != nil {
			// Thiết bị khớp! Cập nhật IP, vị trí và thời điểm hoạt động cuối (đã gia hạn 30 ngày)
			_ = s.repo.UpdateUserDeviceLastLogin(device.ID, ipAddress, location)
			isTrusted = true
		}
	}

	// OTP điện thoại vừa xác minh chính chủ, vì vậy thiết bị mới được đăng ký
	// trực tiếp thay vì tiếp tục yêu cầu phê duyệt qua email.
	if !isTrusted {
		newDeviceID := deviceID
		if newDeviceID == "" {
			newDeviceID = generateUUID()
		}
		device := &UserDevice{
			UserID:         user.ID,
			DeviceID:       newDeviceID,
			UserAgent:      userAgent,
			LastActiveIP:   ipAddress,
			LastLocation:   location,
			LastLoggedInAt: time.Now(),
		}
		if err := s.repo.CreateUserDevice(device); err != nil {
			return nil, err
		}
		deviceID = newDeviceID
	}

	// 5. Nếu là thiết bị quen -> Cho phép đăng nhập bình thường
	// Kiểm tra giới hạn tối đa 3 thiết bị đồng thời
	activeSessions, err := s.repo.GetActiveSessions(user.ID)
	if err == nil && len(activeSessions) >= 3 {
		// Vô hiệu hóa session cũ nhất
		_ = s.repo.RevokeRefreshTokenByID(activeSessions[0].ID)
	}

	accessToken, err := jwtProvider.GenerateAccessToken(
		user.ID,
		user.Phone,
		user.Role,
		user.SessionVersion,
		s.cfg.AccessTokenSecret,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwtProvider.GenerateRefreshToken(
		user.ID,
		user.Phone,
		user.Role,
		user.SessionVersion,
		s.cfg.RefreshTokenSecret,
	)
	if err != nil {
		return nil, err
	}

	refreshTokenHash := sha256.Sum256([]byte(refreshToken))

	savedRefreshToken := &RefreshToken{
		UserID:    user.ID,
		TokenHash: hex.EncodeToString(refreshTokenHash[:]),
		IsRevoked: false,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.repo.CreateRefreshToken(savedRefreshToken); err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		DeviceID:     deviceID,
		User: UserResponse{
			ID:         user.ID,
			FullName:   user.FullName,
			Phone:      user.Phone,
			Role:       user.Role,
			IsVerified: user.IsVerified,
		},
	}, nil
}

func resolveLocation(ip string) string {
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return "Cục bộ (Thiết bị thử nghiệm)"
	}

	// Kiểm tra nếu là IP mạng nội bộ (LAN / Wi-Fi)
	if strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "172.16.") ||
		strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") ||
		strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.2") ||
		strings.HasPrefix(ip, "172.3") ||
		strings.HasPrefix(ip, "169.254.") {
		return "Mạng nội bộ (LAN / Wi-Fi)"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip + "?fields=status,country,regionName,city,isp")
	if err != nil {
		return "Không xác định"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Không xác định"
	}

	var geo struct {
		Status     string `json:"status"`
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
		ISP        string `json:"isp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return "Không xác định"
	}

	if geo.Status != "success" {
		return "Không xác định"
	}

	return fmt.Sprintf("%s, %s (%s)", geo.City, geo.Country, geo.ISP)
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func validateVietnamPhone(phone string) error {
	pattern := `^(03[2-9]|086|09[6-8]|08[1-5]|087|088|091|094|070|076|077|078|079|089|090|093|052|056|058|092|059|099)[0-9]{7}$`

	matched, err := regexp.MatchString(pattern, phone)
	if err != nil {
		return err
	}

	if !matched {
		return errors.New("số điện thoại không đúng định dạng nhà mạng Việt Nam")
	}

	return nil
}

func normalizePhone(phone string) string {
	reg := regexp.MustCompile(`\D`)
	digits := reg.ReplaceAllString(phone, "")

	if len(digits) >= 11 && digits[:2] == "84" {
		return digits[2:]
	}
	if len(digits) > 0 && digits[:1] == "0" {
		return digits[1:]
	}
	return digits
}

func parseUserAgent(ua string) string {
	if ua == "" {
		return "Thiết bị không xác định"
	}

	// Xác định hệ điều hành
	os := "Thiết bị không xác định"
	if strings.Contains(ua, "Windows NT") {
		os = "Máy tính Windows"
	} else if strings.Contains(ua, "Macintosh") || strings.Contains(ua, "Mac OS X") {
		os = "Máy tính macOS"
	} else if strings.Contains(ua, "iPhone") {
		os = "Điện thoại iPhone"
	} else if strings.Contains(ua, "iPad") {
		os = "Máy tính bảng iPad"
	} else if strings.Contains(ua, "Android") {
		os = "Điện thoại Android"
	} else if strings.Contains(ua, "Linux") {
		os = "Thiết bị Linux"
	}

	// Xác định trình duyệt
	browser := "Trình duyệt ẩn danh/khác"
	if strings.Contains(ua, "Chrome") && !strings.Contains(ua, "Edg") && !strings.Contains(ua, "OPR") {
		browser = "Google Chrome"
	} else if strings.Contains(ua, "Safari") && !strings.Contains(ua, "Chrome") {
		browser = "Apple Safari"
	} else if strings.Contains(ua, "Firefox") {
		browser = "Mozilla Firefox"
	} else if strings.Contains(ua, "Edg") {
		browser = "Microsoft Edge"
	} else if strings.Contains(ua, "OPR") || strings.Contains(ua, "Opera") {
		browser = "Opera"
	}

	return fmt.Sprintf("%s (%s)", browser, os)
}
