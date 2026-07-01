package savings

import "time"

type CreateSavingsRequest struct {
	Amount              int64  `json:"amount" binding:"required,min=5000000"`
	TermMonths          int    `json:"term_months" binding:"required"`
	MaturityInstruction string `json:"maturity_instruction" binding:"required"`
	TransactionPIN      string `json:"transaction_pin" binding:"required,len=6,numeric"`
	IdempotencyKey      string `json:"-"`
}

type EarlyWithdrawalRequest struct {
	Amount         int64  `json:"amount" binding:"required,gt=0"`
	TransactionPIN string `json:"transaction_pin" binding:"required,len=6,numeric"`
	IdempotencyKey string `json:"-"`
}

type EarlyWithdrawalResponse struct {
	AccountNumber      string    `json:"account_number"`
	WithdrawnPrincipal int64     `json:"withdrawn_principal"`
	DemandInterestRate float64   `json:"demand_interest_rate"`
	AccruedDays        int       `json:"accrued_days"`
	Interest           int64     `json:"interest"`
	TotalReceived      int64     `json:"total_received"`
	RemainingPrincipal int64     `json:"remaining_principal"`
	IsFullWithdrawal   bool      `json:"is_full_withdrawal"`
	ProcessedAt        time.Time `json:"processed_at"`
}

type SavingsProductResponse struct {
	TermMonths         int     `json:"term_months"`
	InterestRate       float64 `json:"interest_rate"`
	MinimumAmount      int64   `json:"minimum_amount"`
	DemandInterestRate float64 `json:"demand_interest_rate"`
}

type SavingsResponse struct {
	AccountNumber       string                         `json:"account_number"`
	OriginalPrincipal   int64                          `json:"original_principal"`
	InterestRate        float64                        `json:"interest_rate"`
	TermMonths          int                            `json:"term_months"`
	StartDate           time.Time                      `json:"start_date"`
	EndDate             time.Time                      `json:"end_date"`
	ExpectedInterest    int64                          `json:"expected_interest"`
	MaturityAmount      int64                          `json:"maturity_amount"`
	MaturityInstruction string                         `json:"maturity_instruction"`
	IsSettled           bool                           `json:"is_settled"`
	Status              string                         `json:"status"`
	RenewalCount        int                            `json:"renewal_count"`
	LastMaturedAt       *time.Time                     `json:"last_matured_at,omitempty"`
	MaturityHistory     []SavingsMaturityEventResponse `json:"maturity_history"`
	ClosedAt            *time.Time                     `json:"closed_at,omitempty"`
	ClosureReason       string                         `json:"closure_reason,omitempty"`
	WithdrawalHistory   []SavingsWithdrawalResponse    `json:"withdrawal_history"`
	DemandInterestRate  float64                        `json:"demand_interest_rate"`
}

type SavingsWithdrawalResponse struct {
	Amount             int64     `json:"amount"`
	DemandInterestRate float64   `json:"demand_interest_rate"`
	AccruedDays        int       `json:"accrued_days"`
	Interest           int64     `json:"interest"`
	RemainingPrincipal int64     `json:"remaining_principal"`
	IsFullWithdrawal   bool      `json:"is_full_withdrawal"`
	ProcessedAt        time.Time `json:"processed_at"`
}

type SavingsMaturityEventResponse struct {
	CycleNumber  int       `json:"cycle_number"`
	Principal    int64     `json:"principal"`
	Interest     int64     `json:"interest"`
	InterestRate float64   `json:"interest_rate"`
	TermMonths   int       `json:"term_months"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	Instruction  string    `json:"instruction"`
	Outcome      string    `json:"outcome"`
	ProcessedAt  time.Time `json:"processed_at"`
}
