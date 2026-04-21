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
		return errors.Join(repo.ErrNotFound, err)
	}

	return errors.Join(repo.ErrDB, err)
}
