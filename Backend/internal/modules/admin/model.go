package admin

import "time"

// AuditLog lưu vết bất biến cho các thao tác quản trị nhạy cảm.
type AuditLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ActorUserID uint      `gorm:"not null;index" json:"actor_user_id"`
	Action      string    `gorm:"type:varchar(50);not null;index" json:"action"`
	TargetType  string    `gorm:"type:varchar(50);not null" json:"target_type"`
	TargetID    string    `gorm:"type:varchar(100);not null" json:"target_id"`
	Summary     string    `gorm:"type:varchar(500);not null" json:"summary"`
	IPAddress   string    `gorm:"type:varchar(45)" json:"ip_address"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

type StepUpChallenge struct {
	ID           uint       `gorm:"primaryKey" json:"-"`
	ActorUserID  uint       `gorm:"not null;index;uniqueIndex:idx_admin_totp_once" json:"-"`
	Action       string     `gorm:"type:varchar(50);not null;index" json:"-"`
	TokenHash    string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	BindingHash  string     `gorm:"type:char(64);not null;index" json:"-"`
	TOTPTimeStep int64      `gorm:"not null;uniqueIndex:idx_admin_totp_once" json:"-"`
	ExpiresAt    time.Time  `gorm:"not null;index" json:"-"`
	UsedAt       *time.Time `gorm:"index" json:"-"`
	CreatedAt    time.Time  `json:"-"`
}
