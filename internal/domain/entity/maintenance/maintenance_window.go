package maintenance

import (
	"WatchTower/internal/domain/entity/user"
	"time"

	"github.com/google/uuid"
)

type WindowType string

const (
	WindowTypeOneTime WindowType = "one_time"
	WindowTypeManual  WindowType = "manual"
)

type MaintenanceWindow struct {
	ID          uuid.UUID
	User        *user.User
	Title       string
	Description string
	Type        WindowType
	Config      MaintenanceWindowConfig
}

// NewOneTimeMaintenanceWindow creates a one-time maintenance window with a fixed start and end time.
func NewOneTimeMaintenanceWindow(
	user *user.User,
	title string,
	description string,
	startTime time.Time,
	endTime time.Time,
) (*MaintenanceWindow, error) {
	if user == nil {
		return nil, wrapValidation("user is required")
	}
	if title == "" {
		return nil, wrapValidation("title is required")
	}
	if startTime.IsZero() || endTime.IsZero() {
		return nil, wrapValidation("start_time and end_time are required for one-time maintenance window")
	}
	if !endTime.After(startTime) {
		return nil, wrapValidation("end_time must be after start_time")
	}
	if endTime.Before(time.Now()) {
		return nil, wrapValidation("end_time must be in the future")
	}

	return &MaintenanceWindow{
		ID:          uuid.New(),
		User:        user,
		Title:       title,
		Description: description,
		Type:        WindowTypeOneTime,
		Config: OneTimeMaintenanceWindowConfig{
			StartTime: startTime,
			EndTime:   endTime,
		},
	}, nil
}

// NewManualMaintenanceWindow creates a manually controlled maintenance window (initially inactive).
func NewManualMaintenanceWindow(
	user *user.User,
	title string,
	description string,
) (*MaintenanceWindow, error) {
	if user == nil {
		return nil, wrapValidation("user is required")
	}
	if title == "" {
		return nil, wrapValidation("title is required")
	}

	return &MaintenanceWindow{
		ID:          uuid.New(),
		User:        user,
		Title:       title,
		Description: description,
		Type:        WindowTypeManual,
		Config: ManualMaintenanceWindowConfig{
			Active: false,
		},
	}, nil
}

// MaintenanceWindowUpdate represents a partial update of a maintenance window.
// nil fields are not changed. ConfigUpdate is type-specific and optional.
type MaintenanceWindowUpdate struct {
	Title        *string
	Description  *string
	ConfigUpdate MaintenanceWindowConfigUpdate // nil if no config change needed
}

// ApplyUpdate applies a partial update to the maintenance window.
func (mw *MaintenanceWindow) ApplyUpdate(upd MaintenanceWindowUpdate) error {
	if upd.Title != nil {
		if *upd.Title == "" {
			return wrapValidation("title cannot be empty")
		}
		mw.Title = *upd.Title
	}

	if upd.Description != nil {
		mw.Description = *upd.Description
	}

	if upd.ConfigUpdate != nil {
		newCfg, err := upd.ConfigUpdate.Apply(mw.Config)
		if err != nil {
			return err
		}
		mw.Config = newCfg
		mw.Type = newCfg.Type()
	}

	return nil
}

// IsActive checks if the maintenance window is currently active based on its configuration.
func (mw *MaintenanceWindow) IsActive() bool {
	return mw.Config.IsActive()
}
