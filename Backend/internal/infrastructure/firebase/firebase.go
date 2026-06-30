package firebase

import (
	"context"
	"errors"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type Client struct {
	App             *firebase.App
	AuthClient      *auth.Client
	MessagingClient *messaging.Client
}

// InitFirebase khởi tạo kết nối với Firebase Admin SDK
func InitFirebase(credentialsPath string) (*Client, error) {
	if credentialsPath == "" {
		return nil, errors.New("Đường dẫn credentials Firebase trống")
	}

	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("Lỗi khởi tạo Firebase App: %v", err)
	}

	authClient, err := app.Auth(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Lỗi khởi tạo Firebase Auth client: %v", err)
	}
	messagingClient, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Lỗi khởi tạo Firebase Messaging client: %v", err)
	}

	log.Println("Đã kết nối thành công Firebase Admin SDK!")
	return &Client{
		App:             app,
		AuthClient:      authClient,
		MessagingClient: messagingClient,
	}, nil
}

// VerifyIDToken xác thực Firebase ID Token từ client và trả về số điện thoại đã được xác minh
func (c *Client) VerifyIDToken(idToken string) (string, error) {
	token, err := c.AuthClient.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		return "", fmt.Errorf("ID Token không hợp lệ hoặc đã hết hạn: %v", err)
	}

	// Lấy số điện thoại từ các claims của token
	phoneVal, exists := token.Claims["phone_number"]
	if !exists {
		return "", errors.New("Không tìm thấy thông tin số điện thoại trong token")
	}

	phoneNumber, ok := phoneVal.(string)
	if !ok || phoneNumber == "" {
		return "", errors.New("Số điện thoại trong token không đúng định dạng")
	}

	return phoneNumber, nil
}
