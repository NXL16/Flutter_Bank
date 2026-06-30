package admin

import "time"

type AdminUserResponse struct {
	ID         uint   `json:"id"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	IsVerified bool   `json:"is_verified"`
	IsLocked   bool   `json:"is_locked"`
}

type CreateAdminRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type CreateAdminResponse struct {
	ID         uint      `json:"id"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone"`
	Role       string    `json:"role"`
	TOTPSecret string    `json:"totp_secret"`
	CreatedAt  time.Time `json:"created_at"`
}
