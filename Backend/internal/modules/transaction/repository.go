package transaction

import (
	"errors"

	"bank-service/internal/modules/account"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
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

func (r *Repository) FindAccountByNumberForUpdate(
	tx *gorm.DB,
	accountNumber string,
) (*account.Account, error) {
	var acc account.Account

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
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

func (r *Repository) WithTx(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

func (r *Repository) FindPaymentAccountByUserIDForUpdate(
	tx *gorm.DB,
	userID uint,
) (*account.Account, error) {

	var acc account.Account

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"user_id = ? AND account_type = ?",
			userID,
			"PAYMENT",
		).
		First(&acc).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &acc, nil
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

func (r *Repository) FindTransactionsByAccountID(
	accountID uint,
) ([]Transaction, error) {

	var transactions []Transaction

	err := r.db.
		Where(
			"sender_account_id = ? OR receiver_account_id = ?",
			accountID,
			accountID,
		).
		Order("created_at desc").
		Find(&transactions).Error

	return transactions, err
}

func (r *Repository) FindTransactionByReferenceCode(
	referenceCode string,
) (*Transaction, error) {

	var transaction Transaction

	err := r.db.
		Where("reference_code = ?", referenceCode).
		First(&transaction).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &transaction, nil
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

func (r *Repository) GetUserPhone(userID uint) (string, error) {
	var user struct {
		Phone string
	}
	err := r.db.Table("users").Where("id = ?", userID).First(&user).Error
	return user.Phone, err
}
