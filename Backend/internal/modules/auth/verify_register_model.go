package auth

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VerifyRegister struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	FullName      string             `bson:"full_name"`
	Email         string             `bson:"email"`
	Phone         string             `bson:"phone"`
	OTPChannel    string             `bson:"otp_channel"`
	PasswordHash  string             `bson:"password_hash"`
	OTPHash       string             `bson:"otp_hash"`
	CreatedAt     time.Time          `bson:"created_at"`
	ExpiresAt     time.Time          `bson:"expires_at"`
}
