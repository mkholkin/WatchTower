package postgres

import (
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
	"WatchTower/internal/service/metrics"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricsRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	log     *slog.Logger
}

func NewMetricsRepository(pool *pgxpool.Pool, logger *slog.Logger) *MetricsRepository {
	return &MetricsRepository{
		pool:    pool,
		queries: sqlcgen.New(pool),
		log:     logger.With("component", "metrics_repository"),
	}
}

func (m *MetricsRepository) GetStatusEvents(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]metrics.StatusEvent, error) {
	rows, err := m.queries.GetStatusHistory(ctx, sqlcgen.GetStatusHistoryParams{
		MonitorID: pgtype.UUID{Bytes: monitorID, Valid: true},
		EndTime:   pgtype.Timestamptz{Time: from, Valid: true},
		StartTime: pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		m.log.Error("failed to get status history", "monitor_id", monitorID, "from", from, "to", to, "error", err)
		return nil, mapPGXErrorToRepo(err)
	}

	result := make([]metrics.StatusEvent, 0, len(rows))
	for _, row := range rows {
		domainStatus, err := mapDBStatusToDomain(row.Status)
		if err != nil {
			m.log.Error("failed to map status in metrics history", "status", row.Status, "error", err)
			return nil, err
		}

		result = append(result, metrics.StatusEvent{
			Status:    domainStatus,
			StartTime: row.StartTime.Time,
			EndTime:   row.EndTime.Time,
		})
	}

	return result, nil
}

func (m *MetricsRepository) GetSLAAggregation(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (metrics.SLAStats, error) {
	row, err := m.queries.GetSLAStat(ctx, sqlcgen.GetSLAStatParams{
		Column1: pgtype.UUID{Bytes: monitorID, Valid: true},
		Column2: pgtype.Timestamptz{Time: from, Valid: true},
		Column3: pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return metrics.SLAStats{}, mapPGXErrorToRepo(err)
	}

	uptimePercent, err := asFloat64(row.UptimePercent)
	if err != nil {
		m.log.Error("failed to parse uptime_percent", "value", row.UptimePercent, "error", err)
		return metrics.SLAStats{}, err
	}

	return metrics.SLAStats{
		MonitorID:        row.MonitorID.Bytes,
		UptimePercent:    uptimePercent,
		TotalDowntimeSec: int(row.TotalDowntimeSec),
		PeriodStart:      row.PeriodStart.Time,
		PeriodEnd:        row.PeriodEnd.Time,
	}, nil
}

func asFloat64(v interface{}) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		return strconv.ParseFloat(x, 64)
	case []byte:
		return strconv.ParseFloat(string(x), 64)
	default:
		return 0, fmt.Errorf("unsupported float type: %T", v)
	}
}
