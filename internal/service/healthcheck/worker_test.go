package service

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

func TestWorkerPoolRun_SavesProbeResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	targetID := uuid.New()
	tgt := target.Target{ID: targetID, Endpoint: "https://example.com", Config: target.HTTPConfig{Method: "GET"}, ProbeIntervalSec: 10}

	registry := NewProberRegistry()
	prober := testmocks.NewMockProber(ctrl)
	registry.Register(target.ProtocolHTTP, prober)
	prober.EXPECT().Probe(gomock.Any(), gomock.AssignableToTypeOf(&target.Target{})).DoAndReturn(func(_ context.Context, probed *target.Target) (probe.Result, error) {
		return probe.Result{ID: uuid.New(), Target: probed, ProbeTime: time.Now()}, nil
	})

	saved := make(chan struct{}, 1)
	repo := testmocks.NewMockProbeResultRepository(ctrl)
	repo.EXPECT().Create(gomock.AssignableToTypeOf(&probe.Result{})).DoAndReturn(func(result *probe.Result) error {
		if result.Target == nil || result.Target.ID != targetID {
			t.Fatalf("unexpected probe result target in repository create")
		}
		saved <- struct{}{}
		return nil
	})

	queue := make(chan target.Target, 1)
	wp := NewWorkerPool(registry, repo, queue, 1, testutil.NoopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go wp.Run(ctx)
	queue <- tgt

	select {
	case <-saved:
	case <-time.After(2 * time.Second):
		t.Fatal("expected probe result to be saved")
	}
}

func TestWorkerPoolRun_MissingProberDoesNotSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tgt := target.Target{ID: uuid.New(), Endpoint: "127.0.0.1", Config: target.TCPConfig{Port: 80}, ProbeIntervalSec: 10}
	registry := NewProberRegistry()

	repo := testmocks.NewMockProbeResultRepository(ctrl)
	repo.EXPECT().Create(gomock.Any()).Times(0)

	queue := make(chan target.Target, 1)
	wp := NewWorkerPool(registry, repo, queue, 1, testutil.NoopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go wp.Run(ctx)
	queue <- tgt
	time.Sleep(700 * time.Millisecond)
}

func TestWorkerPoolRun_StopsOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registry := NewProberRegistry()
	repo := testmocks.NewMockProbeResultRepository(ctrl)
	queue := make(chan target.Target)
	wp := NewWorkerPool(registry, repo, queue, 2, testutil.NoopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		wp.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker pool did not stop after context cancellation")
	}
}
