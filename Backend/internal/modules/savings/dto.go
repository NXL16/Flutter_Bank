package savings

import "time"

type CreateSavingsRequest struct {
	Amount              int64  `json:"amount" binding:"required,min=5000000"`
	TermMonths          int    `json:"term_months" binding:"required"`
	MaturityInstruction string `json:"maturity_instruction" binding:"required"`
	TransactionPIN      string `json:"transaction_pin" binding:"required,len=6,numeric"`
	IdempotencyKey      string `json:"-"`
}

type SavingsProductResponse struct {
	TermMonths    int     `json:"term_months"`
	InterestRate  float64 `json:"interest_rate"`
	MinimumAmount int64   `json:"minimum_amount"`
}

type SavingsResponse struct {
	AccountNumber       string    `json:"account_number"`
	OriginalPrincipal   int64     `json:"original_principal"`
	InterestRate        float64   `json:"interest_rate"`
	TermMonths          int       `json:"term_months"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
	ExpectedInterest    int64     `json:"expected_interest"`
	MaturityAmount      int64     `json:"maturity_amount"`
	MaturityInstruction string    `json:"maturity_instruction"`
	IsSettled           bool      `json:"is_settled"`
	Status              string    `json:"status"`
}
