package metrics

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	baseservice "WatchTower/internal/service"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

type analyticsStub struct {
	events    []StatusEvent
	eventsErr error
	stats     SLAStats
	statsErr  error
}

func (s analyticsStub) GetStatusEvents(context.Context, uuid.UUID, time.Time, time.Time) ([]StatusEvent, error) {
	return s.events, s.eventsErr
}
func (s analyticsStub) GetSLAAggregation(context.Context, uuid.UUID, time.Time, time.Time) (SLAStats, error) {
	return s.stats, s.statsErr
}

type summariesStub struct {
	latest       []*probe.Summary
	latestErr    error
	period       []*probe.Summary
	periodErr    error
	latestLimit  int
	periodFrom   time.Time
	periodTo     time.Time
	latestCalled bool
}

func (s *summariesStub) GetMonitorLatestSummaries(_ context.Context, _ uuid.UUID, limit int) ([]*probe.Summary, error) {
	s.latestCalled = true
	s.latestLimit = limit
	return s.latest, s.latestErr
}
func (s *summariesStub) GetMonitorSummariesForPeriod(_ context.Context, _ uuid.UUID, from, to time.Time) ([]*probe.Summary, error) {
	s.periodFrom, s.periodTo = from, to
	return s.period, s.periodErr
}

func ownedMetricsService(t *testing.T, analytics AnalyticsRepository, summaries ProbeSummaryReadRepository, owner string) (MetricQueryService, uuid.UUID) {
	t.Helper()
	ctrl := gomock.NewController(t)
	provider := testmocks.NewMockUserProvider(ctrl)
	monitors := testmocks.NewMockMonitorRepository(ctrl)
	monitorID := uuid.New()
	provider.EXPECT().GetAuthorizedUser(gomock.Any()).Return(&user.User{Login: "alice"}, nil)
	monitors.EXPECT().GetByID(gomock.Any(), monitorID).Return(testutil.NewMonitorBuilder().WithOwner(owner).Build(), nil)
	return NewMetricsQueryService(monitors, provider, analytics, summaries), monitorID
}

func TestMetricsGetLastSummaries(t *testing.T) {
	t.Run("returns requested latest summaries", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetLastSummaries", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			expected := []*probe.Summary{{MonitorID: uuid.New()}}
			repository := &summariesStub{latest: expected}
			svc, monitorID := ownedMetricsService(t, analyticsStub{}, repository, "alice")
			got, err := svc.GetLastSummaries(context.Background(), monitorID, 3)
			if err != nil || !reflect.DeepEqual(got, expected) || repository.latestLimit != 3 {
				t.Fatalf("GetLastSummaries() = (%#v, %v), limit=%d", got, err, repository.latestLimit)
			}
		})
	})
	t.Run("non-positive limit returns empty without querying summaries", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetLastSummaries", "boundary", "london", func(t *testing.T, _ *allure.Context) {
			repository := &summariesStub{}
			svc, monitorID := ownedMetricsService(t, analyticsStub{}, repository, "alice")
			got, err := svc.GetLastSummaries(context.Background(), monitorID, 0)
			if err != nil || len(got) != 0 || repository.latestCalled {
				t.Fatalf("GetLastSummaries() = (%#v, %v), repository called=%v", got, err, repository.latestCalled)
			}
		})
	})
	t.Run("denies another user's monitor", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetLastSummaries", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, monitorID := ownedMetricsService(t, analyticsStub{}, &summariesStub{}, "bob")
			_, err := svc.GetLastSummaries(context.Background(), monitorID, 3)
			if !errors.Is(err, baseservice.ErrPermissionDenied) {
				t.Fatalf("GetLastSummaries() error = %v", err)
			}
		})
	})
	t.Run("returns summary repository failure", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetLastSummaries", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, monitorID := ownedMetricsService(t, analyticsStub{}, &summariesStub{latestErr: repo.ErrDB}, "alice")
			_, err := svc.GetLastSummaries(context.Background(), monitorID, 3)
			if !errors.Is(err, repo.ErrDB) {
				t.Fatalf("GetLastSummaries() error = %v", err)
			}
		})
	})
}

