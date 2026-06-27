package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OTPRepository struct {
	collection *mongo.Collection
}

func NewOTPRepository(db *mongo.Database) *OTPRepository {
	return &OTPRepository{
		collection: db.Collection("otps"),
	}
}

func (r *OTPRepository) CreateTTLIndex(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "expires_at", Value: 1},
		},
		Options: options.Index().
			SetExpireAfterSeconds(0).
			SetName("otps_ttl"),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (r *OTPRepository) CreateOTP(ctx context.Context, otp *OTP) error {
	_, err := r.collection.InsertOne(ctx, otp)
	return err
}

func (r *OTPRepository) FindValidOTPByEmailAndPurpose(ctx context.Context, email string, purpose string) (*OTP, error) {
	var otp OTP

	filter := bson.M{
		"email":   email,
		"purpose": purpose,
		"expires_at": bson.M{
			"$gt": time.Now(),
		},
	}

	opts := options.FindOne().
		SetSort(bson.D{
			{Key: "created_at", Value: -1},
		})

	err := r.collection.FindOne(ctx, filter, opts).Decode(&otp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	return &otp, nil
}

func (r *OTPRepository) DeleteOTP(ctx context.Context, id interface{}) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{
		"_id": id,
	})
	return err
}
