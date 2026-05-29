package mongodb

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"WatchTower/internal/domain/repo"
)

func mapMongoError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, mongo.ErrNoDocuments) {
		return errors.Join(repo.ErrNotFound, err)
	}

	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Code == 11000 {
		return errors.Join(repo.ErrAlreadyExists, err)
	}

	return errors.Join(repo.ErrDB, err)
}
