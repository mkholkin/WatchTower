package handler

import (
	"context"
	"fmt"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	alert "WatchTower/internal/domain/entity/alert_contact"
	contactdto "WatchTower/internal/service/contacts/dto"
)

func mapDomainAlertContact(contact *alert.Contact) apiv2gen.AlertContact {
	result := apiv2gen.AlertContact{
		Id:        contact.ID,
		Name:      contact.Name,
		Platform:  apiv2gen.AlertContactPlatform(contact.Type),
		IsEnabled: contact.IsActive,
	}
	switch config := contact.Config.(type) {
	case alert.TelegramContactConfig:
		result.ChatId = int(config.ChatID)
	case *alert.TelegramContactConfig:
		if config != nil {
			result.ChatId = int(config.ChatID)
		}
	}
	return result
}

func (a *APIHandler) ListAlertContacts(ctx context.Context, _ apiv2gen.ListAlertContactsRequestObject) (apiv2gen.ListAlertContactsResponseObject, error) {
	contacts, err := a.alertContactsSvc.GetAllAlertContacts(ctx)
	if err != nil {
		status, body := errorFrom(err)
		if status == 401 {
			return apiv2gen.ListAlertContacts401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		}
		return apiv2gen.ListAlertContacts500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
	}

	response := make(apiv2gen.ListAlertContacts200JSONResponse, 0, len(contacts))
	for i := range contacts {
		response = append(response, mapDomainAlertContact(&contacts[i]))
	}
	return response, nil
}

func (a *APIHandler) CreateAlertContact(ctx context.Context, request apiv2gen.CreateAlertContactRequestObject) (apiv2gen.CreateAlertContactResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.CreateAlertContact400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	if request.Body.Platform != apiv2gen.AlertContactCreatePlatformTelegram {
		return apiv2gen.CreateAlertContact400JSONResponse{BadRequestJSONResponse: badRequest("unsupported alert contact platform")}, nil
	}
	if message := validateRequiredText(request.Body.Name, "name", 120); message != "" {
		return apiv2gen.CreateAlertContact400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}
	if request.Body.Credentials.BotToken == nil || *request.Body.Credentials.BotToken == "" {
		return apiv2gen.CreateAlertContact400JSONResponse{BadRequestJSONResponse: badRequest("bot_token is required")}, nil
	}

	contact, err := a.alertContactsSvc.CreateTelegramAlertContact(ctx, contactdto.CreateTelegramAlertContactDTO{
		Name:     request.Body.Name,
		ChatID:   int64(request.Body.Credentials.ChatId),
		BotToken: *request.Body.Credentials.BotToken,
	})
	if err != nil {
		status, body := errorFrom(err)
		switch status {
		case 400:
			return apiv2gen.CreateAlertContact400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}, nil
		case 401:
			return apiv2gen.CreateAlertContact401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		default:
			return apiv2gen.CreateAlertContact500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}

	location := fmt.Sprintf("/api/v2/alert-contacts/%s", contact.ID)
	return apiv2gen.CreateAlertContact201JSONResponse{
		Body:    mapDomainAlertContact(contact),
		Headers: apiv2gen.CreateAlertContact201ResponseHeaders{Location: &location},
	}, nil
}

func (a *APIHandler) GetAlertContact(ctx context.Context, request apiv2gen.GetAlertContactRequestObject) (apiv2gen.GetAlertContactResponseObject, error) {
	contact, err := a.alertContactsSvc.GetAlertContact(ctx, request.ContactId)
	if err != nil {
		return getAlertContactError(err), nil
	}
	return apiv2gen.GetAlertContact200JSONResponse(mapDomainAlertContact(contact)), nil
}

func getAlertContactError(err error) apiv2gen.GetAlertContactResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 401:
		return apiv2gen.GetAlertContact401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.GetAlertContact403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.GetAlertContact404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.GetAlertContact500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) PatchAlertContact(ctx context.Context, request apiv2gen.PatchAlertContactRequestObject) (apiv2gen.PatchAlertContactResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.PatchAlertContact400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	body := request.Body
	if body.Name == nil && body.ChatId == nil && body.BotToken == nil && body.IsEnabled == nil {
		return apiv2gen.PatchAlertContact400JSONResponse{BadRequestJSONResponse: badRequest("at least one field is required")}, nil
	}
	if body.Name != nil {
		if message := validateRequiredText(*body.Name, "name", 120); message != "" {
			return apiv2gen.PatchAlertContact400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
		}
	}
	if body.BotToken != nil && *body.BotToken == "" {
		return apiv2gen.PatchAlertContact400JSONResponse{BadRequestJSONResponse: badRequest("bot_token must not be empty")}, nil
	}

	update := contactdto.UpdateAlertContactDTO{
		ContactID: request.ContactId,
		Name:      body.Name,
		IsActive:  body.IsEnabled,
	}
	if body.ChatId != nil || body.BotToken != nil {
		config := alert.TelegramConfigUpdate{BotToken: body.BotToken}
		if body.ChatId != nil {
			chatID := int64(*body.ChatId)
			config.ChatID = &chatID
		}
		update.ConfigUpdate = config
	}

	if err := a.alertContactsSvc.UpdateAlertContact(ctx, update); err != nil {
		return patchAlertContactError(err), nil
	}
	contact, err := a.alertContactsSvc.GetAlertContact(ctx, request.ContactId)
	if err != nil {
		return patchAlertContactError(err), nil
	}
	return apiv2gen.PatchAlertContact200JSONResponse(mapDomainAlertContact(contact)), nil
}

func patchAlertContactError(err error) apiv2gen.PatchAlertContactResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 400:
		return apiv2gen.PatchAlertContact400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}
	case 401:
		return apiv2gen.PatchAlertContact401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.PatchAlertContact403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.PatchAlertContact404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.PatchAlertContact500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) DeleteAlertContact(ctx context.Context, request apiv2gen.DeleteAlertContactRequestObject) (apiv2gen.DeleteAlertContactResponseObject, error) {
	if err := a.alertContactsSvc.DeleteAlertContact(ctx, request.ContactId); err != nil {
		status, body := errorFrom(err)
		switch status {
		case 401:
			return apiv2gen.DeleteAlertContact401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 403:
			return apiv2gen.DeleteAlertContact403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}, nil
		case 404:
			return apiv2gen.DeleteAlertContact404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}, nil
		default:
			return apiv2gen.DeleteAlertContact500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}
	return apiv2gen.DeleteAlertContact204Response{}, nil
}
