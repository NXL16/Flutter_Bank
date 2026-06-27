package credit

import "time"

type CreditDetail struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	AccountID          uint      `gorm:"not null;uniqueIndex" json:"account_id"`
	CreditLimit        int64     `gorm:"not null;default:50000000" json:"credit_limit"`
	OutstandingBalance int64     `gorm:"not null;default:0" json:"outstanding_balance"`
	InterestRate       float64   `gorm:"not null;type:decimal(5,2);default:18.00" json:"interest_rate"`
	DueDate            time.Time `gorm:"not null" json:"due_date"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
