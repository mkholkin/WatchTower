package maintenance_service

import (
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func maintenanceTestService(t *testing.T) (*maintenanceService, *testmocks.MockMaintenanceWindowRepository, *testmocks.MockMonitorRepository, *testmocks.MockUserProvider) {
	t.Helper()
	ctrl := gomock.NewController(t)
	windows := testmocks.NewMockMaintenanceWindowRepository(ctrl)
	monitors := testmocks.NewMockMonitorRepository(ctrl)
	provider := testmocks.NewMockUserProvider(ctrl)
	return NewMaintenanceService(windows, monitors, provider, testutil.NoopLogger()).(*maintenanceService), windows, monitors, provider
}

func TestMaintenanceCreateOneTimeWindow(t *testing.T) {
	t.Run("persists valid future window", func(t *testing.T) {
		testutil.Case(t, "maintenance", "CreateOneTimeMaintenanceWindow", "boundary", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, _, provider := maintenanceTestService(t)
			start, end := time.Now().Add(time.Hour), time.Now().Add(2*time.Hour)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *maintenance.MaintenanceWindow) error {
				cfg, ok := got.Config.(maintenance.OneTimeMaintenanceWindowConfig)
				if got.User.Login != "alice" || got.Title != "deploy" || !ok || !cfg.StartTime.Equal(start) || !cfg.EndTime.Equal(end) {
					t.Fatalf("persisted window = %#v, config=%#v", got, got.Config)
				}
				return nil
			})
			got, err := svc.CreateOneTimeMaintenanceWindow(context.Background(), CreateOneTimeMaintenanceWindowDTO{Title: "deploy", StartTime: start, EndTime: end})
			if err != nil || got == nil {
				t.Fatalf("CreateOneTimeMaintenanceWindow() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("rejects end before start", func(t *testing.T) {
		testutil.Case(t, "maintenance", "CreateOneTimeMaintenanceWindow", "boundary", "london", func(t *testing.T, _ *allure.Context) {
			svc, _, _, provider := maintenanceTestService(t)
			start := time.Now().Add(time.Hour)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			if got, err := svc.CreateOneTimeMaintenanceWindow(context.Background(), CreateOneTimeMaintenanceWindowDTO{Title: "deploy", StartTime: start, EndTime: start}); err == nil || got != nil {
				t.Fatalf("CreateOneTimeMaintenanceWindow() = (%#v, %v)", got, err)
			}
		})
	})
}

func TestMaintenanceCreateManualWindow(t *testing.T) {
	t.Run("persists inactive manual window", func(t *testing.T) {
		testutil.Case(t, "maintenance", "CreateManualMaintenanceWindow", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, _, provider := maintenanceTestService(t)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *maintenance.MaintenanceWindow) error {
				if got.User.Login != "alice" || got.Title != "manual" || got.IsActive() {
					t.Fatalf("persisted manual window = %#v", got)
				}
				return nil
			})
			got, err := svc.CreateManualMaintenanceWindow(context.Background(), CreateManualMaintenanceWindowDTO{Title: "manual"})
			if err != nil || got == nil {
				t.Fatalf("CreateManualMaintenanceWindow() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("returns repository failure", func(t *testing.T) {
		testutil.Case(t, "maintenance", "CreateManualMaintenanceWindow", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, _, provider := maintenanceTestService(t)
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().Create(gomock.Any(), gomock.Any()).Return(repo.ErrDB)
			_, err := svc.CreateManualMaintenanceWindow(context.Background(), CreateManualMaintenanceWindowDTO{Title: "manual"})
			if !errors.Is(err, repo.ErrDB) {
				t.Fatalf("CreateManualMaintenanceWindow() error = %v", err)
			}
		})
	})
}

func TestMaintenanceGetAllWindows(t *testing.T) {
	tests := []struct {
		name    string
		result  []maintenance.MaintenanceWindow
		repoErr error
	}{
		{name: "returns authorized user's windows", result: []maintenance.MaintenanceWindow{testutil.ObjectMother{}.Window(false)}},
		{name: "returns repository failure", repoErr: repo.ErrDB},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "maintenance", "GetAllMaintenanceWindows", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
				svc, windows, _, provider := maintenanceTestService(t)
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				windows.EXPECT().GetByUserLogin(gomock.Any(), "alice").Return(tc.result, tc.repoErr)
				got, err := svc.GetAllMaintenanceWindows(context.Background())
				if !errors.Is(err, tc.repoErr) || (tc.repoErr == nil && len(got) != 1) {
					t.Fatalf("GetAllMaintenanceWindows() = (%#v, %v)", got, err)
				}
			})
		})
	}
}

