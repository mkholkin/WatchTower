package healtcheck_service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/testmocks"
	"WatchTower/internal/testutil"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/golang/mock/gomock"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestProberRegistryRegisterAndGet(t *testing.T) {
	t.Run("returns registered prober", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "ProberRegistry.Register/Get", "equivalence", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			expected := testmocks.NewMockProber(ctrl)
			registry := NewProberRegistry()
			registry.Register(target.ProtocolHTTP, expected)
			actual, err := registry.Get(target.ProtocolHTTP)
			if err != nil || actual != expected {
				t.Fatalf("Get() = (%v, %v)", actual, err)
			}
		})
	})
	t.Run("later registration replaces earlier prober", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "ProberRegistry.Register/Get", "state-transition", "london", func(t *testing.T, _ *allure.Context) {
			ctrl := gomock.NewController(t)
			registry := NewProberRegistry()
			registry.Register(target.ProtocolHTTP, testmocks.NewMockProber(ctrl))
			expected := testmocks.NewMockProber(ctrl)
			registry.Register(target.ProtocolHTTP, expected)
			actual, err := registry.Get(target.ProtocolHTTP)
			if err != nil || actual != expected {
				t.Fatalf("Get() = (%v, %v)", actual, err)
			}
		})
	})
	t.Run("missing protocol returns error", func(t *testing.T) {
		testutil.Case(t, "healthcheck", "ProberRegistry.Get", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			actual, err := NewProberRegistry().Get(target.ProtocolTCP)
			if err == nil || actual != nil {
				t.Fatalf("Get() = (%v, %v)", actual, err)
			}
		})
	})
}
