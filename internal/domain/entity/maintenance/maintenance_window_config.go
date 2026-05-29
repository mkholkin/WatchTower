package maintenance

import (
	"errors"
	"time"
)

type MaintenanceWindowConfig interface {
	IsActive() bool
	Type() WindowType
}

// MaintenanceWindowConfigUpdate represents a type-specific partial update for a maintenance window config.
// Each config type provides its own implementation.
type MaintenanceWindowConfigUpdate interface {
	// Apply applies the update to the given config and returns the new config.
	Apply(cfg MaintenanceWindowConfig) (MaintenanceWindowConfig, error)
}

// ---- One Time ----

// OneTimeMaintenanceWindowConfig represents a maintenance window that occurs once at a specific time.
type OneTimeMaintenanceWindowConfig struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (c OneTimeMaintenanceWindowConfig) IsActive() bool {
	return time.Now().After(c.StartTime) && time.Now().Before(c.EndTime)
}

func (c OneTimeMaintenanceWindowConfig) Type() WindowType {
	return WindowTypeOneTime
}

// OneTimeConfigUpdate is a partial update for OneTimeMaintenanceWindowConfig.
type OneTimeConfigUpdate struct {
	StartTime *time.Time
	EndTime   *time.Time
}

func (u OneTimeConfigUpdate) Apply(cfg MaintenanceWindowConfig) (MaintenanceWindowConfig, error) {
	c, ok := cfg.(OneTimeMaintenanceWindowConfig)
	if !ok {
		return nil, errors.New("one-time config update is not applicable to this maintenance window type")
	}

	if u.StartTime != nil {
		c.StartTime = *u.StartTime
	}
	if u.EndTime != nil {
		c.EndTime = *u.EndTime
	}

	if c.StartTime.IsZero() || c.EndTime.IsZero() {
		return nil, errors.New("start_time and end_time are required")
	}
	if !c.EndTime.After(c.StartTime) {
		return nil, errors.New("end_time must be after start_time")
	}
	if c.EndTime.Before(time.Now()) {
		return nil, errors.New("end_time must be in the future")
	}

	return c, nil
}

// ---- Manual ----

// ManualMaintenanceWindowConfig represents a maintenance window that is manually activated and deactivated by the user.
type ManualMaintenanceWindowConfig struct {
	Active bool `json:"active"`
}

func (c ManualMaintenanceWindowConfig) IsActive() bool {
	return c.Active
}

func (c ManualMaintenanceWindowConfig) Type() WindowType {
	return WindowTypeManual
}

// ManualConfigUpdate is a partial update for ManualMaintenanceWindowConfig.
type ManualConfigUpdate struct {
	Active *bool
}

func (u ManualConfigUpdate) Apply(cfg MaintenanceWindowConfig) (MaintenanceWindowConfig, error) {
	c, ok := cfg.(ManualMaintenanceWindowConfig)
	if !ok {
		return nil, errors.New("manual config update is not applicable to this maintenance window type")
	}

	if u.Active != nil {
		c.Active = *u.Active
	}

	return c, nil
}
