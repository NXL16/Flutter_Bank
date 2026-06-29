package notification

import "time"

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Type      string    `gorm:"type:varchar(50);not null;default:'BALANCE_FLUCTUATION'" json:"type"`
	Title     string    `gorm:"type:varchar(255);not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PushToken struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;index" json:"user_id"`
	Token      string    `gorm:"type:varchar(512);not null;uniqueIndex" json:"-"`
	Platform   string    `gorm:"type:varchar(20);not null" json:"platform"`
	LastSeenAt time.Time `gorm:"not null;index" json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
