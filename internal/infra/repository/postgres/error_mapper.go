package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"

	"WatchTower/internal/domain/repo"
)

func mapPGXErrorToRepo(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return repo.ErrNotFound
	}

	return repo.ErrDB
}

