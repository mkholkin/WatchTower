package monitoring_service

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	monitordto "WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func newMonitoringManagementTestService(ctrl *gomock.Controller) (
	*monitoringManagementService,
	*testmocks.MockMonitorRepository,
	*testmocks.MockTargetRepository,
	*testmocks.MockAlertContactRepository,
	*testmocks.MockMaintenanceWindowRepository,
	*testmocks.MockUserProvider,
) {
	monitorRepo := testmocks.NewMockMonitorRepository(ctrl)
	targetRepo := testmocks.NewMockTargetRepository(ctrl)
	alertRepo := testmocks.NewMockAlertContactRepository(ctrl)
	windowRepo := testmocks.NewMockMaintenanceWindowRepository(ctrl)
	provider := testmocks.NewMockUserProvider(ctrl)
	logger := testutil.NoopLogger()

	svc := NewMonitoringManagementService(
		monitorRepo,
		targetRepo,
		alertRepo,
		windowRepo,
		provider,
		nil,
		logger,
	).(*monitoringManagementService)

	return svc, monitorRepo, targetRepo, alertRepo, windowRepo, provider
}

func TestMonitoringManagementService_GetAllMonitors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, _, _, provider := newMonitoringManagementTestService(ctrl)
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetAllByUser(gomock.Any(), gomock.AssignableToTypeOf(&user.User{})).Return([]*monitor.Monitor{{Label: "m1"}}, nil)

	mons, err := svc.GetAllMonitors(context.Background())
	if err != nil {
		t.Fatalf("GetAllMonitors() error = %v", err)
	}
	if len(mons) != 1 {
		t.Fatalf("GetAllMonitors() len = %d, want 1", len(mons))
	}
}

func TestMonitoringManagementService_DisableMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, _, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(&monitor.Monitor{ID: monitorID, IsActive: false, User: &user.User{Login: "alice"}}, nil)

	if err := svc.DisableMonitor(context.Background(), monitorID); err != nil {
		t.Fatalf("DisableMonitor() error = %v", err)
	}
}

func TestMonitoringManagementService_EnableMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, _, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(&monitor.Monitor{ID: monitorID, IsActive: true, User: &user.User{Login: "alice"}}, nil)

	if err := svc.EnableMonitor(context.Background(), monitorID); err != nil {
		t.Fatalf("EnableMonitor() error = %v", err)
	}
}

func TestMonitoringManagementService_DeleteMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, _, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()
	targetID := uuid.New()
	tgt := &target.Target{ID: targetID, ProbeIntervalSec: 10, IsActive: true}

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(&monitor.Monitor{ID: monitorID, User: &user.User{Login: "alice"}, Target: tgt}, nil)
	monitorRepo.EXPECT().DeleteByID(gomock.Any(), monitorID).Return(nil)
	monitorRepo.EXPECT().GetAllByTargetID(gomock.Any(), targetID).Return([]*monitor.Monitor{{ProbeIntervalSec: 10}}, nil)

	if err := svc.DeleteMonitor(context.Background(), monitorID); err != nil {
		t.Fatalf("DeleteMonitor() error = %v", err)
	}
}

func TestMonitoringManagementService_CreateMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, targetRepo, alertRepo, windowRepo, provider := newMonitoringManagementTestService(ctrl)
	existingTarget := &target.Target{ID: uuid.New(), Endpoint: "example.com", ProbeIntervalSec: 5, IsActive: true, Config: target.HTTPConfig{}}

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	windowRepo.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]maintenance.MaintenanceWindow{}, nil)
	alertRepo.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]alert.Contact{}, nil)
	targetRepo.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(existingTarget, nil)
	monitorRepo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&monitor.Monitor{})).Return(nil)

	_, err := svc.CreateMonitor(context.Background(), monitordto.CreateMonitorDTO{
		Label:            "test",
		Endpoint:         "example.com",
		ProbeIntervalSec: 10,
		NetworkConfig: monitordto.HTTPMonitorNetworkConfig{
			Method: "GET",
		},
		Expectations: monitordto.HTTPMonitorExpectations{MaxLatencyMs: 1000},
	})
	if err != nil {
		t.Fatalf("CreateMonitor() error = %v", err)
	}
}

func TestMonitoringManagementService_CreateMonitorDoesNotCommitBeforeTargetSync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, targetRepo, alertRepo, windowRepo, provider := newMonitoringManagementTestService(ctrl)
	existingTarget := &target.Target{ID: uuid.New(), Endpoint: "https://example.com", ProbeIntervalSec: 60, IsActive: false, Config: target.HTTPConfig{Method: "GET"}}
	syncErr := errors.New("target update failed")
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	windowRepo.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]maintenance.MaintenanceWindow{}, nil)
	alertRepo.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]alert.Contact{}, nil)
	targetRepo.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(existingTarget, nil)
	gomock.InOrder(
		targetRepo.EXPECT().Update(gomock.Any(), existingTarget).Return(syncErr),
		targetRepo.EXPECT().Update(gomock.Any(), existingTarget).Return(nil),
	)

	_, err := svc.CreateMonitor(context.Background(), monitordto.CreateMonitorDTO{
		Label: "test", Endpoint: existingTarget.Endpoint, ProbeIntervalSec: 10,
		NetworkConfig: monitordto.HTTPMonitorNetworkConfig{Method: "GET"},
		Expectations:  monitordto.HTTPMonitorExpectations{StatusCodes: []int{200}, MaxLatencyMs: 1000},
	})
	if !errors.Is(err, syncErr) {
		t.Fatalf("CreateMonitor() error = %v, want target sync error", err)
	}
}

