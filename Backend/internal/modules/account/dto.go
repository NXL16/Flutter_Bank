package account

type AccountResponse struct {
	ID            uint   `json:"id"`
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	Balance       int64  `json:"balance"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
}
