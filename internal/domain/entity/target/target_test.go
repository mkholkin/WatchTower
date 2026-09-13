package target_test

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service"
	"WatchTower/internal/testutil"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTarget_NewTarget(t *testing.T) {
	cases := []struct {
		name     string
		interval int32
		config   target.NetworkConfig
		bad      bool
	}{{"valid", 1, target.HTTPConfig{Method: "GET"}, false}, {"zero", 0, target.HTTPConfig{Method: "GET"}, true}, {"negative", -1, target.HTTPConfig{Method: "GET"}, true}, {"missing_config", 1, nil, true}, {"invalid_method", 1, target.HTTPConfig{Method: "invalid"}, true}}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "Target", "NewTarget", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var got *target.Target
				var err error
				a.Step("Arrange", func(*allure.Context) { /* Inputs are the immutable table entry. */ })
				a.Step("Act", func(*allure.Context) { got, err = target.NewTarget("https://example.test", tc.interval, tc.config) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						require.True(t, got.IsActive)
						require.Equal(t, tc.interval, got.ProbeIntervalSec)
						require.Len(t, got.ConfigHash, 64)
					}
				})
			})
		})
	}
}
func TestTarget_UpdateProbeInterval(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name string
		n    int32
		bad  bool
	}{{"minimum", 1, false}, {"zero", 0, true}, {"negative", -1, true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "Target", "UpdateProbeInterval", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var tgt *target.Target
				var err error
				a.Step("Arrange", func(*allure.Context) { tgt = (testutil.ObjectMother{}).Target() })
				a.Step("Act", func(*allure.Context) { err = tgt.UpdateProbeInterval(tc.n) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
						require.EqualValues(t, 30, tgt.ProbeIntervalSec)
					} else {
						require.NoError(t, err)
						require.Equal(t, tc.n, tgt.ProbeIntervalSec)
					}
				})
			})
		})
	}
}
func TestTarget_ComputeHash(t *testing.T) {
	for _, tc := range []struct {
		name, endpoint, method string
		equal                  bool
	}{{"same_target", "https://example.test", "GET", true}, {"different_endpoint", "https://other.test", "GET", false}, {"different_method", "https://example.test", "POST", false}} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "Target", "ComputeHash", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var baseline, got string
				a.Step("Arrange", func(*allure.Context) {
					baseline = "cc96193f43e85e53b94e3ea3decc4cf4fc7b310d1dc1aedff6e0a7c22019021c"
				})
				a.Step("Act", func(*allure.Context) {
					got = target.ComputeHash(tc.endpoint, target.HTTPConfig{Method: tc.method, Headers: map[string]string{"A": "1", "B": "2"}})
				})
				a.Step("Assert", func(*allure.Context) { require.Equal(t, tc.equal, baseline == got) })
			})
		})
	}
}
func TestNetworkConfig_HTTP(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name, method string
		bad          bool
	}{{"valid", "GET", false}, {"invalid", "get", true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "HTTPConfig", "NewHTTPConfig/Validate", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var cfg target.NetworkConfig
				var err error
				a.Step("Arrange", func(*allure.Context) {})
				a.Step("Act", func(*allure.Context) { cfg, err = target.NewHTTPConfig(tc.method, nil, "", false) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
						require.Equal(t, target.ProtocolHTTP, cfg.Protocol())
					}
				})
			})
		})
	}
}
func TestNetworkConfig_TCP(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name string
		port int
		bad  bool
	}{{"below", 0, true}, {"minimum", 1, false}, {"maximum", 65535, false}, {"above", 65536, true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "TCPConfig", "NewTCPConfig/Validate", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var cfg target.NetworkConfig
				var err error
				a.Step("Arrange", func(*allure.Context) {})
				a.Step("Act", func(*allure.Context) { cfg, err = target.NewTCPConfig(tc.port) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
						require.Equal(t, target.ProtocolTCP, cfg.Protocol())
					}
				})
			})
		})
	}
}

func TestHTTPConfig_Validate(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name, method string
		bad          bool
	}{{"valid", "POST", false}, {"unknown_method", "INVALID", true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "HTTPConfig", "Validate", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var cfg target.HTTPConfig
				var err error
				a.Step("Arrange", func(*allure.Context) { cfg = target.HTTPConfig{Method: tc.method} })
				a.Step("Act", func(*allure.Context) { err = cfg.Validate() })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
					}
				})
			})
		})
	}
}
