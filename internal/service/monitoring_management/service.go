package monitoring_service

import (
	"WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service"
	"WatchTower/internal/service/common/ownership"
	"WatchTower/internal/service/common/provider"
	healthchecksvc "WatchTower/internal/service/healthcheck"
	monitordto "WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/internal/service/monitoring_management/mappers"
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

type MonitoringManagementService interface {
	GetAllMonitors(ctx context.Context) ([]*monitor.Monitor, error)
	GetMonitor(ctx context.Context, monitorId uuid.UUID) (*monitor.Monitor, error)
	DisableMonitor(ctx context.Context, monitorId uuid.UUID) error
	EnableMonitor(ctx context.Context, monitorId uuid.UUID) error
	DeleteMonitor(ctx context.Context, monitorId uuid.UUID) error
	CreateMonitor(ctx context.Context, createDTO monitordto.CreateMonitorDTO) (*monitor.Monitor, error)
	UpdateMonitor(ctx context.Context, dto monitordto.UpdateMonitorDTO) error
	LinkAlertContact(ctx context.Context, monitorID uuid.UUID, alertContactID uuid.UUID) error
	UnlinkAlertContact(ctx context.Context, monitorID uuid.UUID, alertContactID uuid.UUID) error
}

type monitoringManagementService struct {
	monitorRepo          repo.MonitorRepository
	targetRepo           repo.TargetRepository
	alertContactRepo     repo.AlertContactRepository
	windowRepo           repo.MaintenanceWindowRepository
	userProvider         provider.UserProvider
	publisher            message.Publisher
	networkConfigMappers map[target.Protocol]monitordto.NetworkConfigMapper
	expectationsMappers  map[target.Protocol]monitordto.ExpectationsMapper
	log                  *slog.Logger
}

// NewMonitoringManagementService creates a new MonitoringManagementService.
func NewMonitoringManagementService(
	monitorRepo repo.MonitorRepository,
	targetRepo repo.TargetRepository,
	alertContactRepo repo.AlertContactRepository,
	windowRepo repo.MaintenanceWindowRepository,
	userProvider provider.UserProvider,
	publisher message.Publisher,
	logger *slog.Logger,
) MonitoringManagementService {
	return &monitoringManagementService{
		monitorRepo:      monitorRepo,
		targetRepo:       targetRepo,
		alertContactRepo: alertContactRepo,
		windowRepo:       windowRepo,
		userProvider:     userProvider,
		publisher:        publisher,
		networkConfigMappers: map[target.Protocol]monitordto.NetworkConfigMapper{
			target.ProtocolHTTP: mappers.HTTPNetworkConfigMapper{},
			target.ProtocolTCP:  mappers.TCPNetworkConfigMapper{},
			target.ProtocolICMP: mappers.ICMPNetworkConfigMapper{},
		},
		expectationsMappers: map[target.Protocol]monitordto.ExpectationsMapper{
			target.ProtocolHTTP: mappers.HTTPExpectationsMapper{},
			target.ProtocolTCP:  mappers.TCPExpectationsMapper{},
			target.ProtocolICMP: mappers.ICMPExpectationsMapper{},
		},
		log: logger.With("service", "monitoring_management"),
	}
}

// GetAllMonitors returns all monitors obtained by authorized user.
func (s *monitoringManagementService) GetAllMonitors(ctx context.Context) ([]*monitor.Monitor, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}
	slog.Debug("getting all monitors for user", "user", usr.Login)

	monitors, err := s.monitorRepo.GetAllByUser(ctx, usr)
	if err != nil {
		slog.Error("failed to get all monitors for user", "user", usr, "error", err)
		return nil, err
	}

	return monitors, nil
}

func (s *monitoringManagementService) GetMonitor(ctx context.Context, monitorId uuid.UUID) (*monitor.Monitor, error) {
	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}
	slog.Debug("getting monitor", "monitor_id", monitorId, "user", usr.Login)

	mon, err := s.monitorRepo.GetByID(ctx, monitorId)
	if err != nil {
		slog.Error("failed to get monitor", "monitor_id", monitorId, "error", err)
		return nil, err
	}

	if mon.User.Login != usr.Login {
		slog.Warn("user attempted to access monitor they do not own", "monitor_id", monitorId, "user", usr.Login)
		return nil, errors.Join(service.ErrPermissionDenied, errors.New("monitor does not belong to user"))
	}

	return mon, nil
}

