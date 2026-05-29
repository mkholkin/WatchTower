package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
)

type probeSummaryRepositoryPG struct {
	queries *sqlcgen.Queries
	log     *slog.Logger
}

func NewProbeSummaryRepository(pool *pgxpool.Pool, logger *slog.Logger) repo.ProbeSummaryRepository {
	return &probeSummaryRepositoryPG{
		queries: sqlcgen.New(pool),
		log:     logger,
	}
}

func (r *probeSummaryRepositoryPG) GetMonitorLatestSummaries(ctx context.Context, monitorID uuid.UUID, limit int) ([]*probe.Summary, error) {
	if limit <= 0 {
		return []*probe.Summary{}, nil
	}

	rows, err := r.queries.GetProbeSummaryByMonitorID(ctx, sqlcgen.GetProbeSummaryByMonitorIDParams{
		ID:    pgtype.UUID{Bytes: monitorID, Valid: true},
		Limit: int32(limit),
	})
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	result := make([]*probe.Summary, len(rows))
	for i, row := range rows {
		status, err := mapDBStatusToDomain(row.MonitorStatus)
		if err != nil {
			r.log.Error("failed to map monitor status for probe summary", "status", row.MonitorStatus, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}

		result[i] = &probe.Summary{
			MonitorID:      row.MonitorID.Bytes,
			LatencyMs:      row.LatencyMs,
			ProbeTime:      row.ProbeTime.Time,
			MonitorStatus:  status,
			StatusCode:     row.StatusCode,
			NetworkFailure: row.NetworkFailure,
			FailureReason:  row.FailureReason,
		}
	}

	return result, nil
}

func (r *probeSummaryRepositoryPG) GetMonitorSummariesForPeriod(
	ctx context.Context,
	monitorID uuid.UUID,
	from, to time.Time,
) ([]*probe.Summary, error) {
	if to.Before(from) {
		from, to = to, from
	}

	rows, err := r.queries.GetProbeSummaryByMonitorIDForPeriod(ctx, sqlcgen.GetProbeSummaryByMonitorIDForPeriodParams{
		ID:          pgtype.UUID{Bytes: monitorID, Valid: true},
		ProbeTime:   pgtype.Timestamptz{Time: from, Valid: true},
		ProbeTime_2: pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	result := make([]*probe.Summary, len(rows))
	for i, row := range rows {
		status, err := mapDBStatusToDomain(row.MonitorStatus)
		if err != nil {
			r.log.Error("failed to map monitor status for probe summary", "status", row.MonitorStatus, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}

		result[i] = &probe.Summary{
			MonitorID:      row.MonitorID.Bytes,
			LatencyMs:      row.LatencyMs,
			ProbeTime:      row.ProbeTime.Time,
			MonitorStatus:  status,
			StatusCode:     row.StatusCode,
			NetworkFailure: row.NetworkFailure,
			FailureReason:  row.FailureReason,
		}
	}

	return result, nil
}

// Create is intentionally a no-op for the postgres summary repository in current design.
func (r *probeSummaryRepositoryPG) Create(ctx context.Context, summary *probe.Summary) error {
	return nil
}

// BulkCreate is intentionally a no-op for the postgres summary repository in current design.
func (r *probeSummaryRepositoryPG) BulkCreate(ctx context.Context, summaries []*probe.Summary) error {
	return nil
}

func mapDBStatusToDomain(status sqlcgen.StatusType) (monitor.Status, error) {
	switch status {
	case sqlcgen.StatusTypeUP:
		return monitor.StatusUp, nil
	case sqlcgen.StatusTypeDOWN:
		return monitor.StatusDown, nil
	case sqlcgen.StatusTypeMAINTENANCE:
		return monitor.StatusMaintenance, nil
	case sqlcgen.StatusTypeUNKNOWN:
		return monitor.StatusUnknown, nil
	default:
		return "", repo.ErrInternal
	}
}
