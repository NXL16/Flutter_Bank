package notification

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

// CreateNotification lưu thông tin thông báo, hỗ trợ luồng transaction dùng chung (tx)
func (r *Repository) CreateNotification(tx *gorm.DB, noti *Notification) error {
	dbConn := r.db
	if tx != nil {
		dbConn = tx
	}
	return dbConn.Create(noti).Error
}

// FindNotificationsByUserID lấy toàn bộ thông báo của 1 user, sắp xếp mới nhất lên đầu
func (r *Repository) FindNotificationsByUserID(userID uint) ([]Notification, error) {
	var notifications []Notification
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&notifications).Error
	return notifications, err
}

// MarkAsRead đánh dấu đã đọc 1 thông báo của 1 user nhất định
func (r *Repository) MarkAsRead(id uint, userID uint) error {
	res := r.db.
		Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("không tìm thấy thông báo hoặc thông báo không thuộc quyền sở hữu của bạn")
	}
	return nil
}

// MarkAllAsRead đánh dấu đã đọc toàn bộ thông báo chưa đọc của user
func (r *Repository) MarkAllAsRead(userID uint) error {
	return r.db.
		Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
