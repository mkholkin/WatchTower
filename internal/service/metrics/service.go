package metrics

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service/common/ownership"
	"WatchTower/internal/service/common/provider"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MetricQueryService interface {
	GetLastSummaries(ctx context.Context, monitorID uuid.UUID, n int) ([]*probe.Summary, error)
	GetSummariesForPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*probe.Summary, error)
	GetSLA(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (SLAStats, error)
	GetStatusHistory(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]StatusEvent, error)
}

type AnalyticsRepository interface {
	GetStatusEvents(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]StatusEvent, error)
	GetSLAAggregation(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (SLAStats, error)
}

type ProbeSummaryRepository interface {
	GetByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]*probe.Summary, error)
	GetByMonitorIDForPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*probe.Summary, error)
}

type metricsQueryService struct {
	monitorRepo      repo.MonitorRepository
	userProvider     provider.UserProvider
	analyticsRepo    AnalyticsRepository
	probeSummaryRepo ProbeSummaryRepository
}

func NewMetricsQueryService(
	monitorRepo repo.MonitorRepository,
	userProvider provider.UserProvider,
	analyticsRepo AnalyticsRepository,
	probeSummaryRepo ProbeSummaryRepository,
) MetricQueryService {
	return &metricsQueryService{
		monitorRepo:      monitorRepo,
		userProvider:     userProvider,
		analyticsRepo:    analyticsRepo,
		probeSummaryRepo: probeSummaryRepo,
	}
}

func (m *metricsQueryService) GetLastSummaries(ctx context.Context, monitorID uuid.UUID, n int) ([]*probe.Summary, error) {
	if err := m.ensureOwnedMonitor(ctx, monitorID); err != nil {
		return nil, err
	}
	if n <= 0 {
		return []*probe.Summary{}, nil
	}

	return m.probeSummaryRepo.GetByMonitorID(ctx, monitorID, n)

}

func (m *metricsQueryService) GetSummariesForPeriod(
	ctx context.Context,
	monitorID uuid.UUID,
	from, to time.Time,
) ([]*probe.Summary, error) {
	if err := m.ensureOwnedMonitor(ctx, monitorID); err != nil {
		return nil, err
	}

	if to.Before(from) {
		from, to = to, from
	}

	return m.probeSummaryRepo.GetByMonitorIDForPeriod(ctx, monitorID, from, to)
}

func (m *metricsQueryService) GetSLA(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (SLAStats, error) {
	if err := m.ensureOwnedMonitor(ctx, monitorID); err != nil {
		return SLAStats{}, err
	}

	return m.analyticsRepo.GetSLAAggregation(ctx, monitorID, from, to)
}

func (m *metricsQueryService) GetStatusHistory(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]StatusEvent, error) {
	if err := m.ensureOwnedMonitor(ctx, monitorID); err != nil {
		return nil, err
	}

	events, err := m.analyticsRepo.GetStatusEvents(ctx, monitorID, from, to)
	if err != nil {
		return nil, err
	}

	return clipStatusEvents(events, from, to), nil
}

func (m *metricsQueryService) ensureOwnedMonitor(ctx context.Context, monitorID uuid.UUID) error {
	if m.monitorRepo == nil {
		return fmt.Errorf("monitor repository is not configured")
	}
	if m.userProvider == nil {
		return fmt.Errorf("user provider is not configured")
	}

	_, err := ownership.GetOwnedMonitor(ctx, m.userProvider, m.monitorRepo, monitorID)
	return err
}

func clipStatusEvents(events []StatusEvent, from, to time.Time) []StatusEvent {
	clipped := make([]StatusEvent, 0, len(events))

	for i := range events {
		e := events[i]
		start := maxTime(e.StartTime, from)

		end := e.EndTime
		if end.IsZero() {
			end = to
		}
		end = minTime(end, to)

		if !end.After(start) {
			continue
		}

		e.StartTime = start
		e.EndTime = end
		clipped = append(clipped, e)
	}

	return clipped
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}

	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}

	return b
}
