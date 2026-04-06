package dto

import (
	"WatchTower/internal/domain/entity/alert_contact"

	"github.com/google/uuid"
)

// UpdateAlertContactDTO represents a partial update of an alert contact.
// nil fields are not changed.
type UpdateAlertContactDTO struct {
	ContactID    uuid.UUID                 `json:"contact_id"`
	Name         *string                   `json:"name,omitempty"`
	IsActive     *bool                     `json:"is_active,omitempty"`
	ConfigUpdate alert.ContactConfigUpdate // nil if no config change needed
}
