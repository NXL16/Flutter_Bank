package admin

import (
	"bank-service/internal/modules/auth"

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

func (r *Repository) FindAllUsers() ([]auth.User, error) {
	var users []auth.User

	err := r.db.Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repository) FindUserByID(
	userID uint,
) (*auth.User, error) {

	var user auth.User

	err := r.db.First(&user, userID).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) LockUser(userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&auth.User{}).
			Where("id = ?", userID).
			Updates(map[string]interface{}{
				"is_locked":       true,
				"session_version": gorm.Expr("session_version + 1"),
			}).Error; err != nil {
			return err
		}
		return tx.Model(&auth.RefreshToken{}).
			Where("user_id = ? AND is_revoked = ?", userID, false).
			Update("is_revoked", true).Error
	})
}

func (r *Repository) UnlockUser(userID uint) error {
	return r.db.
		Model(&auth.User{}).
		Where("id = ?", userID).
		Update("is_locked", false).Error
}

func (r *Repository) CreateAdminUser(user *auth.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) FindUserByPhone(phone string) (*auth.User, error) {
	var user auth.User
	err := r.db.Where("phone = ?", phone).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
