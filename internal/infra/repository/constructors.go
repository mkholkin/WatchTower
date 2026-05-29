package repository

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"WatchTower/configs"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/mongodb"
	"WatchTower/internal/infra/repository/postgres"
	"WatchTower/internal/service/metrics"
)

// --- internal closer ---

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// --- Postgres ---

func connectPgPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}
	return pool, nil
}

// --- MongoDB ---

func connectMongoDB(ctx context.Context, dsn string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongodb.NewClient(ctx, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}
	dbName := dbNameFromDSN(dsn)
	return client, client.Database(dbName), nil
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

// pgCloser wraps pgxpool.Close (no error return) into io.Closer.
func pgCloser(pool *pgxpool.Pool) io.Closer {
	return closerFunc(func() error { pool.Close(); return nil })
}

// mongoCloser wraps mongo.Client.Disconnect into io.Closer.
func mongoCloser(client *mongo.Client) io.Closer {
	return closerFunc(func() error { return client.Disconnect(context.Background()) })
}

// --- Repository constructors ---

func newRepo[T any](ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger, build func(*pgxpool.Pool) T, buildMongo func(*mongo.Database) T) (T, io.Closer, error) {
	if cfg.DBType() == "mongodb" {
		client, db, err := connectMongoDB(ctx, cfg.DSN)
		if err != nil {
			var zero T
			return zero, nil, err
		}
		return buildMongo(db), mongoCloser(client), nil
	}
	pool, err := connectPgPool(ctx, cfg.DSN)
	if err != nil {
		var zero T
		return zero, nil, err
	}
	return build(pool), pgCloser(pool), nil
}

func NewMonitorRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (repo.MonitorRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) repo.MonitorRepository   { return postgres.NewMonitorRepository(p, logger) },
		func(db *mongo.Database) repo.MonitorRepository { return mongodb.NewMonitorRepository(db, logger) },
	)
}

func NewTargetRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (repo.TargetRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) repo.TargetRepository   { return postgres.NewTargetRepository(p, logger) },
		func(db *mongo.Database) repo.TargetRepository { return mongodb.NewTargetRepository(db, logger) },
	)
}

func NewAlertContactRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (repo.AlertContactRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) repo.AlertContactRepository   { return postgres.NewAlertContactRepository(p, logger) },
		func(db *mongo.Database) repo.AlertContactRepository { return mongodb.NewAlertContactRepository(db, logger) },
	)
}

func NewMaintenanceWindowRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (repo.MaintenanceWindowRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) repo.MaintenanceWindowRepository   { return postgres.NewMaintenanceWindowRepository(p, logger) },
		func(db *mongo.Database) repo.MaintenanceWindowRepository { return mongodb.NewMaintenanceWindowRepository(db, logger) },
	)
}

func NewProbeResultRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (repo.ProbeResultRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) repo.ProbeResultRepository   { return postgres.NewProbeResultRepository(p, logger) },
		func(db *mongo.Database) repo.ProbeResultRepository { return mongodb.NewProbeResultRepository(db, logger) },
	)
}

func NewProbeSummaryRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (repo.ProbeSummaryRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) repo.ProbeSummaryRepository   { return postgres.NewProbeSummaryRepository(p, logger) },
		func(db *mongo.Database) repo.ProbeSummaryRepository { return mongodb.NewProbeSummaryRepository(db, logger) },
	)
}

func NewUserRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (repo.UserRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) repo.UserRepository   { return postgres.NewUserRepository(p, logger) },
		func(db *mongo.Database) repo.UserRepository { return mongodb.NewUserRepository(db, logger) },
	)
}

func NewAnalyticsRepository(ctx context.Context, cfg configs.ServiceDBConfig, logger *slog.Logger) (metrics.AnalyticsRepository, io.Closer, error) {
	return newRepo(ctx, cfg, logger,
		func(p *pgxpool.Pool) metrics.AnalyticsRepository   { return postgres.NewMetricsRepository(p, logger) },
		func(db *mongo.Database) metrics.AnalyticsRepository { return mongodb.NewAnalyticsRepository(db, logger) },
	)
}
