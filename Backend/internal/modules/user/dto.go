package user

import "time"

type UserProfileResponse struct {
	UserID      uint       `json:"user_id"`
	FullName    string     `json:"full_name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Role        string     `json:"role"`
	IsVerified  bool       `json:"is_verified"`
	IsLocked    bool       `json:"is_locked"`
	Address     string     `json:"address"`
	AvatarURL   string     `json:"avatar_url"`
	Gender      string     `json:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Age         int        `json:"age"`
}

type UpdateUserProfileRequest struct {
	Address     string     `json:"address"`
	AvatarURL   string     `json:"avatar_url"`
	Gender      string     `json:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth"`
}
