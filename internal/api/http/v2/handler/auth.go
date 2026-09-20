package handler

import (
	"context"
	"unicode/utf8"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
)

func (a *APIHandler) LoginUser(ctx context.Context, request apiv2gen.LoginUserRequestObject) (apiv2gen.LoginUserResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.LoginUser400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	if message := validateCredentials(*request.Body); message != "" {
		return apiv2gen.LoginUser400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}

	token, err := a.authSvc.Login(ctx, request.Body.Login, request.Body.Password)
	if err != nil {
		status, body := errorFrom(err)
		switch status {
		case 401:
			return apiv2gen.LoginUser401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 400:
			return apiv2gen.LoginUser400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}, nil
		default:
			return apiv2gen.LoginUser500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}

	return apiv2gen.LoginUser200JSONResponse{AccessToken: token, TokenType: apiv2gen.Bearer}, nil
}

func (a *APIHandler) RegisterUser(ctx context.Context, request apiv2gen.RegisterUserRequestObject) (apiv2gen.RegisterUserResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.RegisterUser400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	if message := validateCredentials(*request.Body); message != "" {
		return apiv2gen.RegisterUser400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}

	if err := a.authSvc.Register(ctx, request.Body.Login, request.Body.Password); err != nil {
		status, body := errorFrom(err)
		switch status {
		case 400:
			return apiv2gen.RegisterUser400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}, nil
		case 409:
			return apiv2gen.RegisterUser409JSONResponse{ConflictJSONResponse: apiv2gen.ConflictJSONResponse(body)}, nil
		default:
			return apiv2gen.RegisterUser500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}

	return apiv2gen.RegisterUser201JSONResponse{Login: request.Body.Login}, nil
}

func validateCredentials(credentials apiv2gen.UserCredentials) string {
	loginLength := utf8.RuneCountInString(credentials.Login)
	if loginLength < 3 || loginLength > 64 {
		return "login length must be between 3 and 64 characters"
	}
	passwordLength := utf8.RuneCountInString(credentials.Password)
	if passwordLength < 8 || passwordLength > 128 {
		return "password length must be between 8 and 128 characters"
	}
	return ""
}
