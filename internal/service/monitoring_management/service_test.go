package monitoring_service

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	healthchecksvc "WatchTower/internal/service/healthcheck"
	monitordto "WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

type recordingPublisher struct {
	topic    string
	messages []*message.Message
	err      error
}

func (p *recordingPublisher) Publish(topic string, messages ...*message.Message) error {
	p.topic = topic
	p.messages = append(p.messages, messages...)
	return p.err
}
func (*recordingPublisher) Close() error { return nil }

type monitoringDeps struct {
	service  *monitoringManagementService
	monitors *testmocks.MockMonitorRepository
	targets  *testmocks.MockTargetRepository
	contacts *testmocks.MockAlertContactRepository
	windows  *testmocks.MockMaintenanceWindowRepository
	users    *testmocks.MockUserProvider
}

func newMonitoringService(t *testing.T, publisher message.Publisher) monitoringDeps {
	t.Helper()
	ctrl := gomock.NewController(t)
	d := monitoringDeps{
		monitors: testmocks.NewMockMonitorRepository(ctrl),
		targets:  testmocks.NewMockTargetRepository(ctrl),
		contacts: testmocks.NewMockAlertContactRepository(ctrl),
		windows:  testmocks.NewMockMaintenanceWindowRepository(ctrl),
		users:    testmocks.NewMockUserProvider(ctrl),
	}
	d.service = NewMonitoringManagementService(d.monitors, d.targets, d.contacts, d.windows, d.users, publisher, testutil.NoopLogger()).(*monitoringManagementService)
	return d
}

func expectOwnedMonitor(d monitoringDeps, mon *monitor.Monitor) {
	d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	d.monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
}

func validCreateDTO() monitordto.CreateMonitorDTO {
	return monitordto.CreateMonitorDTO{
		Label:            "API",
		Endpoint:         "https://example.test/health",
		ProbeIntervalSec: 30,
		NetworkConfig:    monitordto.HTTPMonitorNetworkConfig{Method: "GET"},
		Expectations:     monitordto.HTTPMonitorExpectations{StatusCodes: []int{200}, MaxLatencyMs: 1000},
	}
}

func TestMonitoringGetAllMonitors(t *testing.T) {
	tests := []struct {
		name    string
		result  []*monitor.Monitor
		repoErr error
	}{
		{name: "returns authorized user's monitors", result: []*monitor.Monitor{testutil.NewMonitorBuilder().Build()}},
		{name: "returns repository failure", repoErr: repo.ErrDB},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "monitoring", "GetAllMonitors", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
				d := newMonitoringService(t, nil)
				d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				d.monitors.EXPECT().GetAllByUser(gomock.Any(), gomock.Any()).Return(tc.result, tc.repoErr)
				got, err := d.service.GetAllMonitors(context.Background())
				if !errors.Is(err, tc.repoErr) || (tc.repoErr == nil && len(got) != 1) {
					t.Fatalf("GetAllMonitors() = (%#v, %v)", got, err)
				}
			})
		})
	}
}

func TestMonitoringGetMonitor(t *testing.T) {
	for _, owner := range testutil.Shuffle(t, []string{"alice", "bob"}) {
		owner := owner
		t.Run("owner="+owner, func(t *testing.T) {
			testutil.Case(t, "monitoring", "GetMonitor", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
				d := newMonitoringService(t, nil)
				mon := testutil.NewMonitorBuilder().WithOwner(owner).Build()
				d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				d.monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)
				got, err := d.service.GetMonitor(context.Background(), mon.ID)
				if owner == "alice" && (err != nil || got != mon) {
					t.Fatalf("GetMonitor() = (%#v, %v)", got, err)
				}
				if owner == "bob" && !errors.Is(err, baseservice.ErrPermissionDenied) {
					t.Fatalf("GetMonitor() error = %v", err)
				}
			})
		})
	}
}

