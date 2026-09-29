//go:build lab2integration

package integration

import (
	"errors"
	"testing"

	"WatchTower/internal/service"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestMonitorAndMetricsRejectOtherUser(t *testing.T) {
	testsupport.Case(t, "integration", "cross-user-ownership", func(t *testing.T, a *allure.Context) {
		f := setup(t, false)
		alice := f.user(t, "alice")
		bob := f.user(t, "bob")
		m := f.createMonitor(t, alice, "private", "http://probe:8081/health", 30)
		a.Step("Чужой список пуст, прямое чтение запрещено", func(*allure.Context) {
			list, err := f.monitoring.GetAllMonitors(bob)
			if err != nil || len(list) != 0 {
				t.Fatalf("bob list = %d, %v", len(list), err)
			}
			if _, err := f.monitoring.GetMonitor(bob, m.ID); !errors.Is(err, service.ErrPermissionDenied) {
				t.Fatalf("foreign monitor: %v", err)
			}
		})
		a.Step("Чужой пользователь не меняет монитор и не читает метрики", func(*allure.Context) {
			if err := f.monitoring.DisableMonitor(bob, m.ID); !errors.Is(err, service.ErrPermissionDenied) {
				t.Fatalf("foreign disable: %v", err)
			}
			if _, err := f.metrics.GetLastSummaries(bob, m.ID, 10); !errors.Is(err, service.ErrPermissionDenied) {
				t.Fatalf("foreign metrics: %v", err)
			}
			got, err := f.monitoring.GetMonitor(alice, m.ID)
			if err != nil || !got.IsActive {
				t.Fatalf("owner monitor changed = %#v, %v", got, err)
			}
		})
	})
}
