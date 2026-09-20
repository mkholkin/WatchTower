package client

import (
	"context"
	"fmt"
	"net/http"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	alert "WatchTower/internal/domain/entity/alert_contact"
	contactdto "WatchTower/internal/service/contacts/dto"

	"github.com/google/uuid"
)

func mapAPIAlertContact(value apiv2gen.AlertContact) alert.Contact {
	return alert.Contact{
		ID:       value.Id,
		Name:     value.Name,
		Type:     alert.ContactType(value.Platform),
		Config:   alert.TelegramContactConfig{ChatID: int64(value.ChatId)},
		IsActive: value.IsEnabled,
	}
}

func (c *Client) CreateTelegramAlertContact(ctx context.Context, dto contactdto.CreateTelegramAlertContactDTO) (*alert.Contact, error) {
	token := dto.BotToken
	response, err := c.api.CreateAlertContactWithResponse(ctx, apiv2gen.CreateAlertContactJSONRequestBody{
		Name:     dto.Name,
		Platform: apiv2gen.AlertContactCreatePlatformTelegram,
		Credentials: apiv2gen.TelegramCredentials{
			ChatId: int(dto.ChatID), BotToken: &token,
		},
	}, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("create alert contact: %w", err)
	}
	if response.StatusCode() != http.StatusCreated || response.JSON201 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	contact := mapAPIAlertContact(*response.JSON201)
	return &contact, nil
}

func (c *Client) GetAllAlertContacts(ctx context.Context) ([]alert.Contact, error) {
	response, err := c.api.ListAlertContactsWithResponse(ctx, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("list alert contacts: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	result := make([]alert.Contact, 0, len(*response.JSON200))
	for _, value := range *response.JSON200 {
		result = append(result, mapAPIAlertContact(value))
	}
	return result, nil
}

func (c *Client) GetAlertContact(ctx context.Context, contactID uuid.UUID) (*alert.Contact, error) {
	response, err := c.api.GetAlertContactWithResponse(ctx, contactID, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("get alert contact: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	contact := mapAPIAlertContact(*response.JSON200)
	return &contact, nil
}

func (c *Client) UpdateAlertContact(ctx context.Context, dto contactdto.UpdateAlertContactDTO) error {
	body := apiv2gen.PatchAlertContactJSONRequestBody{Name: dto.Name, IsEnabled: dto.IsActive}
	if dto.ConfigUpdate != nil {
		var update alert.TelegramConfigUpdate
		switch typed := dto.ConfigUpdate.(type) {
		case alert.TelegramConfigUpdate:
			update = typed
		case *alert.TelegramConfigUpdate:
			if typed == nil {
				return fmt.Errorf("telegram config update is nil")
			}
			update = *typed
		default:
			return fmt.Errorf("unsupported contact config update %T", dto.ConfigUpdate)
		}
		if update.ChatID != nil {
			chatID := int(*update.ChatID)
			body.ChatId = &chatID
		}
		body.BotToken = update.BotToken
	}
	response, err := c.api.PatchAlertContactWithResponse(ctx, dto.ContactID, body, c.withAuth)
	if err != nil {
		return fmt.Errorf("patch alert contact: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) setAlertContactEnabled(ctx context.Context, contactID uuid.UUID, enabled bool) error {
	return c.UpdateAlertContact(ctx, contactdto.UpdateAlertContactDTO{ContactID: contactID, IsActive: &enabled})
}

func (c *Client) EnableAlertContact(ctx context.Context, contactID uuid.UUID) error {
	return c.setAlertContactEnabled(ctx, contactID, true)
}

func (c *Client) DisableAlertContact(ctx context.Context, contactID uuid.UUID) error {
	return c.setAlertContactEnabled(ctx, contactID, false)
}

func (c *Client) DeleteAlertContact(ctx context.Context, contactID uuid.UUID) error {
	response, err := c.api.DeleteAlertContactWithResponse(ctx, contactID, c.withAuth)
	if err != nil {
		return fmt.Errorf("delete alert contact: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}
