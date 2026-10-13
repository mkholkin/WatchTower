package probe_test

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/testutil"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProbe_NewProbeResult(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name          string
		latency, code int32
		bad           bool
	}{{"normal", 10, 200, false}, {"zero_latency", 0, 200, false}, {"negative_latency", -1, 200, true}, {"negative_status", 10, -1, true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "ProbeResult", "NewProbeResult", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var tgt *target.Target
				var got *probe.Result
				var err error
				a.Step("Arrange", func(*allure.Context) { tgt = (testutil.ObjectMother{}).Target() })
				a.Step("Act", func(*allure.Context) { got, err = probe.NewProbeResult(*tgt, tc.latency, tc.code, []byte("payload")) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.Error(t, err)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						require.False(t, got.NetworkFailure)
						require.True(t, got.StatusCode.Valid)
						require.Equal(t, tc.code, got.StatusCode.Int32)
						require.Equal(t, probe.ProcessingStatusNew, got.ProcessingStatus)
						require.Equal(t, []byte("payload"), got.Meta)
					}
				})
			})
		})
	}
}
func TestProbe_NewProbeResultWithNetworkFailure(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name    string
		latency int32
		bad     bool
	}{{"network_error", 10, false}, {"zero_latency", 0, false}, {"negative_latency", -1, true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "ProbeResult", "NewProbeResultWithNetworkFailure", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var tgt *target.Target
				var got *probe.Result
				var err error
				a.Step("Arrange", func(*allure.Context) { tgt = (testutil.ObjectMother{}).Target() })
				a.Step("Act", func(*allure.Context) { got, err = probe.NewProbeResultWithNetworkFailure(tgt, tc.latency, "timeout") })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.Error(t, err)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						require.True(t, got.NetworkFailure)
						require.False(t, got.StatusCode.Valid)
						require.Equal(t, "timeout", *got.ErrorMessage)
						require.Equal(t, probe.ProcessingStatusNew, got.ProcessingStatus)
					}
				})
			})
		})
	}
}
