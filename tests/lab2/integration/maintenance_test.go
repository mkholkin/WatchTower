//go:build lab2integration

package integration

import (
	"errors"
	"testing"

	"WatchTower/internal/service"
	maintenance "WatchTower/internal/service/maintenance"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestMaintenanceWindowAssociationAndOwner(t *testing.T) {
	testsupport.Case(t, "integration", "maintenance-link-ownership", func(t *testing.T, a *allure.Context) {
		f := setup(t, false)
		alice := f.user(t, "alice")
		bob := f.user(t, "bob")
		m := f.createMonitor(t, alice, "API", "http://probe:8081/health", 30)
		window, err := f.maintenance.CreateManualMaintenanceWindow(alice, maintenance.CreateManualMaintenanceWindowDTO{Title: "Deploy", Description: "Lab 2"})
		if err != nil {
			t.Fatal(err)
		}
		a.Step("Окно прикрепляется и сохраняется", func(*allure.Context) {
			if err := f.maintenance.AddMonitorToMaintenanceWindow(alice, m.ID, window.ID); err != nil {
				t.Fatal(err)
			}
			got, err := f.monitoring.GetMonitor(alice, m.ID)
			if err != nil || len(got.MaintenanceWindows) != 1 || got.MaintenanceWindows[0].ID != window.ID {
				t.Fatalf("linked window = %#v, %v", got, err)
			}
		})
		a.Step("Чужое окно и изменение связи запрещены", func(*allure.Context) {
			if _, err := f.maintenance.GetMaintenanceWindow(bob, window.ID); !errors.Is(err, service.ErrPermissionDenied) {
				t.Fatalf("foreign window: %v", err)
			}
			if err := f.maintenance.RemoveMonitorFromMaintenanceWindow(bob, m.ID, window.ID); !errors.Is(err, service.ErrPermissionDenied) {
				t.Fatalf("foreign remove: %v", err)
			}
		})
		a.Step("Владелец удаляет связь, окно остается", func(*allure.Context) {
			if err := f.maintenance.RemoveMonitorFromMaintenanceWindow(alice, m.ID, window.ID); err != nil {
				t.Fatal(err)
			}
			got, err := f.monitoring.GetMonitor(alice, m.ID)
			if err != nil || len(got.MaintenanceWindows) != 0 {
				t.Fatalf("windows after remove = %#v, %v", got, err)
			}
			if _, err := f.maintenance.GetMaintenanceWindow(alice, window.ID); err != nil {
				t.Fatal(err)
			}
		})
	})
}
