package auth

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OTP struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    uint               `bson:"user_id"`
	Email     string             `bson:"email"`
	OTPHash   string             `bson:"otp_hash"`
	Purpose   string             `bson:"purpose"`
	CreatedAt time.Time          `bson:"created_at"`
	ExpiresAt time.Time          `bson:"expires_at"`
}
