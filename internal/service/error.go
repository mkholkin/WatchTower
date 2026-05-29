package service

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized: missing or invalid user info")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrNotFound           = errors.New("not found")
	ErrInvalidData        = errors.New("invalid data")
	ErrEventMarshalFailed = errors.New("failed to marshal event")
	ErrEventPublishFailed = errors.New("failed to publish event")
)

type Error struct {
	svcError   error
	innerError error
}

func NewError(svcError, innerError error) error {
	return Error{
		svcError:   svcError,
		innerError: innerError,
	}
}

func (err Error) Error() string {
	return errors.Join().Error()
}
