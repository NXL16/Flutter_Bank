package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	jwtProvider "bank-service/internal/infrastructure/jwt"
	"bank-service/internal/shared/response"

	"github.com/gin-gonic/gin"
)

// Handler chịu trách nhiệm xử lý HTTP request
type Handler struct {
	service *Service
}

// NewHandler tạo auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Register xử lý API đăng ký
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	// Bind JSON request vào struct
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	if err := h.service.Register(req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusCreated, "OTP xác thực đăng ký đã được gửi đến email", nil)
}

// Login xử lý API đăng nhập
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	deviceID, _ := c.Cookie("device_id")
	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	res, err := h.service.Login(req, userAgent, ipAddress, deviceID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// Mật khẩu Admin hợp lệ, yêu cầu nhập mã TOTP ở bước tiếp theo.
	if res.TOTPRequired {
		response.Success(c, http.StatusOK, "Vui lòng nhập mã xác thực TOTP", res)
		return
	}

	// Nếu là Admin/Super Admin đăng nhập thành công trực tiếp (có AccessToken)
	if res.AccessToken != "" {
		c.SetCookie(
			"device_id",
			res.DeviceID,
			365*24*60*60,
			"/api/v1/auth",
			"",
			false,
			true,
		)

		setRefreshTokenCookie(c, res.RefreshToken)

		response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
		return
	}

	response.Success(c, http.StatusOK, "Mật khẩu đúng, tiến hành xác thực số điện thoại", gin.H{
		"sms_auth_required": true,
		"phone":             res.Phone,
	})
}

// setRefreshTokenCookie thiết lập refresh token trong cookie
func setRefreshTokenCookie(c *gin.Context, refreshToken string) {
	c.SetCookie(
		"refresh_token",
		refreshToken,
		7*24*60*60, // 7 ngày
		"/api/v1/auth",
		"",
		false, // local dev dùng false, production HTTPS đổi thành true
		true,  // HttpOnly
	)
}

// Logout xử lý API logout
func (h *Handler) Logout(c *gin.Context) {

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Không tìm thấy refresh token", nil)
		return
	}

	err = h.service.Logout(refreshToken)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Đăng xuất thất bại", err)
		return
	}

	// Xóa refresh token cookie
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/v1/auth",
		"",
		false,
		true,
	)

	response.Success(c, http.StatusOK, "Đăng xuất thành công", nil)
}

// Refresh xử lý cấp access token mới từ refresh token cookie
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Không tìm thấy refresh token", nil)
		return
	}

	res, err := h.service.RefreshAccessToken(refreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Refresh token thành công", res)
}

// ChangePassword xử lý đổi mật khẩu
func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		response.Error(c, http.StatusUnauthorized, "Không xác định được người dùng", nil)
		return
	}

	if err := h.service.ChangePassword(userID, req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Đổi mật khẩu thành công", nil)
}

// ForgotPassword xử lý yêu cầu quên mật khẩu
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	if err := h.service.ForgotPassword(req); err != nil {
		response.Error(c, http.StatusInternalServerError, "Không thể gửi OTP", err)
		return
	}

	response.Success(c, http.StatusOK, "Nếu email tồn tại, OTP đã được gửi", nil)
}

// ResetPassword xử lý yêu cầu reset mật khẩu
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	if err := h.service.ResetPassword(req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Đặt lại mật khẩu thành công", nil)
}

