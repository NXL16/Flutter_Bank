package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID         uint   `json:"user_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	SessionVersion int    `json:"session_version"`

	jwt.RegisteredClaims
}

func GenerateAccessToken(
	userID uint,
	email string,
	role string,
	sessionVersion int,
	secret string,
) (string, error) {

	claims := CustomClaims{
		UserID:         userID,
		Email:          email,
		Role:           role,
		SessionVersion: sessionVersion,

		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(24* 60 * time.Minute),
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
	email string,
	role string,
	sessionVersion int,
	secret string,
) (string, error) {

	claims := CustomClaims{
		UserID:         userID,
		Email:          email,
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
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
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
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