func TestMaintenanceGetWindow(t *testing.T) {
	for _, owner := range testutil.Shuffle(t, []string{"alice", "bob"}) {
		owner := owner
		t.Run("owner="+owner, func(t *testing.T) {
			testutil.Case(t, "maintenance", "GetMaintenanceWindow", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
				svc, windows, _, provider := maintenanceTestService(t)
				window := testutil.ObjectMother{}.Window(false)
				window.User.Login = owner
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
				got, err := svc.GetMaintenanceWindow(context.Background(), window.ID)
				if owner == "alice" && (err != nil || got != &window) {
					t.Fatalf("GetMaintenanceWindow() = (%#v, %v)", got, err)
				}
				if owner != "alice" && !errors.Is(err, baseservice.ErrPermissionDenied) {
					t.Fatalf("GetMaintenanceWindow() error = %v", err)
				}
			})
		})
	}
}

func TestMaintenanceDeleteWindow(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		deleteErr error
		wantErr   error
	}{
		{name: "deletes owned window", owner: "alice"},
		{name: "denies another user's window", owner: "bob", wantErr: baseservice.ErrPermissionDenied},
		{name: "returns delete failure", owner: "alice", deleteErr: repo.ErrDB, wantErr: repo.ErrDB},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "maintenance", "DeleteMaintenanceWindow", "decision-table", "london", func(t *testing.T, _ *allure.Context) {
				svc, windows, _, provider := maintenanceTestService(t)
				window := testutil.ObjectMother{}.Window(false)
				window.User.Login = tc.owner
				provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
				if tc.owner == "alice" {
					windows.EXPECT().DeleteByID(gomock.Any(), window.ID).Return(tc.deleteErr)
				}
				err := svc.DeleteMaintenanceWindow(context.Background(), window.ID)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("DeleteMaintenanceWindow() error = %v, want %v", err, tc.wantErr)
				}
			})
		})
	}
}

