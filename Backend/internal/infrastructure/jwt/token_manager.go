package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID         uint   `json:"user_id"`
	Phone          string `json:"phone"`
	Role           string `json:"role"`
	SessionVersion int    `json:"session_version"`

	jwt.RegisteredClaims
}

func GenerateAccessToken(
	userID uint,
	phone string,
	role string,
	sessionVersion int,
	secret string,
) (string, error) {

	claims := CustomClaims{
		UserID:         userID,
		Phone:          phone,
		Role:           role,
		SessionVersion: sessionVersion,

		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(15 * time.Minute),
			),

			IssuedAt: jwt.NewNumericDate(time.Now()),

			NotBefore: jwt.NewNumericDate(time.Now()),

			Issuer: "NF-Bank",
		},
	}

	// Tạo token với thuật toán HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Ký token bằng secret key
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func GenerateRefreshToken(
	userID uint,
	phone string,
	role string,
	sessionVersion int,
	secret string,
) (string, error) {

	claims := CustomClaims{
		UserID:         userID,
		Phone:          phone,
		Role:           role,
		SessionVersion: sessionVersion,

		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(7 * 24 * time.Hour),
			),

			IssuedAt: jwt.NewNumericDate(time.Now()),

			NotBefore: jwt.NewNumericDate(time.Now()),

			Issuer: "NF-Bank",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateToken(
	tokenString string,
	secret string,
) (*CustomClaims, error) {

	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(token *jwt.Token) (interface{}, error) {

			// Chỉ chấp nhận HS256
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("Invalid signing method")
			}

			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	// Ép kiểu claims
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("Invalid token")
	}

	return claims, nil
}