func TestMonitoringDisableMonitor(t *testing.T) {
	t.Run("persists disabled state", func(t *testing.T) {
		testutil.Case(t, "monitoring", "DisableMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().WithStatus(monitor.StatusUp).Build()
			expectOwnedMonitor(d, mon)
			d.monitors.EXPECT().Disable(gomock.Any(), mon.ID).Return(nil)
			d.monitors.EXPECT().GetAllByTargetID(gomock.Any(), mon.Target.ID).Return([]*monitor.Monitor{{IsActive: true, ProbeIntervalSec: mon.Target.ProbeIntervalSec}}, nil)
			if err := d.service.DisableMonitor(context.Background(), mon.ID); err != nil || mon.IsActive || mon.CurrentStatus != monitor.StatusUnknown {
				t.Fatalf("DisableMonitor() error=%v monitor=%#v", err, mon)
			}
		})
	})
	t.Run("already disabled is idempotent", func(t *testing.T) {
		testutil.Case(t, "monitoring", "DisableMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().WithActive(false).Build()
			expectOwnedMonitor(d, mon)
			if err := d.service.DisableMonitor(context.Background(), mon.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("denies another user's monitor", func(t *testing.T) {
		testutil.Case(t, "monitoring", "DisableMonitor", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().WithOwner("bob").Build()
			expectOwnedMonitor(d, mon)
			if err := d.service.DisableMonitor(context.Background(), mon.ID); !errors.Is(err, baseservice.ErrPermissionDenied) {
				t.Fatalf("DisableMonitor() error = %v", err)
			}
		})
	})
}

func TestMonitoringDisableMonitorReconcilesActiveMonitors(t *testing.T) {
	tests := []struct {
		name         string
		peerActive   bool
		wantInterval int32
	}{
		{name: "uses remaining active monitor interval", peerActive: true, wantInterval: 30},
		{name: "disables target when all monitors are inactive", wantInterval: 10},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "monitoring", "DisableMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
				publisher := &recordingPublisher{}
				d := newMonitoringService(t, publisher)
				mon := testutil.NewMonitorBuilder().Build()
				mon.ProbeIntervalSec = 10
				mon.Target.ProbeIntervalSec = 10
				expectOwnedMonitor(d, mon)
				d.monitors.EXPECT().Disable(gomock.Any(), mon.ID).Return(nil)
				d.monitors.EXPECT().GetAllByTargetID(gomock.Any(), mon.Target.ID).Return([]*monitor.Monitor{
					mon,
					{IsActive: tc.peerActive, ProbeIntervalSec: 30},
				}, nil)
				if !tc.peerActive {
					d.targets.EXPECT().Disable(gomock.Any(), mon.Target.ID).Return(nil)
				}
				d.targets.EXPECT().Update(gomock.Any(), mon.Target).DoAndReturn(func(_ context.Context, got *target.Target) error {
					if got.IsActive != tc.peerActive || got.ProbeIntervalSec != tc.wantInterval {
						t.Fatalf("persisted target active=%v interval=%d, want active=%v interval=%d", got.IsActive, got.ProbeIntervalSec, tc.peerActive, tc.wantInterval)
					}
					return nil
				})

				if err := d.service.DisableMonitor(context.Background(), mon.ID); err != nil {
					t.Fatal(err)
				}
				if mon.Target.IsActive != tc.peerActive || mon.Target.ProbeIntervalSec != tc.wantInterval {
					t.Fatalf("target active=%v interval=%d, want active=%v interval=%d", mon.Target.IsActive, mon.Target.ProbeIntervalSec, tc.peerActive, tc.wantInterval)
				}
				if publisher.topic != healthchecksvc.TopicTargetUpdated || len(publisher.messages) != 1 {
					t.Fatalf("target update event topic=%q messages=%d", publisher.topic, len(publisher.messages))
				}
			})
		})
	}
}

