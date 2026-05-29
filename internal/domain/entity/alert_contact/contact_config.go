package alert

import "errors"

// ContactConfig is the polymorphic interface for alert contact configurations.
// Each contact type (Telegram, etc.) provides its own implementation.
type ContactConfig interface {
	Type() ContactType
}

// ContactConfigUpdate represents a type-specific partial update for a contact config.
// Each config type provides its own implementation.
type ContactConfigUpdate interface {
	// Apply applies the update to the given config and returns the new config.
	Apply(cfg ContactConfig) (ContactConfig, error)
}

// --- Telegram ---

// TelegramContactConfig represents the configuration for a Telegram alert contact.
type TelegramContactConfig struct {
	ChatID   int64  `json:"chat_id"`
	BotToken string `json:"bot_token"`
}

// Type returns the contact type for Telegram.
func (c TelegramContactConfig) Type() ContactType {
	return ContactTypeTelegram
}

// TelegramConfigUpdate is a partial update for TelegramContactConfig.
type TelegramConfigUpdate struct {
	ChatID   *int64
	BotToken *string
}

// Apply applies the partial update to a TelegramContactConfig and returns the new config.
func (u TelegramConfigUpdate) Apply(cfg ContactConfig) (ContactConfig, error) {
	c, ok := cfg.(TelegramContactConfig)
	if !ok {
		return nil, errors.New("telegram config update is not applicable to this contact type")
	}

	if u.ChatID != nil {
		if *u.ChatID == 0 {
			return nil, errors.New("chat_id cannot be zero")
		}
		c.ChatID = *u.ChatID
	}
	if u.BotToken != nil {
		if *u.BotToken == "" {
			return nil, errors.New("bot_token cannot be empty")
		}
		c.BotToken = *u.BotToken
	}

	return c, nil
}
