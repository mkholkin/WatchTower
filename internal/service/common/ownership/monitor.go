package ownership

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service"
	"WatchTower/internal/service/common/provider"
	"context"
	"fmt"

	"github.com/google/uuid"
)

// GetOwnedMonitor returns the monitor only if it belongs to the authorized user.
func GetOwnedMonitor(
	ctx context.Context,
	userProvider provider.UserProvider,
	monitorRepo repo.MonitorRepository,
	monitorID uuid.UUID,
) (*monitor.Monitor, error) {
	usr, err := userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	mon, err := monitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	if usr.Login != mon.User.Login {
		return nil, fmt.Errorf("user doesn't own monitor %w", service.ErrPermissionDenied)
	}

	return mon, nil
}