func (s *monitoringManagementService) getOwnedMonitor(ctx context.Context, monitorID uuid.UUID) (*monitor.Monitor, error) {
	mon, err := ownership.GetOwnedMonitor(ctx, s.userProvider, s.monitorRepo, monitorID)
	if err != nil {
		s.log.Error("failed to get monitor", "monitor_id", monitorID, "error", err)
		return nil, err
	}

	return mon, nil
}

// DisableMonitor disables monitor with the given ID. If the monitor is already disabled, it does nothing.
func (s *monitoringManagementService) DisableMonitor(ctx context.Context, monitorId uuid.UUID) error {
	s.log.Debug("disabling monitor", "monitor_id", monitorId)

	mon, err := s.getOwnedMonitor(ctx, monitorId)
	if err != nil {
		return err
	}

	if !mon.IsActive {
		s.log.Debug("monitor already disabled", "monitor_id", monitorId)
		return nil
	}

	mon.Disable()

	if err := s.monitorRepo.Disable(ctx, monitorId); err != nil {
		s.log.Error("failed to disable monitor", "monitor_id", monitorId, "error", err)
		return err
	}

	if err := s.maintainTargetConsistency(ctx, mon.Target); err != nil {
		return err
	}

	s.log.Info("monitor disabled", "monitor_id", monitorId)
	return nil
}

// EnableMonitor enables monitor with the given ID. If the monitor is already enabled, it does nothing.
func (s *monitoringManagementService) EnableMonitor(ctx context.Context, monitorId uuid.UUID) error {
	s.log.Debug("enabling monitor", "monitor_id", monitorId)

	mon, err := s.getOwnedMonitor(ctx, monitorId)
	if err != nil {
		return err
	}

	if mon.IsActive {
		s.log.Debug("monitor already enabled", "monitor_id", monitorId)
		return nil
	}

	mon.Enable()

	if err := s.monitorRepo.Enable(ctx, monitorId); err != nil {
		s.log.Error("failed to enable monitor", "monitor_id", monitorId, "error", err)
		return err
	}

	if err := s.syncTargetAfterMonitorLinkage(ctx, mon.Target, mon.ProbeIntervalSec); err != nil {
		return nil
	}

	s.log.Info("monitor enabled", "monitor_id", monitorId)
	return nil
}

// DeleteMonitor deletes monitor with the given ID. It checks if the user with given login owns the monitor.
func (s *monitoringManagementService) DeleteMonitor(ctx context.Context, monitorId uuid.UUID) error {
	s.log.Debug("deleting monitor", "monitor_id", monitorId)

	mon, err := s.getOwnedMonitor(ctx, monitorId)
	if err != nil {
		return err
	}

	if err := s.monitorRepo.DeleteByID(ctx, monitorId); err != nil {
		s.log.Error("failed to delete monitor", "monitor_id", monitorId, "error", err)
		return err
	}
	_ = mon

	if err := s.maintainTargetConsistency(ctx, mon.Target); err != nil {
		return err
	}

	s.log.Info("monitor deleted", "monitor_id", monitorId)
	return nil
}

// CreateMonitor creates a new monitor.
func (s *monitoringManagementService) CreateMonitor(ctx context.Context, createDTO monitordto.CreateMonitorDTO) (*monitor.Monitor, error) {
	s.log.Debug("creating monitor", "name", createDTO.Label, "endpoint", createDTO.Endpoint)

	usr, err := s.userProvider.GetAuthorizedUser(ctx)
	if err != nil {
		return nil, err
	}

	networkConfig, expectations, err := s.resolveCreateMonitorConfig(createDTO)
	if err != nil {
		return nil, err
	}

	windows, alertContacts, err := s.loadCreateMonitorRelations(ctx, createDTO)
	if err != nil {
		return nil, err
	}

	tgt, err := s.getOrCreateTarget(ctx, createDTO.Endpoint, createDTO.ProbeIntervalSec, networkConfig)
	if err != nil {
		return nil, err
	}

	mon, err := s.createAndStoreMonitor(ctx, createDTO, tgt, alertContacts, windows, expectations, usr)
	if err != nil {
		return nil, err
	}

	if err := s.syncTargetAfterMonitorLinkage(ctx, tgt, createDTO.ProbeIntervalSec); err != nil {
		return nil, err
	}

	s.log.Info("monitor created", "monitor_id", mon.ID, "target_id", tgt.ID)
	return mon, nil
}

