package monitor

import (
	"WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusUp          Status = "up"
	StatusDown        Status = "down"
	StatusMaintenance Status = "maintenance"
	StatusUnknown     Status = "unknown"
)

type Monitor struct {
	ID                 uuid.UUID
	Label              string
	Target             *target.Target
	User               *user.User
	AlertContacts      []alert.Contact
	MaintenanceWindows []maintenance.MaintenanceWindow
	CurrentStatus      Status
	LastEvaluatedAt    time.Time
	ProbeIntervalSec   int32
	IsActive           bool
	CreatedAt          time.Time
	Expectations       Expectations
}

func NewMonitor(
	name string,
	target *target.Target,
	user *user.User,
	alertContacts []alert.Contact,
	maintenanceWindows []maintenance.MaintenanceWindow,
	ProbeIntervalSec int32,
	expectations Expectations,
) (*Monitor, error) {
	if name == "" {
		return nil, wrapValidation("monitor name is required")
	}

	if target == nil {
		return nil, wrapValidation("monitor target is required")
	}

	if user == nil {
		return nil, wrapValidation("monitor user is required")
	}

	return &Monitor{
		ID:                 uuid.New(),
		Label:              name,
		Target:             target,
		User:               user,
		AlertContacts:      alertContacts,
		MaintenanceWindows: maintenanceWindows,
		CurrentStatus:      StatusUnknown,
		LastEvaluatedAt:    time.Time{},
		ProbeIntervalSec:   ProbeIntervalSec,
		IsActive:           true,
		CreatedAt:          time.Now(),
		Expectations:       expectations,
	}, nil
}

// OnMaintenance checks if the monitor is currently in a maintenance window.
// It returns true if any of the maintenance windows are active.
func (m *Monitor) OnMaintenance() bool {
	for _, mw := range m.MaintenanceWindows {
		if mw.IsActive() {
			return true
		}
	}

	return false
}

func (m *Monitor) Enable() {
	m.IsActive = true
}

func (m *Monitor) Disable() {
	m.IsActive = false
	m.CurrentStatus = StatusUnknown
}
