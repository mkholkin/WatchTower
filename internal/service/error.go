package service

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized: missing or invalid user info")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrEventMarshalFailed = errors.New("failed to marshal event")
	ErrEventPublishFailed = errors.New("failed to publish event")
)
