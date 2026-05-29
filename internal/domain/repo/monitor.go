package repo

import (
	"context"

	"github.com/google/uuid"

	"WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/user"
)

type MonitorRepository interface {
	Create(ctx context.Context, mon *monitor.Monitor) error
	GetByID(ctx context.Context, id uuid.UUID) (*monitor.Monitor, error)
	Update(ctx context.Context, mon *monitor.Monitor) error
	DeleteByID(ctx context.Context, id uuid.UUID) error

	// GetAllByUser retrieves all monitors associated with a given user.
	GetAllByUser(ctx context.Context, usr *user.User) ([]*monitor.Monitor, error)

	// GetAllByTargetID retrieves all monitors associated with a target with given id.
	GetAllByTargetID(ctx context.Context, targetID uuid.UUID) ([]*monitor.Monitor, error)

	// GetMonitorsToEvaluate retrieves all monitors for given target id's that are active and last evaluation time + probe interval is lower than current.
	// Monitors are grouped by target id.
	GetMonitorsToEvaluate(ctx context.Context, targetIDs []uuid.UUID) (map[uuid.UUID][]*monitor.Monitor, error)

	// BulkUpdateEvaluation updates last_evaluated_at and current_status for multiple monitors in a single batch.
	BulkUpdateEvaluation(ctx context.Context, monitors []*monitor.Monitor) error

	// AddAlertContact adds an alert contact to a monitor.
	AddAlertContact(ctx context.Context, mon *monitor.Monitor, contact *alert.Contact) error

	// RemoveAlertContact removes an alert contact from a monitor.
	RemoveAlertContact(ctx context.Context, mon *monitor.Monitor, contact *alert.Contact) error

	Enable(ctx context.Context, monitorID uuid.UUID) error
	Disable(ctx context.Context, monitorID uuid.UUID) error
}
