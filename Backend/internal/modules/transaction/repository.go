package transaction

import (
	"errors"
	"time"

	"bank-service/internal/modules/account"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

type transactionView struct {
	ID                        uint
	ReferenceCode             string
	SenderAccountID           *uint
	ReceiverAccountID         uint
	Amount                    int64
	Currency                  string
	Type                      string
	Status                    string
	Description               string
	Direction                 string
	CounterpartyName          string
	CounterpartyAccountNumber string
	BalanceAfter              *int64
	CreatedAt                 time.Time
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) FindAccountByIDForUpdate(
	tx *gorm.DB,
	accountID uint,
) (*account.Account, error) {
	var acc account.Account

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&acc, accountID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &acc, nil
}

func (r *Repository) UpdateAccountBalance(
	tx *gorm.DB,
	accountID uint,
	newBalance int64,
) error {
	return tx.
		Model(&account.Account{}).
		Where("id = ?", accountID).
		Update("balance", newBalance).Error
}

func (r *Repository) CreateTransaction(
	tx *gorm.DB,
	transaction *Transaction,
) error {
	return tx.Create(transaction).Error
}

func (r *Repository) FindTransactionByIdempotencyKey(
	userID uint,
	key string,
) (*Transaction, error) {
	var transaction Transaction
	err := r.db.
		Where("initiator_user_id = ? AND idempotency_key = ?", userID, key).
		First(&transaction).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Repository) SumSuccessfulOutgoingTransfers(
	tx *gorm.DB,
	accountID uint,
	from time.Time,
	to time.Time,
) (int64, error) {
	var total int64
	err := tx.Model(&Transaction{}).
		Select("COALESCE(SUM(amount), 0)").
		Where(
			"sender_account_id = ? AND type = ? AND status = ? AND created_at >= ? AND created_at < ?",
			accountID,
			"TRANSFER",
			"SUCCESS",
			from,
			to,
		).
		Scan(&total).Error
	return total, err
}

func (r *Repository) ResolveActivePaymentAccount(
	accountNumber string,
) (*AccountResolutionResponse, error) {
	var result AccountResolutionResponse
	err := r.db.Table("accounts AS a").
		Select(
			"a.account_number, u.full_name AS account_name, "+
				"COALESCE(up.avatar_url, '') AS avatar_url, a.currency",
		).
		Joins("JOIN users AS u ON u.id = a.user_id").
		Joins("LEFT JOIN user_profiles AS up ON up.user_id = u.id").
		Where(
			"a.account_number = ? AND a.account_type = ? AND a.status = ? AND u.is_locked = ?",
			accountNumber,
			"PAYMENT",
			"ACTIVE",
			false,
		).
		Take(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	result.BankName = "NF Bank"
	return &result, nil
}

func (r *Repository) WithTx(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

func (r *Repository) FindPaymentAccountByUserID(
	userID uint,
) (*account.Account, error) {

	var acc account.Account

	err := r.db.
		Where("user_id = ? AND account_type = ?", userID, "PAYMENT").
		First(&acc).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &acc, nil
}

func (r *Repository) FindTransactionViewsByAccountID(
	accountID uint,
) ([]transactionView, error) {
	var rows []transactionView
	err := r.transactionViewQuery(accountID).
		Order("t.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) FindTransactionViewByReferenceCode(
	accountID uint,
	referenceCode string,
) (*transactionView, error) {
	var row transactionView
	err := r.transactionViewQuery(accountID).
		Where("t.reference_code = ?", referenceCode).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *Repository) transactionViewQuery(accountID uint) *gorm.DB {
	var ownerUserID uint
	_ = r.db.Table("accounts").Select("user_id").Where("id = ?", accountID).Row().Scan(&ownerUserID)

	return r.db.Table("transactions AS t").
		Select(`
			t.id,
			t.reference_code,
			t.sender_account_id,
			t.receiver_account_id,
			t.amount,
			t.currency,
			t.type,
			t.status,
			t.description,
			t.created_at,
			CASE 
				WHEN t.sender_account_id = ? OR (t.initiator_user_id = ? AND t.type = 'DEPOSIT') THEN 'OUT' 
				ELSE 'IN' 
			END AS direction,
			CASE 
				WHEN t.sender_account_id = ? OR (t.initiator_user_id = ? AND t.type = 'DEPOSIT') THEN receiver_user.full_name 
				ELSE sender_user.full_name 
			END AS counterparty_name,
			CASE 
				WHEN t.sender_account_id = ? OR (t.initiator_user_id = ? AND t.type = 'DEPOSIT') THEN receiver_account.account_number 
				ELSE sender_account.account_number 
			END AS counterparty_account_number,
			ledger.balance_after AS balance_after
		`, accountID, ownerUserID, accountID, ownerUserID, accountID, ownerUserID).
		Joins("LEFT JOIN accounts AS sender_account ON sender_account.id = t.sender_account_id").
		Joins("JOIN accounts AS receiver_account ON receiver_account.id = t.receiver_account_id").
		Joins("LEFT JOIN users AS sender_user ON sender_user.id = sender_account.user_id").
		Joins("JOIN users AS receiver_user ON receiver_user.id = receiver_account.user_id").
		Joins("LEFT JOIN ledger_entries AS ledger ON ledger.transaction_id = t.id AND ledger.account_id = ?", accountID).
		Where("t.sender_account_id = ? OR t.receiver_account_id = ? OR (t.initiator_user_id = ? AND t.type = 'DEPOSIT')", accountID, accountID, ownerUserID)
}

func (r *Repository) FindAccountByNumber(
	accountNumber string,
) (*account.Account, error) {

	var acc account.Account

	err := r.db.
		Where("account_number = ?", accountNumber).
		First(&acc).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &acc, nil
}

func (r *Repository) GetUserFullName(userID uint) (string, error) {
	var user struct {
		FullName string
	}
	err := r.db.Table("users").Where("id = ?", userID).First(&user).Error
	return user.FullName, err
}

func (r *Repository) GetUserRole(userID uint) (string, error) {
	var role string
	err := r.db.Table("users").
		Select("role").
		Where("id = ?", userID).
		Scan(&role).Error
	return role, err
}
