package auth

type RegisterRequest struct {
	Email      string `json:"email" binding:"required,email"`
	FullName   string `json:"full_name" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Password   string `json:"password" binding:"required,min=8"`
	OTPChannel string `json:"otp_channel" binding:"required,oneof=email sms"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken         string       `json:"access_token,omitempty"`
	RefreshToken        string       `json:"-"`
	User                UserResponse `json:"user,omitempty"`
	PendingVerification bool         `json:"pending_verification,omitempty"`
	PendingID           string       `json:"pending_id,omitempty"`
	DeviceID            string       `json:"device_id,omitempty"`
	SMSAuthRequired     bool         `json:"sms_auth_required,omitempty"`
	TOTPRequired        bool         `json:"totp_required,omitempty"`
	Phone               string       `json:"phone,omitempty"`
}

type UserResponse struct {
	ID         uint   `json:"id"`
	FullName   string `json:"full_name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	IsVerified bool   `json:"is_verified"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ConfirmRegisterRequest struct {
	Email   string `json:"email" binding:"required"`
	OTP     string `json:"otp"`
	IDToken string `json:"id_token"`
}

type ConfirmLoginRequest struct {
	Email   string `json:"email" binding:"required"`
	OTP     string `json:"otp"`
	IDToken string `json:"id_token"`
}
