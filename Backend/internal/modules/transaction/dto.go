package transaction

type TransferRequest struct {
	ReceiverAccountNumber string `json:"receiver_account_number" binding:"required"`
	Amount                int64  `json:"amount" binding:"required,gt=0"`
	Description           string `json:"description"`
	IDToken               string `json:"id_token"`
}

type TransactionResponse struct {
	ID                uint   `json:"id"`
	ReferenceCode     string `json:"reference_code"`
	SenderAccountID   *uint  `json:"sender_account_id,omitempty"`
	ReceiverAccountID uint   `json:"receiver_account_id"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
	Type              string `json:"type"`
	Status            string `json:"status"`
	Description       string `json:"description"`
}

type DepositRequest struct {
	ReceiverAccountNumber string `json:"receiver_account_number" binding:"required"`
	Amount                int64  `json:"amount" binding:"required,gt=0"`
	Description           string `json:"description"`
}
