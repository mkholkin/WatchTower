package target

import (
	"WatchTower/internal/service"
	"fmt"
)

func wrapValidation(reason string) error {
	return fmt.Errorf("%w: %s", service.ErrInvalidData, reason)
}

func wrapValidationf(reasonFmt string, args ...interface{}) error {
	return wrapValidation(fmt.Sprintf(reasonFmt, args...))
}