// ConfirmRegister xác thực OTP đăng ký
func (h *Handler) ConfirmRegister(c *gin.Context) {
	var req ConfirmRegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	if err := h.service.ConfirmRegister(req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Xác thực đăng ký thành công, vui lòng đăng nhập", nil)
}

// ConfirmLogin xác thực OTP đăng nhập
func (h *Handler) ConfirmLogin(c *gin.Context) {
	var req ConfirmLoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	// Đọc device_id từ cookie (nếu có)
	deviceID, _ := c.Cookie("device_id")

	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	res, err := h.service.ConfirmLogin(req, userAgent, ipAddress, deviceID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// Nếu đang chờ xác thực thiết bị lạ
	if res.PendingVerification {
		// Set device_id cookie dài hạn (1 năm) với Path /api/v1/auth
		c.SetCookie(
			"device_id",
			res.DeviceID,
			365*24*60*60, // 1 năm
			"/api/v1/auth",
			"",
			false, // local dev false, production true
			true,  // HttpOnly
		)
		response.Success(c, http.StatusOK, "Thiết bị mới, vui lòng xác nhận qua email", res)
		return
	}

	// Set device_id cookie dài hạn
	c.SetCookie(
		"device_id",
		res.DeviceID,
		365*24*60*60,
		"/api/v1/auth",
		"",
		false,
		true,
	)

	setRefreshTokenCookie(c, res.RefreshToken)

	response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
}

// GetLoginStatus kiểm tra trạng thái của phiên chờ đăng nhập (polling)
func (h *Handler) GetLoginStatus(c *gin.Context) {
	pendingID := c.Query("pending_id")
	if pendingID == "" {
		response.Error(c, http.StatusBadRequest, "Thiếu pending_id", nil)
		return
	}

	pending, err := h.service.repo.FindPendingLogin(pendingID)
	if err != nil || pending == nil {
		response.Error(c, http.StatusNotFound, "Không tìm thấy yêu cầu xác thực hoặc đã hết hạn", err)
		return
	}

	if pending.Status == "PENDING" {
		c.JSON(http.StatusOK, gin.H{"status": "PENDING"})
		return
	}

	if pending.Status == "REJECTED" || pending.Status == "EXPIRED" {
		c.JSON(http.StatusOK, gin.H{"status": pending.Status, "message": "Đăng nhập bị từ chối hoặc yêu cầu đã hết hạn"})
		return
	}

	if pending.Status == "APPROVED" {
		// Thành công! Tiến hành cấp Access Token và Refresh Token cho thiết bị
		user, err := h.service.repo.FindUserByID(pending.UserID)
		if err != nil || user == nil {
			response.Error(c, http.StatusInternalServerError, "Lỗi đăng nhập", err)
			return
		}

		// Kiểm tra giới hạn 3 thiết bị đồng thời
		activeSessions, err := h.service.repo.GetActiveSessions(user.ID)
		if err == nil && len(activeSessions) >= 3 {
			_ = h.service.repo.RevokeRefreshTokenByID(activeSessions[0].ID)
		}

		accessToken, err := jwtProvider.GenerateAccessToken(
			user.ID,
			user.Email,
			user.Role,
			user.SessionVersion,
			h.service.cfg.AccessTokenSecret,
		)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Lỗi tạo token", err)
			return
		}

		refreshToken, err := jwtProvider.GenerateRefreshToken(
			user.ID,
			user.Email,
			user.Role,
			user.SessionVersion,
			h.service.cfg.RefreshTokenSecret,
		)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Lỗi tạo token", err)
			return
		}

		refreshTokenHash := sha256.Sum256([]byte(refreshToken))

		savedRefreshToken := &RefreshToken{
			UserID:    user.ID,
			TokenHash: hex.EncodeToString(refreshTokenHash[:]),
			IsRevoked: false,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		if err := h.service.repo.CreateRefreshToken(savedRefreshToken); err != nil {
			response.Error(c, http.StatusInternalServerError, "Lỗi lưu token", err)
			return
		}

		// Set cookies
		c.SetCookie(
			"device_id",
			pending.DeviceID,
			365*24*60*60,
			"/api/v1/auth",
			"",
			false,
			true,
		)

		setRefreshTokenCookie(c, refreshToken)

		// Xóa PendingLogin sau khi đăng nhập thành công
		_ = h.service.repo.DeletePendingLogin(pending.ID)

		response.Success(c, http.StatusOK, "Đăng nhập thành công", &AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			DeviceID:     pending.DeviceID,
			User: UserResponse{
				ID:         user.ID,
				FullName:   user.FullName,
				Email:      user.Email,
				Phone:      user.Phone,
				Role:       user.Role,
				IsVerified: user.IsVerified,
			},
		})
	}
}

// ConfirmDeviceVerification xác nhận thiết bị lạ từ link email
func (h *Handler) ConfirmDeviceVerification(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte("<h1>Lỗi: Thiếu token xác thực</h1>"))
		return
	}

	err := h.service.ConfirmDeviceVerification(token)
	if err != nil {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(fmt.Sprintf("<h1>Xác thực thất bại</h1><p>%s</p>", err.Error())))
		return
	}

	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>NF-Bank - Xác thực thành công</title>
		<meta charset="utf-8">
		<style>
			body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; background-color: #f8f9fa; margin: 0; }
			.card { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); text-align: center; max-width: 400px; }
			h1 { color: #28a745; margin-bottom: 20px; }
			p { color: #6c757d; line-height: 1.6; }
		</style>
	</head>
	<body>
		<div class="card">
			<h1>✓ Xác thực thành công!</h1>
			<p>Thiết bị mới của bạn đã được thêm vào danh sách tin cậy.</p>
			<p>Bạn có thể quay lại trình duyệt đăng nhập để sử dụng dịch vụ ngay lập tức.</p>
		</div>
	</body>
	</html>
	`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// RejectDeviceVerification từ chối thiết bị lạ và khóa tài khoản khẩn cấp
func (h *Handler) RejectDeviceVerification(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte("<h1>Lỗi: Thiếu token xác thực</h1>"))
		return
	}

	err := h.service.RejectDeviceVerification(token)
	if err != nil {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(fmt.Sprintf("<h1>Xử lý thất bại</h1><p>%s</p>", err.Error())))
		return
	}

	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>NF-Bank - Đã khóa tài khoản</title>
		<meta charset="utf-8">
		<style>
			body { font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; background-color: #fff5f5; margin: 0; }
			.card { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); text-align: center; max-width: 500px; border-top: 5px solid #dc3545; }
			h1 { color: #dc3545; margin-bottom: 20px; }
			p { color: #495057; line-height: 1.6; }
			.hotline { font-size: 20px; font-weight: bold; color: #dc3545; margin-top: 15px; }
		</style>
	</head>
	<body>
		<div class="card">
			<h1>⚠ Đã khóa tài khoản khẩn cấp</h1>
			<p>Chúng tôi đã tạm khóa tài khoản của bạn và đăng xuất toàn bộ các thiết bị đang online để bảo vệ tài sản của bạn.</p>
			<p>Vui lòng liên hệ Hotline hỗ trợ khẩn cấp để xác minh và mở khóa tài khoản:</p>
			<div class="hotline">1900-XXXX</div>
		</div>
	</body>
	</html>
	`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
