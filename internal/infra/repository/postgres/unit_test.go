package postgres

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
	"WatchTower/internal/service/metrics"
	"WatchTower/internal/testutil"
	"context"
	"errors"
	"fmt"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type pgFixture struct {
	db        pgxmock.PgxPoolIface
	users     *userRepositoryPG
	targets   *targetRepositoryPG
	monitors  *monitorRepositoryPG
	contacts  *alertContactRepositoryPG
	windows   *maintenanceWindowRepositoryPG
	results   *probeResultRepositoryPG
	summaries *probeSummaryRepositoryPG
	metrics   *MetricsRepository
	usr       *user.User
	tgt       *target.Target
	mon       *monitor.Monitor
	contact   *alert.Contact
	window    maintenance.MaintenanceWindow
	result    *probe.Result
}

func newPGFixture(t *testing.T) *pgFixture {
	t.Helper()
	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.ExpectationsWereMet()); db.Close() })
	log := testutil.NoopLogger()
	q := sqlcgen.New(db)
	f := &pgFixture{db: db, users: NewUserRepository(nil, log).(*userRepositoryPG), targets: NewTargetRepository(nil, log).(*targetRepositoryPG), monitors: NewMonitorRepository(nil, log).(*monitorRepositoryPG), contacts: NewAlertContactRepository(nil, log).(*alertContactRepositoryPG), windows: NewMaintenanceWindowRepository(nil, log).(*maintenanceWindowRepositoryPG), results: NewProbeResultRepository(nil, log).(*probeResultRepositoryPG), summaries: NewProbeSummaryRepository(nil, log).(*probeSummaryRepositoryPG), metrics: NewMetricsRepository(nil, log)}
	f.users.queries = q
	f.targets.queries = q
	f.monitors.queries = q
	f.contacts.queries = q
	f.windows.queries = q
	f.results.queries = q
	f.summaries.queries = q
	f.metrics.queries = q
	mother := testutil.ObjectMother{}
	f.usr = mother.User()
	f.tgt = mother.Target()
	f.mon = testutil.NewMonitorBuilder().Build()
	f.contact = mother.Contact()
	f.window = mother.Window(false)
	f.result = mother.Result()
	return f
}
func pgID(id uuid.UUID) pgtype.UUID         { return pgtype.UUID{Bytes: id, Valid: true} }
func pgTime(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

var databaseFailure = errors.New("database unavailable")

type writeCase struct {
	component, method, query string
	args                     func(*pgFixture) []any
	call                     func(*pgFixture) error
}

func TestPostgres_WriteOperations(t *testing.T) {
	ctx := context.Background()
	cases := []writeCase{
		{"UserRepository", "Create", "CreateUser", func(f *pgFixture) []any { return []any{f.usr.Login, f.usr.PasswordHash} }, func(f *pgFixture) error { return f.users.Create(ctx, f.usr) }},
		{"TargetRepository", "Create", "CreateTarget", func(f *pgFixture) []any {
			return []any{pgID(f.tgt.ID), "", sqlcgen.ProtocolTypeHTTP, f.tgt.Endpoint, []byte(`{"method":"GET","follow_redirects":false}`), true, int32(30)}
		}, func(f *pgFixture) error { return f.targets.Create(ctx, f.tgt) }},
		{"TargetRepository", "Update", "UpdateTarget", func(f *pgFixture) []any {
			return []any{pgID(f.tgt.ID), "", sqlcgen.ProtocolTypeHTTP, f.tgt.Endpoint, []byte(`{"method":"GET","follow_redirects":false}`), true, int32(30)}
		}, func(f *pgFixture) error { return f.targets.Update(ctx, f.tgt) }},
		{"TargetRepository", "UpdateProbeInterval", "UpdateTargetProbeInterval", func(f *pgFixture) []any { return []any{pgID(f.tgt.ID), int32(5)} }, func(f *pgFixture) error { return f.targets.UpdateProbeInterval(ctx, f.tgt.ID, 5) }},
		{"MonitorRepository", "Create", "CreateMonitor", func(f *pgFixture) []any {
			return []any{pgID(f.mon.ID), pgID(f.mon.Target.ID), "alice", "API", true, int32(30), []byte(`{"status_code":[200],"max_latency_ms":1000}`), sqlcgen.StatusTypeUNKNOWN, pgtype.Timestamptz{}, pgTime(f.mon.CreatedAt)}
		}, func(f *pgFixture) error { return f.monitors.Create(ctx, f.mon) }},
		{"MonitorRepository", "Update", "UpdateMonitor", func(f *pgFixture) []any {
			return []any{pgID(f.mon.ID), pgID(f.mon.Target.ID), "alice", "API", true, int32(30), []byte(`{"status_code":[200],"max_latency_ms":1000}`), sqlcgen.StatusTypeUNKNOWN, pgTime(f.mon.LastEvaluatedAt)}
		}, func(f *pgFixture) error { return f.monitors.Update(ctx, f.mon) }},
		{"MonitorRepository", "BulkUpdateEvaluation", "BulkUpdateEvaluation", func(f *pgFixture) []any {
			return []any{[]pgtype.UUID{pgID(f.mon.ID)}, []string{"UNKNOWN"}, []pgtype.Timestamp{{Time: f.mon.LastEvaluatedAt, Valid: true}}}
		}, func(f *pgFixture) error { return f.monitors.BulkUpdateEvaluation(ctx, []*monitor.Monitor{f.mon}) }},
		{"MonitorRepository", "AddAlertContact", "AddAlertContactToMonitor", func(f *pgFixture) []any { return []any{pgID(f.mon.ID), pgID(f.contact.ID)} }, func(f *pgFixture) error { return f.monitors.AddAlertContact(ctx, f.mon, f.contact) }},
		{"MonitorRepository", "RemoveAlertContact", "RemoveAlertContactFromMonitor", func(f *pgFixture) []any { return []any{pgID(f.mon.ID), pgID(f.contact.ID)} }, func(f *pgFixture) error { return f.monitors.RemoveAlertContact(ctx, f.mon, f.contact) }},
		{"AlertContactRepository", "Create", "CreateAlertContact", func(f *pgFixture) []any {
			return []any{pgID(f.contact.ID), "alice", sqlcgen.ContactTypeTELEGRAM, "Operations", []byte(`{"chat_id":42,"bot_token":"test-token"}`), true}
		}, func(f *pgFixture) error { return f.contacts.Create(ctx, f.contact) }},
		{"AlertContactRepository", "Update", "UpdateAlertContact", func(f *pgFixture) []any {
			return []any{pgID(f.contact.ID), "alice", sqlcgen.ContactTypeTELEGRAM, "Operations", []byte(`{"chat_id":42,"bot_token":"test-token"}`)}
		}, func(f *pgFixture) error { return f.contacts.Update(ctx, f.contact) }},
		{"MaintenanceWindowRepository", "Create", "CreateMaintenanceWindow", func(f *pgFixture) []any {
			return []any{pgID(f.window.ID), "alice", "Maintenance", pgtype.Text{}, sqlcgen.MaintenanceTypeMANUAL, []byte(`{"active":false}`)}
		}, func(f *pgFixture) error { return f.windows.Create(ctx, &f.window) }},
		{"MaintenanceWindowRepository", "Update", "UpdateMaintenanceWindow", func(f *pgFixture) []any {
			return []any{pgID(f.window.ID), "alice", "Maintenance", pgtype.Text{}, sqlcgen.MaintenanceTypeMANUAL, []byte(`{"active":false}`)}
		}, func(f *pgFixture) error { return f.windows.Update(ctx, &f.window) }},
		{"MaintenanceWindowRepository", "LinkMonitor", "LinkMonitor", func(f *pgFixture) []any { return []any{pgID(f.mon.ID), pgID(f.window.ID)} }, func(f *pgFixture) error { return f.windows.LinkMonitor(ctx, &f.window, f.mon.ID) }},
		{"MaintenanceWindowRepository", "UnlinkMonitor", "UnlinkMonitor", func(f *pgFixture) []any { return []any{pgID(f.mon.ID), pgID(f.window.ID)} }, func(f *pgFixture) error { return f.windows.UnlinkMonitor(ctx, &f.window, f.mon.ID) }},
		{"ProbeResultRepository", "Create", "CreateProbeResult", func(f *pgFixture) []any {
			return []any{pgID(f.result.ID), pgID(f.result.Target.ID), pgTime(f.result.ProbeTime), int32(100), pgtype.Int4{Int32: 200, Valid: true}, false, pgtype.Text{}, []byte(nil)}
		}, func(f *pgFixture) error { return f.results.Create(f.result) }},
		{"ProbeResultRepository", "BulkUpdateStatus", "BulkUpdateProbeResultStatus", func(f *pgFixture) []any {
			return []any{sqlcgen.ProcessingStatusTypePROCESSED, []pgtype.UUID{pgID(f.result.ID)}}
		}, func(f *pgFixture) error {
			return f.results.BulkUpdateStatus(ctx, []uuid.UUID{f.result.ID}, probe.ProcessingStatusProcessed)
		}},
		{"TargetRepository", "DeleteByID", "DeleteTargetByID", func(f *pgFixture) []any { return []any{pgID(f.tgt.ID)} }, func(f *pgFixture) error { return f.targets.DeleteByID(ctx, f.tgt.ID) }},
		{"TargetRepository", "Enable", "EnableTarget", func(f *pgFixture) []any { return []any{pgID(f.tgt.ID)} }, func(f *pgFixture) error { return f.targets.Enable(ctx, f.tgt.ID) }},
		{"TargetRepository", "Disable", "DisableTarget", func(f *pgFixture) []any { return []any{pgID(f.tgt.ID)} }, func(f *pgFixture) error { return f.targets.Disable(ctx, f.tgt.ID) }},
		{"MonitorRepository", "DeleteByID", "DeleteMonitorByID", func(f *pgFixture) []any { return []any{pgID(f.mon.ID)} }, func(f *pgFixture) error { return f.monitors.DeleteByID(ctx, f.mon.ID) }},
		{"MonitorRepository", "Enable", "EnableMonitor", func(f *pgFixture) []any { return []any{pgID(f.mon.ID)} }, func(f *pgFixture) error { return f.monitors.Enable(ctx, f.mon.ID) }},
		{"MonitorRepository", "Disable", "DisableMonitor", func(f *pgFixture) []any { return []any{pgID(f.mon.ID)} }, func(f *pgFixture) error { return f.monitors.Disable(ctx, f.mon.ID) }},
		{"AlertContactRepository", "DeleteByID", "DeleteAlertContactByID", func(f *pgFixture) []any { return []any{pgID(f.contact.ID)} }, func(f *pgFixture) error { return f.contacts.DeleteByID(ctx, f.contact.ID) }},
		{"AlertContactRepository", "Enable", "EnableAlertContact", func(f *pgFixture) []any { return []any{pgID(f.contact.ID)} }, func(f *pgFixture) error { return f.contacts.Enable(ctx, f.contact.ID) }},
		{"AlertContactRepository", "Disable", "DisableAlertContact", func(f *pgFixture) []any { return []any{pgID(f.contact.ID)} }, func(f *pgFixture) error { return f.contacts.Disable(ctx, f.contact.ID) }},
		{"MaintenanceWindowRepository", "DeleteByID", "DeleteMaintenanceWindowByID", func(f *pgFixture) []any { return []any{pgID(f.window.ID)} }, func(f *pgFixture) error { return f.windows.DeleteByID(ctx, f.window.ID) }},
	}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.component+"/"+tc.method, func(t *testing.T) {
			for _, bad := range []bool{false, true} {
				t.Run(map[bool]string{false: "success", true: "database_error"}[bad], func(t *testing.T) {
					testutil.Case(t, tc.component, tc.method, "equivalence", "london", func(t *testing.T, a *allure.Context) {
						var f *pgFixture
						var err error
						a.Step("Arrange", func(*allure.Context) {
							f = newPGFixture(t)
							expected := f.db.ExpectExec("-- name: " + tc.query + " ").WithArgs(tc.args(f)...)
							if bad {
								expected.WillReturnError(databaseFailure)
							} else {
								expected.WillReturnResult(pgxmock.NewResult("UPDATE", 1))
							}
						})
						a.Step("Act", func(*allure.Context) { err = tc.call(f) })
						a.Step("Assert", func(*allure.Context) {
							if bad {
								require.ErrorIs(t, err, databaseFailure)
								require.ErrorIs(t, err, repo.ErrDB)
							} else {
								require.NoError(t, err)
							}
							require.NoError(t, f.db.ExpectationsWereMet())
						})
					})
				})
			}
		})
	}
}

