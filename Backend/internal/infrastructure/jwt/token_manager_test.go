package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestAccessTokenLifetimeAndClaims(t *testing.T) {
	const secret = "a-production-length-test-secret"
	token, err := GenerateAccessToken(42, "+84901234567", "user", 3, secret)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != 42 || claims.SessionVersion != 3 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	lifetime := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if lifetime < 14*time.Minute || lifetime > 16*time.Minute {
		t.Fatalf("access token lifetime = %v, want about 15 minutes", lifetime)
	}
}

func TestValidateTokenRejectsNonHS256(t *testing.T) {
	claims := CustomClaims{
		UserID: 1,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS512, claims).
		SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(token, "secret"); err == nil {
		t.Fatal("ValidateToken() accepted HS512 token")
	}
}
