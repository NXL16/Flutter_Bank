package savings

import (
	"bank-service/internal/modules/account"
	"time"
)

type SavingsDetail struct {
	AccountID           uint            `gorm:"primaryKey" json:"account_id"`
	Account             account.Account `gorm:"foreignKey:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"account,omitempty"`
	InterestRate        float64         `gorm:"not null;type:decimal(5,2);default:5.50" json:"interest_rate"`
	TermMonths          int             `gorm:"not null;default:12" json:"term_months"`
	StartDate           time.Time       `gorm:"not null" json:"start_date"`
	EndDate             time.Time       `gorm:"not null" json:"end_date"`
	OriginalPrincipal   int64           `gorm:"not null" json:"original_principal"`
	MaturityInstruction string          `gorm:"type:varchar(30);not null;default:PAYOUT" json:"maturity_instruction"`
	IsSettled           bool            `gorm:"default:false" json:"is_settled"`
	RenewalCount        int             `gorm:"not null;default:0" json:"renewal_count"`
	LastMaturedAt       *time.Time      `json:"last_matured_at,omitempty"`
	ClosedAt            *time.Time      `json:"closed_at,omitempty"`
	ClosureReason       string          `gorm:"type:varchar(30)" json:"closure_reason,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// SavingsWithdrawal lưu lịch sử rút trước hạn, bao gồm cả rút một phần.
type SavingsWithdrawal struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	AccountID              uint      `gorm:"not null;index" json:"account_id"`
	UserID                 uint      `gorm:"not null;index" json:"user_id"`
	IdempotencyKey         string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"`
	Amount                 int64     `gorm:"not null" json:"amount"`
	DemandInterestRate     float64   `gorm:"not null;type:decimal(5,2)" json:"demand_interest_rate"`
	AccruedDays            int       `gorm:"not null" json:"accrued_days"`
	Interest               int64     `gorm:"not null" json:"interest"`
	RemainingPrincipal     int64     `gorm:"not null" json:"remaining_principal"`
	IsFullWithdrawal       bool      `gorm:"not null" json:"is_full_withdrawal"`
	PrincipalTransactionID uint      `gorm:"not null;index" json:"principal_transaction_id"`
	InterestTransactionID  *uint     `gorm:"index" json:"interest_transaction_id,omitempty"`
	ProcessedAt            time.Time `gorm:"not null" json:"processed_at"`
	CreatedAt              time.Time `json:"created_at"`
}

// SavingsMaturityEvent là lịch sử bất biến của từng kỳ đáo hạn.
// Unique(AccountID, CycleNumber) là lớp bảo vệ cuối cùng chống xử lý hai lần.
type SavingsMaturityEvent struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	AccountID              uint      `gorm:"not null;uniqueIndex:idx_savings_maturity_cycle" json:"account_id"`
	CycleNumber            int       `gorm:"not null;uniqueIndex:idx_savings_maturity_cycle" json:"cycle_number"`
	Principal              int64     `gorm:"not null" json:"principal"`
	Interest               int64     `gorm:"not null" json:"interest"`
	InterestRate           float64   `gorm:"not null;type:decimal(5,2)" json:"interest_rate"`
	TermMonths             int       `gorm:"not null" json:"term_months"`
	PeriodStart            time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd              time.Time `gorm:"not null" json:"period_end"`
	Instruction            string    `gorm:"type:varchar(30);not null" json:"instruction"`
	Outcome                string    `gorm:"type:varchar(30);not null" json:"outcome"`
	InterestTransactionID  uint      `gorm:"not null;index" json:"interest_transaction_id"`
	PrincipalTransactionID *uint     `gorm:"index" json:"principal_transaction_id,omitempty"`
	ProcessedAt            time.Time `gorm:"not null" json:"processed_at"`
	CreatedAt              time.Time `json:"created_at"`
}
