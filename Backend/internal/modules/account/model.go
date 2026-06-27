package account

import (
	"bank-service/internal/modules/credit"
	"time"
)

type Account struct {
	ID            uint                 `gorm:"primaryKey" json:"id"`
	UserID        uint                 `gorm:"not null;uniqueIndex:idx_user_account_type" json:"user_id"`
	AccountNumber string               `gorm:"type:varchar(20);uniqueIndex;not null" json:"account_number"`
	AccountType   string               `gorm:"type:varchar(50);not null;uniqueIndex:idx_user_account_type" json:"account_type"`
	Balance       int64                `gorm:"not null;default:0" json:"balance"`
	Currency      string               `gorm:"type:varchar(10);not null;default:VND" json:"currency"`
	Status        string               `gorm:"type:varchar(50);not null;default:ACTIVE" json:"status"`
	CreditDetail  *credit.CreditDetail `gorm:"foreignKey:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"credit_detail,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}