func TestMonitoringEnableMonitor(t *testing.T) {
	t.Run("persists enabled state", func(t *testing.T) {
		testutil.Case(t, "monitoring", "EnableMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().WithActive(false).Build()
			mon.Target.IsActive = true
			expectOwnedMonitor(d, mon)
			d.monitors.EXPECT().Enable(gomock.Any(), mon.ID).Return(nil)
			if err := d.service.EnableMonitor(context.Background(), mon.ID); err != nil || !mon.IsActive {
				t.Fatalf("EnableMonitor() error=%v active=%v", err, mon.IsActive)
			}
		})
	})
	t.Run("already enabled is idempotent", func(t *testing.T) {
		testutil.Case(t, "monitoring", "EnableMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().Build()
			expectOwnedMonitor(d, mon)
			if err := d.service.EnableMonitor(context.Background(), mon.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("returns target synchronization failure", func(t *testing.T) {
		testutil.Case(t, "monitoring", "EnableMonitor", "equivalence", "london", func(t *testing.T, a *allure.Context) {
			var d monitoringDeps
			var mon *monitor.Monitor
			var err error
			a.Step("Arrange inactive monitor and failing target repository", func(*allure.Context) {
				d = newMonitoringService(t, nil)
				mon = testutil.NewMonitorBuilder().WithActive(false).Build()
				mon.Target.IsActive = false
				expectOwnedMonitor(d, mon)
				d.monitors.EXPECT().Enable(gomock.Any(), mon.ID).Return(nil)
				d.targets.EXPECT().Update(gomock.Any(), mon.Target).Return(repo.ErrDB)
			})
			a.Step("Act: enable monitor", func(*allure.Context) {
				err = d.service.EnableMonitor(context.Background(), mon.ID)
			})
			a.Step("Assert: target synchronization failure is returned", func(*allure.Context) {
				if !errors.Is(err, repo.ErrDB) {
					t.Fatalf("EnableMonitor() error = %v, want %v", err, repo.ErrDB)
				}
			})
		})
	})
}

func TestMonitoringDeleteMonitor(t *testing.T) {
	t.Run("deletes owned monitor and preserves shared target", func(t *testing.T) {
		testutil.Case(t, "monitoring", "DeleteMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().Build()
			expectOwnedMonitor(d, mon)
			d.monitors.EXPECT().DeleteByID(gomock.Any(), mon.ID).Return(nil)
			d.monitors.EXPECT().GetAllByTargetID(gomock.Any(), mon.Target.ID).Return([]*monitor.Monitor{{IsActive: true, ProbeIntervalSec: mon.Target.ProbeIntervalSec}}, nil)
			if err := d.service.DeleteMonitor(context.Background(), mon.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("returns delete failure", func(t *testing.T) {
		testutil.Case(t, "monitoring", "DeleteMonitor", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().Build()
			expectOwnedMonitor(d, mon)
			d.monitors.EXPECT().DeleteByID(gomock.Any(), mon.ID).Return(repo.ErrDB)
			if err := d.service.DeleteMonitor(context.Background(), mon.ID); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("DeleteMonitor() error = %v", err)
			}
		})
	})
}

func TestMonitoringCreateMonitor(t *testing.T) {
	t.Run("persists monitor on an existing target", func(t *testing.T) {
		testutil.Case(t, "monitoring", "CreateMonitor", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			dto := validCreateDTO()
			existing := testutil.ObjectMother{}.Target()
			existing.ProbeIntervalSec = dto.ProbeIntervalSec
			d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			d.windows.EXPECT().GetByIDBulk(gomock.Any(), dto.MaintenanceWindowIDs).Return([]maintenance.MaintenanceWindow{}, nil)
			d.contacts.EXPECT().GetByIDBulk(gomock.Any(), dto.AlertContactIDs).Return([]alert.Contact{}, nil)
			d.targets.EXPECT().GetByHash(gomock.Any(), target.ComputeHash(dto.Endpoint, target.HTTPConfig{Method: "GET"})).Return(existing, nil)
			d.monitors.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *monitor.Monitor) error {
				if got.Label != dto.Label || got.User.Login != "alice" || got.Target != existing || !got.IsActive {
					t.Fatalf("persisted monitor = %#v", got)
				}
				return nil
			})
			got, err := d.service.CreateMonitor(context.Background(), dto)
			if err != nil || got == nil {
				t.Fatalf("CreateMonitor() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("creates target and publishes creation event", func(t *testing.T) {
		testutil.Case(t, "monitoring", "CreateMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			publisher := &recordingPublisher{}
			d := newMonitoringService(t, publisher)
			dto := validCreateDTO()
			d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			d.windows.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]maintenance.MaintenanceWindow{}, nil)
			d.contacts.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]alert.Contact{}, nil)
			d.targets.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(nil, repo.ErrNotFound)
			d.targets.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			d.monitors.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			got, err := d.service.CreateMonitor(context.Background(), dto)
			if err != nil || got == nil || publisher.topic != healthchecksvc.TopicTargetCreated || len(publisher.messages) != 1 {
				t.Fatalf("CreateMonitor() = (%#v, %v), topic=%q messages=%d", got, err, publisher.topic, len(publisher.messages))
			}
			var event healthchecksvc.TargetEvent
			if err := json.Unmarshal(publisher.messages[0].Payload, &event); err != nil || event.ID != got.Target.ID {
				t.Fatalf("published event = %#v, error=%v", event, err)
			}
		})
	})
	t.Run("rejects protocol mismatch", func(t *testing.T) {
		testutil.Case(t, "monitoring", "CreateMonitor", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			dto := validCreateDTO()
			dto.Expectations = monitordto.TCPMonitorExpectations{}
			d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
			if got, err := d.service.CreateMonitor(context.Background(), dto); err == nil || got != nil {
				t.Fatalf("CreateMonitor() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("returns target lookup failure", func(t *testing.T) {
		testutil.Case(t, "monitoring", "CreateMonitor", "equivalence", "london", func(t *testing.T, a *allure.Context) {
			var d monitoringDeps
			var dto monitordto.CreateMonitorDTO
			var got *monitor.Monitor
			var err error
			a.Step("Arrange valid monitor input and failing target lookup", func(*allure.Context) {
				d = newMonitoringService(t, nil)
				dto = validCreateDTO()
				d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
				d.windows.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]maintenance.MaintenanceWindow{}, nil)
				d.contacts.EXPECT().GetByIDBulk(gomock.Any(), gomock.Any()).Return([]alert.Contact{}, nil)
				d.targets.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(nil, repo.ErrDB)
			})
			a.Step("Act: create monitor", func(*allure.Context) {
				got, err = d.service.CreateMonitor(context.Background(), dto)
			})
			a.Step("Assert: target lookup failure is preserved", func(*allure.Context) {
				if got != nil || !errors.Is(err, repo.ErrDB) {
					t.Fatalf("CreateMonitor() = (%#v, %v), want target lookup failure", got, err)
				}
			})
		})
	})
}

func TestMonitoringCreateMonitorRejectsForeignRelations(t *testing.T) {
	tests := []struct {
		name           string
		foreignContact bool
		foreignWindow  bool
	}{
		{name: "rejects foreign alert contact", foreignContact: true},
		{name: "rejects foreign maintenance window", foreignWindow: true},
	}
	for _, tc := range testutil.Shuffle(t, tests) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "monitoring", "CreateMonitor", "decision-table", "london", func(t *testing.T, a *allure.Context) {
				var d monitoringDeps
				var dto monitordto.CreateMonitorDTO
				var got *monitor.Monitor
				var publisher *recordingPublisher
				var err error
				a.Step("Arrange relations owned by another user", func(*allure.Context) {
					publisher = &recordingPublisher{}
					d = newMonitoringService(t, publisher)
					dto = validCreateDTO()
					contact := testutil.ObjectMother{}.Contact()
					window := testutil.ObjectMother{}.Window(false)
					if tc.foreignContact {
						contact.User.Login = "bob"
						dto.AlertContactIDs = []uuid.UUID{contact.ID}
					}
					if tc.foreignWindow {
						window.User.Login = "bob"
						dto.MaintenanceWindowIDs = []uuid.UUID{window.ID}
					}
					d.users.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
					if tc.foreignWindow {
						d.windows.EXPECT().GetByIDBulk(gomock.Any(), dto.MaintenanceWindowIDs).Return([]maintenance.MaintenanceWindow{window}, nil)
					} else {
						d.windows.EXPECT().GetByIDBulk(gomock.Any(), dto.MaintenanceWindowIDs).Return(nil, nil)
						d.contacts.EXPECT().GetByIDBulk(gomock.Any(), dto.AlertContactIDs).Return([]alert.Contact{*contact}, nil)
					}
				})
				a.Step("Act: create a monitor with the foreign relation", func(*allure.Context) {
					got, err = d.service.CreateMonitor(context.Background(), dto)
				})
				a.Step("Assert: creation stops before target or monitor persistence", func(*allure.Context) {
					if got != nil || !errors.Is(err, baseservice.ErrPermissionDenied) {
						t.Fatalf("CreateMonitor() = (%#v, %v), want permission denied", got, err)
					}
					if len(publisher.messages) != 0 {
						t.Fatalf("rejected creation published %d target events", len(publisher.messages))
					}
				})
			})
		})
	}
}

