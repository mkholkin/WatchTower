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
	causes := err.Unwrap()
	if len(causes) == 0 {
		return ""
	}
	if len(causes) == 1 {
		return causes[0].Error()
	}
	return errors.Join(causes...).Error()
}

func (err Error) Unwrap() []error {
	causes := make([]error, 0, 2)
	if err.svcError != nil {
		causes = append(causes, err.svcError)
	}
	if err.innerError != nil {
		causes = append(causes, err.innerError)
	}
	return causes
}
