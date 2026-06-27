package savings

import "time"

type CreateSavingsRequest struct {
	Amount int64 `json:"amount" binding:"required,min=5000000"`
}

type SavingsResponse struct {
	ID                uint      `json:"id"`
	AccountNumber     string    `json:"account_number"`
	OriginalPrincipal int64     `json:"original_principal"`
	InterestRate      float64   `json:"interest_rate"`
	TermMonths        int       `json:"term_months"`
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
	IsSettled         bool      `json:"is_settled"`
	Status            string    `json:"status"`
}
