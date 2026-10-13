//go:build lab2integration

package integration

import (
	"testing"
	"time"

	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/google/uuid"
)

func TestProbeResultFeedsHistoryAndRedisMetrics(t *testing.T) {
	testsupport.Case(t, "integration", "probe-history-cache", func(t *testing.T, a *allure.Context) {
		f := setup(t, true)
		ctx := f.user(t, "alice")
		m := f.createMonitor(t, ctx, "API", "http://probe:8081/health", 30)
		probeAt := time.Now().UTC().Truncate(time.Microsecond)
		result, err := probe.NewProbeResult(*m.Target, 27, 200, nil)
		if err != nil {
			t.Fatal(err)
		}
		result.ProbeTime = probeAt
		a.Step("Результат создается и выбирается для обработки", func(*allure.Context) {
			if err := f.results.Create(result); err != nil {
				t.Fatal(err)
			}
			pending, err := f.results.FetchUnprocessed(ctx, 10)
			if err != nil || len(pending) != 1 || pending[0].ID != result.ID {
				t.Fatalf("pending results = %#v, %v", pending, err)
			}
		})
		a.Step("Смена статуса формирует историю, результат помечается обработанным", func(*allure.Context) {
			_, err := f.pool.Exec(ctx, `UPDATE monitor SET current_status='UP', last_evaluated_at=$2 WHERE id=$1`, m.ID, probeAt.Add(-time.Microsecond))
			if err != nil {
				t.Fatal(err)
			}
			if err := f.results.BulkUpdateStatus(ctx, []uuid.UUID{result.ID}, probe.ProcessingStatusProcessed); err != nil {
				t.Fatal(err)
			}
			pending, err := f.results.FetchUnprocessed(ctx, 10)
			if err != nil || len(pending) != 0 {
				t.Fatalf("pending after processing = %#v, %v", pending, err)
			}
			history, err := f.metrics.GetStatusHistory(ctx, m.ID, probeAt.Add(-time.Hour), probeAt.Add(time.Hour))
			if err != nil || len(history) != 1 || history[0].Status != monitor.StatusUp {
				t.Fatalf("history = %#v, %v", history, err)
			}
		})
		a.Step("Метрики читаются из PostgreSQL и прогревают Redis", func(*allure.Context) {
			got, err := f.metrics.GetLastSummaries(ctx, m.ID, 10)
			if err != nil || len(got) != 1 || got[0].MonitorStatus != monitor.StatusUp || got[0].StatusCode != 200 || got[0].LatencyMs != 27 {
				t.Fatalf("summaries = %#v, %v", got, err)
			}
			key := "probe_summary:monitor:" + m.ID.String()
			if n, err := f.redis.LLen(ctx, key).Result(); err != nil || n != 1 {
				t.Fatalf("Redis cache size = %d, %v", n, err)
			}
			period, err := f.metrics.GetSummariesForPeriod(ctx, m.ID, probeAt.Add(-time.Minute), probeAt.Add(time.Minute))
			if err != nil || len(period) != 1 || period[0].StatusCode != 200 {
				t.Fatalf("period summaries = %#v, %v", period, err)
			}
		})
		a.Step("Отключение после UP закрывает предыдущий интервал истории", func(*allure.Context) {
			if err := f.monitors.Disable(ctx, m.ID); err != nil {
				t.Fatalf("disable monitor after UP: %v", err)
			}
			history, err := f.metrics.GetStatusHistory(ctx, m.ID, probeAt.Add(-time.Hour), time.Now().Add(time.Hour))
			if err != nil || len(history) != 2 {
				t.Fatalf("history after disable = %#v, %v", history, err)
			}
			latest, previous := history[0], history[1]
			if latest.Status != monitor.StatusUnknown || previous.Status != monitor.StatusUp ||
				!latest.StartTime.After(previous.StartTime) || !previous.EndTime.Equal(latest.StartTime) {
				t.Fatalf("non-monotonic status intervals: previous=%#v latest=%#v", previous, latest)
			}
		})
	})
}
