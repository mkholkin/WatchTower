package maintenance_test

import (
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/service"
	"WatchTower/internal/testutil"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"testing"
	"testing/synctest"
	"time"
)

func ptr[T any](v T) *T { return &v }
func TestMaintenance_NewOneTimeMaintenanceWindow(t *testing.T) {
	cases := []struct {
		name                               string
		start, end                         time.Duration
		missingUser, emptyTitle, zero, bad bool
	}{
		{"valid", 0, time.Hour, false, false, false, false}, {"missing_user", 0, time.Hour, true, false, false, true}, {"empty_title", 0, time.Hour, false, true, false, true}, {"zero_time", 0, time.Hour, false, false, true, true}, {"equal_bounds", time.Hour, time.Hour, false, false, false, true}, {"reversed", time.Hour, 0, false, false, false, true}, {"past", -2 * time.Hour, -time.Hour, false, false, false, true},
	}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "MaintenanceWindow", "NewOneTimeMaintenanceWindow", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var usr *user.User
				var title string
				var start, end time.Time
				var got *maintenance.MaintenanceWindow
				var err error
				a.Step("Arrange", func(*allure.Context) {
					usr = (testutil.ObjectMother{}).User()
					title = "Work"
					now := time.Now()
					start = now.Add(tc.start)
					end = now.Add(tc.end)
					if tc.missingUser {
						usr = nil
					}
					if tc.emptyTitle {
						title = ""
					}
					if tc.zero {
						start = time.Time{}
					}
				})
				a.Step("Act", func(*allure.Context) { got, err = maintenance.NewOneTimeMaintenanceWindow(usr, title, "", start, end) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						require.Equal(t, maintenance.WindowTypeOneTime, got.Type)
						require.Equal(t, maintenance.OneTimeMaintenanceWindowConfig{StartTime: start, EndTime: end}, got.Config)
					}
				})
			})
		})
	}
}
func TestMaintenance_NewManualMaintenanceWindow(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name, title      string
		missingUser, bad bool
	}{{"valid", "Work", false, false}, {"empty_title", "", false, true}, {"missing_user", "Work", true, true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "MaintenanceWindow", "NewManualMaintenanceWindow", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var usr *user.User
				var got *maintenance.MaintenanceWindow
				var err error
				a.Step("Arrange", func(*allure.Context) {
					usr = (testutil.ObjectMother{}).User()
					if tc.missingUser {
						usr = nil
					}
				})
				a.Step("Act", func(*allure.Context) { got, err = maintenance.NewManualMaintenanceWindow(usr, tc.title, "description") })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
						require.Equal(t, maintenance.WindowTypeManual, got.Type)
						require.Equal(t, maintenance.ManualMaintenanceWindowConfig{Active: false}, got.Config)
						require.Equal(t, "description", got.Description)
					}
				})
			})
		})
	}
}
func TestMaintenance_ApplyUpdate(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name   string
		update maintenance.MaintenanceWindowUpdate
		bad    bool
	}{{"change_fields", maintenance.MaintenanceWindowUpdate{Title: ptr("New"), Description: ptr("Changed"), ConfigUpdate: maintenance.ManualConfigUpdate{Active: ptr(true)}}, false}, {"preserve", maintenance.MaintenanceWindowUpdate{}, false}, {"empty_title", maintenance.MaintenanceWindowUpdate{Title: ptr("")}, true}, {"wrong_update_type", maintenance.MaintenanceWindowUpdate{ConfigUpdate: maintenance.OneTimeConfigUpdate{}}, true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "MaintenanceWindow", "ApplyUpdate", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
				var w maintenance.MaintenanceWindow
				var err error
				a.Step("Arrange", func(*allure.Context) { w = (testutil.ObjectMother{}).Window(false) })
				a.Step("Act", func(*allure.Context) { err = w.ApplyUpdate(tc.update) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
						if tc.name == "change_fields" {
							require.Equal(t, "New", w.Title)
							require.Equal(t, "Changed", w.Description)
							require.Equal(t, maintenance.ManualMaintenanceWindowConfig{Active: true}, w.Config)
						} else {
							require.Equal(t, "Maintenance", w.Title)
							require.Equal(t, maintenance.ManualMaintenanceWindowConfig{Active: false}, w.Config)
						}
					}
				})
			})
		})
	}
}
func TestMaintenance_IsActive(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "inactive", true: "active"}[active], func(t *testing.T) {
			testutil.Case(t, "MaintenanceWindow", "IsActive", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
				var w maintenance.MaintenanceWindow
				var got bool
				a.Step("Arrange", func(*allure.Context) { w = (testutil.ObjectMother{}).Window(active) })
				a.Step("Act", func(*allure.Context) { got = w.IsActive() })
				a.Step("Assert", func(*allure.Context) { require.Equal(t, active, got) })
			})
		})
	}
}
func TestOneTimeConfig_IsActive(t *testing.T) {
	cases := []struct {
		name       string
		start, end time.Duration
		want       bool
	}{{"before", time.Second, 2 * time.Second, false}, {"start_boundary", 0, time.Second, false}, {"inside", -time.Second, time.Second, true}, {"end_boundary", -time.Second, 0, false}, {"after", -2 * time.Second, -time.Second, false}}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "OneTimeMaintenanceWindowConfig", "IsActive", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				synctest.Test(t, func(t *testing.T) {
					var c maintenance.OneTimeMaintenanceWindowConfig
					var got bool
					a.Step("Arrange", func(*allure.Context) {
						now := time.Now()
						c = maintenance.OneTimeMaintenanceWindowConfig{StartTime: now.Add(tc.start), EndTime: now.Add(tc.end)}
					})
					a.Step("Act", func(*allure.Context) { got = c.IsActive() })
					a.Step("Assert", func(*allure.Context) {
						require.Equal(t, tc.want, got)
						require.Equal(t, maintenance.WindowTypeOneTime, c.Type())
					})
				})
			})
		})
	}
}
func TestOneTimeConfigUpdate_Apply(t *testing.T) {
	for _, tc := range testutil.Shuffle(t, []struct {
		name string
		bad  bool
	}{{"replace", false}, {"preserve", false}, {"wrong_type", true}, {"zero_time", true}, {"equal_bounds", true}, {"past", true}}) {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "OneTimeConfigUpdate", "Apply", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var cfg maintenance.MaintenanceWindowConfig
				var update maintenance.OneTimeConfigUpdate
				var got maintenance.MaintenanceWindowConfig
				var err error
				var start, end time.Time
				a.Step("Arrange", func(*allure.Context) {
					now := time.Now()
					start = now.Add(time.Hour)
					end = now.Add(2 * time.Hour)
					cfg = maintenance.OneTimeMaintenanceWindowConfig{StartTime: start, EndTime: end}
					switch tc.name {
					case "replace":
						start = now.Add(3 * time.Hour)
						end = now.Add(4 * time.Hour)
						update = maintenance.OneTimeConfigUpdate{StartTime: &start, EndTime: &end}
					case "wrong_type":
						cfg = maintenance.ManualMaintenanceWindowConfig{}
					case "zero_time":
						update.StartTime = ptr(time.Time{})
					case "equal_bounds":
						update.EndTime = &start
					case "past":
						update.StartTime = ptr(now.Add(-2 * time.Hour))
						update.EndTime = ptr(now.Add(-time.Hour))
					}
				})
				a.Step("Act", func(*allure.Context) { got, err = update.Apply(cfg) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.ErrorIs(t, err, service.ErrInvalidData)
						require.Nil(t, got)
					} else {
						require.NoError(t, err)
						require.Equal(t, maintenance.OneTimeMaintenanceWindowConfig{StartTime: start, EndTime: end}, got)
					}
				})
			})
		})
	}
}
func TestManualConfigUpdate_Apply(t *testing.T) {
	for _, tc := range []struct {
		name   string
		update maintenance.ManualConfigUpdate
		wrong  bool
		want   bool
	}{{"activate", maintenance.ManualConfigUpdate{Active: ptr(true)}, false, true}, {"preserve", maintenance.ManualConfigUpdate{}, false, false}, {"wrong_type", maintenance.ManualConfigUpdate{}, true, false}} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "ManualConfigUpdate", "Apply", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
				var cfg maintenance.MaintenanceWindowConfig
				var got maintenance.MaintenanceWindowConfig
				var err error
				a.Step("Arrange", func(*allure.Context) {
					cfg = maintenance.ManualMaintenanceWindowConfig{}
					if tc.wrong {
						cfg = maintenance.OneTimeMaintenanceWindowConfig{}
					}
				})
				a.Step("Act", func(*allure.Context) { got, err = tc.update.Apply(cfg) })
				a.Step("Assert", func(*allure.Context) {
					if tc.wrong {
						require.ErrorIs(t, err, service.ErrInvalidData)
					} else {
						require.NoError(t, err)
						require.Equal(t, maintenance.ManualMaintenanceWindowConfig{Active: tc.want}, got)
						require.Equal(t, maintenance.WindowTypeManual, got.Type())
					}
				})
			})
		})
	}
}

func TestMaintenance_ApplyUpdate_IsAtomic(t *testing.T) {
	testutil.Case(t, "MaintenanceWindow", "ApplyUpdate", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
		var w maintenance.MaintenanceWindow
		var before maintenance.MaintenanceWindow
		var err error
		a.Step("Arrange", func(*allure.Context) { w = (testutil.ObjectMother{}).Window(false); before = w })
		a.Step("Act", func(*allure.Context) {
			err = w.ApplyUpdate(maintenance.MaintenanceWindowUpdate{Title: ptr("Changed"), Description: ptr("Changed"), ConfigUpdate: maintenance.OneTimeConfigUpdate{}})
		})
		a.Step("Assert", func(*allure.Context) { require.ErrorIs(t, err, service.ErrInvalidData); require.Equal(t, before, w) })
	})
}
