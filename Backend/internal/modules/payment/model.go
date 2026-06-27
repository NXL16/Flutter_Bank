package payment

import (
	"time"
)

type Merchant struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	PartnerCode      string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"partner_code"`
	AccessKey        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"access_key"`
	SecretKey        string    `gorm:"type:varchar(100);not null" json:"-"`
	MerchantName     string    `gorm:"type:varchar(255);not null" json:"merchant_name"`
	PaymentAccountID uint      `gorm:"not null" json:"payment_account_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PaymentSession struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	MerchantID    uint      `gorm:"not null" json:"merchant_id"`
	Merchant      Merchant  `gorm:"foreignKey:MerchantID" json:"merchant"`
	PaymentToken  string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"payment_token"`
	Amount        int64     `gorm:"not null" json:"amount"`
	OrderID       string    `gorm:"type:varchar(100);not null" json:"order_id"`
	RequestID     string    `gorm:"type:varchar(100);not null" json:"request_id"`
	OrderInfo     string    `gorm:"type:text;not null" json:"order_info"`
	RedirectURL   string    `gorm:"type:text;not null" json:"redirect_url"`
	IpnURL        string    `gorm:"type:text;not null" json:"ipn_url"`
	ExtraData     string    `gorm:"type:text" json:"extra_data"`
	Status        string    `gorm:"type:varchar(50);not null;default:PENDING" json:"status"` // PENDING, SUCCESS, FAILED
	UserID        *uint     `json:"user_id"`
	ReferenceCode *string   `gorm:"type:varchar(100)" json:"reference_code"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}
