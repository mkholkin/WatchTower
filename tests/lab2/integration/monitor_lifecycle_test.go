//go:build lab2integration

package integration

import (
	"context"
	"errors"
	"testing"

	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/tests/lab2/testsupport"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
)

func TestMonitorsShareTargetAndReconcileActiveInterval(t *testing.T) {
	testsupport.Case(t, "integration", "monitor-target-lifecycle", func(t *testing.T, a *allure.Context) {
		f := setup(t, false)
		alice := f.user(t, "alice")
		bob := f.user(t, "bob")
		endpoint := "http://probe:8081/health"
		scenario := &monitorLifecycleScenario{t: t, f: f, alice: alice, bob: bob, endpoint: endpoint}
		a.Step("Два монитора разделяют цель и минимальный интервал", scenario.shareTarget)
		a.Step("Обновление и отключение пересчитывают состояние цели", scenario.updateAndDisable)
		a.Step("Повторное включение и удаление сохраняют связи", scenario.reenableAndDelete)
	})
}

type monitorLifecycleScenario struct {
	t                           *testing.T
	f                           *fixture
	alice, bob                  context.Context
	endpoint, firstID, secondID string
}

func (s *monitorLifecycleScenario) shareTarget(*allure.Context) {
	s.t.Helper()
	first := s.f.createMonitor(s.t, s.alice, "primary", s.endpoint, 30)
	second := s.f.createMonitor(s.t, s.alice, "secondary", s.endpoint, 10)
	s.firstID, s.secondID = first.ID.String(), second.ID.String()
	require.Equal(s.t, second.Target.ID, first.Target.ID, "%v", "same endpoint/config produced different targets")
	target, err := s.f.targets.GetByID(s.alice, first.Target.ID)
	require.NoError(s.t, err, "shared target = %#v, %v", target, err)
	require.True(s.t, target.IsActive, "shared target = %#v, %v", target, err)
	require.EqualValues(s.t, 10, target.ProbeIntervalSec, "shared target = %#v, %v", target, err)
}

func (s *monitorLifecycleScenario) updateAndDisable(*allure.Context) {
	s.t.Helper()
	first, err := s.f.monitoring.GetAllMonitors(s.alice)
	require.NoError(s.t, err, "%v", err)
	require.Len(s.t, first, 2, "expected two monitors, got %d", len(first))
	var firstMonitor, secondMonitor = first[0], first[1]
	if firstMonitor.ID.String() != s.firstID {
		firstMonitor, secondMonitor = secondMonitor, firstMonitor
	}
	label := "primary-renamed"
	if err := s.f.monitoring.UpdateMonitor(s.alice, dto.UpdateMonitorDTO{ID: firstMonitor.ID, Label: &label}); err != nil {
		s.t.Fatal(err)
	}
	updated, err := s.f.monitoring.GetMonitor(s.alice, firstMonitor.ID)
	require.NoError(s.t, err, "updated monitor = %#v, %v", updated, err)
	require.Equal(s.t, label, updated.Label, "updated monitor = %#v, %v", updated, err)
	err = s.f.monitoring.DisableMonitor(s.alice, secondMonitor.ID)
	require.NoError(s.t, err, "%v", err)
	target, err := s.f.targets.GetByID(s.alice, firstMonitor.Target.ID)
	require.NoError(s.t, err, "after disabling fast monitor = %#v, %v", target, err)
	require.True(s.t, target.IsActive, "after disabling fast monitor = %#v, %v", target, err)
	require.EqualValues(s.t, 30, target.ProbeIntervalSec, "after disabling fast monitor = %#v, %v", target, err)
	err = s.f.monitoring.DisableMonitor(s.alice, firstMonitor.ID)
	require.NoError(s.t, err, "%v", err)
	target, err = s.f.targets.GetByID(s.alice, firstMonitor.Target.ID)
	require.NoError(s.t, err, "after disabling all = %#v, %v", target, err)
	require.False(s.t, target.IsActive, "after disabling all = %#v, %v", target, err)
}

func (s *monitorLifecycleScenario) reenableAndDelete(*allure.Context) {
	s.t.Helper()
	mons, err := s.f.monitoring.GetAllMonitors(s.alice)
	require.NoError(s.t, err, "%v", err)
	for _, m := range mons {
		if m.ID.String() == s.secondID {
			err := s.f.monitoring.EnableMonitor(s.alice, m.ID)
			require.NoError(s.t, err, "%v", err)
		}
	}
	for _, m := range mons {
		err := s.f.monitoring.DeleteMonitor(s.alice, m.ID)
		require.NoError(s.t, err, "%v", err)
	}
	_, err = s.f.monitoring.GetMonitor(s.alice, mons[0].ID)
	require.True(s.t, errors.Is(err, repo.ErrNotFound), "deleted monitor lookup: %v", err)
	target, err := s.f.targets.GetByID(s.alice, mons[0].Target.ID)
	require.NoError(s.t, err, "unreferenced target = %#v, %v", target, err)
	require.False(s.t, target.IsActive, "unreferenced target = %#v, %v", target, err)
	other, err := s.f.monitoring.GetAllMonitors(s.bob)
	require.NoError(s.t, err, "bob monitors = %d, %v", len(other), err)
	require.Len(s.t, other, 0, "bob monitors = %d, %v", len(other), err)
}
