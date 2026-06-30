package auth

type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	IDToken  string `json:"id_token" binding:"required"`
}

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code"`
}

type AuthResponse struct {
	AccessToken     string       `json:"access_token,omitempty"`
	RefreshToken    string       `json:"-"`
	User            UserResponse `json:"user,omitempty"`
	DeviceID        string       `json:"device_id,omitempty"`
	SMSAuthRequired bool         `json:"sms_auth_required,omitempty"`
	TOTPRequired    bool         `json:"totp_required,omitempty"`
	Phone           string       `json:"phone,omitempty"`
}

type UserResponse struct {
	ID         uint   `json:"id"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	IsVerified bool   `json:"is_verified"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
	IDToken     string `json:"id_token" binding:"required"`
}

type ResetPasswordRequest struct {
	Phone       string `json:"phone" binding:"required"`
	IDToken     string `json:"id_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ConfirmLoginRequest struct {
	Phone   string `json:"phone" binding:"required"`
	IDToken string `json:"id_token" binding:"required"`
}