func (s *monitoringManagementService) resolveCreateMonitorConfig(createDTO monitordto.CreateMonitorDTO) (target.NetworkConfig, monitor.Expectations, error) {
	if err := createDTO.ValidateProtocolConsistency(); err != nil {
		return nil, nil, err
	}

	networkConfig, err := createDTO.ToDomainNetworkConfig(s.networkConfigMappers)
	if err != nil {
		return nil, nil, err
	}

	expectations, err := createDTO.ToDomainExpectations(s.expectationsMappers)
	if err != nil {
		return nil, nil, err
	}

	return networkConfig, expectations, nil
}

func (s *monitoringManagementService) loadCreateMonitorRelations(
	ctx context.Context,
	createDTO monitordto.CreateMonitorDTO,
) ([]maintenance.MaintenanceWindow, []alert.Contact, error) {
	windows, err := s.windowRepo.GetByIDBulk(ctx, createDTO.MaintenanceWindowIDs)
	if err != nil {
		return nil, nil, err
	}

	alertContacts, err := s.alertContactRepo.GetByIDBulk(ctx, createDTO.AlertContactIDs)
	if err != nil {
		return nil, nil, err
	}

	return windows, alertContacts, nil
}

func (s *monitoringManagementService) createAndStoreMonitor(
	ctx context.Context,
	createDTO monitordto.CreateMonitorDTO,
	target *target.Target,
	alertContacts []alert.Contact,
	windows []maintenance.MaintenanceWindow,
	expectations monitor.Expectations,
	usr *user.User,
) (*monitor.Monitor, error) {
	mon, err := monitor.NewMonitor(createDTO.Label, target, usr, alertContacts, windows, createDTO.ProbeIntervalSec, expectations)
	if err != nil {
		return nil, err
	}

	if err := s.monitorRepo.Create(ctx, mon); err != nil {
		return nil, err
	}

	return mon, nil
}