func TestMaintenanceAddMonitorToWindow(t *testing.T) {
	t.Run("links an unlinked monitor", func(t *testing.T) {
		testutil.Case(t, "maintenance", "AddMonitorToMaintenanceWindow", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, monitors, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			mon := testutil.NewMonitorBuilder().Build()
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
			windows.EXPECT().LinkMonitor(gomock.Any(), &window, mon.ID).Return(nil)
			if err := svc.AddMonitorToMaintenanceWindow(context.Background(), mon.ID, window.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("already linked monitor is idempotent", func(t *testing.T) {
		testutil.Case(t, "maintenance", "AddMonitorToMaintenanceWindow", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, monitors, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			mon := testutil.NewMonitorBuilder().Build()
			mon.MaintenanceWindows = []maintenance.MaintenanceWindow{window}
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
			if err := svc.AddMonitorToMaintenanceWindow(context.Background(), mon.ID, window.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("returns monitor lookup failure", func(t *testing.T) {
		testutil.Case(t, "maintenance", "AddMonitorToMaintenanceWindow", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, monitors, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			monitorID := uuid.New()
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			monitors.EXPECT().GetByID(gomock.Any(), monitorID).Return(nil, repo.ErrDB)
			if err := svc.AddMonitorToMaintenanceWindow(context.Background(), monitorID, window.ID); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("AddMonitorToMaintenanceWindow() error = %v", err)
			}
		})
	})
}

func TestMaintenanceRemoveMonitorFromWindow(t *testing.T) {
	t.Run("unlinks a linked monitor", func(t *testing.T) {
		testutil.Case(t, "maintenance", "RemoveMonitorFromMaintenanceWindow", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, monitors, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			mon := testutil.NewMonitorBuilder().Build()
			mon.MaintenanceWindows = []maintenance.MaintenanceWindow{window}
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
			windows.EXPECT().UnlinkMonitor(gomock.Any(), &window, mon.ID).Return(nil)
			if err := svc.RemoveMonitorFromMaintenanceWindow(context.Background(), mon.ID, window.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("already unlinked monitor is idempotent", func(t *testing.T) {
		testutil.Case(t, "maintenance", "RemoveMonitorFromMaintenanceWindow", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, monitors, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			mon := testutil.NewMonitorBuilder().Build()
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
			if err := svc.RemoveMonitorFromMaintenanceWindow(context.Background(), mon.ID, window.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("returns unlink failure", func(t *testing.T) {
		testutil.Case(t, "maintenance", "RemoveMonitorFromMaintenanceWindow", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, monitors, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			mon := testutil.NewMonitorBuilder().Build()
			mon.MaintenanceWindows = []maintenance.MaintenanceWindow{window}
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
			windows.EXPECT().UnlinkMonitor(gomock.Any(), &window, mon.ID).Return(repo.ErrDB)
			if err := svc.RemoveMonitorFromMaintenanceWindow(context.Background(), mon.ID, window.ID); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("RemoveMonitorFromMaintenanceWindow() error = %v", err)
			}
		})
	})
}

func TestMaintenanceMonitorLinkAuthorization(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		authErr      error
		windowOwner  string
		monitorOwner string
		wantErr      error
	}{
		{name: "add rejects missing authorization", method: "add", authErr: baseservice.ErrUnauthorized, wantErr: baseservice.ErrUnauthorized},
		{name: "add rejects foreign window", method: "add", windowOwner: "bob", monitorOwner: "alice", wantErr: baseservice.ErrPermissionDenied},
		{name: "add rejects foreign monitor", method: "add", windowOwner: "alice", monitorOwner: "bob", wantErr: baseservice.ErrPermissionDenied},
		{name: "remove rejects missing authorization", method: "remove", authErr: baseservice.ErrUnauthorized, wantErr: baseservice.ErrUnauthorized},
		{name: "remove rejects foreign window", method: "remove", windowOwner: "bob", monitorOwner: "alice", wantErr: baseservice.ErrPermissionDenied},
		{name: "remove rejects foreign monitor", method: "remove", windowOwner: "alice", monitorOwner: "bob", wantErr: baseservice.ErrPermissionDenied},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "maintenance", map[string]string{"add": "AddMonitorToMaintenanceWindow", "remove": "RemoveMonitorFromMaintenanceWindow"}[tc.method], "decision-table", "london", func(t *testing.T, a *allure.Context) {
				var svc *maintenanceService
				var window maintenance.MaintenanceWindow
				var mon *monitor.Monitor
				var err error
				a.Step("Arrange authorization and resource ownership", func(*allure.Context) {
					var windows *testmocks.MockMaintenanceWindowRepository
					var monitors *testmocks.MockMonitorRepository
					var provider *testmocks.MockUserProvider
					svc, windows, monitors, provider = maintenanceTestService(t)
					window = testutil.ObjectMother{}.Window(false)
					mon = testutil.NewMonitorBuilder().Build()
					if tc.windowOwner != "" {
						window.User.Login = tc.windowOwner
					}
					if tc.monitorOwner != "" {
						mon.User.Login = tc.monitorOwner
					}
					provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, tc.authErr)
					if tc.authErr == nil {
						windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
						if tc.windowOwner == "alice" {
							monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
						}
					}
				})
				a.Step("Act: change the monitor-window link", func(*allure.Context) {
					if tc.method == "add" {
						err = svc.AddMonitorToMaintenanceWindow(context.Background(), mon.ID, window.ID)
					} else {
						err = svc.RemoveMonitorFromMaintenanceWindow(context.Background(), mon.ID, window.ID)
					}
				})
				a.Step("Assert: unauthorized ownership cannot mutate links", func(*allure.Context) {
					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("link operation error = %v, want %v", err, tc.wantErr)
					}
				})
			})
		})
	}
}

func TestMaintenanceUpdateWindow(t *testing.T) {
	t.Run("persists title and active state", func(t *testing.T) {
		testutil.Case(t, "maintenance", "UpdateMaintenanceWindow", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, _, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			newTitle, active := "Emergency", true
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			windows.EXPECT().Update(gomock.Any(), &window).DoAndReturn(func(_ context.Context, got *maintenance.MaintenanceWindow) error {
				if got.Title != newTitle || !got.IsActive() {
					t.Fatalf("updated window = %#v", got)
				}
				return nil
			})
			if err := svc.UpdateMaintenanceWindow(context.Background(), UpdateMaintenanceWindowDTO{WindowID: window.ID, Title: &newTitle, ConfigUpdate: maintenance.ManualConfigUpdate{Active: &active}}); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("denies another user's window", func(t *testing.T) {
		testutil.Case(t, "maintenance", "UpdateMaintenanceWindow", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, windows, _, provider := maintenanceTestService(t)
			window := testutil.ObjectMother{}.Window(false)
			window.User.Login = "bob"
			provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			windows.EXPECT().GetByID(gomock.Any(), window.ID).Return(&window, nil)
			if err := svc.UpdateMaintenanceWindow(context.Background(), UpdateMaintenanceWindowDTO{WindowID: window.ID}); !errors.Is(err, baseservice.ErrPermissionDenied) {
				t.Fatalf("UpdateMaintenanceWindow() error = %v", err)
			}
		})
	})
}

var _ = monitor.StatusUnknown
