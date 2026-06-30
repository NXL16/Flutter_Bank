package account

import (
	"time"
)

type Account struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"not null;index:idx_accounts_user_type" json:"user_id"`
	AccountNumber string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"account_number"`
	AccountType   string    `gorm:"type:varchar(50);not null;index:idx_accounts_user_type" json:"account_type"`
	Balance       int64     `gorm:"not null;default:0" json:"balance"`
	Currency      string    `gorm:"type:varchar(10);not null;default:VND" json:"currency"`
	Status        string    `gorm:"type:varchar(50);not null;default:ACTIVE" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
