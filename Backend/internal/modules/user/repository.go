package user

import (
	"errors"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

type BasicUserInfo struct {
	ID         uint
	FullName   string
	Email      string
	Phone      string
	Role       string
	IsVerified bool
	IsLocked   bool
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateProfile(
	profile *UserProfile,
) error {
	return r.db.Create(profile).Error
}

func (r *Repository) FindByUserID(
	userID uint,
) (*UserProfile, error) {

	var profile UserProfile

	err := r.db.
		Where("user_id = ?", userID).
		First(&profile).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &profile, nil
}

func (r *Repository) UpdateProfile(
	userID uint,
	profile *UserProfile,
) error {

	return r.db.
		Model(&UserProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"address":       profile.Address,
			"avatar_url":    profile.AvatarURL,
			"gender":        profile.Gender,
			"date_of_birth": profile.DateOfBirth,
		}).Error
}

func (r *Repository) FindBasicUserInfo(userID uint) (*BasicUserInfo, error) {
	var user BasicUserInfo

	err := r.db.
		Table("users").
		Select("id, full_name, email, phone, role, is_verified, is_locked").
		Where("id = ?", userID).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}
