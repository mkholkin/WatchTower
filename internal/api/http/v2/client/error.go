package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
)

func responseError(status int, payload []byte) error {
	var apiError apiv2gen.Error
	_ = json.Unmarshal(payload, &apiError)
	message := apiError.Message
	if message == "" {
		message = http.StatusText(status)
	}

	var sentinel error
	switch status {
	case http.StatusBadRequest:
		sentinel = baseservice.ErrInvalidData
	case http.StatusUnauthorized:
		sentinel = baseservice.ErrUnauthorized
	case http.StatusForbidden:
		sentinel = baseservice.ErrPermissionDenied
	case http.StatusNotFound:
		sentinel = repo.ErrNotFound
	case http.StatusConflict:
		sentinel = repo.ErrAlreadyExists
	default:
		sentinel = errors.New("API request failed")
	}
	return fmt.Errorf("%w: %s", sentinel, message)
}
