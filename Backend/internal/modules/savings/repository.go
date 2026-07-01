package savings

import (
	"errors"
	"time"

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

// CreateSavingsDetail lưu thông tin chi tiết sổ tiết kiệm, hỗ trợ transaction
func (r *Repository) CreateSavingsDetail(tx *gorm.DB, detail *SavingsDetail) error {
	dbConn := r.db
	if tx != nil {
		dbConn = tx
	}
	return dbConn.Create(detail).Error
}

// FindSavingsDetailByAccountID lấy thông tin chi tiết sổ tiết kiệm theo AccountID
func (r *Repository) FindSavingsDetailByAccountID(accountID uint) (*SavingsDetail, error) {
	var detail SavingsDetail
	err := r.db.Where("account_id = ?", accountID).First(&detail).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &detail, nil
}

func (r *Repository) FindSavingsByUserID(userID uint) ([]SavingsDetail, error) {
	var details []SavingsDetail
	err := r.db.
		Joins("JOIN accounts ON accounts.id = savings_details.account_id").
		Preload("Account").
		Where("accounts.user_id = ? AND accounts.account_type = ?", userID, "SAVINGS").
		Order("savings_details.created_at DESC").
		Find(&details).Error
	return details, err
}

func (r *Repository) FindSavingsByAccountNumber(
	userID uint,
	accountNumber string,
) (*SavingsDetail, error) {
	var detail SavingsDetail
	err := r.db.
		Joins("JOIN accounts ON accounts.id = savings_details.account_id").
		Preload("Account").
		Where(
			"accounts.user_id = ? AND accounts.account_number = ? AND accounts.account_type = ?",
			userID,
			accountNumber,
			"SAVINGS",
		).
		Take(&detail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func (r *Repository) FindMaturityEventsByAccountIDs(
	accountIDs []uint,
) ([]SavingsMaturityEvent, error) {
	if len(accountIDs) == 0 {
		return []SavingsMaturityEvent{}, nil
	}
	var events []SavingsMaturityEvent
	err := r.db.
		Where("account_id IN ?", accountIDs).
		Order("account_id ASC, cycle_number DESC").
		Find(&events).Error
	return events, err
}

func (r *Repository) FindWithdrawalsByAccountIDs(
	accountIDs []uint,
) ([]SavingsWithdrawal, error) {
	if len(accountIDs) == 0 {
		return []SavingsWithdrawal{}, nil
	}
	var withdrawals []SavingsWithdrawal
	err := r.db.
		Where("account_id IN ?", accountIDs).
		Order("account_id ASC, processed_at DESC").
		Find(&withdrawals).Error
	return withdrawals, err
}

func (r *Repository) FindWithdrawalByIdempotencyKey(
	userID uint,
	accountID uint,
	key string,
) (*SavingsWithdrawal, error) {
	var withdrawal SavingsWithdrawal
	err := r.db.
		Where(
			"user_id = ? AND account_id = ? AND idempotency_key = ?",
			userID,
			accountID,
			key,
		).
		Take(&withdrawal).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

func (r *Repository) FindDueSavingsAccountIDs(
	now time.Time,
	limit int,
) ([]uint, error) {
	var accountIDs []uint
	err := r.db.Model(&SavingsDetail{}).
		Select("account_id").
		Where("is_settled = ? AND end_date <= ?", false, now).
		Order("end_date ASC").
		Limit(limit).
		Scan(&accountIDs).Error
	return accountIDs, err
}

func (r *Repository) FindSavingsByIdempotencyKey(
	userID uint,
	key string,
) (*SavingsDetail, error) {
	var detail SavingsDetail
	err := r.db.
		Joins(
			"JOIN transactions ON transactions.receiver_account_id = savings_details.account_id",
		).
		Preload("Account").
		Where(
			"transactions.initiator_user_id = ? AND transactions.idempotency_key = ?",
			userID,
			key,
		).
		Take(&detail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &detail, nil
}
