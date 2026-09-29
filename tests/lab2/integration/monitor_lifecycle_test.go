//go:build lab2integration

package integration

import (
	"errors"
	"testing"

	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestMonitorsShareTargetAndReconcileActiveInterval(t *testing.T) {
	testsupport.Case(t, "integration", "monitor-target-lifecycle", func(t *testing.T, a *allure.Context) {
		f := setup(t, false)
		alice := f.user(t, "alice")
		bob := f.user(t, "bob")
		endpoint := "http://probe:8081/health"
		var firstID, secondID string
		a.Step("Два монитора разделяют цель и минимальный интервал", func(*allure.Context) {
			first := f.createMonitor(t, alice, "primary", endpoint, 30)
			second := f.createMonitor(t, alice, "secondary", endpoint, 10)
			firstID, secondID = first.ID.String(), second.ID.String()
			if first.Target.ID != second.Target.ID {
				t.Fatal("same endpoint/config produced different targets")
			}
			target, err := f.targets.GetByID(alice, first.Target.ID)
			if err != nil || !target.IsActive || target.ProbeIntervalSec != 10 {
				t.Fatalf("shared target = %#v, %v", target, err)
			}
		})
		a.Step("Обновление и отключение пересчитывают состояние цели", func(*allure.Context) {
			first, err := f.monitoring.GetAllMonitors(alice)
			if err != nil {
				t.Fatal(err)
			}
			if len(first) != 2 {
				t.Fatalf("expected two monitors, got %d", len(first))
			}
			var firstMonitor, secondMonitor = first[0], first[1]
			if firstMonitor.ID.String() != firstID {
				firstMonitor, secondMonitor = secondMonitor, firstMonitor
			}
			label := "primary-renamed"
			if err := f.monitoring.UpdateMonitor(alice, dto.UpdateMonitorDTO{ID: firstMonitor.ID, Label: &label}); err != nil {
				t.Fatal(err)
			}
			updated, err := f.monitoring.GetMonitor(alice, firstMonitor.ID)
			if err != nil || updated.Label != label {
				t.Fatalf("updated monitor = %#v, %v", updated, err)
			}
			if err := f.monitoring.DisableMonitor(alice, secondMonitor.ID); err != nil {
				t.Fatal(err)
			}
			target, err := f.targets.GetByID(alice, firstMonitor.Target.ID)
			if err != nil || !target.IsActive || target.ProbeIntervalSec != 30 {
				t.Fatalf("after disabling fast monitor = %#v, %v", target, err)
			}
			if err := f.monitoring.DisableMonitor(alice, firstMonitor.ID); err != nil {
				t.Fatal(err)
			}
			target, err = f.targets.GetByID(alice, firstMonitor.Target.ID)
			if err != nil || target.IsActive {
				t.Fatalf("after disabling all = %#v, %v", target, err)
			}
		})
		a.Step("Повторное включение и удаление сохраняют связи", func(*allure.Context) {
			mons, err := f.monitoring.GetAllMonitors(alice)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range mons {
				if m.ID.String() == secondID {
					if err := f.monitoring.EnableMonitor(alice, m.ID); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, m := range mons {
				if err := f.monitoring.DeleteMonitor(alice, m.ID); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.monitoring.GetMonitor(alice, mons[0].ID); !errors.Is(err, repo.ErrNotFound) {
				t.Fatalf("deleted monitor lookup: %v", err)
			}
			target, err := f.targets.GetByID(alice, mons[0].Target.ID)
			if err != nil || target.IsActive {
				t.Fatalf("unreferenced target = %#v, %v", target, err)
			}
			other, err := f.monitoring.GetAllMonitors(bob)
			if err != nil || len(other) != 0 {
				t.Fatalf("bob monitors = %d, %v", len(other), err)
			}
		})
	})
}
