package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// GenerateSecret sinh một khóa bí mật ngẫu nhiên định dạng Base32 dài 16 ký tự
func GenerateSecret() string {
	randomBytes := make([]byte, 10)
	_, _ = rand.Read(randomBytes)
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	return strings.ToUpper(secret)
}

// ValidateCode kiểm tra mã code 6 số từ Google Authenticator so với khóa bí mật
func ValidateCode(secret string, code string) bool {
	_, valid := MatchCodeAt(secret, code, time.Now())
	return valid
}

// MatchCodeAt trả về time-step đã khớp để tầng gọi có thể chống tái sử dụng
// cùng một TOTP trong cửa sổ hiệu lực.
func MatchCodeAt(secret string, code string, now time.Time) (int64, bool) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	if secret == "" || len(code) != 6 {
		return 0, false
	}

	key, err := decodeBase32(secret)
	if err != nil {
		return 0, false
	}

	currentTime := now.Unix()

	// Cho phép sai số lệch thời gian +/- 30 giây (offset -1, 0, 1)
	for _, offset := range []int64{-1, 0, 1} {
		step := (currentTime / 30) + offset
		if calculateTOTP(key, step) == code {
			return step, true
		}
	}

	return 0, false
}

// decodeBase32 giải mã chuỗi Base32 hỗ trợ tự động bù padding '='
func decodeBase32(secret string) ([]byte, error) {
	secret = strings.ReplaceAll(secret, "=", "")
	pad := len(secret) % 8
	if pad != 0 {
		secret += strings.Repeat("=", 8-pad)
	}
	return base32.StdEncoding.DecodeString(secret)
}

// calculateTOTP tính toán mã OTP dựa trên khóa bí mật và bước thời gian (step)
func calculateTOTP(key []byte, step int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(step))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	// Thuật toán Dynamic Truncation lấy 4 bytes từ HMAC-SHA1
	offset := sum[len(sum)-1] & 0xf
	binaryVal := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	otp := binaryVal % 1000000
	return fmt.Sprintf("%06d", otp)
}
