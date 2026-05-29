package postgres

import (
	"context"
	"log/slog"
	"testing"

	"WatchTower/internal/domain/entity/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Integration(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.Default()
	repo := NewUserRepository(pool, logger)
	ctx := context.Background()

	t.Run("Create and Get user", func(t *testing.T) {
		usr := &user.User{
			Login:        "testuser",
			PasswordHash: "hashedpass",
		}

		err := repo.Create(ctx, usr)
		require.NoError(t, err)

		fetchedUser, err := repo.GetByLogin(ctx, "testuser")
		require.NoError(t, err)
		assert.Equal(t, usr.Login, fetchedUser.Login)
		assert.Equal(t, usr.PasswordHash, fetchedUser.PasswordHash)
	})

	t.Run("Get non-existent user", func(t *testing.T) {
		_, err := repo.GetByLogin(ctx, "nonexistent")
		assert.Error(t, err)
	})

	t.Run("Get user inserted via raw SQL", func(t *testing.T) {
		login := "rawuser"
		passHash := "rawpass"
		_, err := pool.Exec(ctx, `INSERT INTO "user" (login, password_hash) VALUES ($1, $2)`, login, passHash)
		require.NoError(t, err)

		fetchedUser, err := repo.GetByLogin(ctx, login)
		require.NoError(t, err)
		assert.Equal(t, login, fetchedUser.Login)
		assert.Equal(t, passHash, fetchedUser.PasswordHash)
	})
}
