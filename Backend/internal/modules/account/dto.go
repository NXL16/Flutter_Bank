package account

type CreateAccountRequest struct {
	AccountType string `json:"account_type" binding:"required"`
	Currency    string `json:"currency" binding:"required"`
}

type AccountResponse struct {
	ID            uint   `json:"id"`
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	Balance       int64  `json:"balance"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
}
