package repo

import (
	"context"

	"github.com/google/uuid"

	"WatchTower/internal/domain/entity/alert_contact"
)

type AlertContactRepository interface {
	// Create persists a new alert contact and returns the generated ID.
	Create(ctx context.Context, contact *alert.Contact) error
	// GetByID retrieves an alert contact by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*alert.Contact, error)
	// Update persists changes to an existing alert contact.
	Update(ctx context.Context, contact *alert.Contact) error
	// DeleteByID removes an alert contact by its ID.
	DeleteByID(ctx context.Context, id uuid.UUID) error

	// GetByIDBulk retrieves multiple alert contacts by their IDs. Returns a slice of found contacts.
	GetByIDBulk(ctx context.Context, ids []uuid.UUID) ([]alert.Contact, error)

	Enable(ctx context.Context, id uuid.UUID) error
	Disable(ctx context.Context, id uuid.UUID) error

	// GetByUserLogin retrieves all alert contacts associated with the provided user login.
	GetByUserLogin(ctx context.Context, userLogin string) ([]alert.Contact, error)
}
