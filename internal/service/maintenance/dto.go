package maintenance_service

import (
	"WatchTower/internal/domain/entity/maintenance"
	"time"

	"github.com/google/uuid"
)

type CreateOneTimeMaintenanceWindowDTO struct {
	UserLogin   string    `json:"user_login"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

type CreateManualMaintenanceWindowDTO struct {
	UserLogin   string `json:"user_login"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateMaintenanceWindowDTO represents a partial update of a maintenance window.
// nil fields are not changed.
type UpdateMaintenanceWindowDTO struct {
	WindowID     uuid.UUID                                 `json:"window_id"`
	Title        *string                                   `json:"title,omitempty"`
	Description  *string                                   `json:"description,omitempty"`
	ConfigUpdate maintenance.MaintenanceWindowConfigUpdate // nil if no config change needed
}
