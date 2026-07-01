package notification

import (
	"bank-service/internal/infrastructure/firebase"
	"context"
	"errors"
	"strings"
	"time"

	"firebase.google.com/go/v4/messaging"
	"gorm.io/gorm"
)

type Service struct {
	repo           *Repository
	firebaseClient *firebase.Client
}

func NewService(repo *Repository, firebaseClient *firebase.Client) *Service {
	return &Service{
		repo:           repo,
		firebaseClient: firebaseClient,
	}
}

func (s *Service) RegisterPushToken(userID uint, token, platform string) error {
	token = strings.TrimSpace(token)
	platform = strings.ToLower(strings.TrimSpace(platform))
	if userID == 0 || len(token) < 20 {
		return errors.New("Push token không hợp lệ")
	}
	if platform != "android" && platform != "ios" && platform != "web" &&
		platform != "macos" && platform != "windows" && platform != "linux" {
		return errors.New("Nền tảng push notification không hợp lệ")
	}
	return s.repo.UpsertPushToken(&PushToken{
		UserID:     userID,
		Token:      token,
		Platform:   platform,
		LastSeenAt: time.Now(),
	})
}

func (s *Service) UnregisterPushToken(userID uint, token string) error {
	if userID == 0 || strings.TrimSpace(token) == "" {
		return errors.New("Push token không hợp lệ")
	}
	return s.repo.DeletePushToken(userID, strings.TrimSpace(token))
}

// SendPushToUser chạy sau khi giao dịch DB đã commit. Lỗi gửi push được trả về
// cho logging/retry, không được dùng để đảo ngược giao dịch tài chính.
func (s *Service) SendPushToUser(
	userID uint,
	title string,
	body string,
	data map[string]string,
) error {
	tokens, err := s.repo.FindPushTokensByUserID(userID)
	if err != nil || len(tokens) == 0 {
		return err
	}

	values := make([]string, 0, len(tokens))
	for _, token := range tokens {
		values = append(values, token.Token)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	result, err := s.firebaseClient.MessagingClient.SendEachForMulticast(
		ctx,
		&messaging.MulticastMessage{
			Tokens: values,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: data,
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					ChannelID:             "nfbank_transactions_v2",
					Sound:                 "default",
					Priority:              messaging.PriorityHigh,
					DefaultVibrateTimings: true,
					DefaultSound:          true,
					Visibility:            messaging.VisibilityPublic,
				},
			},
		},
	)
	if err != nil {
		return err
	}
	for index, response := range result.Responses {
		if response.Error != nil && messaging.IsRegistrationTokenNotRegistered(response.Error) {
			_ = s.repo.DeletePushTokenValue(values[index])
		}
	}
	return nil
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
		return errors.New("Tiêu đề và nội dung thông báo không được để trống")
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

// CreateUserNotification dùng cho sự kiện đã commit hoặc không nằm trong
// transaction tài chính hiện tại.
func (s *Service) CreateUserNotification(
	userID uint,
	notiType string,
	title string,
	content string,
) error {
	return s.CreateNotification(
		s.repo.db,
		userID,
		notiType,
		title,
		content,
	)
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
		return errors.New("Thông tin ID hoặc UserID không hợp lệ")
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
