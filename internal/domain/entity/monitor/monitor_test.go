package monitor_test

import (
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/service"
	"WatchTower/internal/testutil"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMonitor_NewMonitor(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*string, **target.Target, **user.User, *monitor.Expectations)
		bad    bool
	}{
		{"valid", func(*string, **target.Target, **user.User, *monitor.Expectations) {}, false},
		{"empty_name", func(n *string, _ **target.Target, _ **user.User, _ *monitor.Expectations) { *n = "" }, true},
		{"missing_target", func(_ *string, t **target.Target, _ **user.User, _ *monitor.Expectations) { *t = nil }, true},
		{"missing_user", func(_ *string, _ **target.Target, u **user.User, _ *monitor.Expectations) { *u = nil }, true},
		{"protocol_mismatch", func(_ *string, _ **target.Target, _ **user.User, e *monitor.Expectations) {
			*e = monitor.TCPExpectations{MaxLatencyMs: 10}
		}, true},
		{"invalid_http_expectations", func(_ *string, _ **target.Target, _ **user.User, e *monitor.Expectations) {
			*e = monitor.HTTPExpectations{MaxLatencyMs: -1}
		}, true},
		{"typed_nil_expectations", func(_ *string, _ **target.Target, _ **user.User, e *monitor.Expectations) {
			var ptr *monitor.HTTPExpectations
			*e = ptr
		}, true},
		{"missing_expectations", func(_ *string, _ **target.Target, _ **user.User, e *monitor.Expectations) { *e = nil }, true},
	}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "Monitor", "NewMonitor", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var name string
				var tgt *target.Target
				var usr *user.User
				var exp monitor.Expectations
				a.Step("Arrange", func(*allure.Context) {
					name = "API"
					tgt = (testutil.ObjectMother{}).Target()
					usr = (testutil.ObjectMother{}).User()
					exp = monitor.HTTPExpectations{StatusCodes: []int{200}, MaxLatencyMs: 100}
					tc.mutate(&name, &tgt, &usr, &exp)
				})
				var got *monitor.Monitor
				var err error
				a.Step("Act", func(*allure.Context) { got, err = monitor.NewMonitor(name, tgt, usr, nil, nil, 30, exp) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						require.Equal(t, "API", got.Label)
						require.True(t, got.IsActive)
						require.Equal(t, monitor.StatusUnknown, got.CurrentStatus)
					}
				})
			})
		})
	}
}
func TestMonitor_Enable(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "paused_to_active", true: "already_active"}[active], func(t *testing.T) {
			testutil.Case(t, "Monitor", "Enable", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
				var m *monitor.Monitor
				a.Step("Arrange", func(*allure.Context) { m = testutil.NewMonitorBuilder().WithActive(active).Build() })
				a.Step("Act", func(*allure.Context) { m.Enable() })
				a.Step("Assert", func(*allure.Context) { require.True(t, m.IsActive) })
			})
		})
	}
}
func TestMonitor_Disable(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "already_paused", true: "active_to_paused"}[active], func(t *testing.T) {
			testutil.Case(t, "Monitor", "Disable", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
				var m *monitor.Monitor
				a.Step("Arrange", func(*allure.Context) {
					m = testutil.NewMonitorBuilder().WithActive(active).WithStatus(monitor.StatusUp).Build()
				})
				a.Step("Act", func(*allure.Context) { m.Disable() })
				a.Step("Assert", func(*allure.Context) {
					require.False(t, m.IsActive)
					require.Equal(t, monitor.StatusUnknown, m.CurrentStatus)
				})
			})
		})
	}
}
func TestMonitor_OnMaintenance(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "inactive", true: "active"}[active], func(t *testing.T) {
			testutil.Case(t, "Monitor", "OnMaintenance", "decision-table", "classic", func(t *testing.T, a *allure.Context) {
				var m *monitor.Monitor
				var got bool
				a.Step("Arrange", func(*allure.Context) {
					m = testutil.NewMonitorBuilder().Build()
					m.MaintenanceWindows = []maintenance.MaintenanceWindow{(testutil.ObjectMother{}).Window(false), (testutil.ObjectMother{}).Window(active)}
				})
				a.Step("Act", func(*allure.Context) { got = m.OnMaintenance() })
				a.Step("Assert", func(*allure.Context) { require.Equal(t, active, got) })
			})
		})
	}
}
func TestHTTPExpectations_Validate(t *testing.T) {
	cases := []struct {
		name    string
		codes   []int
		latency int
		bad     bool
	}{
		{"valid", []int{200, 204}, 100, false}, {"zero_latency", []int{200}, 0, false}, {"negative_latency", []int{200}, -1, true}, {"empty_codes", nil, 100, true}, {"below_status_range", []int{99}, 100, true}, {"lower_status_boundary", []int{100}, 100, false}, {"upper_status_boundary", []int{599}, 100, false}, {"above_status_range", []int{600}, 100, true},
	}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "HTTPExpectations", "Validate", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var e monitor.HTTPExpectations
				var err error
				a.Step("Arrange", func(*allure.Context) { e = monitor.HTTPExpectations{StatusCodes: tc.codes, MaxLatencyMs: tc.latency} })
				a.Step("Act", func(*allure.Context) { err = e.Validate() })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
						require.Equal(t, target.ProtocolHTTP, e.Protocol())
					}
				})
			})
		})
	}
}
