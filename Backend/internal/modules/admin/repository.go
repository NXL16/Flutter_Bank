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
	return r.db.
		Model(&auth.User{}).
		Where("id = ?", userID).
		Update("is_locked", true).Error
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

func (r *Repository) FindUserByEmailOrPhone(email string, phone string) (*auth.User, error) {
	var user auth.User
	err := r.db.Where("email = ? OR phone = ?", email, phone).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
