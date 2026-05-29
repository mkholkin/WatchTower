package maintenance_service

import (
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	"WatchTower/internal/service/common/provider"
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
)

type MaintenanceService interface {
	GetAllMaintenanceWindows(ctx context.Context) ([]maintenance.MaintenanceWindow, error)
	GetMaintenanceWindow(ctx context.Context, windowId uuid.UUID) (*maintenance.MaintenanceWindow, error)
	CreateOneTimeMaintenanceWindow(ctx context.Context, dto CreateOneTimeMaintenanceWindowDTO) (*maintenance.MaintenanceWindow, error)
	CreateManualMaintenanceWindow(ctx context.Context, dto CreateManualMaintenanceWindowDTO) (*maintenance.MaintenanceWindow, error)
	AddMonitorToMaintenanceWindow(ctx context.Context, monitorID uuid.UUID, windowID uuid.UUID) error
	RemoveMonitorFromMaintenanceWindow(ctx context.Context, monitorID uuid.UUID, windowID uuid.UUID) error
	UpdateMaintenanceWindow(ctx context.Context, dto UpdateMaintenanceWindowDTO) error
	DeleteMaintenanceWindow(ctx context.Context, windowID uuid.UUID) error
}

type maintenanceService struct {
	MWRepo       repo.MaintenanceWindowRepository
	MonitorRepo  repo.MonitorRepository
	userProvider provider.UserProvider
	log          *slog.Logger
}

// NewMaintenanceService creates a new MaintenanceService.
func NewMaintenanceService(
	mwRepo repo.MaintenanceWindowRepository,
	monitorRepo repo.MonitorRepository,
	userProvider provider.UserProvider,
	logger *slog.Logger,
) MaintenanceService {
	return &maintenanceService{
		MWRepo:       mwRepo,
		MonitorRepo:  monitorRepo,
		userProvider: userProvider,
		log:          logger.With("service", "maintenance"),
	}
}

// CreateOneTimeMaintenanceWindow creates a one-time maintenance window with a fixed start and end time.
func (s *maintenanceService) CreateOneTimeMaintenanceWindow(
	ctx context.Context,
	dto CreateOneTimeMaintenanceWindowDTO,
) (*maintenance.MaintenanceWindow, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debug("creating one-time maintenance window", "user", usr.Login, "title", dto.Title)

	mw, err := maintenance.NewOneTimeMaintenanceWindow(
		usr,
		dto.Title,
		dto.Description,
		dto.StartTime,
		dto.EndTime,
	)
	if err != nil {
		s.log.Error("validation failed for one-time maintenance window", "error", err)
		return nil, err
	}

	if err := s.MWRepo.Create(ctx, mw); err != nil {
		s.log.Error("failed to create one-time maintenance window", "error", err)
		return nil, err
	}

	s.log.Debug("one-time maintenance window created", "id", mw.ID, "user", usr.Login)
	return mw, nil
}

// CreateManualMaintenanceWindow creates a manually controlled maintenance window (initially inactive).
func (s *maintenanceService) CreateManualMaintenanceWindow(
	ctx context.Context,
	dto CreateManualMaintenanceWindowDTO,
) (*maintenance.MaintenanceWindow, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debug("creating manual maintenance window", "user", usr.Login, "title", dto.Title)

	mw, err := maintenance.NewManualMaintenanceWindow(
		usr,
		dto.Title,
		dto.Description,
	)
	if err != nil {
		s.log.Error("validation failed for manual maintenance window", "error", err)
		return nil, err
	}

	if err := s.MWRepo.Create(ctx, mw); err != nil {
		s.log.Error("failed to create manual maintenance window", "error", err)
		return nil, err
	}

	s.log.Debug("manual maintenance window created", "id", mw.ID, "user", usr.Login)
	return mw, nil
}

func (s *maintenanceService) DeleteMaintenanceWindow(ctx context.Context, windowID uuid.UUID) error {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return err
	}

	s.log.Debug("attempting to delete maintenance window", "window_id", windowID, "user", usr.Login)

	mw, err := s.MWRepo.GetByID(ctx, windowID)
	if err != nil {
		s.log.Error("failed to get maintenance window for deletion", "window_id", windowID, "error", err)
		return err
	}

	if mw.User.Login != usr.Login {
		s.log.Warn("user attempted to delete maintenance window they do not own", "window_id", windowID, "user", usr.Login)
		return errors.Join(baseservice.ErrPermissionDenied, err)
	}

	if err := s.MWRepo.DeleteByID(ctx, mw.ID); err != nil {
		s.log.Error("failed to delete maintenance window", "window_id", windowID, "error", err)
		return err
	}

	s.log.Info("maintenance window deleted", "id", mw.ID, "user", usr.Login)
	return nil
}

