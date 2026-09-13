package alert

import (
	"WatchTower/internal/domain/entity/user"

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
		return nil, wrapValidation("user is required")
	}
	if name == "" {
		return nil, wrapValidation("name is required")
	}
	if chatID == 0 {
		return nil, wrapValidation("chat_id is required")
	}
	if botToken == "" {
		return nil, wrapValidation("bot_token is required")
	}

	return &Contact{
		ID:       uuid.New(),
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
	next := *ac
	if upd.Name != nil {
		if *upd.Name == "" {
			return wrapValidation("name cannot be empty")
		}
		next.Name = *upd.Name
	}

	if upd.IsActive != nil {
		next.IsActive = *upd.IsActive
	}

	if upd.ConfigUpdate != nil {
		newCfg, err := upd.ConfigUpdate.Apply(next.Config)
		if err != nil {
			return err
		}
		next.Config = newCfg
		next.Type = newCfg.Type()
	}

	*ac = next
	return nil
}

// AlertContactRepository defines the persistence contract for alert contacts.