func sampleRow(kind string, f *pgFixture) []any {
	switch kind {
	case "user":
		return []any{"alice", "fixture-hash"}
	case "target":
		return []any{pgID(f.tgt.ID), "signature", sqlcgen.ProtocolTypeHTTP, true, "https://example.test/health", []byte(`{"method":"GET","follow_redirects":true}`), int32(30)}
	case "monitor":
		return []any{pgID(f.mon.ID), pgID(f.tgt.ID), "alice", "API", true, int32(30), []byte(`{"status_code":[200],"max_latency_ms":1000}`), sqlcgen.StatusTypeUP, pgTime(f.result.ProbeTime), pgTime(f.mon.CreatedAt), pgID(f.tgt.ID), "signature", sqlcgen.ProtocolTypeHTTP, true, "https://example.test/health", []byte(`{"method":"GET","follow_redirects":true}`), int32(30), "alice", "fixture-hash", []byte(`[]`), []byte(`[]`)}
	case "contact":
		return []any{pgID(f.contact.ID), "alice", sqlcgen.ContactTypeTELEGRAM, "Operations", []byte(`{"chat_id":42,"bot_token":"test-token"}`), true, "alice", "fixture-hash"}
	case "window":
		return []any{pgID(f.window.ID), "alice", "Maintenance", pgtype.Text{String: "planned", Valid: true}, sqlcgen.MaintenanceTypeMANUAL, []byte(`{"active":true}`), "alice", "fixture-hash"}
	case "result":
		return []any{pgID(f.result.ID), pgID(f.tgt.ID), pgTime(f.result.ProbeTime), int32(100), pgtype.Int4{}, true, pgtype.Text{String: "upstream timeout", Valid: true}, []byte("payload"), sqlcgen.ProcessingStatusTypeNEW}
	case "summary":
		return []any{pgID(f.mon.ID), int32(100), pgTime(f.result.ProbeTime), sqlcgen.StatusTypeUP, int32(0), true, "connection reset"}
	case "events":
		return []any{int64(1), pgID(f.mon.ID), sqlcgen.StatusTypeDOWN, pgTime(f.result.ProbeTime), pgTime(f.result.ProbeTime.Add(time.Minute))}
	case "sla":
		return []any{pgID(f.mon.ID), float64(99.5), int64(30), pgTime(f.result.ProbeTime), pgTime(f.result.ProbeTime.Add(time.Hour))}
	default:
		panic("unknown test fixture")
	}
}
func mockRows(values []any) *pgxmock.Rows {
	cols := make([]string, len(values))
	for i := range cols {
		cols[i] = fmt.Sprintf("column_%d", i)
	}
	return pgxmock.NewRows(cols).AddRow(values...)
}
func assertPGRead(t *testing.T, f *pgFixture, got any) {
	t.Helper()
	switch v := got.(type) {
	case *user.User:
		require.Equal(t, "alice", v.Login)
		require.Equal(t, "fixture-hash", v.PasswordHash)
	case *target.Target:
		require.Equal(t, f.tgt.ID, v.ID)
		require.Equal(t, "https://example.test/health", v.Endpoint)
		require.Equal(t, target.HTTPConfig{Method: "GET", FollowRedirects: true}, v.Config)
		require.True(t, v.IsActive)
		require.Equal(t, int32(30), v.ProbeIntervalSec)
		require.Equal(t, "signature", v.ConfigHash)
	case []target.Target:
		require.Len(t, v, 1)
		assertPGRead(t, f, &v[0])
	case *monitor.Monitor:
		require.Equal(t, f.mon.ID, v.ID)
		require.Equal(t, "API", v.Label)
		require.Equal(t, f.tgt.ID, v.Target.ID)
		require.Equal(t, "https://example.test/health", v.Target.Endpoint)
		require.Equal(t, target.HTTPConfig{Method: "GET", FollowRedirects: true}, v.Target.Config)
		require.True(t, v.Target.IsActive)
		require.Equal(t, int32(30), v.Target.ProbeIntervalSec)
		require.Equal(t, "signature", v.Target.ConfigHash)
		require.Equal(t, "alice", v.User.Login)
		require.Equal(t, "fixture-hash", v.User.PasswordHash)
		require.Empty(t, v.AlertContacts)
		require.Empty(t, v.MaintenanceWindows)
		require.Equal(t, monitor.StatusUp, v.CurrentStatus)
		require.Equal(t, f.result.ProbeTime, v.LastEvaluatedAt)
		require.Equal(t, int32(30), v.ProbeIntervalSec)
		require.True(t, v.IsActive)
		require.Equal(t, f.mon.CreatedAt, v.CreatedAt)
		require.Equal(t, monitor.HTTPExpectations{StatusCodes: []int{200}, MaxLatencyMs: 1000}, v.Expectations)
	case []*monitor.Monitor:
		require.Len(t, v, 1)
		assertPGRead(t, f, v[0])
	case map[uuid.UUID][]*monitor.Monitor:
		require.Len(t, v, 1)
		require.Contains(t, v, f.tgt.ID)
		assertPGRead(t, f, v[f.tgt.ID])
	case *alert.Contact:
		require.Equal(t, f.contact.ID, v.ID)
		require.Equal(t, "Operations", v.Name)
		require.Equal(t, "alice", v.User.Login)
		require.Equal(t, "fixture-hash", v.User.PasswordHash)
		require.Equal(t, alert.ContactTypeTelegram, v.Type)
		require.Equal(t, alert.TelegramContactConfig{ChatID: 42, BotToken: "test-token"}, v.Config)
		require.True(t, v.IsActive)
	case []alert.Contact:
		require.Len(t, v, 1)
		assertPGRead(t, f, &v[0])
	case *maintenance.MaintenanceWindow:
		require.Equal(t, f.window.ID, v.ID)
		require.Equal(t, "Maintenance", v.Title)
		require.Equal(t, "planned", v.Description)
		require.Equal(t, "alice", v.User.Login)
		require.Equal(t, "fixture-hash", v.User.PasswordHash)
		require.Equal(t, maintenance.WindowTypeManual, v.Type)
		require.Equal(t, maintenance.ManualMaintenanceWindowConfig{Active: true}, v.Config)
	case []maintenance.MaintenanceWindow:
		require.Len(t, v, 1)
		assertPGRead(t, f, &v[0])
	case []*probe.Result:
		require.Len(t, v, 1)
		require.Equal(t, f.result.ID, v[0].ID)
		require.Equal(t, f.tgt.ID, v[0].Target.ID)
		require.Equal(t, int32(100), v[0].LatencyMs)
		require.Equal(t, f.result.ProbeTime, v[0].ProbeTime)
		require.Equal(t, probe.ProcessingStatusNew, v[0].ProcessingStatus)
		require.Equal(t, int32(0), v[0].StatusCode.Int32)
		require.False(t, v[0].StatusCode.Valid)
		require.True(t, v[0].NetworkFailure)
		require.NotNil(t, v[0].ErrorMessage)
		require.Equal(t, "upstream timeout", *v[0].ErrorMessage)
		require.Equal(t, []byte("payload"), v[0].Meta)
	case []*probe.Summary:
		require.Len(t, v, 1)
		require.Equal(t, f.mon.ID, v[0].MonitorID)
		require.Equal(t, monitor.StatusUp, v[0].MonitorStatus)
		require.Equal(t, int32(100), v[0].LatencyMs)
		require.Equal(t, f.result.ProbeTime, v[0].ProbeTime)
		require.Equal(t, int32(0), v[0].StatusCode)
		require.True(t, v[0].NetworkFailure)
		require.Equal(t, "connection reset", v[0].FailureReason)
	case []metrics.StatusEvent:
		require.Len(t, v, 1)
		require.Equal(t, monitor.StatusDown, v[0].Status)
		require.Equal(t, f.result.ProbeTime, v[0].StartTime)
		require.Equal(t, f.result.ProbeTime.Add(time.Minute), v[0].EndTime)
		require.Empty(t, v[0].Reason)
	case metrics.SLAStats:
		require.Equal(t, f.mon.ID, v.MonitorID)
		require.Equal(t, 99.5, v.UptimePercent)
		require.Equal(t, 30, v.TotalDowntimeSec)
		require.Equal(t, f.result.ProbeTime, v.PeriodStart)
		require.Equal(t, f.result.ProbeTime.Add(time.Hour), v.PeriodEnd)
	default:
		t.Fatalf("unhandled read result %T", got)
	}
}

