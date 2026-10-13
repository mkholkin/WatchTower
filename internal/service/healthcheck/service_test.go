package healtcheck_service

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/jonboulle/clockwork"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestHealthCheckerRun(t *testing.T) {
	t.Run("probes and persists before clean cancellation", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "HealthChecker.Run", "state-transition", "london", func(t *testing.T, a *allure.Context) {
			var checker HealthChecker
			var clock *clockwork.FakeClock
			var cancel context.CancelFunc
			var saved chan *probe.Result
			var runErr error
			a.Step("Arrange one active target and a registered prober", func(*allure.Context) {
				ctrl := gomock.NewController(t)
				tgt := *testutil.ObjectMother{}.Target()
				tgt.ProbeIntervalSec = 1
				result := testutil.ObjectMother{}.Result()
				result.Target = &tgt
				targets := testmocks.NewMockTargetRepository(ctrl)
				probes := testmocks.NewMockProbeResultRepository(ctrl)
				subscriber := testmocks.NewMockSubscriber(ctrl)
				prober := testmocks.NewMockProber(ctrl)
				targets.EXPECT().GetAllActive(gomock.Any()).Return([]target.Target{tgt}, nil)
				subscriber.EXPECT().Subscribe(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, string) (<-chan *message.Message, error) {
					return closedMessageChannel(), nil
				}).AnyTimes()
				prober.EXPECT().Probe(gomock.Any(), gomock.Any()).Return(result, nil)
				saved = make(chan *probe.Result, 1)
				probes.EXPECT().Create(result).DoAndReturn(func(got *probe.Result) error {
					saved <- got
					return nil
				})
				registry := NewProberRegistry()
				registry.Register(target.ProtocolHTTP, prober)
				checker = NewHealthChecker(targets, probes, subscriber, registry, HealthCheckerConfig{WorkerCount: 1, TaskQueueSize: 1}, testutil.NoopLogger())
				clock = clockwork.NewFakeClockAt(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
				checker.(*healthChecker).scheduler.clock = clock
			})
			a.Step("Act: run one scheduled probe, persist it, then cancel", func(*allure.Context) {
				ctx, stop := context.WithCancel(context.Background())
				cancel = stop
				t.Cleanup(cancel)
				done := make(chan error, 1)
				go func() { done <- checker.Run(ctx) }()
				clock.BlockUntil(1)
				clock.Advance(SchedulerTickInterval)
				select {
				case <-saved:
				case <-time.After(2 * time.Second):
					t.Fatal("health checker did not persist the scheduled probe")
				}
				cancel()
				runErr = <-done
			})
			a.Step("Assert: the active pipeline stops cleanly", func(*allure.Context) {
				if runErr != nil {
					t.Fatalf("HealthChecker.Run() error = %v", runErr)
				}
			})
		})
	})
	t.Run("returns scheduler repository failure", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "HealthChecker.Run", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			targets := testmocks.NewMockTargetRepository(ctrl)
			targets.EXPECT().GetAllActive(gomock.Any()).Return(nil, repo.ErrDB)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			checker := NewHealthChecker(targets, testmocks.NewMockProbeResultRepository(ctrl), testmocks.NewMockSubscriber(ctrl), NewProberRegistry(), HealthCheckerConfig{WorkerCount: 1, TaskQueueSize: 1}, testutil.NoopLogger())
			if err := checker.Run(ctx); !errors.Is(err, repo.ErrDB) {
				t.Fatalf("HealthChecker.Run() error = %v", err)
			}
		})
	})
}
