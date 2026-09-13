package healtcheck_service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func closedMessageChannel() <-chan *message.Message {
	ch := make(chan *message.Message)
	close(ch)
	return ch
}

func TestSchedulerRun(t *testing.T) {
	t.Run("dispatches loaded active target", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "Scheduler.Run", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			clock := clockwork.NewFakeClockAt(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
			tgt := *testutil.ObjectMother{}.Target()
			tgt.ProbeIntervalSec = 1
			repository := testmocks.NewMockTargetRepository(ctrl)
			subscriber := testmocks.NewMockSubscriber(ctrl)
			repository.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{tgt}, nil)
			subscriber.EXPECT().Subscribe(gomock.Any(), gomock.Any()).Return(closedMessageChannel(), nil).AnyTimes()
			queue := make(chan target.Target, 1)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- NewScheduler(repository, subscriber, queue, clock, testutil.NoopLogger()).Run(ctx) }()
			clock.BlockUntil(1)
			clock.Advance(SchedulerTickInterval)
			got := <-queue
			cancel()
			if err := <-done; err != nil || got.ID != tgt.ID {
				t.Fatalf("Scheduler.Run() error=%v target=%s, want %s", err, got.ID, tgt.ID)
			}
		})
	})
	t.Run("returns active target repository failure", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "Scheduler.Run", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			repository := testmocks.NewMockTargetRepository(ctrl)
			repository.EXPECT().GetAllActive(gomock.Any()).Return(nil, repo.ErrDB)
			scheduler := NewScheduler(repository, testmocks.NewMockSubscriber(ctrl), make(chan target.Target, 1), clockwork.NewFakeClock(), testutil.NoopLogger())
			if err := scheduler.Run(context.Background()); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("Scheduler.Run() error = %v", err)
			}
		})
	})
	t.Run("created event loads and dispatches target", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "Scheduler.Run", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			clock := clockwork.NewFakeClock()
			tgt := *testutil.ObjectMother{}.Target()
			tgt.ProbeIntervalSec = 1
			created := make(chan *message.Message, 1)
			repository := testmocks.NewMockTargetRepository(ctrl)
			subscriber := testmocks.NewMockSubscriber(ctrl)
			repository.EXPECT().GetAllActive(gomock.Any()).Return(nil, nil)
			repository.EXPECT().GetByID(gomock.Any(), tgt.ID).Return(&tgt, nil)
			subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetCreated).Return(created, nil)
			subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetUpdated).Return(closedMessageChannel(), nil)
			subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetDeleted).Return(closedMessageChannel(), nil)
			queue := make(chan target.Target, 1)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- NewScheduler(repository, subscriber, queue, clock, testutil.NoopLogger()).Run(ctx) }()
			clock.BlockUntil(1)
			payload, err := json.Marshal(TargetEvent{ID: tgt.ID})
			if err != nil {
				t.Fatal(err)
			}
			msg := message.NewMessage(uuid.NewString(), payload)
			created <- msg
			<-msg.Acked()
			clock.Advance(SchedulerTickInterval)
			got := <-queue
			cancel()
			if err := <-done; err != nil || got.ID != tgt.ID {
				t.Fatalf("Scheduler.Run() error=%v target=%s, want %s", err, got.ID, tgt.ID)
			}
		})
	})
	t.Run("updated event replaces scheduled target", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "Scheduler.Run", "state-transition", "london", func(t *testing.T, a *allure.Context) {
			var clock *clockwork.FakeClock
			var updated chan *message.Message
			var updatedTarget target.Target
			var queue chan target.Target
			var cancel context.CancelFunc
			var done chan error
			a.Step("Arrange loaded target and update subscription", func(*allure.Context) {
				ctrl := gomock.NewController(t)
				clock = clockwork.NewFakeClock()
				initial := *testutil.ObjectMother{}.Target()
				initial.ProbeIntervalSec = 30
				updatedTarget = initial
				updatedTarget.ProbeIntervalSec = 5
				updated = make(chan *message.Message, 1)
				repository := testmocks.NewMockTargetRepository(ctrl)
				subscriber := testmocks.NewMockSubscriber(ctrl)
				repository.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{initial}, nil)
				repository.EXPECT().GetByID(gomock.Any(), initial.ID).Return(&updatedTarget, nil)
				subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetCreated).Return(closedMessageChannel(), nil)
				subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetUpdated).Return(updated, nil)
				subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetDeleted).Return(closedMessageChannel(), nil)
				queue = make(chan target.Target, 1)
				scheduler := NewScheduler(repository, subscriber, queue, clock, testutil.NoopLogger())
				ctx, stop := context.WithCancel(context.Background())
				cancel = stop
				done = make(chan error, 1)
				go func() { done <- scheduler.Run(ctx) }()
				clock.BlockUntil(1)
			})
			a.Step("Act: deliver target update", func(*allure.Context) {
				payload, err := json.Marshal(TargetEvent{ID: updatedTarget.ID})
				if err != nil {
					t.Fatal(err)
				}
				msg := message.NewMessage(uuid.NewString(), payload)
				updated <- msg
				<-msg.Acked()
			})
			a.Step("Assert: next fake-clock tick dispatches the updated target", func(*allure.Context) {
				clock.Advance(SchedulerTickInterval)
				got := <-queue
				cancel()
				if err := <-done; err != nil || got.ID != updatedTarget.ID || got.ProbeIntervalSec != 5 {
					t.Fatalf("Scheduler.Run() error=%v dispatched=%#v", err, got)
				}
			})
		})
	})
	t.Run("deleted event removes scheduled target", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "Scheduler.Run", "state-transition", "london", func(t *testing.T, a *allure.Context) {
			var clock *clockwork.FakeClock
			var deleted chan *message.Message
			var targetID uuid.UUID
			var survivorID uuid.UUID
			var queue chan target.Target
			var cancel context.CancelFunc
			var done chan error
			a.Step("Arrange loaded target and delete subscription", func(*allure.Context) {
				ctrl := gomock.NewController(t)
				clock = clockwork.NewFakeClock()
				initial := *testutil.ObjectMother{}.Target()
				targetID = initial.ID
				survivor := *testutil.ObjectMother{}.Target()
				survivorID = survivor.ID
				deleted = make(chan *message.Message, 1)
				repository := testmocks.NewMockTargetRepository(ctrl)
				subscriber := testmocks.NewMockSubscriber(ctrl)
				repository.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{initial, survivor}, nil)
				subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetCreated).Return(closedMessageChannel(), nil)
				subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetUpdated).Return(closedMessageChannel(), nil)
				subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetDeleted).Return(deleted, nil)
				queue = make(chan target.Target, 3)
				scheduler := NewScheduler(repository, subscriber, queue, clock, testutil.NoopLogger())
				ctx, stop := context.WithCancel(context.Background())
				cancel = stop
				done = make(chan error, 1)
				go func() { done <- scheduler.Run(ctx) }()
				clock.BlockUntil(1)
			})
			a.Step("Act: deliver target deletion", func(*allure.Context) {
				payload, err := json.Marshal(TargetEvent{ID: targetID})
				if err != nil {
					t.Fatal(err)
				}
				msg := message.NewMessage(uuid.NewString(), payload)
				deleted <- msg
				<-msg.Acked()
			})
			a.Step("Assert: next fake-clock tick dispatches only the surviving target", func(*allure.Context) {
				clock.Advance(SchedulerTickInterval)
				got := <-queue
				barrierPayload, err := json.Marshal(TargetEvent{ID: uuid.New()})
				if err != nil {
					t.Fatal(err)
				}
				barrier := message.NewMessage(uuid.NewString(), barrierPayload)
				deleted <- barrier
				<-barrier.Acked()
				cancel()
				if runErr := <-done; runErr != nil || got.ID != survivorID {
					t.Fatalf("Scheduler.Run() error=%v dispatched=%s, want survivor %s", runErr, got.ID, survivorID)
				}
				select {
				case extra := <-queue:
					t.Fatalf("deleted target %s was dispatched as extra %#v", targetID, extra)
				default:
				}
			})
		})
	})
}
