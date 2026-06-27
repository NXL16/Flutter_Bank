package notification

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateNotification sinh thông báo mới và lưu vào DB (MySQL)
func (s *Service) CreateNotification(tx *gorm.DB, userID uint, notiType, title, content string) error {
	notiType = strings.TrimSpace(notiType)
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	if userID == 0 {
		return errors.New("userID không hợp lệ")
	}
	if title == "" || content == "" {
		return errors.New("tiêu đề và nội dung thông báo không được để trống")
	}
	if notiType == "" {
		notiType = "BALANCE_FLUCTUATION"
	}

	noti := &Notification{
		UserID:  userID,
		Type:    notiType,
		Title:   title,
		Content: content,
		IsRead:  false,
	}

	return s.repo.CreateNotification(tx, noti)
}

// GetNotifications trả về danh sách toàn bộ thông báo của 1 user
func (s *Service) GetNotifications(userID uint) ([]Notification, error) {
	if userID == 0 {
		return nil, errors.New("userID không hợp lệ")
	}
	return s.repo.FindNotificationsByUserID(userID)
}

// MarkNotificationRead đánh dấu 1 thông báo là đã đọc
func (s *Service) MarkNotificationRead(id uint, userID uint) error {
	if id == 0 || userID == 0 {
		return errors.New("thông tin ID hoặc UserID không hợp lệ")
	}
	return s.repo.MarkAsRead(id, userID)
}

// MarkAllNotificationsRead đánh dấu toàn bộ thông báo là đã đọc
func (s *Service) MarkAllNotificationsRead(userID uint) error {
	if userID == 0 {
		return errors.New("userID không hợp lệ")
	}
	return s.repo.MarkAllAsRead(userID)
}
