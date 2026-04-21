package notification

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	notificationsvc "WatchTower/internal/service/notification"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type telegramProvider struct {
	httpClient *http.Client
}

func NewTelegramNotificationProvider(httpClient *http.Client) notificationsvc.NotificationProvider {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &telegramProvider{httpClient: httpClient}
}

func (p *telegramProvider) SendNotification(contact *alert.Contact, msg string) error {
	if contact == nil {
		return fmt.Errorf("contact is nil")
	}

	cfg, ok := contact.Config.(alert.TelegramContactConfig)
	if !ok {
		return fmt.Errorf("unexpected contact config type: %T", contact.Config)
	}

	baseURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	params := url.Values{}
	params.Add("chat_id", fmt.Sprintf("%d", cfg.ChatID))
	params.Add("text", msg)
	resp, err := http.Get(fmt.Sprintf("%s?%s", baseURL, params.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: status=%d body=%s", resp.StatusCode, string(payload))
	}

	return nil
}
