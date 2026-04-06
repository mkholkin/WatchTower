package repo

import (
	"context"

	"WatchTower/internal/domain/entity/user"
)

type UserRepository interface {
	Create(ctx context.Context, usr *user.User) error
	GetByLogin(ctx context.Context, login string) (*user.User, error)
}

