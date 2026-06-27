package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"bank-service/internal/config"
	"bank-service/internal/infrastructure/email"
	"bank-service/internal/infrastructure/firebase"
	jwtProvider "bank-service/internal/infrastructure/jwt"
	"bank-service/internal/infrastructure/totp"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/user"

	"golang.org/x/crypto/bcrypt"
)

// Service cung cấp các phương thức xử lý logic liên quan đến xác thực và quản lý người dùng
type Service struct {
	repo               *Repository
	otpRepo            *OTPRepository
	verifyRegisterRepo *VerifyRegisterRepository
	emailSender        *email.Sender
	accountService     *account.Service
	userService        *user.Service
	cfg                *config.Config
	firebaseClient     *firebase.Client
}

// NewService khởi tạo một instance của Service với các dependency cần thiết
func NewService(
	repo *Repository,
	otpRepo *OTPRepository,
	verifyRegisterRepo *VerifyRegisterRepository,
	emailSender *email.Sender,
	accountService *account.Service,
	userService *user.Service,
	cfg *config.Config,
	firebaseClient *firebase.Client,
) *Service {
	return &Service{
		repo:               repo,
		otpRepo:            otpRepo,
		accountService:     accountService,
		emailSender:        emailSender,
		userService:        userService,
		cfg:                cfg,
		verifyRegisterRepo: verifyRegisterRepo,
		firebaseClient:     firebaseClient,
	}
}

