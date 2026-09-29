package v1

import (
	apigen "WatchTower/internal/api/http/v1/gen"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service"
	"errors"
)

type ApiError struct {
	Code int                  `json:"code"`
	Body apigen.ErrorResponse `json:"body"`
}

func ResponseFromError(err error) ApiError {
	var response apigen.ErrorResponse
	var apiError ApiError
	switch {
	case errors.Is(err, service.ErrInvalidData):
		apiError.Code = 400
		response = apigen.N400{
			Code:    "INVALID_DATA",
			Message: err.Error(),
		}
	case errors.Is(err, service.ErrUnauthorized):
		apiError.Code = 401
		response = apigen.N401{
			Code:    "UNAUTHORIZED",
			Message: err.Error(),
		}
	case errors.Is(err, service.ErrPermissionDenied):
		apiError.Code = 403
		response = apigen.N403{
			Code:    "PERMISSION_DENIED",
			Message: err.Error(),
		}
	case errors.Is(err, service.ErrNotFound), errors.Is(err, repo.ErrNotFound):
		apiError.Code = 404
		response = apigen.N404{
			Code:    "NOT_FOUND",
			Message: err.Error(),
		}
	default:
		apiError.Code = 500
		response = apigen.N500{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: err.Error(), // TODO: убрать в релизе
		}
	}

	apiError.Body = response

	return apiError
}

type ResponseFactory[T any] map[int]func(response apigen.ErrorResponse) T

func ResponseFromFactory[T any](factory ResponseFactory[T], err error) T {
	apiErr := ResponseFromError(err)
	if f, ok := factory[apiErr.Code]; ok {
		return f(apiErr.Body)
	}
	fallback := apigen.ErrorResponse{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: err.Error(),
	}
	if f, ok := factory[500]; ok {
		return f(fallback)
	}
	return factory[apiErr.Code](apiErr.Body)
}
