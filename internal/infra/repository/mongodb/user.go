package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
)

type userRepositoryMG struct {
	coll *mongo.Collection
	log  *slog.Logger
}

func NewUserRepository(db *mongo.Database, logger *slog.Logger) repo.UserRepository {
	return &userRepositoryMG{
		coll: db.Collection(collUsers),
		log:  logger.With("component", "mongodb_user_repository"),
	}
}

func (r *userRepositoryMG) Create(ctx context.Context, usr *user.User) error {
	_, err := r.coll.InsertOne(ctx, bson.M{
		"_id":          usr.Login,
		"password_hash": usr.PasswordHash,
	})
	return mapMongoError(err)
}

func (r *userRepositoryMG) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	var doc struct {
		ID           string `bson:"_id"`
		PasswordHash string `bson:"password_hash"`
	}
	if err := r.coll.FindOne(ctx, bson.M{"_id": login}).Decode(&doc); err != nil {
		return nil, mapMongoError(err)
	}
	return &user.User{Login: doc.ID, PasswordHash: doc.PasswordHash}, nil
}
