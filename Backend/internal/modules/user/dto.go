package user

import "time"

type UserProfileResponse struct {
	FullName    string     `json:"full_name"`
	Phone       string     `json:"phone"`
	Role        string     `json:"role"`
	Address     string     `json:"address"`
	AvatarURL   string     `json:"avatar_url"`
	Gender      string     `json:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth"`
}

type UpdateUserProfileRequest struct {
	Address     string     `json:"address"`
	Gender      string     `json:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth"`
}
