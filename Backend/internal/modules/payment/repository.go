package payment

import (
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/transaction"
	"errors"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) FindMerchantByCodeAndKey(partnerCode, accessKey string) (*Merchant, error) {
	var m Merchant
	err := r.db.Where("partner_code = ? AND access_key = ?", partnerCode, accessKey).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *Repository) CreateSession(session *PaymentSession) error {
	return r.db.Create(session).Error
}

func (r *Repository) FindSessionByToken(token string) (*PaymentSession, error) {
	var s PaymentSession
	err := r.db.Preload("Merchant").Where("payment_token = ?", token).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) FindSessionByID(id uint) (*PaymentSession, error) {
	var s PaymentSession
	err := r.db.Preload("Merchant").First(&s, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) FindMerchantByID(id uint) (*Merchant, error) {
	var m Merchant
	err := r.db.First(&m, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *Repository) FindAccountByID(id uint) (*account.Account, error) {
	var acc account.Account
	err := r.db.First(&acc, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) FindAccountByIDAndUserID(id, userID uint) (*account.Account, error) {
	var acc account.Account
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&acc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) FindAccountByIDForUpdate(tx *gorm.DB, id uint) (*account.Account, error) {
	var acc account.Account
	err := tx.Set("gorm:query_option", "FOR UPDATE").First(&acc, id).Error
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) UpdateAccountBalance(tx *gorm.DB, id uint, balance int64) error {
	return tx.Model(&account.Account{}).Where("id = ?", id).Update("balance", balance).Error
}

func (r *Repository) CreateTransaction(tx *gorm.DB, t *transaction.Transaction) error {
	return tx.Create(t).Error
}

func (r *Repository) UpdateSessionStatus(id uint, status string) error {
	return r.db.Model(&PaymentSession{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) UpdateSessionSuccess(tx *gorm.DB, id uint, userID uint, referenceCode string) error {
	return tx.Model(&PaymentSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         "SUCCESS",
		"user_id":        userID,
		"reference_code": referenceCode,
	}).Error
}

func (r *Repository) GetUserPhone(userID uint) (string, error) {
	var phone string
	err := r.db.Table("users").Select("phone").Where("id = ?", userID).Row().Scan(&phone)
	return phone, err
}

func (r *Repository) WithTx(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}
