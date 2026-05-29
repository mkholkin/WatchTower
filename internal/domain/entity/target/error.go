package target

import (
	"errors"
	"fmt"
)

var ErrValidation = errors.New("validation failed")

func wrapValidation(reason string) error {
	return fmt.Errorf("%w: %s", ErrValidation, reason)
}

func wrapValidationf(reasonFmt string, args ...interface{}) error {
	return wrapValidation(fmt.Sprintf(reasonFmt, args...))
}