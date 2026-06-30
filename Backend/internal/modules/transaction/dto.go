package transaction

import "time"

type TransferRequest struct {
	ReceiverAccountNumber string `json:"receiver_account_number" binding:"required"`
	Amount                int64  `json:"amount" binding:"required,gt=0"`
	Description           string `json:"description"`
	TransactionPIN        string `json:"transaction_pin" binding:"required"`
	IdempotencyKey        string `json:"-"`
}

type TransactionResponse struct {
	ID                        uint      `json:"id"`
	ReferenceCode             string    `json:"reference_code"`
	SenderAccountID           *uint     `json:"sender_account_id,omitempty"`
	ReceiverAccountID         uint      `json:"receiver_account_id"`
	Amount                    int64     `json:"amount"`
	Currency                  string    `json:"currency"`
	Type                      string    `json:"type"`
	Status                    string    `json:"status"`
	Description               string    `json:"description"`
	Direction                 string    `json:"direction,omitempty"`
	CounterpartyName          string    `json:"counterparty_name,omitempty"`
	CounterpartyAccountNumber string    `json:"counterparty_account_number,omitempty"`
	BalanceAfter              *int64    `json:"balance_after,omitempty"`
	CreatedAt                 time.Time `json:"created_at"`
}

type AccountResolutionResponse struct {
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	AvatarURL     string `json:"avatar_url"`
	BankName      string `json:"bank_name"`
	Currency      string `json:"currency"`
}

type DepositRequest struct {
	ReceiverAccountNumber string `json:"receiver_account_number" binding:"required"`
	Amount                int64  `json:"amount" binding:"required,gt=0"`
	Description           string `json:"description"`
}

type TransactionPINStatusResponse struct {
	HasPIN bool `json:"has_pin"`
}

type SetupTransactionPINRequest struct {
	PIN        string `json:"pin" binding:"required"`
	ConfirmPIN string `json:"confirm_pin" binding:"required"`
}