// AddMonitorToMaintenanceWindow links a monitor to a maintenance window. If the monitor is already linked, it does nothing.
func (s *maintenanceService) AddMonitorToMaintenanceWindow(
	ctx context.Context,
	monitorID uuid.UUID,
	windowID uuid.UUID,
) error {
	s.log.Debug("linking monitor to maintenance window", "window_id", windowID, "monitor_id", monitorID)

	// Check if the maintenance window exists.
	maintenanceWindow, err := s.MWRepo.GetByID(ctx, windowID)
	if err != nil {
		s.log.Error("failed to get maintenance window", "window_id", windowID, "error", err)
		return err
	}

	// Check if the monitor exists.
	mon, err := s.MonitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		s.log.Error("failed to get monitor", "monitor_id", monitorID, "error", err)
		return err
	}

	// If the monitor is already linked to the maintenance window, do nothing.
	// TODO: optimize by checking the relation in the repository instead of fetching the whole monitor with all maintenance windows.
	for i := range mon.MaintenanceWindows {
		if mon.MaintenanceWindows[i].ID == windowID {
			s.log.Debug("monitor already linked to maintenance window", "window_id", windowID, "monitor_id", monitorID)
			return nil
		}
	}

	if err := s.MWRepo.LinkMonitor(ctx, maintenanceWindow, mon.ID); err != nil {
		s.log.Error("failed to link monitor to maintenance window", "window_id", windowID, "monitor_id", monitorID, "error", err)
		return err
	}

	s.log.Debug("monitor linked to maintenance window", "window_id", windowID, "monitor_id", monitorID)
	return nil
}

// RemoveMonitorFromMaintenanceWindow unlinks a monitor from a maintenance window. If the monitor is not linked, it does nothing.
func (s *maintenanceService) RemoveMonitorFromMaintenanceWindow(
	ctx context.Context,
	monitorID uuid.UUID,
	windowID uuid.UUID,
) error {
	s.log.Debug("unlinking monitor from maintenance window", "window_id", windowID, "monitor_id", monitorID)

	// Check if the maintenance window exists.
	maintenanceWindow, err := s.MWRepo.GetByID(ctx, windowID)
	if err != nil {
		s.log.Error("failed to get maintenance window", "window_id", windowID, "error", err)
		return err
	}

	// Check if the monitor exists.
	mon, err := s.MonitorRepo.GetByID(ctx, monitorID)
	if err != nil {
		s.log.Error("failed to get monitor", "monitor_id", monitorID, "error", err)
		return err
	}

	// If the monitor is not linked to the maintenance window, do nothing.
	// TODO: optimize by checking the relation in the repository instead of fetching the whole monitor with all maintenance windows.
	alreadyUnlinked := true
	for i := range mon.MaintenanceWindows {
		if mon.MaintenanceWindows[i].ID == windowID {
			alreadyUnlinked = false
			break
		}
	}
	if alreadyUnlinked {
		s.log.Debug("monitor already unlinked from maintenance window", "window_id", windowID, "monitor_id", monitorID)
		return nil
	}

	if err := s.MWRepo.UnlinkMonitor(ctx, maintenanceWindow, mon.ID); err != nil {
		s.log.Error("failed to unlink monitor from maintenance window", "window_id", windowID, "monitor_id", monitorID, "error", err)
		return err
	}

	s.log.Debug("monitor unlinked from maintenance window", "window_id", windowID, "monitor_id", monitorID)
	return nil
}

// UpdateMaintenanceWindow applies a partial update to a maintenance window.
// Supports renaming, changing description, rescheduling (one-time) and activating/deactivating (manual).
func (s *maintenanceService) UpdateMaintenanceWindow(
	ctx context.Context,
	dto UpdateMaintenanceWindowDTO,
) error {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return err
	}

	s.log.Debug("updating maintenance window", "window_id", dto.WindowID)

	mw, err := s.MWRepo.GetByID(ctx, dto.WindowID)
	if err != nil {
		s.log.Error("failed to get maintenance window for update", "window_id", dto.WindowID, "error", err)
		return err
	}

	if mw.User.Login != usr.Login {
		s.log.Warn("user attempted to update maintenance window they do not own", "window_id", dto.WindowID, "user", usr.Login)
		return errors.Join(baseservice.ErrPermissionDenied, err)
	}

	if err := mw.ApplyUpdate(maintenance.MaintenanceWindowUpdate{
		Title:        dto.Title,
		Description:  dto.Description,
		ConfigUpdate: dto.ConfigUpdate,
	}); err != nil {
		s.log.Error("failed to apply update to maintenance window", "window_id", dto.WindowID, "error", err)
		return err
	}

	if err := s.MWRepo.Update(ctx, mw); err != nil {
		s.log.Error("failed to persist maintenance window update", "window_id", dto.WindowID, "error", err)
		return err
	}

	s.log.Debug("maintenance window updated", "window_id", dto.WindowID)
	return nil
}

func (s *maintenanceService) GetAllMaintenanceWindows(ctx context.Context) ([]maintenance.MaintenanceWindow, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	windows, err := s.MWRepo.GetByUserLogin(ctx, usr.Login)
	if err != nil {
		s.log.Error("failed to get maintenance windows for user", "user", usr.Login, "error", err)
		return nil, err
	}

	return windows, nil
}

func (s *maintenanceService) GetMaintenanceWindow(ctx context.Context, windowId uuid.UUID) (*maintenance.MaintenanceWindow, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	window, err := s.MWRepo.GetByID(ctx, windowId)
	if err != nil {
		s.log.Error("failed to get maintenance window", "window_id", windowId, "error", err)
		return nil, err
	}

	if window.User.Login != usr.Login {
		s.log.Warn("user attempted to access maintenance window they do not own", "window_id", windowId, "user", usr.Login)
		return nil, errors.Join(baseservice.ErrPermissionDenied, err)
	}

	return window, nil
}
