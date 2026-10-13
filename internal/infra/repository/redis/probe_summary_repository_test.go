package redis_test

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/redis"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/go-redis/redismock/v9"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type cacheFixture struct {
	mock         redismock.ClientMock
	fallback     *testmocks.MockProbeSummaryRepository
	repo         repo.ProbeSummaryRepository
	id           uuid.UUID
	summary      *probe.Summary
	key, payload string
}

func newCacheFixture(t *testing.T) *cacheFixture {
	t.Helper()
	client, mock := redismock.NewClientMock()
	ctrl := gomock.NewController(t)
	fallback := testmocks.NewMockProbeSummaryRepository(ctrl)
	id := uuid.New()
	summary := &probe.Summary{MonitorID: id, MonitorStatus: monitor.StatusUp, LatencyMs: 12, ProbeTime: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(summary)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, mock.ExpectationsWereMet()); require.NoError(t, client.Close()) })
	return &cacheFixture{mock: mock, fallback: fallback, repo: redis.NewProbeSummaryRepository(client, fallback, testutil.NoopLogger()), id: id, summary: summary, key: fmt.Sprintf("probe_summary:monitor:%s", id), payload: string(data)}
}
func (f *cacheFixture) expectPush() {
	f.mock.ExpectLPush(f.key, []byte(f.payload)).SetVal(1)
	f.mock.ExpectLTrim(f.key, 0, 99).SetVal("OK")
}
func (f *cacheFixture) expectPushError() {
	f.mock.ExpectLPush(f.key, []byte(f.payload)).SetErr(storageFailure)
}

var storageFailure = errors.New("storage failed")

