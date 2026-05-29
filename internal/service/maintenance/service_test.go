package maintenance_service

import (
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func newMaintenanceTestService(ctrl *gomock.Controller) (*maintenanceService, *testmocks.MockMaintenanceWindowRepository, *testmocks.MockMonitorRepository, *testmocks.MockUserProvider) {
	mwRepo := testmocks.NewMockMaintenanceWindowRepository(ctrl)
	monitorRepo := testmocks.NewMockMonitorRepository(ctrl)
	provider := testmocks.NewMockUserProvider(ctrl)
	logger := testutil.NoopLogger()

	svc := NewMaintenanceService(mwRepo, monitorRepo, provider, logger).(*maintenanceService)
	return svc, mwRepo, monitorRepo, provider
}

func TestMaintenanceService_CreateOneTimeMaintenanceWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, mwRepo, _, provider := newMaintenanceTestService(ctrl)
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	mwRepo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&maintenance.MaintenanceWindow{})).Return(nil)

	start := time.Now().Add(10 * time.Minute)
	end := start.Add(time.Hour)

	_, err := svc.CreateOneTimeMaintenanceWindow(context.Background(), CreateOneTimeMaintenanceWindowDTO{
		Title:       "window",
		Description: "desc",
		StartTime:   start,
		EndTime:     end,
	})
	if err != nil {
		t.Fatalf("CreateOneTimeMaintenanceWindow() error = %v", err)
	}
}

func TestMaintenanceService_CreateManualMaintenanceWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, mwRepo, _, provider := newMaintenanceTestService(ctrl)
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	mwRepo.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&maintenance.MaintenanceWindow{})).Return(nil)

	_, err := svc.CreateManualMaintenanceWindow(context.Background(), CreateManualMaintenanceWindowDTO{Title: "window"})
	if err != nil {
		t.Fatalf("CreateManualMaintenanceWindow() error = %v", err)
	}
}

func TestMaintenanceService_AddMonitorToMaintenanceWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, mwRepo, monitorRepo, _ := newMaintenanceTestService(ctrl)
	windowID := uuid.New()
	monitorID := uuid.New()

	mw := &maintenance.MaintenanceWindow{ID: windowID}
	mon := &monitor.Monitor{ID: monitorID}

	mwRepo.EXPECT().GetByID(gomock.Any(), windowID).Return(mw, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, nil)
	mwRepo.EXPECT().UnlinkMonitor(gomock.Any(), mw, monitorID).Return(nil)

	err := svc.AddMonitorToMaintenanceWindow(context.Background(), monitorID, windowID)
	if err != nil {
		t.Fatalf("AddMonitorToMaintenanceWindow() error = %v", err)
	}
}

func TestMaintenanceService_RemoveMonitorFromMaintenanceWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, mwRepo, monitorRepo, _ := newMaintenanceTestService(ctrl)
	windowID := uuid.New()
	monitorID := uuid.New()

	mw := &maintenance.MaintenanceWindow{ID: windowID}
	mon := &monitor.Monitor{ID: monitorID, MaintenanceWindows: []maintenance.MaintenanceWindow{*mw}}

	mwRepo.EXPECT().GetByID(gomock.Any(), windowID).Return(mw, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), monitorID).Return(mon, nil)
	mwRepo.EXPECT().UnlinkMonitor(gomock.Any(), mw, monitorID).Return(nil)

	err := svc.RemoveMonitorFromMaintenanceWindow(context.Background(), monitorID, windowID)
	if err != nil {
		t.Fatalf("RemoveMonitorFromMaintenanceWindow() error = %v", err)
	}
}

func TestMaintenanceService_UpdateMaintenanceWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, mwRepo, _, provider := newMaintenanceTestService(ctrl)
	windowID := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	mw := &maintenance.MaintenanceWindow{
		ID:     windowID,
		User:   &user.User{Login: "alice"},
		Title:  "old",
		Config: maintenance.ManualMaintenanceWindowConfig{Active: false},
	}
	mwRepo.EXPECT().GetByID(gomock.Any(), windowID).Return(mw, nil)
	mwRepo.EXPECT().Update(gomock.Any(), mw).Return(nil)

	err := svc.UpdateMaintenanceWindow(context.Background(), UpdateMaintenanceWindowDTO{WindowID: windowID})
	if err != nil {
		t.Fatalf("UpdateMaintenanceWindow() error = %v", err)
	}
}

func TestMaintenanceService_DeleteMaintenanceWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, mwRepo, _, provider := newMaintenanceTestService(ctrl)
	windowID := uuid.New()

	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	mw := &maintenance.MaintenanceWindow{ID: windowID, User: &user.User{Login: "alice"}}
	mwRepo.EXPECT().GetByID(gomock.Any(), windowID).Return(mw, nil)
	mwRepo.EXPECT().DeleteByID(gomock.Any(), windowID).Return(nil)

	err := svc.DeleteMaintenanceWindow(context.Background(), windowID)
	if err != nil {
		t.Fatalf("DeleteMaintenanceWindow() error = %v", err)
	}
}
