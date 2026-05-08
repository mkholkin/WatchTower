package app

import (
	"WatchTower/configs"
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/target"
	notifinfra "WatchTower/internal/infra/notification"
	infra "WatchTower/internal/infra/probe"
	"WatchTower/internal/infra/repository/postgres"
	"WatchTower/internal/infra/repository/redis"
	analyzationsvc "WatchTower/internal/service/analyze"
	authsvc "WatchTower/internal/service/auth"
	"WatchTower/internal/service/common/provider"
	contactssvc "WatchTower/internal/service/contacts"
	healthchecksvc "WatchTower/internal/service/healthcheck"
	maintenancesvc "WatchTower/internal/service/maintenance"
	metricssvc "WatchTower/internal/service/metrics"
	monitoringsvc "WatchTower/internal/service/monitoring_management"
	"WatchTower/internal/service/notification"
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"log/slog"
)

type App struct {
	AuthSvc          authsvc.AuthService
	MonitoringSvc    monitoringsvc.MonitoringManagementService
	AlertContactsSvc contactssvc.ContactService
	MaintenanceSvc   maintenancesvc.MaintenanceService
	MetricsQuerySvc  metricssvc.MetricQueryService

	healthchecker healthchecksvc.HealthChecker
	analyzer      *analyzationsvc.ProbeAnalyzationService
	notificator   notification.NotificationService

	eventBus    *gochannel.GoChannel
	redisClient *goredis.Client
	pools       map[string]*pgxpool.Pool
	logger      *slog.Logger

	cancel context.CancelFunc
}

func newPgPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

type watermillLogger struct {
	watermill.LoggerAdapter
}

func newFilteredWatermillLogger(logger watermill.LoggerAdapter) watermill.LoggerAdapter {
	return &watermillLogger{LoggerAdapter: logger}
}

func InitApp(ctx context.Context, cfg *configs.Config) (*App, error) {
	logger, _, err := configs.NewLoggerFromConfig(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %v", err)
	}

	redisOpts, err := goredis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url in config: %v", err)
	}
	redisClient := goredis.NewClient(redisOpts)

	eventBus := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer:            256,
		Persistent:                     false,
		BlockPublishUntilSubscriberAck: false,
		PreserveContext:                false,
	},
		newFilteredWatermillLogger(watermill.NewStdLogger(false, false)),
	)

	app := &App{
		eventBus:    eventBus,
		redisClient: redisClient,
		pools:       make(map[string]*pgxpool.Pool),
		logger:      logger,
	}

	metricsPool, err := newPgPool(ctx, cfg.Database.Metrics.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to init metrics db pool: %v", err)
	}
	app.pools["metrics"] = metricsPool

	userProvider, err := app.initAuthService(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := app.initMonitoringService(ctx, cfg, userProvider); err != nil {
		return nil, err
	}

	if err := app.initMaintenanceService(ctx, cfg, userProvider); err != nil {
		return nil, err
	}

	if err := app.initContactService(ctx, cfg, userProvider); err != nil {
		return nil, err
	}

	if err := app.initMetricsQueryService(ctx, cfg, userProvider); err != nil {
		return nil, err
	}

	if err := app.initHealthChecker(ctx, cfg); err != nil {
		return nil, err
	}

	if err := app.initAnalyzer(ctx, cfg); err != nil {
		return nil, err
	}

	if err := app.initNotificationService(ctx, cfg); err != nil {
		return nil, err
	}

	return app, nil
}

func (a *App) initAuthService(ctx context.Context, cfg *configs.Config) (provider.UserProvider, error) {
	authPool, err := newPgPool(ctx, cfg.Database.Auth.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to init auth db pool: %v", err)
	}
	a.pools["auth"] = authPool

	userRepo := postgres.NewUserRepository(authPool, a.logger)
	jwtTTL := time.Duration(cfg.Auth.JWTTTLHours) * time.Hour
	a.AuthSvc = authsvc.NewService(userRepo, cfg.Auth.JWTSecret, jwtTTL)

	return provider.NewUserProvider(userRepo), nil
}

func (a *App) initMonitoringService(ctx context.Context, cfg *configs.Config, userProvider provider.UserProvider) error {
	monitoringPool, err := newPgPool(ctx, cfg.Database.Monitoring.DSN)
	if err != nil {
		return fmt.Errorf("failed to init monitoring db pool: %v", err)
	}
	a.pools["monitoring"] = monitoringPool

	monitoringMonitorRepo := postgres.NewMonitorRepository(monitoringPool, a.logger)
	monitoringTargetRepo := postgres.NewTargetRepository(monitoringPool, a.logger)
	monitoringAlertContactRepo := postgres.NewAlertContactRepository(monitoringPool, a.logger)
	monitoringMWRepo := postgres.NewMaintenanceWindowRepository(monitoringPool, a.logger)

	a.MonitoringSvc = monitoringsvc.NewMonitoringManagementService(
		monitoringMonitorRepo,
		monitoringTargetRepo,
		monitoringAlertContactRepo,
		monitoringMWRepo,
		userProvider,
		a.eventBus,
		a.logger,
	)
	return nil
}

func (a *App) initMaintenanceService(ctx context.Context, cfg *configs.Config, userProvider provider.UserProvider) error {
	maintenancePool, err := newPgPool(ctx, cfg.Database.Maintenance.DSN)
	if err != nil {
		return fmt.Errorf("failed to init maintenance db pool: %v", err)
	}
	a.pools["maintenance"] = maintenancePool

	maintenanceMonitorRepo := postgres.NewMonitorRepository(maintenancePool, a.logger)
	maintenanceMWRepo := postgres.NewMaintenanceWindowRepository(maintenancePool, a.logger)

	a.MaintenanceSvc = maintenancesvc.NewMaintenanceService(
		maintenanceMWRepo,
		maintenanceMonitorRepo,
		userProvider,
		a.logger,
	)
	return nil
}