type readCase struct {
	component, method, query, kind string
	single                         bool
	args                           func(*pgFixture) []any
	call                           func(*pgFixture) (any, error)
}

func TestPostgres_ReadOperations(t *testing.T) {
	ctx := context.Background()
	cases := []readCase{
		{"UserRepository", "GetByLogin", "GetUserByLogin", "user", true, func(f *pgFixture) []any { return []any{"alice"} }, func(f *pgFixture) (any, error) { return f.users.GetByLogin(ctx, "alice") }},
		{"TargetRepository", "GetByID", "GetTargetByID", "target", true, func(f *pgFixture) []any { return []any{pgID(f.tgt.ID)} }, func(f *pgFixture) (any, error) { return f.targets.GetByID(ctx, f.tgt.ID) }},
		{"TargetRepository", "GetByHash", "GetTargetByHash", "target", true, func(f *pgFixture) []any { return []any{"signature"} }, func(f *pgFixture) (any, error) { return f.targets.GetByHash(ctx, "signature") }},
		{"TargetRepository", "GetAllActive", "GetAllActiveTargets", "target", false, func(f *pgFixture) []any { return nil }, func(f *pgFixture) (any, error) { return f.targets.GetAllActive(ctx) }},
		{"MonitorRepository", "GetByID", "GetMonitorByID", "monitor", true, func(f *pgFixture) []any { return []any{pgID(f.mon.ID)} }, func(f *pgFixture) (any, error) { return f.monitors.GetByID(ctx, f.mon.ID) }},
		{"MonitorRepository", "GetAllByUser", "GetAllMonitorsByUser", "monitor", false, func(f *pgFixture) []any { return []any{"alice"} }, func(f *pgFixture) (any, error) { return f.monitors.GetAllByUser(ctx, f.usr) }},
		{"MonitorRepository", "GetAllByTargetID", "GetAllMonitorsByTargetID", "monitor", false, func(f *pgFixture) []any { return []any{pgID(f.tgt.ID)} }, func(f *pgFixture) (any, error) { return f.monitors.GetAllByTargetID(ctx, f.tgt.ID) }},
		{"MonitorRepository", "GetMonitorsToEvaluate", "GetMonitorsToEvaluate", "monitor", false, func(f *pgFixture) []any { return []any{[]pgtype.UUID{pgID(f.tgt.ID)}} }, func(f *pgFixture) (any, error) { return f.monitors.GetMonitorsToEvaluate(ctx, []uuid.UUID{f.tgt.ID}) }},
		{"AlertContactRepository", "GetByID", "GetAlertContactByID", "contact", true, func(f *pgFixture) []any { return []any{pgID(f.contact.ID)} }, func(f *pgFixture) (any, error) { return f.contacts.GetByID(ctx, f.contact.ID) }},
		{"AlertContactRepository", "GetByIDBulk", "GetAlertContactsByIDBulk", "contact", false, func(f *pgFixture) []any { return []any{[]pgtype.UUID{pgID(f.contact.ID)}} }, func(f *pgFixture) (any, error) { return f.contacts.GetByIDBulk(ctx, []uuid.UUID{f.contact.ID}) }},
		{"AlertContactRepository", "GetByUserLogin", "GetAlertContactsByUserLogin", "contact", false, func(f *pgFixture) []any { return []any{"alice"} }, func(f *pgFixture) (any, error) { return f.contacts.GetByUserLogin(ctx, "alice") }},
		{"MaintenanceWindowRepository", "GetByID", "GetMaintenanceWindowByID", "window", true, func(f *pgFixture) []any { return []any{pgID(f.window.ID)} }, func(f *pgFixture) (any, error) { return f.windows.GetByID(ctx, f.window.ID) }},
		{"MaintenanceWindowRepository", "GetByIDBulk", "GeMaintenanceWindowsByIDBulk", "window", false, func(f *pgFixture) []any { return []any{[]pgtype.UUID{pgID(f.window.ID)}} }, func(f *pgFixture) (any, error) { return f.windows.GetByIDBulk(ctx, []uuid.UUID{f.window.ID}) }},
		{"MaintenanceWindowRepository", "GetByUserLogin", "GetMaintenanceWindowsByUserLogin", "window", false, func(f *pgFixture) []any { return []any{"alice"} }, func(f *pgFixture) (any, error) { return f.windows.GetByUserLogin(ctx, "alice") }},
		{"ProbeResultRepository", "FetchUnprocessed", "GetUnprocessedProbeResults", "result", false, func(f *pgFixture) []any { return []any{int32(10)} }, func(f *pgFixture) (any, error) { return f.results.FetchUnprocessed(ctx, 10) }},
		{"ProbeSummaryRepository", "GetMonitorLatestSummaries", "GetProbeSummaryByMonitorID", "summary", false, func(f *pgFixture) []any { return []any{pgID(f.mon.ID), int32(10)} }, func(f *pgFixture) (any, error) { return f.summaries.GetMonitorLatestSummaries(ctx, f.mon.ID, 10) }},
		{"ProbeSummaryRepository", "GetMonitorSummariesForPeriod", "GetProbeSummaryByMonitorIDForPeriod", "summary", false, func(f *pgFixture) []any {
			return []any{pgID(f.mon.ID), pgTime(f.result.ProbeTime), pgTime(f.result.ProbeTime.Add(time.Hour))}
		}, func(f *pgFixture) (any, error) {
			return f.summaries.GetMonitorSummariesForPeriod(ctx, f.mon.ID, f.result.ProbeTime, f.result.ProbeTime.Add(time.Hour))
		}},
		{"MetricsRepository", "GetStatusEvents", "GetStatusHistory", "events", false, func(f *pgFixture) []any {
			return []any{pgID(f.mon.ID), pgTime(f.result.ProbeTime), pgTime(f.result.ProbeTime.Add(time.Hour))}
		}, func(f *pgFixture) (any, error) {
			return f.metrics.GetStatusEvents(ctx, f.mon.ID, f.result.ProbeTime, f.result.ProbeTime.Add(time.Hour))
		}},
		{"MetricsRepository", "GetSLAAggregation", "GetSLAStat", "sla", true, func(f *pgFixture) []any {
			return []any{pgID(f.mon.ID), pgTime(f.result.ProbeTime), pgTime(f.result.ProbeTime.Add(time.Hour))}
		}, func(f *pgFixture) (any, error) {
			return f.metrics.GetSLAAggregation(ctx, f.mon.ID, f.result.ProbeTime, f.result.ProbeTime.Add(time.Hour))
		}},
	}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(tc.component+"/"+tc.method, func(t *testing.T) {
			for _, outcome := range []string{"success", "database_error", "empty"} {
				t.Run(outcome, func(t *testing.T) {
					testutil.Case(t, tc.component, tc.method, "equivalence", "london", func(t *testing.T, a *allure.Context) {
						var f *pgFixture
						var got any
						var err error
						a.Step("Arrange", func(*allure.Context) {
							f = newPGFixture(t)
							expected := f.db.ExpectQuery("-- name: " + tc.query + " ").WithArgs(tc.args(f)...)
							switch outcome {
							case "database_error":
								expected.WillReturnError(databaseFailure)
							case "empty":
								expected.WillReturnRows(pgxmock.NewRows([]string{"unused"}))
							default:
								expected.WillReturnRows(mockRows(sampleRow(tc.kind, f)))
							}
						})
						a.Step("Act", func(*allure.Context) { got, err = tc.call(f) })
						a.Step("Assert", func(*allure.Context) {
							switch outcome {
							case "database_error":
								require.ErrorIs(t, err, databaseFailure)
								require.ErrorIs(t, err, repo.ErrDB)
							case "empty":
								if tc.single {
									require.ErrorIs(t, err, repo.ErrNotFound)
									require.ErrorIs(t, err, pgx.ErrNoRows)
								} else {
									require.NoError(t, err)
									require.Empty(t, got)
								}
							default:
								require.NoError(t, err)
								assertPGRead(t, f, got)
							}
							require.NoError(t, f.db.ExpectationsWereMet())
						})
					})
				})
			}
		})
	}
}

