package provider

import (
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	auth "WatchTower/internal/service/auth"
	"context"
	"fmt"
)

// UserProvider is an interface that provides a user from a given context.
type UserProvider interface {
	GetAuthorizedUser(ctx context.Context) (*user.User, error)
}

// UserProviderImpl is the default implementation of UserProvider.
type UserProviderImpl struct {
	userRepo repo.UserRepository
}

// NewUserProvider creates a new instance of UserProvider given a UserRepository.
func NewUserProvider(userRepo repo.UserRepository) UserProvider {
	return &UserProviderImpl{userRepo: userRepo}
}

// GetAuthorizedUser extracts the user login from context and fetches the full user profile from the specific repository.
func (p *UserProviderImpl) GetAuthorizedUser(ctx context.Context) (*user.User, error) {
	login, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, baseservice.ErrUnauthorized
	}

	usr, err := p.userRepo.GetByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %w", err)
	}

	return usr, nil
}