func TestMonitoringUpdateMonitor(t *testing.T) {
	t.Run("persists simple field changes", func(t *testing.T) {
		testutil.Case(t, "monitoring", "UpdateMonitor", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().Build()
			label := "Payments API"
			expectOwnedMonitor(d, mon)
			d.monitors.EXPECT().Update(gomock.Any(), mon).DoAndReturn(func(_ context.Context, got *monitor.Monitor) error {
				if got.Label != label {
					t.Fatalf("updated label = %q", got.Label)
				}
				return nil
			})
			if err := d.service.UpdateMonitor(context.Background(), monitordto.UpdateMonitorDTO{ID: mon.ID, Label: &label}); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("returns persistence failure", func(t *testing.T) {
		testutil.Case(t, "monitoring", "UpdateMonitor", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().Build()
			expectOwnedMonitor(d, mon)
			d.monitors.EXPECT().Update(gomock.Any(), mon).Return(repo.ErrDB)
			if err := d.service.UpdateMonitor(context.Background(), monitordto.UpdateMonitorDTO{ID: mon.ID}); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("UpdateMonitor() error = %v", err)
			}
		})
	})
	for _, name := range testutil.Shuffle(t, []string{"invalid HTTP expectations", "protocol mismatch"}) {
		t.Run("rejects "+name+" without mutation", func(t *testing.T) {
			testutil.Case(t, "monitoring", "UpdateMonitor", "decision-table", "london", func(t *testing.T, a *allure.Context) {
				var d monitoringDeps
				var mon *monitor.Monitor
				var dto monitordto.UpdateMonitorDTO
				var original monitor.Monitor
				var publisher *recordingPublisher
				var err error
				a.Step("Arrange an owned monitor and invalid effective protocol state", func(*allure.Context) {
					publisher = &recordingPublisher{}
					d = newMonitoringService(t, publisher)
					mon = testutil.NewMonitorBuilder().Build()
					original = *mon
					dto.ID = mon.ID
					label := "must not be applied"
					dto.Label = &label
					if name == "invalid HTTP expectations" {
						var expectations monitordto.MonitorExpectations = monitordto.HTTPMonitorExpectations{StatusCodes: nil, MaxLatencyMs: -1}
						dto.Expectations = &expectations
					} else {
						protocol := target.ProtocolTCP
						var config monitordto.MonitorNetworkConfig = monitordto.TCPMonitorNetworkConfig{Port: 443}
						dto.Protocol = &protocol
						dto.NetworkConfig = &config
					}
					expectOwnedMonitor(d, mon)
				})
				a.Step("Act: update the monitor", func(*allure.Context) {
					err = d.service.UpdateMonitor(context.Background(), dto)
				})
				a.Step("Assert: invalid state is rejected before mutation or persistence", func(*allure.Context) {
					if err == nil {
						t.Fatal("UpdateMonitor() unexpectedly succeeded")
					}
					if !reflect.DeepEqual(*mon, original) {
						t.Fatalf("monitor mutated on rejected update: got %#v, want %#v", *mon, original)
					}
					if len(publisher.messages) != 0 {
						t.Fatalf("rejected update published %d target events", len(publisher.messages))
					}
				})
			})
		})
	}
}

func TestMonitoringLinkAlertContact(t *testing.T) {
	t.Run("links contact to owned monitor", func(t *testing.T) {
		testutil.Case(t, "monitoring", "LinkAlertContact", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon, contact := testutil.NewMonitorBuilder().Build(), testutil.ObjectMother{}.Contact()
			expectOwnedMonitor(d, mon)
			d.contacts.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
			d.monitors.EXPECT().AddAlertContact(gomock.Any(), mon, contact).Return(nil)
			if err := d.service.LinkAlertContact(context.Background(), mon.ID, contact.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("denies another user's monitor", func(t *testing.T) {
		testutil.Case(t, "monitoring", "LinkAlertContact", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon := testutil.NewMonitorBuilder().WithOwner("bob").Build()
			expectOwnedMonitor(d, mon)
			if err := d.service.LinkAlertContact(context.Background(), mon.ID, uuid.New()); !errors.Is(err, baseservice.ErrPermissionDenied) {
				t.Fatalf("LinkAlertContact() error = %v", err)
			}
		})
	})
}

func TestMonitoringUnlinkAlertContact(t *testing.T) {
	t.Run("unlinks contact from owned monitor", func(t *testing.T) {
		testutil.Case(t, "monitoring", "UnlinkAlertContact", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon, contact := testutil.NewMonitorBuilder().Build(), testutil.ObjectMother{}.Contact()
			expectOwnedMonitor(d, mon)
			d.contacts.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
			d.monitors.EXPECT().RemoveAlertContact(gomock.Any(), mon, contact).Return(nil)
			if err := d.service.UnlinkAlertContact(context.Background(), mon.ID, contact.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("returns contact lookup failure", func(t *testing.T) {
		testutil.Case(t, "monitoring", "UnlinkAlertContact", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			d := newMonitoringService(t, nil)
			mon, contactID := testutil.NewMonitorBuilder().Build(), uuid.New()
			expectOwnedMonitor(d, mon)
			d.contacts.EXPECT().GetByID(gomock.Any(), contactID).Return(nil, repo.ErrDB)
			if err := d.service.UnlinkAlertContact(context.Background(), mon.ID, contactID); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("UnlinkAlertContact() error = %v", err)
			}
		})
	})
}

func TestMonitoringAlertContactLinkRejectsForeignContact(t *testing.T) {
	for _, method := range testutil.Shuffle(t, []string{"LinkAlertContact", "UnlinkAlertContact"}) {
		t.Run(method, func(t *testing.T) {
			testutil.Case(t, "monitoring", method, "equivalence", "london", func(t *testing.T, a *allure.Context) {
				var d monitoringDeps
				var mon *monitor.Monitor
				var contact *alert.Contact
				var err error
				a.Step("Arrange an owned monitor and foreign contact", func(*allure.Context) {
					d = newMonitoringService(t, nil)
					mon = testutil.NewMonitorBuilder().Build()
					contact = testutil.ObjectMother{}.Contact()
					contact.User.Login = "bob"
					expectOwnedMonitor(d, mon)
					d.contacts.EXPECT().GetByID(gomock.Any(), contact.ID).Return(contact, nil)
				})
				a.Step("Act: change the contact link", func(*allure.Context) {
					if method == "LinkAlertContact" {
						err = d.service.LinkAlertContact(context.Background(), mon.ID, contact.ID)
					} else {
						err = d.service.UnlinkAlertContact(context.Background(), mon.ID, contact.ID)
					}
				})
				a.Step("Assert: foreign contact cannot be linked or unlinked", func(*allure.Context) {
					if !errors.Is(err, baseservice.ErrPermissionDenied) {
						t.Fatalf("%s() error = %v, want permission denied", method, err)
					}
				})
			})
		})
	}
}
