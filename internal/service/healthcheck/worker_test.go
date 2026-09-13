package healtcheck_service

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestWorkerPoolRun(t *testing.T) {
	t.Run("probes target and persists timestamped result", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "WorkerPool.Run", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			tgt := *testutil.ObjectMother{}.Target()
			result := &probe.Result{Target: &tgt}
			registry := NewProberRegistry()
			prober := testmocks.NewMockProber(ctrl)
			registry.Register(target.ProtocolHTTP, prober)
			repository := testmocks.NewMockProbeResultRepository(ctrl)
			queue := make(chan target.Target, 1)
			queue <- tgt
			ctx, cancel := context.WithCancel(context.Background())
			prober.EXPECT().Probe(gomock.Any(), gomock.Any()).Return(result, nil)
			repository.EXPECT().Create(result).DoAndReturn(func(got *probe.Result) error {
				if got.Target.ID != tgt.ID || got.ProbeTime.IsZero() {
					t.Fatalf("persisted result = %#v", got)
				}
				cancel()
				return nil
			})

			NewWorkerPool(registry, repository, queue, 1, testutil.NoopLogger()).Run(ctx)
		})
	})
	t.Run("probe failure is not persisted", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "WorkerPool.Run", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			tgt := *testutil.ObjectMother{}.Target()
			registry := NewProberRegistry()
			prober := testmocks.NewMockProber(ctrl)
			registry.Register(target.ProtocolHTTP, prober)
			repository := testmocks.NewMockProbeResultRepository(ctrl)
			queue := make(chan target.Target, 1)
			queue <- tgt
			ctx, cancel := context.WithCancel(context.Background())
			prober.EXPECT().Probe(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *target.Target) (*probe.Result, error) {
				cancel()
				return nil, errors.New("dial failed")
			})

			NewWorkerPool(registry, repository, queue, 1, testutil.NoopLogger()).Run(ctx)
		})
	})
	t.Run("missing protocol prober is skipped", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "WorkerPool.Run", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			repository := testmocks.NewMockProbeResultRepository(ctrl)
			queue := make(chan target.Target, 1)
			queue <- target.Target{Config: target.TCPConfig{Port: 80}}
			close(queue)

			NewWorkerPool(NewProberRegistry(), repository, queue, 1, testutil.NoopLogger()).Run(context.Background())
		})
	})
}
