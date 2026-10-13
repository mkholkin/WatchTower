package notification

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/repo"
	analyzationsvc "WatchTower/internal/service/analyze"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

type notificationProviderSpy struct {
	mu      sync.Mutex
	contact *alert.Contact
	message string
	err     error
	after   func()
}

func (p *notificationProviderSpy) SendNotification(contact *alert.Contact, msg string) error {
	p.mu.Lock()
	p.contact, p.message = contact, msg
	p.mu.Unlock()
	if p.after != nil {
		p.after()
	}
	return p.err
}

func TestNotificationProviderRegistry(t *testing.T) {
	t.Run("returns registered provider", func(t *testing.T) {
		testutil.Case(t, "notification", "ProviderRegistry.Register/Get", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			registry := NewProviderRegistry()
			expected := &notificationProviderSpy{}
			registry.Register(alert.ContactTypeTelegram, expected)
			got, err := registry.Get(alert.ContactTypeTelegram)
			if err != nil || got != expected {
				t.Fatalf("Get() = (%v, %v)", got, err)
			}
		})
	})
	t.Run("later registration replaces provider", func(t *testing.T) {
		testutil.Case(t, "notification", "ProviderRegistry.Register/Get", "state-transition", "classic", func(t *testing.T, _ *allure.Context) {
			registry := NewProviderRegistry()
			registry.Register(alert.ContactTypeTelegram, &notificationProviderSpy{})
			expected := &notificationProviderSpy{}
			registry.Register(alert.ContactTypeTelegram, expected)
			got, err := registry.Get(alert.ContactTypeTelegram)
			if err != nil || got != expected {
				t.Fatalf("Get() = (%v, %v)", got, err)
			}
		})
	})
	t.Run("missing contact type returns error", func(t *testing.T) {
		testutil.Case(t, "notification", "ProviderRegistry.Get", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			got, err := NewProviderRegistry().Get(alert.ContactType("email"))
			if err == nil || got != nil {
				t.Fatalf("Get() = (%v, %v)", got, err)
			}
		})
	})
}

func TestNotificationServiceRun(t *testing.T) {
	t.Run("sends status event to active contacts with formatted contents", func(t *testing.T) {
		testutil.Case(t, "notification", "NotificationService.Run", "state-transition", "london", func(t *testing.T, a *allure.Context) {
			ctrl := gomock.NewController(t)
			monitors := testmocks.NewMockMonitorRepository(ctrl)
			subscriber := testmocks.NewMockSubscriber(ctrl)
			mon := testutil.NewMonitorBuilder().WithLabel("Payments").Build()
			active := testutil.ObjectMother{}.Contact()
			inactive := *testutil.ObjectMother{}.Contact()
			inactive.IsActive = false
			mon.AlertContacts = []alert.Contact{*active, inactive}
			event := analyzationsvc.MonitorStatusChangedEvent{MonitorID: mon.ID, OldStatus: monitor.StatusUp, NewStatus: monitor.StatusDown, OccurredAt: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}
			payload, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			msg := message.NewMessage(uuid.NewString(), payload)
			messages := make(chan *message.Message, 1)
			messages <- msg
			ctx, cancel := context.WithCancel(context.Background())
			provider := &notificationProviderSpy{after: cancel}
			registry := NewProviderRegistry()
			registry.Register(alert.ContactTypeTelegram, provider)
			subscriber.EXPECT().Subscribe(gomock.Any(), analyzationsvc.TopicMonitorStatusChanged).Return(messages, nil)
			monitors.EXPECT().GetByID(gomock.Any(), mon.ID).Return(mon, nil)

			svc := NewNotificationService(registry, subscriber, monitors, testutil.NoopLogger(), WithWorkerCount(1), WithTaskQueueSize(2))
			var runErr error
			a.Step("Act: run notification service", func(*allure.Context) { runErr = svc.Run(ctx) })
			a.Step("Assert: active contact receives exact transition message", func(*allure.Context) {
				if runErr != nil {
					t.Fatalf("Run() error = %v", runErr)
				}
				provider.mu.Lock()
				defer provider.mu.Unlock()
				if provider.contact == nil || provider.contact.ID != active.ID || !strings.Contains(provider.message, "Monitor 'Payments' status changed: up -> down") || !strings.Contains(provider.message, event.OccurredAt.Format(time.RFC3339)) {
					t.Fatalf("sent contact=%#v message=%q", provider.contact, provider.message)
				}
				select {
				case <-msg.Acked():
				default:
					t.Fatal("status event was not acknowledged")
				}
			})
		})
	})
	t.Run("returns subscription failure", func(t *testing.T) {
		testutil.Case(t, "notification", "NotificationService.Run", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			subscriber := testmocks.NewMockSubscriber(ctrl)
			subscriber.EXPECT().Subscribe(gomock.Any(), analyzationsvc.TopicMonitorStatusChanged).Return(nil, repo.ErrDB)
			svc := NewNotificationService(NewProviderRegistry(), subscriber, testmocks.NewMockMonitorRepository(ctrl), testutil.NoopLogger(), WithWorkerCount(1))
			if err := svc.Run(context.Background()); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("Run() error = %v", err)
			}
		})
	})
	t.Run("acknowledges malformed event without repository access", func(t *testing.T) {
		testutil.Case(t, "notification", "NotificationService.Run", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			subscriber := testmocks.NewMockSubscriber(ctrl)
			msg := message.NewMessage(uuid.NewString(), []byte("not-json"))
			messages := make(chan *message.Message, 1)
			messages <- msg
			close(messages)
			subscriber.EXPECT().Subscribe(gomock.Any(), analyzationsvc.TopicMonitorStatusChanged).Return(messages, nil)
			svc := NewNotificationService(NewProviderRegistry(), subscriber, testmocks.NewMockMonitorRepository(ctrl), testutil.NoopLogger(), WithWorkerCount(1))
			if err := svc.Run(context.Background()); err != nil {
				t.Fatal(err)
			}
			select {
			case <-msg.Acked():
			default:
				t.Fatal("malformed event was not acknowledged")
			}
		})
	})
}

func TestNotificationWorkerPoolStartLoop(t *testing.T) {
	t.Run("sends queued task", func(t *testing.T) {
		testutil.Case(t, "notification", "WorkerPool.StartLoop", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			contact := testutil.ObjectMother{}.Contact()
			provider := &notificationProviderSpy{}
			registry := NewProviderRegistry()
			registry.Register(contact.Type, provider)
			queue := make(chan Task, 1)
			queue <- Task{contact: contact, message: "hello"}
			close(queue)
			NewWorkerPool(registry, 1, queue, testutil.NoopLogger()).StartLoop(context.Background())
			provider.mu.Lock()
			defer provider.mu.Unlock()
			if provider.contact != contact || provider.message != "hello" {
				t.Fatalf("sent contact=%#v message=%q", provider.contact, provider.message)
			}
		})
	})
	t.Run("missing provider drops task and continues", func(t *testing.T) {
		testutil.Case(t, "notification", "WorkerPool.StartLoop", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			queue := make(chan Task, 1)
			queue <- Task{contact: testutil.ObjectMother{}.Contact(), message: "hello"}
			close(queue)
			NewWorkerPool(NewProviderRegistry(), 1, queue, testutil.NoopLogger()).StartLoop(context.Background())
		})
	})
}
