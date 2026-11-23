package analyzation_service

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

type analyzationPublisher struct {
	topic  string
	event  MonitorStatusChangedEvent
	err    error
	called bool
}

func (p *analyzationPublisher) Publish(topic string, messages ...*message.Message) error {
	p.called = true
	p.topic = topic
	if len(messages) == 1 {
		_ = json.Unmarshal(messages[0].Payload, &p.event)
	}
	return p.err
}
func (*analyzationPublisher) Close() error { return nil }

func TestProbeAnalyzationServiceRun(t *testing.T) {
	t.Run("returns cleanly when canceled after empty cycle", analysisCanceledAfterEmptyCycle)

	t.Run("returns fetch failure", analysisReturnsFetchFailure)

	t.Run("load sheds, persists evaluation, and publishes status transition", analysisPersistsStatusTransition)
}

func analysisCanceledAfterEmptyCycle(t *testing.T) {
	testutil.Case(t, "analyze", "ProbeAnalyzationService.Run", "state-transition", "london", func(t *testing.T, a *allure.Context) {
		ctrl := gomock.NewController(t)
		probes := testmocks.NewMockProbeResultRepository(ctrl)
		ctx, cancel := context.WithCancel(context.Background())
		probes.EXPECT().FetchUnprocessed(gomock.Any(), 10).DoAndReturn(func(context.Context, int) ([]*probe.Result, error) {
			cancel()
			return nil, nil
		})
		svc := NewProbeAnalyzationService(testmocks.NewMockMonitorRepository(ctrl), probes, testmocks.NewMockProbeSummaryRepository(ctrl), testmocks.NewMockProbeEvaluator(ctrl), &analyzationPublisher{}, ProbeAnalyzationServiceConfig{FetchLimit: 10, LoadSheddingThreshold: 5}, testutil.NoopLogger())
		var runErr error
		a.Step("Act: run one empty cycle", func(*allure.Context) { runErr = svc.Run(ctx) })
		a.Step("Assert: cancellation exits cleanly", func(*allure.Context) {
			if runErr != nil {
				t.Fatalf("Run() error = %v", runErr)
			}
		})
	})
}

func analysisReturnsFetchFailure(t *testing.T) {
	testutil.Case(t, "analyze", "ProbeAnalyzationService.Run", "equivalence", "london", func(t *testing.T, a *allure.Context) {
		ctrl := gomock.NewController(t)
		probes := testmocks.NewMockProbeResultRepository(ctrl)
		probes.EXPECT().FetchUnprocessed(gomock.Any(), 10).Return(nil, repo.ErrDB)
		svc := NewProbeAnalyzationService(testmocks.NewMockMonitorRepository(ctrl), probes, testmocks.NewMockProbeSummaryRepository(ctrl), testmocks.NewMockProbeEvaluator(ctrl), &analyzationPublisher{}, ProbeAnalyzationServiceConfig{FetchLimit: 10, LoadSheddingThreshold: 5}, testutil.NoopLogger())
		var runErr error
		a.Step("Act: run analysis", func(*allure.Context) { runErr = svc.Run(context.Background()) })
		a.Step("Assert: fetch failure is preserved", func(*allure.Context) {
			if !errors.Is(runErr, repo.ErrDB) {
				t.Fatalf("Run() error = %v", runErr)
			}
		})
	})
}

func analysisPersistsStatusTransition(t *testing.T) {
	testutil.Case(t, "analyze", "ProbeAnalyzationService.Run", "state-transition", "london", func(t *testing.T, a *allure.Context) {
		ctrl := gomock.NewController(t)
		monitors := testmocks.NewMockMonitorRepository(ctrl)
		probes := testmocks.NewMockProbeResultRepository(ctrl)
		summaries := testmocks.NewMockProbeSummaryRepository(ctrl)
		evaluator := testmocks.NewMockProbeEvaluator(ctrl)
		publisher := &analyzationPublisher{}
		stale := testutil.ObjectMother{}.Result()
		active := testutil.ObjectMother{}.Result()
		mon := testutil.NewMonitorBuilder().WithStatus(monitor.StatusUnknown).Build()
		mon.Target = active.Target
		ctx, cancel := context.WithCancel(context.Background())

		probes.EXPECT().FetchUnprocessed(gomock.Any(), 2).Return([]*probe.Result{stale, active}, nil)
		probes.EXPECT().BulkUpdateStatus(gomock.Any(), []uuid.UUID{stale.ID}, probe.ProcessingStatusCanceled).Return(nil)
		monitors.EXPECT().GetMonitorsToEvaluate(gomock.Any(), []uuid.UUID{active.Target.ID}).Return(map[uuid.UUID][]*monitor.Monitor{active.Target.ID: {mon}}, nil)
		evaluator.EXPECT().Evaluate(gomock.Any(), active, mon).Return(monitor.StatusUp, nil)
		summaries.EXPECT().BulkCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got []*probe.Summary) error {
			assertPersistedSummary(t, got, mon.ID, active.LatencyMs)
			return nil
		})
		probes.EXPECT().BulkUpdateStatus(gomock.Any(), []uuid.UUID{active.ID}, probe.ProcessingStatusProcessed).Return(nil)
		monitors.EXPECT().BulkUpdateEvaluation(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got []*monitor.Monitor) error {
			if len(got) != 1 || got[0].CurrentStatus != monitor.StatusUp || !got[0].LastEvaluatedAt.Equal(active.ProbeTime) {
				t.Fatalf("updated monitors = %#v", got)
			}
			cancel()
			return nil
		})

		svc := NewProbeAnalyzationService(monitors, probes, summaries, evaluator, publisher, ProbeAnalyzationServiceConfig{FetchLimit: 2, LoadSheddingThreshold: 1}, testutil.NoopLogger())
		var runErr error
		a.Step("Act: run controlled analysis cycle", func(*allure.Context) { runErr = svc.Run(ctx) })
		a.Step("Assert: persistence and status event completed", func(*allure.Context) {
			if runErr != nil {
				t.Fatalf("Run() error = %v", runErr)
			}
			assertPublishedTransition(t, publisher, mon.ID)
		})
	})
}

func assertPersistedSummary(t *testing.T, got []*probe.Summary, monitorID uuid.UUID, latencyMs int32) {
	t.Helper()
	if len(got) != 1 || got[0].MonitorID != monitorID || got[0].MonitorStatus != monitor.StatusUp || got[0].LatencyMs != latencyMs || got[0].StatusCode != 200 {
		t.Fatalf("persisted summaries = %#v", got)
	}
}

func assertPublishedTransition(t *testing.T, publisher *analyzationPublisher, monitorID uuid.UUID) {
	t.Helper()
	if !publisher.called || publisher.topic != TopicMonitorStatusChanged || publisher.event.MonitorID != monitorID || publisher.event.OldStatus != monitor.StatusUnknown || publisher.event.NewStatus != monitor.StatusUp || publisher.event.OccurredAt.IsZero() {
		t.Fatalf("published status event = %#v topic=%q", publisher.event, publisher.topic)
	}
}
