//go:build lab2integration

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres"
	"WatchTower/internal/infra/repository/redis"
	auth "WatchTower/internal/service/auth"
	"WatchTower/internal/service/common/provider"
	contacts "WatchTower/internal/service/contacts"
	healthcheck "WatchTower/internal/service/healthcheck"
	maintenance "WatchTower/internal/service/maintenance"
	"WatchTower/internal/service/metrics"
	monitoring_management "WatchTower/internal/service/monitoring_management"
	mdto "WatchTower/internal/service/monitoring_management/dto"
	"WatchTower/tests/lab2/testsupport"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	gredis "github.com/redis/go-redis/v9"
)

type fixture struct {
	pool         *pgxpool.Pool
	redis        *gredis.Client
	users        repo.UserRepository
	monitors     repo.MonitorRepository
	targets      repo.TargetRepository
	contactsRepo repo.AlertContactRepository
	windows      repo.MaintenanceWindowRepository
	results      repo.ProbeResultRepository
	summaries    repo.ProbeSummaryRepository
	auth         auth.AuthService
	monitoring   monitoring_management.MonitoringManagementService
	contacts     contacts.ContactService
	maintenance  maintenance.MaintenanceService
	metrics      metrics.MetricQueryService
}

func setup(t *testing.T, withRedis bool) *fixture {
	t.Helper()
	p := testsupport.Database(t)
	l := testsupport.Logger()
	f := &fixture{pool: p,
		users:        postgres.NewUserRepository(p, l),
		monitors:     postgres.NewMonitorRepository(p, l),
		targets:      postgres.NewTargetRepository(p, l),
		contactsRepo: postgres.NewAlertContactRepository(p, l),
		windows:      postgres.NewMaintenanceWindowRepository(p, l),
		results:      postgres.NewProbeResultRepository(p, l),
		summaries:    postgres.NewProbeSummaryRepository(p, l),
	}
	if withRedis {
		f.redis = testsupport.Redis(t)
		f.summaries = redis.NewProbeSummaryRepository(f.redis, f.summaries, l)
	}
	bus := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer:            64,
		BlockPublishUntilSubscriberAck: true,
	}, watermill.NewStdLogger(false, false))
	ctx, cancel := context.WithCancel(context.Background())
	var consumers sync.WaitGroup
	t.Cleanup(func() {
		// Every completed Publish has already been Acked by the subscribers.
		if err := bus.Close(); err != nil {
			t.Errorf("close event bus: %v", err)
		}
		cancel()
		consumers.Wait()
	})
	for _, topic := range []string{healthcheck.TopicTargetCreated, healthcheck.TopicTargetUpdated} {
		ch, err := bus.Subscribe(ctx, topic)
		if err != nil {
			t.Fatal(err)
		}
		consumers.Add(1)
		go func() {
			defer consumers.Done()
			for msg := range ch {
				msg.Ack()
			}
		}()
	}
	up := provider.NewUserProvider(f.users)
	f.auth = auth.NewService(f.users, "lab2-integration-secret", time.Hour)
	f.monitoring = monitoring_management.NewMonitoringManagementService(f.monitors, f.targets, f.contactsRepo, f.windows, up, bus, l)
	f.contacts = contacts.NewContactService(f.contactsRepo, up, l)
	f.maintenance = maintenance.NewMaintenanceService(f.windows, f.monitors, up, l)
	f.metrics = metrics.NewMetricsQueryService(f.monitors, up, postgres.NewMetricsRepository(p, l), f.summaries)
	return f
}

func (f *fixture) user(t *testing.T, login string) context.Context {
	t.Helper()
	if err := f.auth.Register(context.Background(), login, "Lab2Pass-123!"); err != nil {
		t.Fatal(err)
	}
	return auth.ContextWithUser(context.Background(), login)
}

func (f *fixture) createMonitor(t *testing.T, ctx context.Context, label, endpoint string, interval int32) *monitor.Monitor {
	t.Helper()
	m, err := f.monitoring.CreateMonitor(ctx, mdto.CreateMonitorDTO{
		Label: label, Endpoint: endpoint, ProbeIntervalSec: interval,
		NetworkConfig: mdto.HTTPMonitorNetworkConfig{Method: "GET"},
		Expectations:  mdto.HTTPMonitorExpectations{StatusCodes: []int{200}, MaxLatencyMs: 5000},
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
