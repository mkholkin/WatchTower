package handler

import (
	v1 "WatchTower/internal/api/http/v1"
	apigen "WatchTower/internal/api/http/v1/gen"
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/service/contacts/dto"
	"context"
	"fmt"
)

var getAlertContactListResponseFactory = v1.ResponseFactory[apigen.GetAlertContactsListResponseObject]{
	401: func(er apigen.ErrorResponse) apigen.GetAlertContactsListResponseObject {
		return apigen.GetAlertContactsList401JSONResponse{apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetAlertContactsListResponseObject {
		return apigen.GetAlertContactsList500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var createAlertContactResponseFactory = v1.ResponseFactory[apigen.CreateAlertContactResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.CreateAlertContactResponseObject {
		return apigen.CreateAlertContact400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.CreateAlertContactResponseObject {
		return apigen.CreateAlertContact401JSONResponse{apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.CreateAlertContactResponseObject {
		return apigen.CreateAlertContact500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var getAlertContactDetailsResponseFactory = v1.ResponseFactory[apigen.GetAlertContactDetailsResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.GetAlertContactDetailsResponseObject {
		return apigen.GetAlertContactDetails400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.GetAlertContactDetailsResponseObject {
		return apigen.GetAlertContactDetails401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.GetAlertContactDetailsResponseObject {
		return apigen.GetAlertContactDetails403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.GetAlertContactDetailsResponseObject {
		return apigen.GetAlertContactDetails404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetAlertContactDetailsResponseObject {
		return apigen.GetAlertContactDetails500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var deleteAlertContactResponseFactory = v1.ResponseFactory[apigen.DeleteAlertContactResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.DeleteAlertContactResponseObject {
		return apigen.DeleteAlertContact400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.DeleteAlertContactResponseObject {
		return apigen.DeleteAlertContact401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.DeleteAlertContactResponseObject {
		return apigen.DeleteAlertContact403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.DeleteAlertContactResponseObject {
		return apigen.DeleteAlertContact404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.DeleteAlertContactResponseObject {
		return apigen.DeleteAlertContact500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var disableAlertContactResponseFactory = v1.ResponseFactory[apigen.DisableAlertContactResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.DisableAlertContactResponseObject {
		return apigen.DisableAlertContact400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.DisableAlertContactResponseObject {
		return apigen.DisableAlertContact401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.DisableAlertContactResponseObject {
		return apigen.DisableAlertContact403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.DisableAlertContactResponseObject {
		return apigen.DisableAlertContact404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.DisableAlertContactResponseObject {
		return apigen.DisableAlertContact500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var enableAlertContactResponseFactory = v1.ResponseFactory[apigen.EnableAlertContactResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.EnableAlertContactResponseObject {
		return apigen.EnableAlertContact400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.EnableAlertContactResponseObject {
		return apigen.EnableAlertContact401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.EnableAlertContactResponseObject {
		return apigen.EnableAlertContact403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.EnableAlertContactResponseObject {
		return apigen.EnableAlertContact404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.EnableAlertContactResponseObject {
		return apigen.EnableAlertContact500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var updateAlertContactResponseFactory = v1.ResponseFactory[apigen.UpdateAlertContactResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.UpdateAlertContactResponseObject {
		return apigen.UpdateAlertContact400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.UpdateAlertContactResponseObject {
		return apigen.UpdateAlertContact401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.UpdateAlertContactResponseObject {
		return apigen.UpdateAlertContact403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.UpdateAlertContactResponseObject {
		return apigen.UpdateAlertContact404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.UpdateAlertContactResponseObject {
		return apigen.UpdateAlertContact500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

func mapDomainAlertContactToAPI(c *alert.Contact) (apigen.AlertContact, error) {
	apiContact := apigen.AlertContact{
		Id:        c.ID,
		Name:      c.Name,
		IsEnabled: c.IsActive,
	}

	switch cfg := c.Config.(type) {
	case alert.TelegramContactConfig:
		token := cfg.BotToken
		telegramCfg := apigen.TelegramConfig{
			Platform: apigen.Telegram,
			ChatId:   int(cfg.ChatID),
			Token:    token,
		}
		if err := apiContact.Config.FromTelegramConfig(telegramCfg); err != nil {
			return apigen.AlertContact{}, err
		}
	default:
		return apigen.AlertContact{}, fmt.Errorf("unsupported alert contact config type: %T", c.Config)
	}

	return apiContact, nil
}

func (a *ApiHandler) GetAlertContactsList(ctx context.Context, request apigen.GetAlertContactsListRequestObject) (apigen.GetAlertContactsListResponseObject, error) {
	_ = request
	contacts, err := a.alertContactsSvc.GetAllAlertContacts(ctx)
	if err != nil {
		return v1.ResponseFromFactory(getAlertContactListResponseFactory, err), nil
	}

	response := apigen.GetAlertContactsList200JSONResponse{}
	for _, c := range contacts {
		apiContact, mapErr := mapDomainAlertContactToAPI(&c)
		if mapErr != nil {
			return v1.ResponseFromFactory(getAlertContactListResponseFactory, mapErr), nil
		}
		response = append(response, apiContact)
	}

	return response, nil
}

func (a *ApiHandler) CreateAlertContact(ctx context.Context, request apigen.CreateAlertContactRequestObject) (apigen.CreateAlertContactResponseObject, error) {
	if request.Body == nil {
		return createAlertContactResponseFactory[400](errorResponse("INVALID_DATA", "request body is required")), nil
	}

	var contact *alert.Contact
	var err error

	switch platform, discrErr := request.Body.Config.Discriminator(); {
	case discrErr != nil:
		return createAlertContactResponseFactory[400](errorResponse("INVALID_DATA", "invalid platform")), nil
	case platform == string(apigen.Telegram):
		config, cfgErr := request.Body.Config.AsTelegramConfig()
		if cfgErr != nil {
			return createAlertContactResponseFactory[400](errorResponse("INVALID_DATA", cfgErr.Error())), nil
		}
		if config.ChatId == 0 || config.Token == "" {
			return createAlertContactResponseFactory[400](errorResponse("INVALID_DATA", "telegram chat_id and token are required")), nil
		}

		contact, err = a.alertContactsSvc.CreateTelegramAlertContact(ctx, dto.CreateTelegramAlertContactDTO{
			Name:     request.Body.Name,
			ChatID:   int64(config.ChatId),
			BotToken: config.Token,
		})
	default:
		return createAlertContactResponseFactory[400](errorResponse("INVALID_DATA", "unsupported alert contact platform")), nil
	}

	if err != nil {
		return v1.ResponseFromFactory(createAlertContactResponseFactory, err), nil
	}

	response, mapErr := mapDomainAlertContactToAPI(contact)
	if mapErr != nil {
		return v1.ResponseFromFactory(createAlertContactResponseFactory, mapErr), nil
	}

	return apigen.CreateAlertContact201JSONResponse(response), nil
}

func (a *ApiHandler) DeleteAlertContact(ctx context.Context, request apigen.DeleteAlertContactRequestObject) (apigen.DeleteAlertContactResponseObject, error) {
	if err := a.alertContactsSvc.DeleteAlertContact(ctx, request.ContactId); err != nil {
		return v1.ResponseFromFactory(deleteAlertContactResponseFactory, err), nil
	}

	return apigen.DeleteAlertContact204Response{}, nil
}

func (a *ApiHandler) GetAlertContactDetails(ctx context.Context, request apigen.GetAlertContactDetailsRequestObject) (apigen.GetAlertContactDetailsResponseObject, error) {
	contact, err := a.alertContactsSvc.GetAlertContact(ctx, request.ContactId)
	if err != nil {
		return v1.ResponseFromFactory(getAlertContactDetailsResponseFactory, err), nil
	}

	apiContact, mapErr := mapDomainAlertContactToAPI(contact)
	if mapErr != nil {
		return v1.ResponseFromFactory(getAlertContactDetailsResponseFactory, mapErr), nil
	}

	return apigen.GetAlertContactDetails200JSONResponse(apiContact), nil
}

func (a *ApiHandler) UpdateAlertContact(ctx context.Context, request apigen.UpdateAlertContactRequestObject) (apigen.UpdateAlertContactResponseObject, error) {
	if request.Body == nil {
		return updateAlertContactResponseFactory[400](errorResponse("INVALID_DATA", "request body is required")), nil
	}

	var cfgUpdate alert.ContactConfigUpdate
	if request.Body.Config != nil {
		switch platform, discrErr := request.Body.Config.Discriminator(); {
		case discrErr != nil:
			return updateAlertContactResponseFactory[400](errorResponse("INVALID_DATA", "invalid platform")), nil
		case platform == string(apigen.Telegram):
			config, cfgErr := request.Body.Config.AsTelegramConfig()
			if cfgErr != nil {
				return updateAlertContactResponseFactory[400](errorResponse("INVALID_DATA", cfgErr.Error())), nil
			}
			cfgUpdate = &alert.TelegramConfigUpdate{
				ChatID:   new(int64(config.ChatId)),
				BotToken: new(config.Token),
			}
		default:
			return updateAlertContactResponseFactory[400](errorResponse("INVALID_DATA", "unsupported alert contact platform")), nil
		}
	}

	err := a.alertContactsSvc.UpdateAlertContact(ctx, dto.UpdateAlertContactDTO{
		ContactID:    request.ContactId,
		Name:         request.Body.Name,
		IsActive:     request.Body.IsEnabled,
		ConfigUpdate: cfgUpdate,
	})

	if err != nil {
		return v1.ResponseFromFactory(updateAlertContactResponseFactory, err), nil
	}

	updatedContact, err := a.alertContactsSvc.GetAlertContact(ctx, request.ContactId)
	if err != nil {
		apiErr := v1.ResponseFromError(err)
		return updateAlertContactResponseFactory[apiErr.Code](apiErr.Body), nil
	}

	apiContact, err := mapDomainAlertContactToAPI(updatedContact)
	if err != nil {
		return v1.ResponseFromFactory(updateAlertContactResponseFactory, err), nil
	}

	return apigen.UpdateAlertContact200JSONResponse(apiContact), nil
}

func (a *ApiHandler) DisableAlertContact(ctx context.Context, request apigen.DisableAlertContactRequestObject) (apigen.DisableAlertContactResponseObject, error) {
	if err := a.alertContactsSvc.DisableAlertContact(ctx, request.ContactId); err != nil {
		return v1.ResponseFromFactory(disableAlertContactResponseFactory, err), nil
	}

	return apigen.DisableAlertContact200JSONResponse{}, nil
}

func (a *ApiHandler) EnableAlertContact(ctx context.Context, request apigen.EnableAlertContactRequestObject) (apigen.EnableAlertContactResponseObject, error) {
	if err := a.alertContactsSvc.EnableAlertContact(ctx, request.ContactId); err != nil {
		return v1.ResponseFromFactory(enableAlertContactResponseFactory, err), nil
	}

	return apigen.EnableAlertContact200JSONResponse{}, nil
}
