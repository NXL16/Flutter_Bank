package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type VerifyRegisterRepository struct {
	collection *mongo.Collection
}

func NewVerifyRegisterRepository(db *mongo.Database) *VerifyRegisterRepository {
	return &VerifyRegisterRepository{
		collection: db.Collection("verify_registers"),
	}
}

func (r *VerifyRegisterRepository) CreateTTLIndex(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "expires_at", Value: 1},
		},
		Options: options.Index().
			SetExpireAfterSeconds(0).
			SetName("verify_registers_ttl"),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

func (r *VerifyRegisterRepository) Create(
	ctx context.Context,
	data *VerifyRegister,
) error {
	_, err := r.collection.InsertOne(ctx, data)
	return err
}

func (r *VerifyRegisterRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*VerifyRegister, error) {
	var data VerifyRegister

	filter := bson.M{
		"email": email,
		"expires_at": bson.M{
			"$gt": time.Now(),
		},
	}

	opts := options.FindOne().
		SetSort(bson.D{
			{Key: "created_at", Value: -1},
		})

	err := r.collection.FindOne(ctx, filter, opts).Decode(&data)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	return &data, nil
}

func (r *VerifyRegisterRepository) Delete(
	ctx context.Context,
	id interface{},
) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{
		"_id": id,
	})

	return err
}
