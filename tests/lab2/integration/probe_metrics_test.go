//go:build lab2integration

package integration

import (
	"context"
	"testing"
	"time"

	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestProbeResultFeedsHistoryAndRedisMetrics(t *testing.T) {
	testsupport.Case(t, "integration", "probe-history-cache", func(t *testing.T, a *allure.Context) {
		f := setup(t, true)
		ctx := f.user(t, "alice")
		m := f.createMonitor(t, ctx, "API", "http://probe:8081/health", 30)
		probeAt := time.Now().UTC().Truncate(time.Microsecond)
		result, err := probe.NewProbeResult(*m.Target, 27, 200, nil)
		require.NoError(t, err, "%v", err)
		result.ProbeTime = probeAt
		scenario := &probeMetricsScenario{t: t, f: f, ctx: ctx, m: m, probeAt: probeAt, result: result}
		a.Step("Результат создается и выбирается для обработки", scenario.createPendingResult)
		a.Step("Смена статуса формирует историю, результат помечается обработанным", scenario.processResultAndHistory)
		a.Step("Метрики читаются из PostgreSQL и прогревают Redis", scenario.warmMetricsCache)
		a.Step("Отключение после UP закрывает предыдущий интервал истории", scenario.disableClosesHistory)
	})
}

type probeMetricsScenario struct {
	t       *testing.T
	f       *fixture
	ctx     context.Context
	m       *monitor.Monitor
	probeAt time.Time
	result  *probe.Result
}

func (s *probeMetricsScenario) createPendingResult(*allure.Context) {
	s.t.Helper()
	err := s.f.results.Create(s.result)
	require.NoError(s.t, err, "%v", err)
	pending, err := s.f.results.FetchUnprocessed(s.ctx, 10)
	require.NoError(s.t, err, "pending results = %#v, %v", pending, err)
	require.Len(s.t, pending, 1, "pending results = %#v, %v", pending, err)
	require.Equal(s.t, s.result.ID, pending[0].ID, "pending results = %#v, %v", pending, err)
}

func (s *probeMetricsScenario) processResultAndHistory(*allure.Context) {
	s.t.Helper()
	_, err := s.f.pool.Exec(s.ctx, `UPDATE monitor SET current_status='UP', last_evaluated_at=$2 WHERE id=$1`, s.m.ID, s.probeAt.Add(-time.Microsecond))
	require.NoError(s.t, err, "%v", err)
	if err := s.f.results.BulkUpdateStatus(s.ctx, []uuid.UUID{s.result.ID}, probe.ProcessingStatusProcessed); err != nil {
		s.t.Fatal(err)
	}
	pending, err := s.f.results.FetchUnprocessed(s.ctx, 10)
	require.NoError(s.t, err, "pending after processing = %#v, %v", pending, err)
	require.Len(s.t, pending, 0, "pending after processing = %#v, %v", pending, err)
	history, err := s.f.metrics.GetStatusHistory(s.ctx, s.m.ID, s.probeAt.Add(-time.Hour), s.probeAt.Add(time.Hour))
	require.NoError(s.t, err, "history = %#v, %v", history, err)
	require.Len(s.t, history, 1, "history = %#v, %v", history, err)
	require.Equal(s.t, monitor.StatusUp, history[0].Status, "history = %#v, %v", history, err)
}

func (s *probeMetricsScenario) warmMetricsCache(*allure.Context) {
	s.t.Helper()
	got, err := s.f.metrics.GetLastSummaries(s.ctx, s.m.ID, 10)
	require.NoError(s.t, err, "summaries = %#v, %v", got, err)
	require.Len(s.t, got, 1, "summaries = %#v, %v", got, err)
	require.Equal(s.t, monitor.StatusUp, got[0].MonitorStatus, "summaries = %#v, %v", got, err)
	require.EqualValues(s.t, 200, got[0].StatusCode, "summaries = %#v, %v", got, err)
	require.EqualValues(s.t, 27, got[0].LatencyMs, "summaries = %#v, %v", got, err)
	key := "probe_summary:monitor:" + s.m.ID.String()
	n, err := s.f.redis.LLen(s.ctx, key).Result()
	require.NoError(s.t, err, "Redis cache size = %d, %v", n, err)
	require.EqualValues(s.t, 1, n, "Redis cache size = %d, %v", n, err)
	period, err := s.f.metrics.GetSummariesForPeriod(s.ctx, s.m.ID, s.probeAt.Add(-time.Minute), s.probeAt.Add(time.Minute))
	require.NoError(s.t, err, "period summaries = %#v, %v", period, err)
	require.Len(s.t, period, 1, "period summaries = %#v, %v", period, err)
	require.EqualValues(s.t, 200, period[0].StatusCode, "period summaries = %#v, %v", period, err)
}

func (s *probeMetricsScenario) disableClosesHistory(*allure.Context) {
	s.t.Helper()
	err := s.f.monitors.Disable(s.ctx, s.m.ID)
	require.NoError(s.t, err, "disable monitor after UP: %v", err)
	history, err := s.f.metrics.GetStatusHistory(s.ctx, s.m.ID, s.probeAt.Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(s.t, err, "history after disable = %#v, %v", history, err)
	require.Len(s.t, history, 2, "history after disable = %#v, %v", history, err)
	latest, previous := history[0], history[1]
	require.Equal(s.t, monitor.StatusUnknown, latest.Status, "non-monotonic status intervals: previous=%#v latest=%#v", previous, latest)
	require.Equal(s.t, monitor.StatusUp, previous.Status, "non-monotonic status intervals: previous=%#v latest=%#v", previous, latest)
	require.True(s.t, latest.StartTime.After(previous.StartTime), "non-monotonic status intervals: previous=%#v latest=%#v", previous, latest)
	require.True(s.t, previous.EndTime.Equal(latest.StartTime), "non-monotonic status intervals: previous=%#v latest=%#v", previous, latest)
}
