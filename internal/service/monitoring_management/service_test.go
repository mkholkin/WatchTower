package service

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	monitordto "WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
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

	err := svc.CreateMonitor(context.Background(), monitordto.CreateMonitorDTO{
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

func TestMonitoringManagementService_UpdateMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, monitorRepo, _, _, _, provider := newMonitoringManagementTestService(ctrl)
	monitorID := uuid.New()
	mon := &monitor.Monitor{ID: monitorID, User: &user.User{Login: "alice"}, Target: &target.Target{ID: uuid.New(), Endpoint: "old", ProbeIntervalSec: 10}}

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, nil)
	monitorRepo.EXPECT().Update(gomock.Any(), mon).Return(nil)

	err := svc.UpdateMonitor(context.Background(), monitordto.UpdateMonitorDTO{ID: monitorID})
	if err != nil {
		t.Fatalf("UpdateMonitor() error = %v", err)
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