func TestMetricsGetSummariesForPeriod(t *testing.T) {
	from := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	t.Run("normalizes reversed range and returns summaries", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetSummariesForPeriod", "boundary", "london", func(t *testing.T, _ *allure.Context) {
			expected := []*probe.Summary{{MonitorID: uuid.New()}}
			repository := &summariesStub{period: expected}
			svc, monitorID := ownedMetricsService(t, analyticsStub{}, repository, "alice")
			got, err := svc.GetSummariesForPeriod(context.Background(), monitorID, to, from)
			if err != nil || !reflect.DeepEqual(got, expected) || !repository.periodFrom.Equal(from) || !repository.periodTo.Equal(to) {
				t.Fatalf("GetSummariesForPeriod() = (%#v, %v), range=(%v,%v)", got, err, repository.periodFrom, repository.periodTo)
			}
		})
	})
	t.Run("returns repository failure", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetSummariesForPeriod", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, monitorID := ownedMetricsService(t, analyticsStub{}, &summariesStub{periodErr: repo.ErrDB}, "alice")
			_, err := svc.GetSummariesForPeriod(context.Background(), monitorID, from, to)
			if !errors.Is(err, repo.ErrDB) {
				t.Fatalf("GetSummariesForPeriod() error = %v", err)
			}
		})
	})
}

func TestMetricsGetSLA(t *testing.T) {
	from := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	t.Run("returns aggregation", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetSLA", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			expected := SLAStats{MonitorID: uuid.New(), UptimePercent: 99.5, TotalDowntimeSec: 18, PeriodStart: from, PeriodEnd: to}
			svc, monitorID := ownedMetricsService(t, analyticsStub{stats: expected}, &summariesStub{}, "alice")
			got, err := svc.GetSLA(context.Background(), monitorID, from, to)
			if err != nil || got != expected {
				t.Fatalf("GetSLA() = (%#v, %v)", got, err)
			}
		})
	})
	t.Run("returns analytics failure", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetSLA", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, monitorID := ownedMetricsService(t, analyticsStub{statsErr: repo.ErrDB}, &summariesStub{}, "alice")
			_, err := svc.GetSLA(context.Background(), monitorID, from, to)
			if !errors.Is(err, repo.ErrDB) {
				t.Fatalf("GetSLA() error = %v", err)
			}
		})
	})
}

func TestMetricsGetStatusHistory(t *testing.T) {
	from := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	t.Run("clips events to query range and drops empty overlaps", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetStatusHistory", "boundary", "london", func(t *testing.T, a *allure.Context) {
			events := []StatusEvent{
				{Status: monitor.StatusDown, StartTime: from.Add(-time.Hour), EndTime: from.Add(15 * time.Minute), Reason: "before"},
				{Status: monitor.StatusUp, StartTime: from.Add(30 * time.Minute), Reason: "open"},
				{Status: monitor.StatusDown, StartTime: to, EndTime: to.Add(time.Minute), Reason: "outside"},
			}
			svc, monitorID := ownedMetricsService(t, analyticsStub{events: events}, &summariesStub{}, "alice")
			var got []StatusEvent
			var err error
			a.Step("Act: query bounded status history", func(*allure.Context) {
				got, err = svc.GetStatusHistory(context.Background(), monitorID, from, to)
			})
			a.Step("Assert: intervals are clipped and empty overlap removed", func(*allure.Context) {
				if err != nil || len(got) != 2 || !got[0].StartTime.Equal(from) || !got[0].EndTime.Equal(from.Add(15*time.Minute)) || !got[1].EndTime.Equal(to) {
					t.Fatalf("GetStatusHistory() = (%#v, %v)", got, err)
				}
			})
		})
	})
	t.Run("returns analytics failure", func(t *testing.T) {
		testutil.Case(t, "metrics", "GetStatusHistory", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			svc, monitorID := ownedMetricsService(t, analyticsStub{eventsErr: repo.ErrDB}, &summariesStub{}, "alice")
			_, err := svc.GetStatusHistory(context.Background(), monitorID, from, to)
			if !errors.Is(err, repo.ErrDB) {
				t.Fatalf("GetStatusHistory() error = %v", err)
			}
		})
	})
}
