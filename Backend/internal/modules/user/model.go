package user

import "time"

type UserProfile struct {
	UserID      uint       `gorm:"primaryKey" json:"user_id"`
	Address     string     `gorm:"type:varchar(500)" json:"address"`
	AvatarURL   string     `gorm:"type:varchar(500)" json:"avatar_url"`
	Gender      string     `gorm:"type:varchar(20)" json:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
