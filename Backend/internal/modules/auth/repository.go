package auth

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Repository chịu trách nhiệm thao tác database cho auth
type Repository struct {
	db *gorm.DB
}

// NewRepository tạo auth repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

// CreateUser tạo user mới trong database
func (r *Repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

// FindUserByEmail tìm user theo email
func (r *Repository) FindUserByEmail(email string) (*User, error) {
	var user User

	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

// FindUserByID tìm user theo ID
func (r *Repository) FindUserByID(id uint) (*User, error) {
	var user User

	err := r.db.First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

// UpdatePassword cập nhật mật khẩu mới cho user
func (r *Repository) UpdatePassword(userID uint, passwordHash string) error {
	return r.db.Model(&User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash).Error
}

func (r *Repository) FindUserByPhone(phone string) (*User, error) {
	var user User

	err := r.db.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *Repository) IncreaseSessionVersion(userID uint) error {
	return r.db.Model(&User{}).
		Where("id = ?", userID).
		Update("session_version", gorm.Expr("session_version + ?", 1)).
		Error
}

func (r *Repository) FindSessionVersionByUserID(
	userID uint,
) (int, error) {

	var user User

	err := r.db.
		Select("session_version").
		First(&user, userID).Error

	if err != nil {
		return 0, err
	}

	return user.SessionVersion, nil
}

func (r *Repository) CreateRefreshToken(
	refreshToken *RefreshToken,
) error {
	return r.db.Create(refreshToken).Error
}

func (r *Repository) HasActiveSession(
	userID uint,
) (bool, error) {

	var count int64

	err := r.db.
		Model(&RefreshToken{}).
		Where(
			"user_id = ? AND is_revoked = ? AND expires_at > ?",
			userID,
			false,
			time.Now(),
		).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repository) RevokeAllUserRefreshTokens(
	userID uint,
) error {

	return r.db.
		Model(&RefreshToken{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Update("is_revoked", true).Error
}

func (r *Repository) RevokeRefreshToken(
	tokenHash string,
) error {

	return r.db.
		Model(&RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("is_revoked", true).Error
}

func (r *Repository) GetActiveSessions(userID uint) ([]RefreshToken, error) {
	var tokens []RefreshToken
	err := r.db.
		Where("user_id = ? AND is_revoked = ? AND expires_at > ?", userID, false, time.Now()).
		Order("created_at asc").
		Find(&tokens).Error
	return tokens, err
}

func (r *Repository) RevokeRefreshTokenByID(id uint) error {
	return r.db.
		Model(&RefreshToken{}).
		Where("id = ?", id).
		Update("is_revoked", true).Error
}

func (r *Repository) FindUserDevice(userID uint, deviceID string) (*UserDevice, error) {
	var dev UserDevice
	err := r.db.
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&dev).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dev, nil
}

func (r *Repository) CreateUserDevice(device *UserDevice) error {
	return r.db.Create(device).Error
}

func (r *Repository) UpdateUserDeviceLastLogin(id uint, ip string, location string) error {
	return r.db.
		Model(&UserDevice{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_active_ip":    ip,
			"last_location":     location,
			"last_logged_in_at": time.Now(),
		}).Error
}

func (r *Repository) DeleteStaleUserDevices(userID uint, beforeTime time.Time) error {
	return r.db.
		Where("user_id = ? AND last_logged_in_at < ?", userID, beforeTime).
		Delete(&UserDevice{}).Error
}

func (r *Repository) CreatePendingLogin(pending *PendingLogin) error {
	return r.db.Create(pending).Error
}

func (r *Repository) FindPendingLogin(id string) (*PendingLogin, error) {
	var pending PendingLogin
	err := r.db.
		Where("id = ?", id).
		First(&pending).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pending, nil
}

func (r *Repository) UpdatePendingLoginStatus(id string, status string) error {
	return r.db.
		Model(&PendingLogin{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *Repository) LockUser(userID uint) error {
	return r.db.
		Model(&User{}).
		Where("id = ?", userID).
		Update("is_locked", true).Error
}

func (r *Repository) DeletePendingLogin(id string) error {
	return r.db.
		Where("id = ?", id).
		Delete(&PendingLogin{}).Error
}

func (r *Repository) CountUserDevices(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&UserDevice{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *Repository) DeleteUserDeviceByID(id uint) error {
	return r.db.Delete(&UserDevice{}, id).Error
}