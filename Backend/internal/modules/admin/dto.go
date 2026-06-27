package admin

import "time"

type AdminUserResponse struct {
	ID             uint      `json:"id"`
	FullName       string    `json:"full_name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	Role           string    `json:"role"`
	IsVerified     bool      `json:"is_verified"`
	IsLocked       bool      `json:"is_locked"`
	SessionVersion int       `json:"session_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateAdminRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type CreateAdminResponse struct {
	ID         uint      `json:"id"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Role       string    `json:"role"`
	TOTPSecret string    `json:"totp_secret"`
	CreatedAt  time.Time `json:"created_at"`
}
