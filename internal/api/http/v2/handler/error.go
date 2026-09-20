package handler

import (
	"errors"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	authsvc "WatchTower/internal/service/auth"
)

func errorFrom(err error) (int, apiv2gen.Error) {
	status := 500
	code := "internal_error"

	switch {
	case errors.Is(err, baseservice.ErrInvalidData):
		status, code = 400, "invalid_request"
	case errors.Is(err, authsvc.ErrInvalidCredentials), errors.Is(err, authsvc.ErrTokenInvalid), errors.Is(err, baseservice.ErrUnauthorized):
		status, code = 401, "unauthorized"
	case errors.Is(err, baseservice.ErrPermissionDenied):
		status, code = 403, "forbidden"
	case errors.Is(err, repo.ErrNotFound), errors.Is(err, baseservice.ErrNotFound):
		status, code = 404, "not_found"
	case errors.Is(err, authsvc.ErrUserAlreadyExists), errors.Is(err, repo.ErrAlreadyExists):
		status, code = 409, "conflict"
	}

	message := "an unexpected error occurred"
	if status != 500 && err != nil {
		message = err.Error()
	}
	return status, apiv2gen.Error{Code: code, Message: message}
}

func badRequest(message string) apiv2gen.BadRequestJSONResponse {
	return apiv2gen.BadRequestJSONResponse{Code: "invalid_request", Message: message}
}

func unauthorized(errorBody apiv2gen.Error) apiv2gen.UnauthorizedJSONResponse {
	scheme := "Bearer"
	return apiv2gen.UnauthorizedJSONResponse{
		Body:    errorBody,
		Headers: apiv2gen.UnauthorizedResponseHeaders{WWWAuthenticate: &scheme},
	}
}
