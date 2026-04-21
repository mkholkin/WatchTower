package healtcheck_service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/golang/mock/gomock"
)

func TestHealthChecker_Run_ReturnsSchedulerError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("failed to load targets")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	targetRepo := testmocks.NewMockTargetRepository(ctrl)
	probeRepo := testmocks.NewMockProbeResultRepository(ctrl)
	subscriber := testmocks.NewMockSubscriber(ctrl)

	targetRepo.EXPECT().GetAllActive(gomock.Any()).Return(nil, expectedErr)

	checker := NewHealthChecker(
		targetRepo,
		probeRepo,
		subscriber,
		NewProberRegistry(),
		HealthCheckerConfig{WorkerCount: 1, TaskQueueSize: 1},
		testutil.NoopLogger(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := checker.Run(ctx)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestHealthChecker_Run_StopsOnCanceledContext(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	targetRepo := testmocks.NewMockTargetRepository(ctrl)
	probeRepo := testmocks.NewMockProbeResultRepository(ctrl)
	subscriber := testmocks.NewMockSubscriber(ctrl)

	targetRepo.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{}, nil)
	subscriber.EXPECT().Subscribe(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ string) (<-chan *message.Message, error) {
		ch := make(chan *message.Message)
		close(ch)
		return ch, nil
	}).AnyTimes()

	checker := NewHealthChecker(
		targetRepo,
		probeRepo,
		subscriber,
		NewProberRegistry(),
		HealthCheckerConfig{WorkerCount: 1, TaskQueueSize: 1},
		testutil.NoopLogger(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		done <- checker.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("health checker did not stop on canceled context")
	}
}
