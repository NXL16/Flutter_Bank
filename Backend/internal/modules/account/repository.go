package account

import (
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

func (r *Repository) CreateAccount(account *Account) error {
	return r.db.Create(account).Error
}

func (r *Repository) FindAccountsByUserID(userID uint) ([]Account, error) {
	var accounts []Account

	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at asc").
		Find(&accounts).Error

	return accounts, err
}

func (r *Repository) FindAccountByID(accountID uint) (*Account, error) {
	var account Account

	err := r.db.First(&account, accountID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

func (r *Repository) FindAccountByNumber(accountNumber string) (*Account, error) {
	var account Account

	err := r.db.Where("account_number = ?", accountNumber).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

func (r *Repository) FindByUserIDAndType(
	userID uint,
	accountType string,
) (*Account, error) {

	var account Account

	err := r.db.
		Where("user_id = ? AND account_type = ?", userID, accountType).
		First(&account).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &account, nil
}

func (r *Repository) FindUserRoleByID(userID uint) (string, error) {
	var role string
	err := r.db.Table("users").
		Select("role").
		Where("id = ?", userID).
		Row().
		Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

