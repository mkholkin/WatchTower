package postgres

import (
	"context"
	"log/slog"
	"github.com/jackc/pgx/v5/pgxpool"

	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
)

type userRepositoryPG struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	log     *slog.Logger
}

func NewUserRepository(pool *pgxpool.Pool, logger *slog.Logger) repo.UserRepository {
	return &userRepositoryPG{
		pool:    pool,
		queries: sqlcgen.New(pool),
		log:     logger,
	}
}

func (r *userRepositoryPG) Create(ctx context.Context, usr *user.User) error {
	params := sqlcgen.CreateUserParams{
		Login:        usr.Login,
		PasswordHash: usr.PasswordHash,
	}

	err := r.queries.CreateUser(ctx, params)
	if err != nil {
		r.log.Error("create user failed", "login", usr.Login, "error", err)
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *userRepositoryPG) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	usrRow, err := r.queries.GetUserByLogin(ctx, login)
	if err != nil {
		r.log.Error("get user by login failed", "login", login, "error", err)
		return nil, mapPGXErrorToRepo(err)
	}

	return &user.User{
		Login:        usrRow.Login,
		PasswordHash: usrRow.PasswordHash,
	}, nil
}
