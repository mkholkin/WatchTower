package service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

func TestSchedulerRun_DispatchesLoadedTarget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeClock := clockwork.NewFakeClockAt(time.Now())
	tgt := target.Target{ID: uuid.New(), Endpoint: "https://example.com", Config: target.HTTPConfig{Method: "GET"}, ProbeIntervalSec: 1}
	repo := testmocks.NewMockTargetRepository(ctrl)
	subscriber := testmocks.NewMockSubscriber(ctrl)

	repo.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{tgt}, nil)
	subscriber.EXPECT().Subscribe(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	}).AnyTimes()

	queue := make(chan target.Target, 1)
	s := NewScheduler(repo, subscriber, queue, fakeClock, testutil.NoopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	fakeClock.BlockUntil(1)
	fakeClock.Advance(2 * time.Second)

	select {
	case got := <-queue:
		if got.ID != tgt.ID {
			t.Fatalf("unexpected target id: got %s, want %s", got.ID, tgt.ID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected scheduler to dispatch loaded target")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}

func TestSchedulerRun_ReturnsRepoError(t *testing.T) {
	expectedErr := errors.New("db down")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := testmocks.NewMockTargetRepository(ctrl)
	subscriber := testmocks.NewMockSubscriber(ctrl)

	repo.EXPECT().GetAllActive(gomock.Any()).Return(nil, expectedErr)

	s := NewScheduler(repo, subscriber, make(chan target.Target, 1), clockwork.NewFakeClockAt(time.Now()), testutil.NoopLogger())

	err := s.Run(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestSchedulerRun_HandlesTargetCreatedEvent(t *testing.T) {
	targetID := uuid.New()
	tgt := target.Target{ID: targetID, Endpoint: "https://example.com", Config: target.HTTPConfig{Method: "GET"}, IsActive: true, ProbeIntervalSec: 1}
	createdCh := make(chan *message.Message, 1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeClock := clockwork.NewFakeClockAt(time.Now())
	repo := testmocks.NewMockTargetRepository(ctrl)
	subscriber := testmocks.NewMockSubscriber(ctrl)

	repo.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{}, nil)
	repo.EXPECT().GetByID(gomock.Any(), targetID).Return(&tgt, nil)
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetCreated).Return(createdCh, nil)
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetUpdated).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	})
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetDeleted).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	})

	queue := make(chan target.Target, 1)
	s := NewScheduler(repo, subscriber, queue, fakeClock, testutil.NoopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	fakeClock.BlockUntil(1)

	payload, _ := json.Marshal(TargetEvent{ID: targetID})
	msg := message.NewMessage(uuid.NewString(), payload)
	createdCh <- msg

	select {
	case <-msg.Acked():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected created event to be acknowledged")
	}

	fakeClock.Advance(2 * time.Second)

	select {
	case got := <-queue:
		if got.ID != tgt.ID {
			t.Fatalf("unexpected target id: got %s, want %s", got.ID, tgt.ID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected created target to be dispatched")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}

func TestSchedulerRun_HandlesTargetUpdatedEvent(t *testing.T) {
	targetID := uuid.New()
	initial := target.Target{ID: targetID, Endpoint: "https://example.com", Config: target.HTTPConfig{Method: "GET"}, IsActive: true, ProbeIntervalSec: 2}
	updated := initial
	updated.ProbeIntervalSec = 1
	updatedCh := make(chan *message.Message, 1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeClock := clockwork.NewFakeClockAt(time.Now())
	repo := testmocks.NewMockTargetRepository(ctrl)
	subscriber := testmocks.NewMockSubscriber(ctrl)

	repo.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{initial}, nil)
	repo.EXPECT().GetByID(gomock.Any(), targetID).Return(&updated, nil)
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetCreated).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	})
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetUpdated).Return(updatedCh, nil)
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetDeleted).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	})

	queue := make(chan target.Target, 1)
	s := NewScheduler(repo, subscriber, queue, fakeClock, testutil.NoopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	fakeClock.BlockUntil(1)

	payload, _ := json.Marshal(TargetEvent{ID: targetID})
	msg := message.NewMessage(uuid.NewString(), payload)
	updatedCh <- msg

	select {
	case <-msg.Acked():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected updated event to be acknowledged")
	}

	fakeClock.Advance(2 * time.Second)

	select {
	case got := <-queue:
		if got.ProbeIntervalSec != 1 {
			t.Fatalf("expected updated interval 1, got %d", got.ProbeIntervalSec)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected updated target to be dispatched")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}

func TestSchedulerRun_HandlesTargetDeleted(t *testing.T) {
	targetID := uuid.New()
	tgt := target.Target{ID: targetID, Endpoint: "https://example.com", Config: target.HTTPConfig{Method: "GET"}, IsActive: true, ProbeIntervalSec: 1}
	deletedCh := make(chan *message.Message, 1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fakeClock := clockwork.NewFakeClockAt(time.Now())
	repo := testmocks.NewMockTargetRepository(ctrl)
	subscriber := testmocks.NewMockSubscriber(ctrl)

	repo.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{tgt}, nil)
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetCreated).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	})
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetUpdated).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	})
	subscriber.EXPECT().Subscribe(gomock.Any(), TopicTargetDeleted).Return(deletedCh, nil)

	queue := make(chan target.Target, 1)
	s := NewScheduler(repo, subscriber, queue, fakeClock, testutil.NoopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	fakeClock.BlockUntil(1)

	payload, _ := json.Marshal(TargetEvent{ID: targetID})
	msg := message.NewMessage(uuid.NewString(), payload)
	deletedCh <- msg

	select {
	case <-msg.Acked():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected delete event to be acknowledged")
	}

	fakeClock.Advance(2 * time.Second)

	select {
	case got := <-queue:
		t.Fatalf("expected no target after deletion, got %s", got.ID)
	case <-time.After(150 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("scheduler did not stop after context cancellation")
	}
}