func TestMonitoringManagementService_CreateMonitorRemovesNewTargetOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, targetRepo, alertRepo, windowRepo, provider := newMonitoringManagementTestService(ctrl)
	createErr := errors.New("monitor create failed")
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	windowRepo.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]maintenance.MaintenanceWindow{}, nil)
	alertRepo.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]alert.Contact{}, nil)
	targetRepo.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(nil, repo.ErrNotFound)
	targetRepo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&target.Target{})).Return(nil)
	monitorRepo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&monitor.Monitor{})).Return(createErr)
	targetRepo.EXPECT().DeleteByID(gomock.Any(), gomock.Any()).Return(nil)

	_, err := svc.CreateMonitor(context.Background(), monitordto.CreateMonitorDTO{
		Label: "test", Endpoint: "https://new.example.com", ProbeIntervalSec: 10,
		NetworkConfig: monitordto.HTTPMonitorNetworkConfig{Method: "GET"},
		Expectations:  monitordto.HTTPMonitorExpectations{StatusCodes: []int{200}, MaxLatencyMs: 1000},
	})
	if !errors.Is(err, createErr) {
		t.Fatalf("CreateMonitor() error = %v, want monitor create error", err)
	}
}

func TestMonitoringManagementService_UpdateMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, targetRepo, _, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()
	mon := &monitor.Monitor{ID: monitorID, User: &user.User{Login: "alice"}, Target: &target.Target{ID: uuid.New(), Endpoint: "old", ProbeIntervalSec: 10}, IsActive: true}
	disabled := false

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, nil)
	monitorRepo.EXPECT().Update(gomock.Any(), mon).Return(nil)
	monitorRepo.EXPECT().GetAllByTargetID(gomock.Any(), mon.Target.ID).Return(nil, nil)
	targetRepo.EXPECT().Disable(gomock.Any(), mon.Target.ID).Return(nil)
	targetRepo.EXPECT().Update(gomock.Any(), mon.Target).Return(nil)

	err := svc.UpdateMonitor(context.Background(), monitordto.UpdateMonitorDTO{ID: monitorID, IsEnabled: &disabled})
	if err != nil {
		t.Fatalf("UpdateMonitor() error = %v", err)
	}
	if mon.IsActive {
		t.Fatal("UpdateMonitor() left monitor enabled")
	}
}

func TestMonitoringManagementService_LinkAlertContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, alertRepo, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()
	contactID := uuid.New()
	mon := &monitor.Monitor{ID: monitorID, User: &user.User{Login: "alice"}}
	contact := &alert.Contact{ID: contactID, User: &user.User{Login: "alice"}}

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, nil)
	alertRepo.EXPECT().GetByID(gomock.Any(), contactID).Return(contact, nil)
	monitorRepo.EXPECT().AddAlertContact(gomock.Any(), mon, contact).Return(nil)

	if err := svc.LinkAlertContact(context.Background(), monitorID, contactID); err != nil {
		t.Fatalf("LinkAlertContact() error = %v", err)
	}
}

func TestMonitoringManagementService_LinkAlertContactRejectsForeignContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, alertRepo, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()
	contactID := uuid.New()
	mon := &monitor.Monitor{ID: monitorID, User: &user.User{Login: "alice"}}
	contact := &alert.Contact{ID: contactID, User: &user.User{Login: "bob"}}

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, nil)
	alertRepo.EXPECT().GetByID(gomock.Any(), contactID).Return(contact, nil)

	err := svc.LinkAlertContact(context.Background(), monitorID, contactID)
	if !errors.Is(err, baseservice.ErrPermissionDenied) {
		t.Fatalf("LinkAlertContact() error = %v, want ErrPermissionDenied", err)
	}
}

func TestMonitoringManagementService_CreateMonitorRejectsForeignRelations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, alertRepo, windowRepo, provider := newMonitoringManagementTestService(ctrl)
	contactID := uuid.New()
	windowID := uuid.New()
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	windowRepo.EXPECT().GetByIDBulk(gomock.Any(), []uuid.UUID{windowID}).Return([]maintenance.MaintenanceWindow{{ID: windowID, User: &user.User{Login: "alice"}}}, nil)
	alertRepo.EXPECT().GetByIDBulk(gomock.Any(), []uuid.UUID{contactID}).Return([]alert.Contact{{ID: contactID, User: &user.User{Login: "bob"}}}, nil)

	_, err := svc.CreateMonitor(context.Background(), monitordto.CreateMonitorDTO{
		Label: "test", Endpoint: "https://example.com", ProbeIntervalSec: 10,
		AlertContactIDs: []uuid.UUID{contactID}, MaintenanceWindowIDs: []uuid.UUID{windowID},
		NetworkConfig: monitordto.HTTPMonitorNetworkConfig{Method: "GET"},
		Expectations:  monitordto.HTTPMonitorExpectations{StatusCodes: []int{200}, MaxLatencyMs: 1000},
	})
	if !errors.Is(err, baseservice.ErrPermissionDenied) {
		t.Fatalf("CreateMonitor() error = %v, want ErrPermissionDenied", err)
	}
}

func TestMonitoringManagementService_UnlinkAlertContact(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, alertRepo, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()
	contactID := uuid.New()
	mon := &monitor.Monitor{ID: monitorID, User: &user.User{Login: "alice"}}
	contact := &alert.Contact{ID: contactID, User: &user.User{Login: "alice"}}

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, nil)
	alertRepo.EXPECT().GetByID(gomock.Any(), contactID).Return(contact, nil)
	monitorRepo.EXPECT().RemoveAlertContact(gomock.Any(), mon, contact).Return(nil)

	if err := svc.UnlinkAlertContact(context.Background(), monitorID, contactID); err != nil {
		t.Fatalf("UnlinkAlertContact() error = %v", err)
	}
}