// Register xử lý đăng ký tài khoản
func (s *Service) Register(req RegisterRequest) error {
	if err := validatePassword(req.Password); err != nil {
		return err
	}

	existingUser, err := s.repo.FindUserByEmail(req.Email)
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

	existingPhone, err := s.repo.FindUserByPhone(formattedPhone)
	if err != nil {
		return err
	}

	if existingPhone != nil {
		return errors.New("số điện thoại đã được sử dụng")
	}

	if existingUser != nil {
		return errors.New("email đã được sử dụng")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	otp, err := generateOTP()
	if err != nil {
		return err
	}

	otpHash, err := bcrypt.GenerateFromPassword(
		[]byte(otp),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	verifyRegister := &VerifyRegister{
		Email:        req.Email,
		FullName:     req.FullName,
		Phone:        formattedPhone,
		PasswordHash: string(hashedPassword),
		OTPHash:      string(otpHash),
		OTPChannel:   req.OTPChannel,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.verifyRegisterRepo.Create(ctx, verifyRegister); err != nil {
		return err
	}

	// In OTP ra console để hỗ trợ test local / debug nếu email gặp sự cố
	fmt.Printf("\n🔑 [TEST/DEBUG] Đăng ký tài khoản: %s | OTP: %s\n\n", req.Email, otp)

	if err := s.emailSender.SendRegisterOTP(req.Email, otp); err != nil {
		fmt.Printf("⚠️ Lỗi gửi Email OTP (SMTP): %v. Nhưng vẫn tiếp tục ở chế độ test (Đọc OTP từ log trên).\n", err)
	}

	return nil
}

// Login xử lý đăng nhập: trả về AuthResponse chứa cờ yêu cầu OTP SMS với user thường hoặc token trực tiếp với admin/super admin
func (s *Service) Login(req LoginRequest, userAgent string, ipAddress string, deviceID string) (*AuthResponse, error) {
	var user *User
	var err error

	// Thử tìm theo Email/Username trước (hỗ trợ cả custom username không có ký tự @)
	user, err = s.repo.FindUserByEmail(req.Email)
	if err != nil {
		return nil, err
	}

	// Nếu không tìm thấy và không chứa ký tự @, tiến hành tìm theo số điện thoại
	if user == nil && !strings.Contains(req.Email, "@") {
		phoneFormatted := normalizePhone(req.Email)
		if len(phoneFormatted) > 0 {
			phoneFormatted = "+84" + phoneFormatted
			user, err = s.repo.FindUserByPhone(phoneFormatted)
			if err != nil {
				return nil, err
			}
		}
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
			user.Email,
			user.Role,
			user.SessionVersion,
			s.cfg.AccessTokenSecret,
		)
		if err != nil {
			return nil, err
		}

		refreshToken, err := jwtProvider.GenerateRefreshToken(
			user.ID,
			user.Email,
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
				Email:      user.Email,
				Phone:      user.Phone,
				Role:       user.Role,
				IsVerified: user.IsVerified,
			},
		}, nil
	}

	// Đối với user bình thường: tự động tạo OTP dự phòng gửi qua email và log ra console để test
	otp, err := generateOTP()
	if err == nil {
		otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			loginOTP := &OTP{
				Email:     user.Email,
				OTPHash:   string(otpHash),
				Purpose:   "login",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}
			_ = s.otpRepo.CreateOTP(ctx, loginOTP)

			fmt.Printf("\n🔑 [TEST/DEBUG] OTP Đăng nhập của %s: %s\n\n", user.Email, otp)

			// Gửi Email
			if err := s.emailSender.SendLoginOTP(user.Email, otp); err != nil {
				fmt.Printf("⚠️ Lỗi gửi Email OTP Đăng nhập: %v. Đăng nhập vẫn tiếp tục ở chế độ debug.\n", err)
			}
		}
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

	accessToken, err := jwtProvider.GenerateAccessToken(
		user.ID,
		user.Email,
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
			Email:      user.Email,
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

	return s.repo.UpdatePassword(userID, string(hashedPassword))
}

// generateOTP tạo một mã OTP ngẫu nhiên 6 chữ số
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

// ForgotPassword xử lý yêu cầu quên mật khẩu
func (s *Service) ForgotPassword(req ForgotPasswordRequest) error {
	user, err := s.repo.FindUserByEmail(req.Email)
	if err != nil {
		return err
	}

	// Không leak email tồn tại hay không
	if user == nil {
		return nil
	}

	otp, err := generateOTP()
	if err != nil {
		return err
	}

	otpHash, err := bcrypt.GenerateFromPassword(
		[]byte(otp),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	resetOTP := &OTP{
		UserID:    user.ID,
		Email:     user.Email,
		OTPHash:   string(otpHash),
		Purpose:   "password_reset",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.otpRepo.CreateOTP(ctx, resetOTP); err != nil {
		return err
	}

	if err := s.emailSender.SendResetPasswordOTP(user.Email, otp); err != nil {
		return err
	}
	return nil
}

// ResetPassword xử lý yêu cầu đặt lại mật khẩu
func (s *Service) ResetPassword(req ResetPasswordRequest) error {
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}

	user, err := s.repo.FindUserByEmail(req.Email)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("otp không hợp lệ hoặc đã hết hạn")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resetOTP, err := s.otpRepo.FindValidOTPByEmailAndPurpose(ctx, req.Email, "password_reset")
	if err != nil {
		return err
	}

	if resetOTP == nil {
		return errors.New("otp không hợp lệ hoặc đã hết hạn")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(resetOTP.OTPHash),
		[]byte(req.OTP),
	)
	if err != nil {
		return errors.New("otp không hợp lệ hoặc đã hết hạn")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.NewPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	_ = s.otpRepo.DeleteOTP(ctx, resetOTP.ID)

	return s.repo.UpdatePassword(user.ID, string(hashedPassword))
}

// ConfirmRegister xác thực OTP đăng ký tài khoản (hỗ trợ Firebase SMS OTP)
func (s *Service) ConfirmRegister(req ConfirmRegisterRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	verifyRegister, err := s.verifyRegisterRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return err
	}

	if verifyRegister == nil {
		return errors.New("yêu cầu đăng ký không tồn tại hoặc đã hết hạn")
	}

	if req.IDToken != "" {
		// Xác thực bằng Firebase ID Token (SĐT)
		verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
		if err != nil {
			return err
		}
		if normalizePhone(verifiedPhone) != normalizePhone(verifyRegister.Phone) {
			return errors.New("số điện thoại xác thực không khớp với số đăng ký")
		}
	} else if req.OTP != "" {
		// Fallback cho email OTP
		err = bcrypt.CompareHashAndPassword(
			[]byte(verifyRegister.OTPHash),
			[]byte(req.OTP),
		)
		if err != nil {
			return errors.New("otp không hợp lệ hoặc đã hết hạn")
		}
	} else {
		return errors.New("thiếu thông tin xác thực OTP hoặc ID Token")
	}

	user := &User{
		Email:        verifyRegister.Email,
		FullName:     verifyRegister.FullName,
		Phone:        verifyRegister.Phone,
		PasswordHash: verifyRegister.PasswordHash,
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

	if err := s.verifyRegisterRepo.Delete(ctx, verifyRegister.ID); err != nil {
		return err
	}

	return nil
}

// ConfirmLogin xác thực OTP đăng nhập và cấp token (đã nâng cấp hỗ trợ Firebase SMS OTP và nhận diện thiết bị đầu tiên)
func (s *Service) ConfirmLogin(req ConfirmLoginRequest, userAgent string, ipAddress string, deviceID string) (*AuthResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user *User
	var err error

	if strings.Contains(req.Email, "@") {
		user, err = s.repo.FindUserByEmail(req.Email)
	} else {
		phoneFormatted := normalizePhone(req.Email)
		if len(phoneFormatted) > 0 {
			phoneFormatted = "+84" + phoneFormatted
		}
		user, err = s.repo.FindUserByPhone(phoneFormatted)
	}

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

	if req.IDToken != "" {
		// Xác thực bằng Firebase ID Token
		verifiedPhone, err := s.firebaseClient.VerifyIDToken(req.IDToken)
		if err != nil {
			return nil, err
		}
		if normalizePhone(verifiedPhone) != normalizePhone(user.Phone) {
			return nil, errors.New("số điện thoại xác thực không khớp với tài khoản")
		}
	} else if req.OTP != "" {
		// Fallback cho email OTP
		loginOTP, err := s.otpRepo.FindValidOTPByEmailAndPurpose(ctx, req.Email, "login")
		if err != nil || loginOTP == nil {
			return nil, errors.New("otp không hợp lệ hoặc đã hết hạn")
		}
		err = bcrypt.CompareHashAndPassword([]byte(loginOTP.OTPHash), []byte(req.OTP))
		if err != nil {
			return nil, errors.New("otp không hợp lệ hoặc đã hết hạn")
		}
		_ = s.otpRepo.DeleteOTP(ctx, loginOTP.ID)
	} else {
		return nil, errors.New("thiếu thông tin xác thực OTP hoặc ID Token")
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

	// 4. Nếu thiết bị lạ (chưa tin cậy) -> Chặn tạm thời và gửi email xác nhận tương tác
	if !isTrusted {
		// Sinh deviceID mới nếu chưa có
		newDeviceID := deviceID
		if newDeviceID == "" {
			newDeviceID = generateUUID()
		}

		// Tạo bản ghi PendingLogin
		pendingID := generateUUID()
		pending := &PendingLogin{
			ID:        pendingID,
			UserID:    user.ID,
			DeviceID:  newDeviceID,
			UserAgent: userAgent,
			IPAddress: ipAddress,
			Location:  location,
			Status:    "PENDING",
			ExpiresAt: time.Now().Add(15 * time.Minute),
		}

		if err := s.repo.CreatePendingLogin(pending); err != nil {
			return nil, err
		}

		// Gửi email chứa 2 nút hành động
		host := s.cfg.AppURL
		if host == "" {
			if s.cfg.ServerPort == "" {
				s.cfg.ServerPort = "8080"
			}
			host = fmt.Sprintf("http://localhost:%s", s.cfg.ServerPort)
		}
		confirmURL := fmt.Sprintf("%s/api/v1/auth/device-verification/confirm?token=%s", host, pendingID)
		rejectURL := fmt.Sprintf("%s/api/v1/auth/device-verification/reject?token=%s", host, pendingID)

		deviceName := parseUserAgent(userAgent)
		_ = s.emailSender.SendNewDeviceAlert(user.Email, ipAddress, location, deviceName, confirmURL, rejectURL)

		return &AuthResponse{
			PendingVerification: true,
			PendingID:           pendingID,
			DeviceID:            newDeviceID,
		}, nil
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
		user.Email,
		user.Role,
		user.SessionVersion,
		s.cfg.AccessTokenSecret,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwtProvider.GenerateRefreshToken(
		user.ID,
		user.Email,
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
			Email:      user.Email,
			Phone:      user.Phone,
			Role:       user.Role,
			IsVerified: user.IsVerified,
		},
	}, nil
}

// ConfirmDeviceVerification phê duyệt thiết bị từ email
func (s *Service) ConfirmDeviceVerification(token string) error {
	pending, err := s.repo.FindPendingLogin(token)
	if err != nil {
		return err
	}
	if pending == nil {
		return errors.New("yêu cầu xác thực không tồn tại")
	}

	if pending.Status != "PENDING" {
		return fmt.Errorf("yêu cầu xác thực đã được xử lý (trạng thái: %s)", pending.Status)
	}

	if time.Now().After(pending.ExpiresAt) {
		_ = s.repo.UpdatePendingLoginStatus(token, "EXPIRED")
		return errors.New("yêu cầu xác thực đã hết hạn (hiệu lực 15 phút)")
	}

	// 1. Phê duyệt trạng thái
	if err := s.repo.UpdatePendingLoginStatus(token, "APPROVED"); err != nil {
		return err
	}

	// 2. Thêm thiết bị mới vào danh sách tin cậy
	device := &UserDevice{
		UserID:         pending.UserID,
		DeviceID:       pending.DeviceID,
		UserAgent:      pending.UserAgent,
		LastActiveIP:   pending.IPAddress,
		LastLocation:   pending.Location,
		LastLoggedInAt: time.Now(),
	}

	return s.repo.CreateUserDevice(device)
}

// RejectDeviceVerification từ chối thiết bị và khóa tài khoản khẩn cấp
func (s *Service) RejectDeviceVerification(token string) error {
	pending, err := s.repo.FindPendingLogin(token)
	if err != nil {
		return err
	}
	if pending == nil {
		return errors.New("yêu cầu xác thực không tồn tại")
	}

	if pending.Status != "PENDING" {
		return fmt.Errorf("yêu cầu xác thực đã được xử lý (trạng thái: %s)", pending.Status)
	}

	// 1. Cập nhật trạng thái từ chối
	if err := s.repo.UpdatePendingLoginStatus(token, "REJECTED"); err != nil {
		return err
	}

	// 2. Khóa tài khoản người dùng ngay lập tức
	if err := s.repo.LockUser(pending.UserID); err != nil {
		return err
	}

	// 3. Vô hiệu hóa toàn bộ refresh token hiện có
	_ = s.repo.RevokeAllUserRefreshTokens(pending.UserID)

	// 4. Tăng session version để lập tức logout tất cả phiên đăng nhập khác
	_ = s.repo.IncreaseSessionVersion(pending.UserID)

	return nil
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