func TestCache_GetMonitorLatestSummaries(t *testing.T) {
	cases := []string{"cache_hit", "cache_miss", "redis_error", "database_error", "corrupt_cache_repaired", "corrupt_cache_fallback_error", "zero_limit", "large_limit"}
	for _, name := range testutil.Shuffle(t, cases) {
		t.Run(name, func(t *testing.T) {
			testutil.Case(t, "RedisProbeSummaryRepository", "GetMonitorLatestSummaries", "equivalence/boundary", "london", func(t *testing.T, a *allure.Context) {
				var f *cacheFixture
				var got []*probe.Summary
				var err error
				limit := 1
				a.Step("Arrange", func(*allure.Context) {
					f = newCacheFixture(t)
					switch name {
					case "cache_hit":
						f.mock.ExpectLRange(f.key, 0, 0).SetVal([]string{f.payload})
						f.mock.ExpectExpire(f.key, 30*time.Minute).SetVal(true)
					case "corrupt_cache_repaired", "corrupt_cache_fallback_error":
						f.mock.ExpectLRange(f.key, 0, 0).SetVal([]string{"{"})
						f.mock.ExpectDel(f.key).SetVal(1)
						if name == "corrupt_cache_fallback_error" {
							f.fallback.EXPECT().GetMonitorLatestSummaries(gomock.Any(), f.id, 1).Return(nil, storageFailure)
						} else {
							f.fallback.EXPECT().GetMonitorLatestSummaries(gomock.Any(), f.id, 1).Return([]*probe.Summary{f.summary}, nil)
							f.expectPush()
							f.mock.ExpectExpire(f.key, 30*time.Minute).SetVal(true)
						}
					case "cache_miss", "redis_error":
						if name == "cache_miss" {
							f.mock.ExpectLRange(f.key, 0, 0).SetVal(nil)
						} else {
							f.mock.ExpectLRange(f.key, 0, 0).SetErr(storageFailure)
						}
						f.fallback.EXPECT().GetMonitorLatestSummaries(gomock.Any(), f.id, 1).Return([]*probe.Summary{f.summary}, nil)
						f.expectPush()
						f.mock.ExpectExpire(f.key, 30*time.Minute).SetVal(true)
					case "database_error":
						f.mock.ExpectLRange(f.key, 0, 0).SetVal(nil)
						f.fallback.EXPECT().GetMonitorLatestSummaries(gomock.Any(), f.id, 1).Return(nil, storageFailure)
					case "zero_limit":
						limit = 0
					case "large_limit":
						limit = 101
						f.fallback.EXPECT().GetMonitorLatestSummaries(gomock.Any(), f.id, 101).Return([]*probe.Summary{f.summary}, nil)
					}
				})
				a.Step("Act", func(*allure.Context) { got, err = f.repo.GetMonitorLatestSummaries(context.Background(), f.id, limit) })
				a.Step("Assert", func(*allure.Context) {
					switch name {
					case "database_error", "corrupt_cache_fallback_error":
						require.ErrorIs(t, err, storageFailure)
					case "zero_limit":
						require.NoError(t, err)
						require.Empty(t, got)
					default:
						require.NoError(t, err)
						require.Len(t, got, 1)
						require.Equal(t, f.id, got[0].MonitorID)
						require.Equal(t, monitor.StatusUp, got[0].MonitorStatus)
						require.EqualValues(t, 12, got[0].LatencyMs)
					}
					require.NoError(t, f.mock.ExpectationsWereMet())
				})
			})
		})
	}
}
func TestCache_GetMonitorSummariesForPeriod(t *testing.T) {
	for _, bad := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "database_error"}[bad], func(t *testing.T) {
			testutil.Case(t, "RedisProbeSummaryRepository", "GetMonitorSummariesForPeriod", "equivalence", "london", func(t *testing.T, a *allure.Context) {
				var f *cacheFixture
				var got []*probe.Summary
				var err error
				var from, to time.Time
				a.Step("Arrange", func(*allure.Context) {
					f = newCacheFixture(t)
					from = f.summary.ProbeTime
					to = from.Add(time.Hour)
					expect := f.fallback.EXPECT().GetMonitorSummariesForPeriod(gomock.Any(), f.id, from, to)
					if bad {
						expect.Return(nil, storageFailure)
					} else {
						expect.Return([]*probe.Summary{f.summary}, nil)
					}
				})
				a.Step("Act", func(*allure.Context) {
					got, err = f.repo.GetMonitorSummariesForPeriod(context.Background(), f.id, from, to)
				})
				a.Step("Assert", func(*allure.Context) {
					if bad {
						require.ErrorIs(t, err, storageFailure)
					} else {
						require.NoError(t, err)
						require.Len(t, got, 1)
						require.Equal(t, f.id, got[0].MonitorID)
					}
					require.NoError(t, f.mock.ExpectationsWereMet())
				})
			})
		})
	}
}
func TestCache_Create(t *testing.T) {
	for _, name := range testutil.Shuffle(t, []string{"cached", "cache_push_error", "cache_absent", "redis_error", "database_error", "nil_summary"}) {
		t.Run(name, func(t *testing.T) {
			testutil.Case(t, "RedisProbeSummaryRepository", "Create", "decision-table", "london", func(t *testing.T, a *allure.Context) {
				var f *cacheFixture
				var summary *probe.Summary
				var err error
				a.Step("Arrange", func(*allure.Context) {
					f = newCacheFixture(t)
					summary = f.summary
					if name == "nil_summary" {
						summary = nil
						return
					}
					switch name {
					case "cached":
						f.mock.ExpectTTL(f.key).SetVal(time.Minute)
						f.expectPush()
					case "cache_push_error":
						f.mock.ExpectTTL(f.key).SetVal(time.Minute)
						f.expectPushError()
					case "redis_error":
						f.mock.ExpectTTL(f.key).SetErr(storageFailure)
					default:
						f.mock.ExpectTTL(f.key).SetVal(-2 * time.Nanosecond)
					}
					expect := f.fallback.EXPECT().Create(gomock.Any(), summary)
					if name == "database_error" {
						expect.Return(storageFailure)
					} else {
						expect.Return(nil)
					}
				})
				a.Step("Act", func(*allure.Context) { err = f.repo.Create(context.Background(), summary) })
				a.Step("Assert", func(*allure.Context) {
					if name == "database_error" {
						require.ErrorIs(t, err, storageFailure)
					} else {
						require.NoError(t, err)
					}
					require.NoError(t, f.mock.ExpectationsWereMet())
				})
			})
		})
	}
}
func TestCache_BulkCreate(t *testing.T) {
	for _, name := range testutil.Shuffle(t, []string{"cached", "cache_push_error", "cache_absent", "redis_error", "database_error", "nil_entry"}) {
		t.Run(name, func(t *testing.T) {
			testutil.Case(t, "RedisProbeSummaryRepository", "BulkCreate", "decision-table", "london", func(t *testing.T, a *allure.Context) {
				var f *cacheFixture
				var summaries []*probe.Summary
				var err error
				a.Step("Arrange", func(*allure.Context) {
					f = newCacheFixture(t)
					summaries = []*probe.Summary{f.summary}
					switch name {
					case "cached":
						f.mock.ExpectTTL(f.key).SetVal(time.Minute)
						f.expectPush()
					case "cache_push_error":
						f.mock.ExpectTTL(f.key).SetVal(time.Minute)
						f.expectPushError()
					case "redis_error":
						f.mock.ExpectTTL(f.key).SetErr(storageFailure)
					case "nil_entry":
						summaries = []*probe.Summary{nil}
					default:
						f.mock.ExpectTTL(f.key).SetVal(-2 * time.Nanosecond)
					}
					expect := f.fallback.EXPECT().BulkCreate(gomock.Any(), summaries)
					if name == "database_error" {
						expect.Return(storageFailure)
					} else {
						expect.Return(nil)
					}
				})
				a.Step("Act", func(*allure.Context) { err = f.repo.BulkCreate(context.Background(), summaries) })
				a.Step("Assert", func(*allure.Context) {
					if name == "database_error" {
						require.ErrorIs(t, err, storageFailure)
					} else {
						require.NoError(t, err)
					}
					require.NoError(t, f.mock.ExpectationsWereMet())
				})
			})
		})
	}
}
