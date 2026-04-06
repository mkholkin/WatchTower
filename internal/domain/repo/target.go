package repo

import (
	"context"

	"github.com/google/uuid"

	"WatchTower/internal/domain/entity/target"
)

type TargetRepository interface {
	Create(ctx context.Context, tgt *target.Target) error
	GetByID(ctx context.Context, id uuid.UUID) (*target.Target, error)
	Update(ctx context.Context, tgt *target.Target) error
	DeleteByID(ctx context.Context, id uuid.UUID) error

	// UpdateProbeInterval updates the probe interval of the target with the given ID. If no such target exists, it returns an error.
	UpdateProbeInterval(ctx context.Context, id uuid.UUID, probeIntervalSec int32) error

	// GetByHash returns the target with the given hash, or nil if no such target exists.
	GetByHash(ctx context.Context, hash string) (*target.Target, error)

	// GetAllActive returns all targets that are currently active.
	GetAllActive(ctx context.Context) ([]target.Target, error)

	// Disable sets the target with the given ID as inactive. If the target is already inactive, it does nothing.
	Disable(ctx context.Context, id uuid.UUID) error
	// Enable sets the target with the given ID as active. If the target is already active, it does nothing.
	Enable(ctx context.Context, id uuid.UUID) error
}