func TestPostgres_InvalidStoredData(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name, component, method, query, kind string
		classification                       string
		column                               int
		invalid                              any
		wantErr                              error
		call                                 func(*pgFixture) (any, error)
	}{
		{"unknown_protocol_from_future_schema", "TargetRepository", "GetByID", "GetTargetByID", "target", "forward-schema defensive", 2, sqlcgen.ProtocolType("FUTURE_PROTOCOL"), repo.ErrInternal, func(f *pgFixture) (any, error) { return f.targets.GetByID(ctx, f.tgt.ID) }},
		{"valid_json_wrong_network_config_shape", "TargetRepository", "GetByID", "GetTargetByID", "target", "valid JSON incompatible shape", 5, []byte(`"future-config"`), repo.ErrDB, func(f *pgFixture) (any, error) { return f.targets.GetByID(ctx, f.tgt.ID) }},
		{"valid_json_wrong_expectations_shape", "MonitorRepository", "GetByID", "GetMonitorByID", "monitor", "valid JSON incompatible shape", 6, []byte(`"future-expectations"`), repo.ErrInternal, func(f *pgFixture) (any, error) { return f.monitors.GetByID(ctx, f.mon.ID) }},
		{"valid_json_wrong_contact_config_shape", "AlertContactRepository", "GetByID", "GetAlertContactByID", "contact", "valid JSON incompatible shape", 4, []byte(`"future-contact"`), repo.ErrDB, func(f *pgFixture) (any, error) { return f.contacts.GetByID(ctx, f.contact.ID) }},
		{"valid_json_wrong_window_config_shape", "MaintenanceWindowRepository", "GetByID", "GetMaintenanceWindowByID", "window", "valid JSON incompatible shape", 5, []byte(`"future-window"`), repo.ErrInternal, func(f *pgFixture) (any, error) { return f.windows.GetByID(ctx, f.window.ID) }},
		{"unknown_processing_status_from_future_schema", "ProbeResultRepository", "FetchUnprocessed", "GetUnprocessedProbeResults", "result", "forward-schema defensive", 8, sqlcgen.ProcessingStatusType("FUTURE_STATUS"), repo.ErrInternal, func(f *pgFixture) (any, error) { return f.results.FetchUnprocessed(ctx, 10) }},
		{"unknown_monitor_status_from_future_schema", "ProbeSummaryRepository", "GetMonitorLatestSummaries", "GetProbeSummaryByMonitorID", "summary", "forward-schema defensive", 3, sqlcgen.StatusType("FUTURE_STATUS"), repo.ErrInternal, func(f *pgFixture) (any, error) { return f.summaries.GetMonitorLatestSummaries(ctx, f.mon.ID, 10) }},
		{"invalid_numeric_sla_value", "MetricsRepository", "GetSLAAggregation", "GetSLAStat", "sla", "defensive conversion", 1, "invalid-number", nil, func(f *pgFixture) (any, error) {
			return f.metrics.GetSLAAggregation(ctx, f.mon.ID, f.result.ProbeTime, f.result.ProbeTime.Add(time.Hour))
		}},
	}
	for _, tc := range testutil.Shuffle(t, cases) {
		t.Run(fmt.Sprintf("%s/%s/%s", tc.component, tc.method, tc.name), func(t *testing.T) {
			testutil.Case(t, tc.component, tc.method, "equivalence", "london", func(t *testing.T, a *allure.Context) {
				var f *pgFixture
				var err error
				a.Step("Arrange", func(*allure.Context) {
					f = newPGFixture(t)
					row := sampleRow(tc.kind, f)
					row[tc.column] = tc.invalid
					f.db.ExpectQuery("-- name: " + tc.query + " ").WithArgs(func() []any {
						switch tc.kind {
						case "target":
							return []any{pgID(f.tgt.ID)}
						case "monitor":
							return []any{pgID(f.mon.ID)}
						case "contact":
							return []any{pgID(f.contact.ID)}
						case "window":
							return []any{pgID(f.window.ID)}
						case "result":
							return []any{int32(10)}
						case "summary":
							return []any{pgID(f.mon.ID), int32(10)}
						default:
							return []any{pgID(f.mon.ID), pgTime(f.result.ProbeTime), pgTime(f.result.ProbeTime.Add(time.Hour))}
						}
					}()...).WillReturnRows(mockRows(row))
					a.Parameter("classification", tc.classification)
				})
				a.Step("Act", func(*allure.Context) { _, err = tc.call(f) })
				a.Step("Assert", func(*allure.Context) {
					require.Error(t, err)
					if tc.wantErr != nil {
						require.ErrorIs(t, err, tc.wantErr)
					}
					require.NoError(t, f.db.ExpectationsWereMet())
				})
			})
		})
	}
}
func TestPostgres_NoOpWrites(t *testing.T) {
	for _, method := range []string{"Create", "BulkCreate"} {
		t.Run(method, func(t *testing.T) {
			for _, empty := range []bool{false, true} {
				t.Run(map[bool]string{false: "summary", true: "empty"}[empty], func(t *testing.T) {
					testutil.Case(t, "ProbeSummaryRepository", method, "equivalence", "london", func(t *testing.T, a *allure.Context) {
						var f *pgFixture
						var summaries []*probe.Summary
						var summary *probe.Summary
						var err error
						a.Step("Arrange", func(*allure.Context) {
							f = newPGFixture(t)
							if !empty {
								summary = &probe.Summary{MonitorID: f.mon.ID, MonitorStatus: monitor.StatusUp}
								summaries = []*probe.Summary{summary}
							}
						})
						a.Step("Act", func(*allure.Context) {
							if method == "Create" {
								err = f.summaries.Create(context.Background(), summary)
							} else {
								err = f.summaries.BulkCreate(context.Background(), summaries)
							}
						})
						a.Step("Assert", func(*allure.Context) { require.NoError(t, err); require.NoError(t, f.db.ExpectationsWereMet()) })
					})
				})
			}
		})
	}
}
func TestPostgres_ZeroLatestLimit(t *testing.T) {
	testutil.Case(t, "ProbeSummaryRepository", "GetMonitorLatestSummaries", "boundary", "london", func(t *testing.T, a *allure.Context) {
		var f *pgFixture
		var got []*probe.Summary
		var err error
		a.Step("Arrange", func(*allure.Context) { f = newPGFixture(t) })
		a.Step("Act", func(*allure.Context) {
			got, err = f.summaries.GetMonitorLatestSummaries(context.Background(), f.mon.ID, 0)
		})
		a.Step("Assert", func(*allure.Context) {
			require.NoError(t, err)
			require.Empty(t, got)
			require.NoError(t, f.db.ExpectationsWereMet())
		})
	})
}
