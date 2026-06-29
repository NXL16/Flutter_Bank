package transaction

import (
	"bank-service/internal/modules/account"
	"time"
)

type Transaction struct {
	ID                uint             `gorm:"primaryKey" json:"id"`
	ReferenceCode     string           `gorm:"type:varchar(50);uniqueIndex;not null" json:"reference_code"`
	InitiatorUserID   *uint            `gorm:"uniqueIndex:idx_transaction_idempotency" json:"-"`
	IdempotencyKey    *string          `gorm:"type:varchar(64);uniqueIndex:idx_transaction_idempotency" json:"-"`
	SenderAccountID   *uint            `gorm:"index" json:"sender_account_id"`
	SenderAccount     *account.Account `gorm:"foreignKey:SenderAccountID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"sender_account,omitempty"`
	ReceiverAccountID uint             `gorm:"not null;index" json:"receiver_account_id"`
	ReceiverAccount   account.Account  `gorm:"foreignKey:ReceiverAccountID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"receiver_account"`
	Amount            int64            `gorm:"not null" json:"amount"`
	Currency          string           `gorm:"type:varchar(10);not null" json:"currency"`
	Type              string           `gorm:"type:varchar(50);not null" json:"type"`
	Status            string           `gorm:"type:varchar(50);not null" json:"status"`
	Description       string           `gorm:"type:varchar(255)" json:"description"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// LedgerEntry là bút toán bất biến. Balance trên Account chỉ là số dư vật chất
// được cập nhật cùng transaction để đọc nhanh; ledger mới là nguồn kiểm toán.
type LedgerEntry struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `gorm:"not null;index;uniqueIndex:idx_ledger_leg" json:"transaction_id"`
	AccountID     uint      `gorm:"not null;index;uniqueIndex:idx_ledger_leg" json:"account_id"`
	Direction     string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_ledger_leg" json:"direction"`
	Amount        int64     `gorm:"not null" json:"amount"`
	Currency      string    `gorm:"type:varchar(10);not null" json:"currency"`
	BalanceAfter  int64     `gorm:"not null" json:"balance_after"`
	CreatedAt     time.Time `json:"created_at"`
}
