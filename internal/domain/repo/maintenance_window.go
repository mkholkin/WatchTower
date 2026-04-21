package repo

import (
	"WatchTower/internal/domain/entity/maintenance"
	"context"

	"github.com/google/uuid"
)

type MaintenanceWindowRepository interface {
	Create(ctx context.Context, mw *maintenance.MaintenanceWindow) error
	GetByID(ctx context.Context, id uuid.UUID) (*maintenance.MaintenanceWindow, error)
	Update(ctx context.Context, mw *maintenance.MaintenanceWindow) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
	GetByIDBulk(ctx context.Context, ids []uuid.UUID) ([]maintenance.MaintenanceWindow, error)
	GetByUserLogin(ctx context.Context, userLogin string) ([]maintenance.MaintenanceWindow, error)
	// LinkMonitor links a monitor to a maintenance window (M:N relation).
	LinkMonitor(ctx context.Context, window *maintenance.MaintenanceWindow, monitorID uuid.UUID) error
	// UnlinkMonitor removes the link between a monitor and a maintenance window.
	UnlinkMonitor(ctx context.Context, window *maintenance.MaintenanceWindow, monitorID uuid.UUID) error
}
