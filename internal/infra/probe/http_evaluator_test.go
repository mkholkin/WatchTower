package infra_test

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	infra "WatchTower/internal/infra/probe"
	analyzation "WatchTower/internal/service/analyze"
	"WatchTower/internal/testutil"
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHTTPProbeEvaluator_Evaluate(t *testing.T) {
	cases := []struct {
		name             string
		latency          int32
		code             int32
		network, badType bool
		want             monitor.Status
		bad              bool
	}{
		{"healthy", 999, 200, false, false, monitor.StatusUp, false}, {"latency_boundary", 1000, 200, false, false, monitor.StatusUp, false}, {"too_slow", 1001, 200, false, false, monitor.StatusDown, false}, {"bad_status", 1, 503, false, false, monitor.StatusDown, false}, {"network_failure", 1, 200, true, false, monitor.StatusDown, false}, {"multiple_failures", 1001, 503, true, false, monitor.StatusDown, false}, {"wrong_expectations", 1, 200, false, true, monitor.StatusUnknown, true},
	}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "HTTPProbeEvaluator", "Evaluate", "decision-table/boundary", "classic", func(t *testing.T, a *allure.Context) {
				var m *monitor.Monitor
				var r *probe.Result
				var e analyzation.ProbeEvaluator
				var got monitor.Status
				var err error
				a.Step("Arrange", func(*allure.Context) {
					m = testutil.NewMonitorBuilder().WithMaxLatency(1000).Build()
					r = (testutil.ObjectMother{}).Result()
					r.LatencyMs = tc.latency
					r.StatusCode.Int32 = tc.code
					r.NetworkFailure = tc.network
					if tc.badType {
						m.Expectations = monitor.TCPExpectations{MaxLatencyMs: 1000}
					}
					e = infra.NewHTTPProbeEvaluator()
				})
				a.Step("Act", func(*allure.Context) { got, err = e.Evaluate(context.Background(), r, m) })
				a.Step("Assert", func(*allure.Context) {
					require.Equal(t, tc.want, got)
					if tc.bad {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				})
			})
		})
	}
}
