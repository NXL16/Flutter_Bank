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
	"gorm.io/gorm"
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
	req.FullName = strings.TrimSpace(req.FullName)
	if len([]rune(req.FullName)) < 2 || len([]rune(req.FullName)) > 100 {
		return errors.New("Họ và tên phải có từ 2 đến 100 ký tự")
	}
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
		return errors.New("Số điện thoại xác thực không khớp với số đăng ký")
	}

	existingPhone, err := s.repo.FindUserByPhone(formattedPhone)
	if err != nil {
		return err
	}
	if existingPhone != nil {
		return errors.New("Số điện thoại đã được sử dụng")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	user := &User{
		FullName:     strings.TrimSpace(req.FullName),
		Phone:        formattedPhone,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		IsVerified:   true,
		IsLocked:     false,
	}
	return s.repo.WithTx(func(tx *gorm.DB) error {
		if err := s.repo.withDB(tx).CreateUser(user); err != nil {
			return err
		}
		if err := s.accountService.
			WithTransaction(tx).
			CreateDefaultPaymentAccount(user.ID); err != nil {
			return err
		}
		return s.userService.
			WithTransaction(tx).
			CreateEmptyProfile(user.ID)
	})
}

// Login xử lý đăng nhập: trả về AuthResponse chứa cờ yêu cầu OTP SMS với user thường hoặc token trực tiếp với admin/super admin
func (s *Service) Login(req LoginRequest, userAgent string, ipAddress string, deviceID string) (*AuthResponse, error) {
	phoneFormatted := "+84" + normalizePhone(req.Phone)
	user, err := s.repo.FindUserByPhone(phoneFormatted)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("Tài khoản hoặc mật khẩu không đúng")
	}

	if user.IsLocked {
		return nil, errors.New("Tài khoản đã bị khóa")
	}

	if !user.IsVerified {
		return nil, errors.New("Tài khoản chưa được xác thực")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	)
	if err != nil {
		return nil, errors.New("Tài khoản hoặc mật khẩu không đúng")
	}

	// Phân loại xử lý dựa trên vai trò
	if user.Role == "admin" || user.Role == "super_admin" {
		// Yêu cầu TOTP code
		if req.TOTPCode == "" {
			return &AuthResponse{TOTPRequired: true}, nil
		}

		now := time.Now()
		if user.TOTPLockedUntil != nil && user.TOTPLockedUntil.After(now) {
			return nil, errors.New(
				"Xác thực TOTP đang tạm khóa do nhập sai quá nhiều lần",
			)
		}
		timeStep, valid := totp.MatchCodeAt(
			user.TOTPSecret,
			req.TOTPCode,
			now,
		)
		if !valid {
			lockedUntil, recordErr := s.repo.RecordTOTPFailure(
				user.ID,
				now,
				5,
				10*time.Minute,
			)
			if recordErr != nil {
				return nil, recordErr
			}
			if lockedUntil != nil {
				return nil, errors.New(
					"Nhập sai TOTP quá nhiều lần, xác thực bị khóa 10 phút",
				)
			}
			return nil, errors.New("Mã xác thực TOTP không đúng hoặc đã hết hạn")
		}
		if err := s.repo.RecordTOTPUsage(
			user.ID,
			timeStep,
			"ADMIN_LOGIN",
		); err != nil {
			return nil, errors.New(
				"Mã TOTP đã được sử dụng, vui lòng chờ mã mới",
			)
		}
		if err := s.repo.ResetTOTPFailures(user.ID); err != nil {
			return nil, err
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
		return errors.New("Mật khẩu phải có ít nhất 8 ký tự")
	}

	// Ít nhất 1 chữ hoa
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)

	// Ít nhất 1 ký tự đặc biệt
	hasSpecialChar := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	if !hasUppercase || !hasSpecialChar {
		return errors.New(
			"Mật khẩu phải có ít nhất 1 chữ hoa và 1 ký tự đặc biệt",
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
		return nil, errors.New("Refresh token không hợp lệ hoặc đã hết hạn")
	}

	refreshTokenHash := sha256.Sum256([]byte(refreshToken))
	storedToken, err := s.repo.FindActiveRefreshToken(hex.EncodeToString(refreshTokenHash[:]))
	if err != nil {
		return nil, err
	}
	if storedToken == nil || storedToken.UserID != claims.UserID {
		return nil, errors.New("Phiên đăng nhập không tồn tại hoặc đã bị thu hồi")
	}

	user, err := s.repo.FindUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("Người dùng không tồn tại")
	}

	if user.IsLocked {
		return nil, errors.New("Tài khoản đã bị khóa")
	}
	if claims.SessionVersion != user.SessionVersion {
		return nil, errors.New("Phiên đăng nhập đã hết hiệu lực")
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
		return errors.New("Tài khoản đã bị khóa")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.OldPassword),
	)
	if err != nil {
		return errors.New("Mật khẩu cũ không đúng")
	}

	if req.OldPassword == req.NewPassword {
		return errors.New("Mật khẩu mới không được trùng mật khẩu cũ")
	}

	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}

	verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
	if err != nil {
		return errors.New("Mã xác thực OTP không hợp lệ hoặc đã hết hạn")
	}
	if normalizePhone(verifiedPhone) != normalizePhone(user.Phone) {
		return errors.New("Số điện thoại xác thực không khớp với tài khoản")
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
		return errors.New("Thông tin xác thực không hợp lệ")
	}

	verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
	if err != nil {
		return err
	}
	if normalizePhone(verifiedPhone) != normalizePhone(user.Phone) {
		return errors.New("Số điện thoại xác thực không khớp với tài khoản")
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
		return nil, errors.New("Người dùng không tồn tại")
	}

	if user.IsLocked {
		return nil, errors.New("Tài khoản đã bị khóa")
	}

	if !user.IsVerified {
		return nil, errors.New("Tài khoản chưa được xác thực")
	}

	verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
	if err != nil {
		return nil, err
	}
	if normalizePhone(verifiedPhone) != normalizePhone(user.Phone) {
		return nil, errors.New("Số điện thoại xác thực không khớp với tài khoản")
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

	// OTP điện thoại vừa xác minh chính chủ, vì vậy đăng ký thiết bị mới.
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
		return "Thiết bị thử nghiệm"
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
		return errors.New("Số điện thoại không đúng định dạng nhà mạng Việt Nam")
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
