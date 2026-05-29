package alert

import (
	"WatchTower/internal/domain/entity/user"
	"errors"

	"github.com/google/uuid"
)

type ContactType string

const (
	ContactTypeTelegram ContactType = "telegram"
)

type Contact struct {
	ID       uuid.UUID
	User     *user.User
	Name     string
	Type     ContactType
	Config   ContactConfig
	IsActive bool
}

// NewTelegramAlertContact creates a new Telegram alert contact.
func NewTelegramAlertContact(
	user *user.User,
	name string,
	chatID int64,
	botToken string,
) (*Contact, error) {
	if user == nil {
		return nil, errors.New("user is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}
	if chatID == 0 {
		return nil, errors.New("chat_id is required")
	}
	if botToken == "" {
		return nil, errors.New("bot_token is required")
	}

	return &Contact{
		User:     user,
		Name:     name,
		Type:     ContactTypeTelegram,
		Config:   TelegramContactConfig{ChatID: chatID, BotToken: botToken},
		IsActive: true,
	}, nil
}

// ContactUpdate represents a partial update of an alert contact.
// nil fields are not changed. ConfigUpdate is type-specific and optional.
type ContactUpdate struct {
	Name         *string
	IsActive     *bool
	ConfigUpdate ContactConfigUpdate // nil if no config change needed
}

// ApplyUpdate applies a partial update to the alert contact.
func (ac *Contact) ApplyUpdate(upd ContactUpdate) error {
	if upd.Name != nil {
		if *upd.Name == "" {
			return errors.New("name cannot be empty")
		}
		ac.Name = *upd.Name
	}

	if upd.IsActive != nil {
		ac.IsActive = *upd.IsActive
	}

	if upd.ConfigUpdate != nil {
		newCfg, err := upd.ConfigUpdate.Apply(ac.Config)
		if err != nil {
			return err
		}
		ac.Config = newCfg
	}

	return nil
}

// AlertContactRepository defines the persistence contract for alert contacts.
