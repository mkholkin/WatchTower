package monitor

import (
	"WatchTower/internal/service"
	"fmt"
)

func wrapValidation(reason string) error {
	return fmt.Errorf("%w: %s", service.ErrInvalidData, reason)
}
