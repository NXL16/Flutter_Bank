package auth

import (
	"net/http"

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

	response.Success(c, http.StatusCreated, "Đăng ký tài khoản thành công", nil)
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
		setDeviceCookie(c, res.DeviceID, h.service.cfg.ServerMode == "production")
		setRefreshTokenCookie(c, res.RefreshToken, h.service.cfg.ServerMode == "production")

		response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
		return
	}

	response.Success(c, http.StatusOK, "Mật khẩu đúng, tiến hành xác thực số điện thoại", gin.H{
		"sms_auth_required": true,
		"phone":             res.Phone,
	})
}

// setRefreshTokenCookie thiết lập refresh token trong cookie
func setRefreshTokenCookie(c *gin.Context, refreshToken string, secure bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		refreshToken,
		7*24*60*60, // 7 ngày
		"/api/v1/auth",
		"",
		secure,
		true,
	)
}

func setDeviceCookie(c *gin.Context, deviceID string, secure bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"device_id",
		deviceID,
		365*24*60*60,
		"/api/v1/auth",
		"",
		secure,
		true,
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
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/v1/auth",
		"",
		h.service.cfg.ServerMode == "production",
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

	setDeviceCookie(c, res.DeviceID, h.service.cfg.ServerMode == "production")
	setRefreshTokenCookie(c, res.RefreshToken, h.service.cfg.ServerMode == "production")

	response.Success(c, http.StatusOK, "Đăng nhập thành công", res)
}