func (a *App) initContactService(ctx context.Context, cfg *configs.Config, userProvider provider.UserProvider) error {
	contactsPool, err := newPgPool(ctx, cfg.Database.Contacts.DSN)
	if err != nil {
		return fmt.Errorf("failed to init contacts db pool: %v", err)
	}
	a.pools["contacts"] = contactsPool

	contactsAlertContactRepo := postgres.NewAlertContactRepository(contactsPool, a.logger)

	a.AlertContactsSvc = contactssvc.NewContactService(
		contactsAlertContactRepo,
		userProvider,
		a.logger,
	)
	return nil
}

func (a *App) initMetricsQueryService(_ context.Context, _ *configs.Config, userProvider provider.UserProvider) error {
	metricsPool, ok := a.pools["metrics"]
	if !ok {
		return fmt.Errorf("metrics pool must be initialized before metrics query service")
	}
	if a.redisClient == nil {
		return fmt.Errorf("redis client must be initialized before metrics query service")
	}

	metricsMonitorRepo := postgres.NewMonitorRepository(metricsPool, a.logger)
	metricsRepo := postgres.NewMetricsRepository(metricsPool, a.logger)
	metricsProbeSummaryRepo := postgres.NewProbeSummaryRepository(metricsPool, a.logger)
	probeSummaryRepo := redis.NewProbeSummaryRepository(
		a.redisClient,
		metricsProbeSummaryRepo,
		a.logger,
	)

	a.MetricsQuerySvc = metricssvc.NewMetricsQueryService(
		metricsMonitorRepo,
		userProvider,
		metricsRepo,
		probeSummaryRepo,
	)
	return nil
}

func (a *App) initHealthChecker(ctx context.Context, cfg *configs.Config) error {
	healthcheckerPool, err := newPgPool(ctx, cfg.Database.Healthchecker.DSN)
	if err != nil {
		return fmt.Errorf("failed to init healthchecker db pool: %v", err)
	}
	a.pools["healthchecker"] = healthcheckerPool

	healthTargetRepo := postgres.NewTargetRepository(healthcheckerPool, a.logger)
	healthProbeResultRepo := postgres.NewProbeResultRepository(healthcheckerPool, a.logger)

	registry := healthchecksvc.NewProberRegistry()
	registry.Register(target.ProtocolHTTP, infra.NewHTTPProber())

	a.healthchecker = healthchecksvc.NewHealthChecker(
		healthTargetRepo,
		healthProbeResultRepo,
		a.eventBus,
		registry,
		healthchecksvc.HealthCheckerConfig{WorkerCount: 5, TaskQueueSize: 100},
		a.logger,
	)
	return nil
}

func (a *App) initAnalyzer(ctx context.Context, cfg *configs.Config) error {
	analyzerPool, err := newPgPool(ctx, cfg.Database.Analyzer.DSN)
	if err != nil {
		return fmt.Errorf("failed to init analyzer db pool: %v", err)
	}
	a.pools["analyzer"] = analyzerPool

	analyzerMonitorRepo := postgres.NewMonitorRepository(analyzerPool, a.logger)
	analyzerProbeResultRepo := postgres.NewProbeResultRepository(analyzerPool, a.logger)
	analyzerProbeSummaryRepo := postgres.NewProbeSummaryRepository(analyzerPool, a.logger)
	probeSummaryRepo := redis.NewProbeSummaryRepository(
		a.redisClient,
		analyzerProbeSummaryRepo,
		a.logger,
	)

	evaluator := infra.NewHTTPProbeEvaluator()
	a.analyzer = analyzationsvc.NewProbeAnalyzationService(
		analyzerMonitorRepo,
		analyzerProbeResultRepo,
		probeSummaryRepo,
		evaluator,
		a.eventBus,
		analyzationsvc.ProbeAnalyzationServiceConfig{-1, -1},
		a.logger,
	)
	return nil
}

func (a *App) initNotificationService(_ context.Context, _ *configs.Config) error {
	monitoringPool, ok := a.pools["monitoring"]
	if !ok {
		return fmt.Errorf("monitoring pool must be initialized before notification service")
	}

	monitoringMonitorRepo := postgres.NewMonitorRepository(monitoringPool, a.logger)

	notificationRegistry := notification.NewProviderRegistry()
	notificationRegistry.Register(alert.ContactTypeTelegram, notifinfra.NewTelegramNotificationProvider(nil))

	a.notificator = notification.NewNotificationService(
		notificationRegistry,
		a.eventBus,
		monitoringMonitorRepo,
		a.logger,
	)
	return nil
}

func (a *App) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	go func() {
		if err := a.healthchecker.Run(ctx); err != nil {
			a.logger.Error("health checker stopped", "error", err)
		}
	}()

	go func() {
		if err := a.analyzer.Run(ctx); err != nil {
			a.logger.Error("analyzer stopped", "error", err)
		}
	}()

	go func() {
		if err := a.notificator.Run(ctx); err != nil {
			a.logger.Error("notification service stopped", "error", err)
		}
	}()
}

func (a *App) Shutdown() {
	a.cancel()
	for _, pool := range a.pools {
		pool.Close()
	}
}
