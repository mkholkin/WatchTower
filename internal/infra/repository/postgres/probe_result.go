package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
	"WatchTower/pkg/mapper"
)

type probeResultRepositoryPG struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	log     *slog.Logger
	mpr     *probeResultTypeMapper
}

func NewProbeResultRepository(pool *pgxpool.Pool, logger *slog.Logger) repo.ProbeResultRepository {
	return &probeResultRepositoryPG{
		pool:    pool,
		queries: sqlcgen.New(pool),
		log:     logger,
		mpr:     newProbeResultTypeMapper(),
	}
}

func (r *probeResultRepositoryPG) Create(probeResult *probe.Result) error {
	if probeResult == nil || probeResult.Target == nil {
		r.log.Error("invalid probe result: nil value")
		return fmt.Errorf("invalid probe result: nil value %w", repo.ErrInternal)
	}

	params := sqlcgen.CreateProbeResultParams{
		ID:             pgtype.UUID{Bytes: probeResult.ID, Valid: true},
		TargetID:       pgtype.UUID{Bytes: probeResult.Target.ID, Valid: true},
		ProbeTime:      pgtype.Timestamp{Time: probeResult.ProbeTime, Valid: true},
		LatencyMs:      probeResult.LatencyMs,
		StatusCode:     toPGInt4(probeResult.StatusCode),
		NetworkFailure: probeResult.NetworkFailure,
		ErrorMessage:   pgtype.Text{},
		Meta:           probeResult.Meta,
	}

	if err := r.queries.CreateProbeResult(context.Background(), params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *probeResultRepositoryPG) FetchUnprocessed(ctx context.Context, limit int) ([]*probe.Result, error) {
	rows, err := r.queries.GetUnprocessedProbeResults(ctx, int32(limit))
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	result := make([]*probe.Result, len(rows))
	for i, row := range rows {
		domainStatus, err := r.mpr.ToDomainProcessingStatus(row.ProcessingStatus)
		if err != nil {
			r.log.Error("failed to map processing status to domain", "status", row.ProcessingStatus, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}

		result[i] = &probe.Result{
			ID:               row.ID.Bytes,
			LatencyMs:        row.LatencyMs,
			Meta:             row.Meta,
			NetworkFailure:   row.NetworkFailure,
			StatusCode:       toNullInt32(row.StatusCode),
			Target:           &target.Target{ID: row.TargetID.Bytes},
			ProbeTime:        row.ProbeTime.Time,
			ProcessingStatus: domainStatus,
		}
	}

	return result, nil
}

func (r *probeResultRepositoryPG) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status probe.ProcessingStatus) error {
	dbStatus, err := r.mpr.ToDBProcessingStatus(status)
	if err != nil {
		r.log.Error("failed to map processing status to DB", "status", status, "error", err)
		return fmt.Errorf("failed to map processing status to DB: %w", repo.ErrInternal)
	}

	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	params := sqlcgen.BulkUpdateProbeResultStatusParams{
		ProcessingStatus: dbStatus,
		Ids:              pgIDs,
	}

	if err := r.queries.BulkUpdateProbeResultStatus(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

type probeResultTypeMapper struct {
	toDBStatusRegistry     mapper.Mapper[probe.ProcessingStatus, sqlcgen.ProcessingStatusType, probe.ProcessingStatus]
	toDomainStatusRegistry mapper.Mapper[sqlcgen.ProcessingStatusType, probe.ProcessingStatus, sqlcgen.ProcessingStatusType]
}

func newProbeResultTypeMapper() *probeResultTypeMapper {
	toDBStatusMap := mapper.New[probe.ProcessingStatus, sqlcgen.ProcessingStatusType, probe.ProcessingStatus]()
	toDBStatusMap.Register(probe.ProcessingStatusNew, func(_ probe.ProcessingStatus) (sqlcgen.ProcessingStatusType, error) {
		return sqlcgen.ProcessingStatusTypeNEW, nil
	})
	toDBStatusMap.Register(probe.ProcessingStatusProcessed, func(_ probe.ProcessingStatus) (sqlcgen.ProcessingStatusType, error) {
		return sqlcgen.ProcessingStatusTypePROCESSED, nil
	})
	toDBStatusMap.Register(probe.ProcessingStatusCanceled, func(_ probe.ProcessingStatus) (sqlcgen.ProcessingStatusType, error) {
		return sqlcgen.ProcessingStatusTypeCANCELLED, nil
	})

	toDomainStatusMap := mapper.New[sqlcgen.ProcessingStatusType, probe.ProcessingStatus, sqlcgen.ProcessingStatusType]()
	toDomainStatusMap.Register(sqlcgen.ProcessingStatusTypeNEW, func(_ sqlcgen.ProcessingStatusType) (probe.ProcessingStatus, error) {
		return probe.ProcessingStatusNew, nil
	})
	toDomainStatusMap.Register(sqlcgen.ProcessingStatusTypePROCESSED, func(_ sqlcgen.ProcessingStatusType) (probe.ProcessingStatus, error) {
		return probe.ProcessingStatusProcessed, nil
	})
	toDomainStatusMap.Register(sqlcgen.ProcessingStatusTypeCANCELLED, func(_ sqlcgen.ProcessingStatusType) (probe.ProcessingStatus, error) {
		return probe.ProcessingStatusCanceled, nil
	})

	return &probeResultTypeMapper{
		toDBStatusRegistry:     toDBStatusMap,
		toDomainStatusRegistry: toDomainStatusMap,
	}
}

func (m probeResultTypeMapper) ToDBProcessingStatus(status probe.ProcessingStatus) (sqlcgen.ProcessingStatusType, error) {
	return m.toDBStatusRegistry.Convert(status, status)
}

func (m probeResultTypeMapper) ToDomainProcessingStatus(status sqlcgen.ProcessingStatusType) (probe.ProcessingStatus, error) {
	return m.toDomainStatusRegistry.Convert(status, status)
}

func toPGInt4(v sql.NullInt32) pgtype.Int4 {
	return pgtype.Int4{Int32: v.Int32, Valid: v.Valid}
}

func toNullInt32(v pgtype.Int4) sql.NullInt32 {
	return sql.NullInt32{Int32: v.Int32, Valid: v.Valid}
}
