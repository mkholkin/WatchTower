package app

import (
	"WatchTower/configs"
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/target"
	notifinfra "WatchTower/internal/infra/notification"
	infra "WatchTower/internal/infra/probe"
	"WatchTower/internal/infra/repository/mongodb"
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
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	goredis "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"

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

	eventBus     *gochannel.GoChannel
	redisClient  *goredis.Client
	pgPools      map[string]*pgxpool.Pool
	mongoDBs     map[string]*mongo.Database
	mongoClients map[string]*mongo.Client
	logger       *slog.Logger

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

func runMigrations(ctx context.Context, cfg *configs.Config, logger *slog.Logger) error {
	db, err := sql.Open("pgx", cfg.Database.Migrations.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration db: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	logger.Info("running postgres migrations", "dir", cfg.MigrationsDir)

	if err := goose.UpContext(ctx, db, cfg.MigrationsDir); err != nil {
		return fmt.Errorf("run goose migrations: %w", err)
	}

	logger.Info("postgres migrations applied successfully")
	return nil
}

func InitApp(ctx context.Context, cfg *configs.Config) (*App, error) {
	logger, _, err := configs.NewLoggerFromConfig(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	if err := runMigrations(ctx, cfg, logger); err != nil {
		return nil, fmt.Errorf("falied to run migrations: %w", err)
	}

	redisOpts, err := goredis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url in config: %w", err)
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
		eventBus:     eventBus,
		redisClient:  redisClient,
		pgPools:      make(map[string]*pgxpool.Pool),
		mongoDBs:     make(map[string]*mongo.Database),
		mongoClients: make(map[string]*mongo.Client),
		logger:       logger,
	}

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
	if cfg.Database.Auth.DBType() == "mongodb" {
		db, err := a.getOrCreateMongoDB(ctx, cfg.Database.Auth.DSN, "auth")
		if err != nil {
			return nil, fmt.Errorf("failed to init auth mongo db: %w", err)
		}
		userRepo := mongodb.NewUserRepository(db, a.logger)
		jwtTTL := time.Duration(cfg.Auth.JWTTTLHours) * time.Hour
		a.AuthSvc = authsvc.NewService(userRepo, cfg.Auth.JWTSecret, jwtTTL)
		return provider.NewUserProvider(userRepo), nil
	}

	pool, err := newPgPool(ctx, cfg.Database.Auth.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to init auth db pool: %w", err)
	}
	a.pgPools["auth"] = pool

	userRepo := postgres.NewUserRepository(pool, a.logger)
	jwtTTL := time.Duration(cfg.Auth.JWTTTLHours) * time.Hour
	a.AuthSvc = authsvc.NewService(userRepo, cfg.Auth.JWTSecret, jwtTTL)

	return provider.NewUserProvider(userRepo), nil
}

func (a *App) initMonitoringService(ctx context.Context, cfg *configs.Config, userProvider provider.UserProvider) error {
	if cfg.Database.Monitoring.DBType() == "mongodb" {
		db, err := a.getOrCreateMongoDB(ctx, cfg.Database.Monitoring.DSN, "monitoring")
		if err != nil {
			return fmt.Errorf("failed to init monitoring mongo db: %w", err)
		}
		a.MonitoringSvc = monitoringsvc.NewMonitoringManagementService(
			mongodb.NewMonitorRepository(db, a.logger),
			mongodb.NewTargetRepository(db, a.logger),
			mongodb.NewAlertContactRepository(db, a.logger),
			mongodb.NewMaintenanceWindowRepository(db, a.logger),
			userProvider, a.eventBus, a.logger,
		)
		return nil
	}

	pool, err := newPgPool(ctx, cfg.Database.Monitoring.DSN)
	if err != nil {
		return fmt.Errorf("failed to init monitoring db pool: %v", err)
	}
	a.pgPools["monitoring"] = pool

	a.MonitoringSvc = monitoringsvc.NewMonitoringManagementService(
		postgres.NewMonitorRepository(pool, a.logger),
		postgres.NewTargetRepository(pool, a.logger),
		postgres.NewAlertContactRepository(pool, a.logger),
		postgres.NewMaintenanceWindowRepository(pool, a.logger),
		userProvider, a.eventBus, a.logger,
	)
	return nil
}

func (a *App) initMaintenanceService(ctx context.Context, cfg *configs.Config, userProvider provider.UserProvider) error {
	if cfg.Database.Maintenance.DBType() == "mongodb" {
		db, err := a.getOrCreateMongoDB(ctx, cfg.Database.Maintenance.DSN, "maintenance")
		if err != nil {
			return fmt.Errorf("failed to init maintenance mongo db: %w", err)
		}
		a.MaintenanceSvc = maintenancesvc.NewMaintenanceService(
			mongodb.NewMaintenanceWindowRepository(db, a.logger),
			mongodb.NewMonitorRepository(db, a.logger),
			userProvider, a.logger,
		)
		return nil
	}

	pool, err := newPgPool(ctx, cfg.Database.Maintenance.DSN)
	if err != nil {
		return fmt.Errorf("failed to init maintenance db pool: %v", err)
	}
	a.pgPools["maintenance"] = pool

	a.MaintenanceSvc = maintenancesvc.NewMaintenanceService(
		postgres.NewMaintenanceWindowRepository(pool, a.logger),
		postgres.NewMonitorRepository(pool, a.logger),
		userProvider, a.logger,
	)
	return nil
}

func (a *App) initContactService(ctx context.Context, cfg *configs.Config, userProvider provider.UserProvider) error {
	if cfg.Database.Contacts.DBType() == "mongodb" {
		db, err := a.getOrCreateMongoDB(ctx, cfg.Database.Contacts.DSN, "contacts")
		if err != nil {
			return fmt.Errorf("failed to init contacts mongo db: %w", err)
		}
		a.AlertContactsSvc = contactssvc.NewContactService(
			mongodb.NewAlertContactRepository(db, a.logger),
			userProvider, a.logger,
		)
		return nil
	}

	pool, err := newPgPool(ctx, cfg.Database.Contacts.DSN)
	if err != nil {
		return fmt.Errorf("failed to init contacts db pool: %v", err)
	}
	a.pgPools["contacts"] = pool

	a.AlertContactsSvc = contactssvc.NewContactService(
		postgres.NewAlertContactRepository(pool, a.logger),
		userProvider, a.logger,
	)
	return nil
}

func (a *App) initMetricsQueryService(ctx context.Context, cfg *configs.Config, userProvider provider.UserProvider) error {
	if cfg.Database.Metrics.DBType() == "mongodb" {
		db, err := a.getOrCreateMongoDB(ctx, cfg.Database.Metrics.DSN, "metrics")
		if err != nil {
			return fmt.Errorf("failed to init metrics mongo db: %w", err)
		}
		probeSummaryRepo := mongodb.NewProbeSummaryRepository(db, a.logger)
		cachedSummaryRepo := redis.NewProbeSummaryRepository(a.redisClient, probeSummaryRepo, a.logger)
		a.MetricsQuerySvc = metricssvc.NewMetricsQueryService(
			mongodb.NewMonitorRepository(db, a.logger),
			userProvider,
			mongodb.NewAnalyticsRepository(db, a.logger),
			cachedSummaryRepo,
		)
		return nil
	}

	pool, err := newPgPool(ctx, cfg.Database.Metrics.DSN)
	if err != nil {
		return fmt.Errorf("failed to init metrics db pool: %v", err)
	}
	a.pgPools["metrics"] = pool

	probeSummaryRepo := postgres.NewProbeSummaryRepository(pool, a.logger)
	cachedSummaryRepo := redis.NewProbeSummaryRepository(a.redisClient, probeSummaryRepo, a.logger)
	a.MetricsQuerySvc = metricssvc.NewMetricsQueryService(
		postgres.NewMonitorRepository(pool, a.logger),
		userProvider,
		postgres.NewMetricsRepository(pool, a.logger),
		cachedSummaryRepo,
	)
	return nil
}

func (a *App) initHealthChecker(ctx context.Context, cfg *configs.Config) error {
	if cfg.Database.Healthchecker.DBType() == "mongodb" {
		db, err := a.getOrCreateMongoDB(ctx, cfg.Database.Healthchecker.DSN, "healthchecker")
		if err != nil {
			return fmt.Errorf("failed to init healthchecker mongo db: %w", err)
		}
		registry := healthchecksvc.NewProberRegistry()
		registry.Register(target.ProtocolHTTP, infra.NewHTTPProber())
		a.healthchecker = healthchecksvc.NewHealthChecker(
			mongodb.NewTargetRepository(db, a.logger),
			mongodb.NewProbeResultRepository(db, a.logger),
			a.eventBus, registry,
			healthchecksvc.HealthCheckerConfig{WorkerCount: 5, TaskQueueSize: 100},
			a.logger,
		)
		return nil
	}

	pool, err := newPgPool(ctx, cfg.Database.Healthchecker.DSN)
	if err != nil {
		return fmt.Errorf("failed to init healthchecker db pool: %v", err)
	}
	a.pgPools["healthchecker"] = pool

	registry := healthchecksvc.NewProberRegistry()
	registry.Register(target.ProtocolHTTP, infra.NewHTTPProber())
	a.healthchecker = healthchecksvc.NewHealthChecker(
		postgres.NewTargetRepository(pool, a.logger),
		postgres.NewProbeResultRepository(pool, a.logger),
		a.eventBus, registry,
		healthchecksvc.HealthCheckerConfig{WorkerCount: 200, TaskQueueSize: 10000},
		a.logger,
	)
	return nil
}

func (a *App) initAnalyzer(ctx context.Context, cfg *configs.Config) error {
	if cfg.Database.Analyzer.DBType() == "mongodb" {
		db, err := a.getOrCreateMongoDB(ctx, cfg.Database.Analyzer.DSN, "analyzer")
		if err != nil {
			return fmt.Errorf("failed to init analyzer mongo db: %w", err)
		}
		probeSummaryRepo := mongodb.NewProbeSummaryRepository(db, a.logger)
		cachedSummaryRepo := redis.NewProbeSummaryRepository(a.redisClient, probeSummaryRepo, a.logger)
		evaluator := infra.NewHTTPProbeEvaluator()
		a.analyzer = analyzationsvc.NewProbeAnalyzationService(
			mongodb.NewMonitorRepository(db, a.logger),
			mongodb.NewProbeResultRepository(db, a.logger),
			cachedSummaryRepo, evaluator, a.eventBus,
			analyzationsvc.ProbeAnalyzationServiceConfig{FetchLimit: -1, LoadSheddingThreshold: -1},
			a.logger,
		)
		return nil
	}

	pool, err := newPgPool(ctx, cfg.Database.Analyzer.DSN)
	if err != nil {
		return fmt.Errorf("failed to init analyzer db pool: %v", err)
	}
	a.pgPools["analyzer"] = pool

	probeSummaryRepo := postgres.NewProbeSummaryRepository(pool, a.logger)
	cachedSummaryRepo := redis.NewProbeSummaryRepository(a.redisClient, probeSummaryRepo, a.logger)
	evaluator := infra.NewHTTPProbeEvaluator()
	a.analyzer = analyzationsvc.NewProbeAnalyzationService(
		postgres.NewMonitorRepository(pool, a.logger),
		postgres.NewProbeResultRepository(pool, a.logger),
		cachedSummaryRepo, evaluator, a.eventBus,
		analyzationsvc.ProbeAnalyzationServiceConfig{FetchLimit: -1, LoadSheddingThreshold: -1},
		a.logger,
	)
	return nil
}

func (a *App) initNotificationService(_ context.Context, _ *configs.Config) error {
	// notification service always uses the monitoring service's DB.
	// Determine which backend monitoring uses by checking our registries.
	pool, pgOk := a.pgPools["monitoring"]
	db, mongoOk := a.mongoDBs["monitoring"]

	if !pgOk && !mongoOk {
		return fmt.Errorf("monitoring db must be initialized before notification service")
	}

	notificationRegistry := notification.NewProviderRegistry()
	notificationRegistry.Register(alert.ContactTypeTelegram, notifinfra.NewTelegramNotificationProvider(nil))

	if pgOk {
		a.notificator = notification.NewNotificationService(
			notificationRegistry, a.eventBus,
			postgres.NewMonitorRepository(pool, a.logger),
			a.logger,
		)
	} else {
		a.notificator = notification.NewNotificationService(
			notificationRegistry, a.eventBus,
			mongodb.NewMonitorRepository(db, a.logger),
			a.logger,
		)
	}
	return nil
}

// --- DB helpers ---

func (a *App) getOrCreateMongoDB(ctx context.Context, dsn, serviceName string) (*mongo.Database, error) {
	if db, ok := a.mongoDBs[serviceName]; ok {
		return db, nil
	}

	// Reuse client if same DSN is already connected.
	client, ok := a.mongoClients[dsn]
	if !ok {
		var err error
		client, err = mongodb.NewClient(ctx, dsn)
		if err != nil {
			return nil, fmt.Errorf("mongo connect: %w", err)
		}
		a.mongoClients[dsn] = client
	}

	dbName := dbNameFromDSN(dsn)
	db := client.Database(dbName)
	a.mongoDBs[serviceName] = db
	return db, nil
}

func dbNameFromDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "watchtower"
	}
	dbName := u.Path
	if len(dbName) > 0 && dbName[0] == '/' {
		dbName = dbName[1:]
	}
	if dbName == "" {
		dbName = "watchtower"
	}
	return dbName
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
	for _, pool := range a.pgPools {
		pool.Close()
	}
	for _, client := range a.mongoClients {
		_ = client.Disconnect(context.Background())
	}
}
