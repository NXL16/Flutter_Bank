package auth

import (
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/notification"
	"bank-service/internal/modules/user"
	"time"
)

type User struct {
	ID             uint                        `gorm:"primaryKey" json:"id"`
	FullName       string                      `gorm:"type:varchar(255);not null" json:"full_name"`
	Email          string                      `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash   string                      `gorm:"type:varchar(255);not null" json:"-"`
	Phone          string                      `gorm:"type:varchar(20);uniqueIndex;not null" json:"phone"`
	Role           string                      `gorm:"type:varchar(50);default:user" json:"role"`
	IsVerified     bool                        `gorm:"default:false" json:"is_verified"`
	IsLocked       bool                        `gorm:"default:false" json:"is_locked"`
	TOTPSecret     string                      `gorm:"type:varchar(255)" json:"-"`
	SessionVersion int                         `gorm:"not null;default:1" json:"-"`
	Accounts       []account.Account           `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"accounts,omitempty"`
	Profile        user.UserProfile            `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"profile,omitempty"`
	Notifications  []notification.Notification `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"notifications,omitempty"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	IsRevoked bool      `gorm:"default:false" json:"is_revoked"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type UserDevice struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	User           User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	DeviceID       string    `gorm:"type:varchar(255);not null;index" json:"device_id"`
	UserAgent      string    `gorm:"type:text;not null" json:"user_agent"`
	LastActiveIP   string    `gorm:"type:varchar(45);not null" json:"last_active_ip"`
	LastLocation   string    `gorm:"type:varchar(255)" json:"last_location"`
	LastLoggedInAt time.Time `gorm:"not null" json:"last_logged_in_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PendingLogin struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)" json:"id"` // Pending UUID
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	DeviceID  string    `gorm:"type:varchar(255);not null" json:"device_id"`
	UserAgent string    `gorm:"type:text;not null" json:"user_agent"`
	IPAddress string    `gorm:"type:varchar(45);not null" json:"ip_address"`
	Location  string    `gorm:"type:varchar(255)" json:"location"`
	Status    string    `gorm:"type:varchar(20);default:'PENDING'" json:"status"` // PENDING, APPROVED, REJECTED
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
