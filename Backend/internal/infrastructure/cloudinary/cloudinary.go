package cloudinary

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Client struct {
	cloudName string
	apiKey    string
	apiSecret string
	http      *http.Client
	uploadURL string
}

func NewClient(cloudName, apiKey, apiSecret string) *Client {
	cloudName = strings.TrimSpace(cloudName)
	return &Client{
		cloudName: cloudName,
		apiKey:    strings.TrimSpace(apiKey),
		apiSecret: strings.TrimSpace(apiSecret),
		http:      &http.Client{Timeout: 20 * time.Second},
		uploadURL: fmt.Sprintf(
			"https://api.cloudinary.com/v1_1/%s/image/upload",
			cloudName,
		),
	}
}

func (c *Client) Configured() bool {
	return c != nil &&
		c.cloudName != "" &&
		c.apiKey != "" &&
		c.apiSecret != ""
}

func (c *Client) UploadAvatar(
	ctx context.Context,
	userID uint,
	file io.Reader,
	filename string,
	contentType string,
) (string, error) {
	if !c.Configured() {
		return "", errors.New("Cloudinary chưa được cấu hình")
	}
	if userID == 0 || file == nil {
		return "", errors.New("Dữ liệu ảnh đại diện không hợp lệ")
	}

	params := map[string]string{
		"invalidate": "true",
		"overwrite":  "true",
		"public_id":  fmt.Sprintf("nf-bank/avatars/user_%d", userID),
		"timestamp":  fmt.Sprintf("%d", time.Now().Unix()),
	}
	signature := signParameters(params, c.apiSecret)
	filename = filepath.Base(filename)
	filename = strings.NewReplacer(
		`"`, "_",
		"\r", "_",
		"\n", "_",
	).Replace(filename)
	if filename == "" || filename == "." {
		filename = "avatar"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set(
		"Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename),
	)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, file); err != nil {
		return "", err
	}
	for key, value := range params {
		if err = writer.WriteField(key, value); err != nil {
			return "", err
		}
	}
	if err = writer.WriteField("api_key", c.apiKey); err != nil {
		return "", err
	}
	if err = writer.WriteField("signature", signature); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.uploadURL,
		&body,
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := c.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("Không thể kết nối Cloudinary: %w", err)
	}
	defer response.Body.Close()

	var payload struct {
		SecureURL string `json:"secure_url"`
		Error     struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).
		Decode(&payload); err != nil {
		return "", errors.New("Cloudinary trả về dữ liệu không hợp lệ")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := strings.TrimSpace(payload.Error.Message)
		if message == "" {
			message = "Upload ảnh lên Cloudinary thất bại"
		}
		return "", errors.New(message)
	}
	if !strings.HasPrefix(payload.SecureURL, "https://") {
		return "", errors.New("Cloudinary không trả về secure_url hợp lệ")
	}
	return payload.SecureURL, nil
}

func signParameters(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+params[key])
	}
	hash := sha1.Sum([]byte(strings.Join(values, "&") + secret))
	return hex.EncodeToString(hash[:])
}
