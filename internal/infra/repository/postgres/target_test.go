package postgres

import (
	"context"
	"log/slog"
	"testing"

	"WatchTower/internal/domain/entity/target"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetRepository_Integration(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.Default()
	repo := NewTargetRepository(pool, logger)
	ctx := context.Background()

	t.Run("Create and Get target", func(t *testing.T) {
		cfg := target.HTTPConfig{
			Method: "GET",
		}
		tgt, err := target.NewTarget(
			"https://example.com",
			60,
			cfg,
		)
		require.NoError(t, err)

		err = repo.Create(ctx, tgt)
		require.NoError(t, err)

		fetchedTarget, err := repo.GetByID(ctx, tgt.ID)
		if err != nil {
			t.Logf("GetByID failed with error: %v, underlying: %v", err, err)
		}
		require.NoError(t, err)
		assert.Equal(t, tgt.Endpoint, fetchedTarget.Endpoint)
		assert.Equal(t, tgt.ConfigHash, fetchedTarget.ConfigHash)
		assert.Equal(t, tgt.IsActive, fetchedTarget.IsActive)
		assert.Equal(t, tgt.ProbeIntervalSec, fetchedTarget.ProbeIntervalSec)
	})

	t.Run("Update target", func(t *testing.T) {
		cfg := target.HTTPConfig{Method: "GET"}
		tgt, err := target.NewTarget(
			"https://example2.com",
			60,
			cfg,
		)
		require.NoError(t, err)

		err = repo.Create(ctx, tgt)
		require.NoError(t, err)

		tgt.Endpoint = "https://updated.com"
		tgt.ProbeIntervalSec = 120
		err = repo.Update(ctx, tgt)
		require.NoError(t, err)

		fetchedTarget, err := repo.GetByID(ctx, tgt.ID)
		require.NoError(t, err)
		assert.Equal(t, "https://updated.com", fetchedTarget.Endpoint)
		assert.Equal(t, int32(120), fetchedTarget.ProbeIntervalSec)
	})

	t.Run("Delete target", func(t *testing.T) {
		cfg := target.HTTPConfig{Method: "GET"}
		tgt, err := target.NewTarget(
			"https://example3.com",
			60,
			cfg,
		)
		require.NoError(t, err)

		err = repo.Create(ctx, tgt)
		require.NoError(t, err)

		err = repo.DeleteByID(ctx, tgt.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, tgt.ID)
		assert.Error(t, err)
	})

	t.Run("Get target inserted via raw SQL", func(t *testing.T) {
		rawId := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO "target" (id, signature_hash, protocol, endpoint, network_config, is_active, probe_interval_sec)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, rawId, "raw_hash", "HTTP", "https://raw.com", []byte(`{"method":"GET"}`), true, 60)
		require.NoError(t, err)

		fetchedTarget, err := repo.GetByID(ctx, rawId)
		require.NoError(t, err)
		assert.Equal(t, "https://raw.com", fetchedTarget.Endpoint)
		assert.Equal(t, "raw_hash", fetchedTarget.ConfigHash)
		assert.Equal(t, true, fetchedTarget.IsActive)
		assert.Equal(t, int32(60), fetchedTarget.ProbeIntervalSec)
	})
}
