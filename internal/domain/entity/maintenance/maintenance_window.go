package maintenance

import (
	"WatchTower/internal/domain/entity/user"
	"errors"
	"time"

	"github.com/google/uuid"
)

type WindowType string

const (
	WindowTypeOneTime WindowType = "one_time"
	WindowTypeManual  WindowType = "manual"
)

type MaintenanceWindow struct {
	ID                      uuid.UUID
	User                    *user.User
	Title                   string
	Description             string
	MaintenanceWindowType   WindowType
	MaintenanceWindowConfig MaintenanceWindowConfig
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
		return nil, errors.New("user is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if startTime.IsZero() || endTime.IsZero() {
		return nil, errors.New("start_time and end_time are required for one-time maintenance window")
	}
	if !endTime.After(startTime) {
		return nil, errors.New("end_time must be after start_time")
	}
	if endTime.Before(time.Now()) {
		return nil, errors.New("end_time must be in the future")
	}

	return &MaintenanceWindow{
		User:                  user,
		Title:                 title,
		Description:           description,
		MaintenanceWindowType: WindowTypeOneTime,
		MaintenanceWindowConfig: OneTimeMaintenanceWindowConfig{
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
		return nil, errors.New("user is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}

	return &MaintenanceWindow{
		User:                  user,
		Title:                 title,
		Description:           description,
		MaintenanceWindowType: WindowTypeManual,
		MaintenanceWindowConfig: ManualMaintenanceWindowConfig{
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
			return errors.New("title cannot be empty")
		}
		mw.Title = *upd.Title
	}

	if upd.Description != nil {
		mw.Description = *upd.Description
	}

	if upd.ConfigUpdate != nil {
		newCfg, err := upd.ConfigUpdate.Apply(mw.MaintenanceWindowConfig)
		if err != nil {
			return err
		}
		mw.MaintenanceWindowConfig = newCfg
	}

	return nil
}

// IsActive checks if the maintenance window is currently active based on its configuration.
func (mw *MaintenanceWindow) IsActive() bool {
	return mw.MaintenanceWindowConfig.IsActive()
}

