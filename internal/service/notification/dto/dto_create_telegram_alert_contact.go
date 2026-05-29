package dto

// CreateTelegramAlertContactDTO contains the data needed to create a Telegram alert contact.
type CreateTelegramAlertContactDTO struct {
	UserLogin string `json:"user_login"`
	Name      string `json:"name"`
	ChatID    int64  `json:"chat_id"`
	BotToken  string `json:"bot_token"`
}
