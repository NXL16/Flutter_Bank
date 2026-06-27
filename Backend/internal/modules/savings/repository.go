package savings

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