func (s *monitoringManagementService) syncTargetAfterMonitorLinkage(ctx context.Context, target *target.Target, newProbeInterval int32) error {
	targetUpdated := false

	// If no new target was created but it was disabled earlier due to no monitors referencing it, we need to enable it again.
	if !target.IsActive {
		target.IsActive = true
		targetUpdated = true
	}

	// If the new monitor has smaller probe interval than found target, we need to update target's probe interval.
	if newProbeInterval < target.ProbeIntervalSec {
		if err := target.UpdateProbeInterval(newProbeInterval); err != nil {
			return err
		}
		targetUpdated = true
	}

	if targetUpdated {
		// TODO: пока так потом возможно записывать только измененные поля
		if err := s.targetRepo.Update(ctx, target); err != nil {
			return err
		}

		if err := s.publishTargetEvent(healthchecksvc.TopicTargetUpdated, target.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *monitoringManagementService) UpdateMonitor(ctx context.Context, dto monitordto.UpdateMonitorDTO) error {
	s.log.Debug("updating monitor", "monitor_id", dto.ID)

	mon, err := s.getOwnedMonitor(ctx, dto.ID)
	if err != nil {
		return err
	}

	targetChanged, endpoint, probeInterval, networkConfig, err := s.resolveTargetUpdateParams(mon, dto)
	if err != nil {
		return err
	}

	if targetChanged {
		if err := s.applyTargetChanges(ctx, mon, endpoint, probeInterval, networkConfig); err != nil {
			return err
		}
	}

	if err := s.applyMonitorSimpleFields(mon, dto); err != nil {
		return err
	}

	if err := s.monitorRepo.Update(ctx, mon); err != nil {
		s.log.Error("failed to update monitor", "monitor_id", mon.ID, "error", err)
		return err
	}

	s.log.Info("monitor updated successfully", "monitor_id", mon.ID)
	return nil
}

func (s *monitoringManagementService) resolveTargetUpdateParams(
	mon *monitor.Monitor,
	dto monitordto.UpdateMonitorDTO,
) (bool, string, int32, target.NetworkConfig, error) {
	targetChanged := false
	oldTarget := mon.Target

	endpoint := oldTarget.Endpoint
	if dto.Endpoint != nil && *dto.Endpoint != oldTarget.Endpoint {
		endpoint = *dto.Endpoint
		targetChanged = true
	}

	probeInterval := mon.ProbeIntervalSec
	if dto.ProbeIntervalSec != nil && *dto.ProbeIntervalSec != mon.ProbeIntervalSec {
		probeInterval = *dto.ProbeIntervalSec
		targetChanged = true
	}

	networkConfig := oldTarget.Config
	if dto.NetworkConfig != nil || dto.Protocol != nil {
		nwCfg, err := dto.ToDomainNetworkConfig(s.networkConfigMappers)
		if err != nil {
			return false, "", 0, nil, err
		}
		if nwCfg != nil {
			networkConfig = nwCfg
			targetChanged = true
		}
	}

	return targetChanged, endpoint, probeInterval, networkConfig, nil
}

func (s *monitoringManagementService) applyTargetChanges(
	ctx context.Context,
	mon *monitor.Monitor,
	endpoint string,
	probeInterval int32,
	networkConfig target.NetworkConfig,
) error {
	oldTarget := mon.Target

	newTgt, err := s.getOrCreateTarget(ctx, endpoint, probeInterval, networkConfig)
	if err != nil {
		return err
	}

	mon.Target = newTgt
	mon.ProbeIntervalSec = probeInterval

	if err := s.syncTargetAfterMonitorLinkage(ctx, newTgt, probeInterval); err != nil {
		return err
	}

	if oldTarget.ID != newTgt.ID {
		if err := s.maintainTargetConsistency(ctx, oldTarget); err != nil {
			return err
		}
	}

	return nil
}

func (s *monitoringManagementService) applyMonitorSimpleFields(
	mon *monitor.Monitor,
	dto monitordto.UpdateMonitorDTO,
) error {
	if dto.Label != nil {
		mon.Label = *dto.Label
	}

	if dto.Expectations != nil {
		exp, err := dto.ToDomainExpectations(s.expectationsMappers)
		if err != nil {
			return err
		}
		if exp != nil {
			mon.Expectations = exp
		}
	}

	return nil
}

// getOrCreateTarget returns the target matching the given endpoint and network config, or creates a new one if no such target exists.
func (s *monitoringManagementService) getOrCreateTarget(
	ctx context.Context,
	endpoint string,
	probeIntervalSec int32,
	config target.NetworkConfig,
) (*target.Target, error) {
	targetHash := target.ComputeHash(endpoint, config)
	tgt, err := s.targetRepo.GetByHash(ctx, targetHash)

	switch {
	case errors.Is(err, repo.ErrNotFound):
		if tgt, err = target.NewTarget(endpoint, probeIntervalSec, config); err != nil {
			return nil, err
		}
		if err := s.targetRepo.Create(ctx, tgt); err != nil {
			return nil, err
		}
		if err := s.publishTargetEvent(healthchecksvc.TopicTargetCreated, tgt.ID); err != nil {
			return nil, err
		}
	case err != nil:
		return tgt, nil
	}

	return tgt, nil
}

func (s *monitoringManagementService) publishTargetEvent(topic string, targetID uuid.UUID) error {
	if s.publisher == nil {
		s.log.Debug("publisher is nil, skip target event", "topic", topic, "target_id", targetID)
		return nil
	}

	payload, err := json.Marshal(healthchecksvc.TargetEvent{ID: targetID})
	if err != nil {
		s.log.Error("marshal target event failed", "target_id", targetID, "error", err)
		return errors.Join(service.ErrEventMarshalFailed, err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	if err := s.publisher.Publish(topic, msg); err != nil {
		return errors.Join(service.ErrEventPublishFailed, err)
		return service.ErrEventPublishFailed
	}

	s.log.Debug("target event published", "topic", topic, "target_id", targetID)
	return nil
}

// maintainTargetConsistency checks if the target with the given ID is still referenced by any active monitors.
// If not, target can be safely disabled.
// If there are any active monitors, sets minimal probe interval of active monitors.
func (s *monitoringManagementService) maintainTargetConsistency(ctx context.Context, target *target.Target) error {
	targetUpdated := false

	associatedMonitors, err := s.monitorRepo.GetAllByTargetID(ctx, target.ID)
	if err != nil {
		return err
	}

	if len(associatedMonitors) == 0 {
		target.IsActive = false
		targetUpdated = true
		if err := s.targetRepo.Disable(ctx, target.ID); err != nil {
			return err
		}
	} else {
		minProbeInterval := associatedMonitors[0].ProbeIntervalSec
		for _, mon := range associatedMonitors[1:] {
			if mon.ProbeIntervalSec < minProbeInterval {
				minProbeInterval = mon.ProbeIntervalSec
			}
		}
		if minProbeInterval != target.ProbeIntervalSec {
			target.ProbeIntervalSec = minProbeInterval
			targetUpdated = true
		}
	}

	if targetUpdated {
		// TODO: пока так, потом возможно записывать только измененные поля
		if err := s.targetRepo.Update(ctx, target); err != nil {
			return err
		}

		if err := s.publishTargetEvent(healthchecksvc.TopicTargetUpdated, target.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *monitoringManagementService) getValidatedMonitorAndContact(
	ctx context.Context,
	monitorID uuid.UUID,
	alertContactID uuid.UUID,
) (*monitor.Monitor, *alert.Contact, error) {
	mon, err := s.getOwnedMonitor(ctx, monitorID)
	if err != nil {
		return nil, nil, err
	}

	contact, err := s.alertContactRepo.GetByID(ctx, alertContactID)
	if err != nil {
		s.log.Error("failed to get alert contact", "alert_contact_id", alertContactID, "error", err)
		return nil, nil, err
	}

	return mon, contact, nil
}

func (s *monitoringManagementService) LinkAlertContact(
	ctx context.Context,
	monitorID uuid.UUID,
	alertContactID uuid.UUID,
) error {
	s.log.Debug(
		"linking alert contact to mon",
		"monitor_id", monitorID,
		"alert_contact_id", alertContactID,
	)

	mon, contact, err := s.getValidatedMonitorAndContact(ctx, monitorID, alertContactID)
	if err != nil {
		return err
	}

	if err := s.monitorRepo.AddAlertContact(ctx, mon, contact); err != nil {
		s.log.Error(
			"failed to link alert contact to mon",
			"monitor_id", monitorID,
			"alert_contact_id", alertContactID,
			"error", err,
		)
		return err
	}

	s.log.Info(
		"alert contact linked to mon",
		"monitor_id", monitorID,
		"alert_contact_id", alertContactID,
	)
	return nil
}

func (s *monitoringManagementService) UnlinkAlertContact(
	ctx context.Context,
	monitorID uuid.UUID,
	alertContactID uuid.UUID,
) error {
	s.log.Debug(
		"unlinking alert contact from mon",
		"monitor_id", monitorID,
		"alert_contact_id", alertContactID,
	)

	mon, contact, err := s.getValidatedMonitorAndContact(ctx, monitorID, alertContactID)
	if err != nil {
		return err
	}

	if err := s.monitorRepo.RemoveAlertContact(ctx, mon, contact); err != nil {
		s.log.Error(
			"failed to unlink alert contact from mon",
			"monitor_id", monitorID,
			"alert_contact_id", alertContactID,
			"error", err,
		)
		return err
	}

	s.log.Info(
		"alert contact unlinked from mon",
		"monitor_id", monitorID,
		"alert_contact_id", alertContactID,
	)
	return nil
}
