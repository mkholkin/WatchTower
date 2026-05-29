package postgres

import (
	"WatchTower/pkg/mapper"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
)

type maintenanceWindowRepositoryPG struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	log     *slog.Logger
	mpr     *maintenanceWindowTypeMapper
}

func NewMaintenanceWindowRepository(pool *pgxpool.Pool, logger *slog.Logger) repo.MaintenanceWindowRepository {
	return &maintenanceWindowRepositoryPG{
		pool:    pool,
		queries: sqlcgen.New(pool),
		log:     logger,
		mpr:     newMaintenanceWindowTypeMapper(logger),
	}
}

func (r *maintenanceWindowRepositoryPG) Create(ctx context.Context, mw *maintenance.MaintenanceWindow) error {
	dbType, err := r.mpr.ToDBMaintenanceWindowType(mw.MaintenanceWindowType)
	if err != nil {
		r.log.Error("failed to convert maintenance window type to DB type", "window_type", mw.MaintenanceWindowType, "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	dbConfig, err := r.mpr.ToDBMaintenanceWindowConfig(mw.MaintenanceWindowConfig)
	if err != nil {
		r.log.Error("failed to convert maintenance window config to DB config", "window_type", mw.MaintenanceWindowType, "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	params := sqlcgen.CreateMaintenanceWindowParams{
		ID:          pgtype.UUID{Bytes: mw.ID, Valid: true},
		UserLogin:   mw.User.Login,
		Title:       mw.Title,
		Description: pgtype.Text{String: mw.Description, Valid: mw.Description != ""},
		Type:        dbType,
		Config:      dbConfig,
	}

	if err := r.queries.CreateMaintenanceWindow(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *maintenanceWindowRepositoryPG) GetByID(ctx context.Context, id uuid.UUID) (*maintenance.MaintenanceWindow, error) {
	row, err := r.queries.GetMaintenanceWindowByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	mw, err := mapMaintenanceWindowToDomain(row.MaintenanceWindow, row.User, r.mpr)
	if err != nil {
		r.log.Error("failed to map maintenance window row to domain", "window_id", row.MaintenanceWindow.ID, "error", err)
		return nil, errors.Join(repo.ErrInternal, err)
	}

	return mw, nil
}

func (r *maintenanceWindowRepositoryPG) Update(ctx context.Context, mw *maintenance.MaintenanceWindow) error {
	dbType, err := r.mpr.ToDBMaintenanceWindowType(mw.MaintenanceWindowType)
	if err != nil {
		r.log.Error("failed to convert maintenance window type to DB type", "window_type", mw.MaintenanceWindowType, "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	dbConfig, err := r.mpr.ToDBMaintenanceWindowConfig(mw.MaintenanceWindowConfig)
	if err != nil {
		r.log.Error("failed to convert maintenance window config to DB config", "window_type", mw.MaintenanceWindowType, "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	params := sqlcgen.UpdateMaintenanceWindowParams{
		ID:          pgtype.UUID{Bytes: mw.ID, Valid: true},
		UserLogin:   mw.User.Login,
		Title:       mw.Title,
		Description: pgtype.Text{String: mw.Description, Valid: mw.Description != ""},
		Type:        dbType,
		Config:      dbConfig,
	}

	if err := r.queries.UpdateMaintenanceWindow(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *maintenanceWindowRepositoryPG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteMaintenanceWindowByID(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *maintenanceWindowRepositoryPG) GetByIDBulk(ctx context.Context, ids []uuid.UUID) ([]maintenance.MaintenanceWindow, error) {
	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	rows, err := r.queries.GeMaintenanceWindowsByIDBulk(ctx, pgIDs)
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	var result []maintenance.MaintenanceWindow
	for _, row := range rows {
		mw, err := mapMaintenanceWindowToDomain(row.MaintenanceWindow, row.User, r.mpr)
		if err != nil {
			r.log.Error("failed to map maintenance window row to domain", "window_id", row.MaintenanceWindow.ID, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}
		result = append(result, *mw)
	}

	return result, nil
}

func (r *maintenanceWindowRepositoryPG) GetByUserLogin(ctx context.Context, userLogin string) ([]maintenance.MaintenanceWindow, error) {
	rows, err := r.queries.GetMaintenanceWindowsByUserLogin(ctx, userLogin)
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	result := make([]maintenance.MaintenanceWindow, 0, len(rows))
	for _, row := range rows {
		mw, err := mapMaintenanceWindowToDomain(row.MaintenanceWindow, row.User, r.mpr)
		if err != nil {
			r.log.Error("failed to map maintenance window row to domain", "window_id", row.MaintenanceWindow.ID, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}

		result = append(result, *mw)
	}

	return result, nil
}

func (r *maintenanceWindowRepositoryPG) LinkMonitor(ctx context.Context, window *maintenance.MaintenanceWindow, monitorID uuid.UUID) error {
	params := sqlcgen.LinkMonitorParams{
		MonitorID: pgtype.UUID{Bytes: monitorID, Valid: true},
		WindowID:  pgtype.UUID{Bytes: window.ID, Valid: true},
	}

	if err := r.queries.LinkMonitor(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *maintenanceWindowRepositoryPG) UnlinkMonitor(ctx context.Context, window *maintenance.MaintenanceWindow, monitorID uuid.UUID) error {
	params := sqlcgen.UnlinkMonitorParams{
		MonitorID: pgtype.UUID{Bytes: monitorID, Valid: true},
		WindowID:  pgtype.UUID{Bytes: window.ID, Valid: true},
	}

	if err := r.queries.UnlinkMonitor(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

// ----------------- Helpers -----------------

func mapMaintenanceWindowToDomain(
	dbWindow sqlcgen.MaintenanceWindow,
	dbUser sqlcgen.User,
	mpr *maintenanceWindowTypeMapper,
) (*maintenance.MaintenanceWindow, error) {
	domainType, err := mpr.ToDomainMaintenanceWindowType(dbWindow.Type)
	if err != nil {
		return nil, err
	}

	config, err := mpr.ToDomainMaintenanceWindowConfig(domainType, dbWindow.Config)
	if err != nil {
		return nil, err
	}

	return &maintenance.MaintenanceWindow{
		ID:                      dbWindow.ID.Bytes,
		Title:                   dbWindow.Title,
		Description:             dbWindow.Description.String,
		MaintenanceWindowType:   domainType,
		MaintenanceWindowConfig: config,
		User: &user.User{
			Login:        dbUser.Login,
			PasswordHash: dbUser.PasswordHash,
		},
	}, nil
}

type maintenanceWindowTypeMapper struct {
	log                    *slog.Logger
	toDBTypeRegistry       mapper.Mapper[maintenance.WindowType, sqlcgen.MaintenanceType, maintenance.WindowType]
	toDBConfigRegistry     mapper.Mapper[maintenance.MaintenanceWindowConfig, []byte, maintenance.WindowType]
	toDomainTypeRegistry   mapper.Mapper[sqlcgen.MaintenanceType, maintenance.WindowType, sqlcgen.MaintenanceType]
	toDomainConfigRegistry mapper.Mapper[[]byte, maintenance.MaintenanceWindowConfig, maintenance.WindowType]
}

func newMaintenanceWindowTypeMapper(log *slog.Logger) *maintenanceWindowTypeMapper {
	toDBTypeMap := mapper.New[maintenance.WindowType, sqlcgen.MaintenanceType, maintenance.WindowType]()
	toDBTypeMap.Register(maintenance.WindowTypeOneTime, func(_ maintenance.WindowType) (sqlcgen.MaintenanceType, error) {
		return sqlcgen.MaintenanceTypeONCE, nil
	})
	toDBTypeMap.Register(maintenance.WindowTypeManual, func(_ maintenance.WindowType) (sqlcgen.MaintenanceType, error) {
		return sqlcgen.MaintenanceTypeMANUAL, nil
	})

	toDBConfigMap := mapper.New[maintenance.MaintenanceWindowConfig, []byte, maintenance.WindowType]()
	toDBConfigMap.Register(maintenance.WindowTypeOneTime, func(c maintenance.MaintenanceWindowConfig) ([]byte, error) {
		onceCfg, ok := c.(maintenance.OneTimeMaintenanceWindowConfig)
		if !ok {
			if onceCfgPtr, ok := c.(*maintenance.OneTimeMaintenanceWindowConfig); ok && onceCfgPtr != nil {
				onceCfg = *onceCfgPtr
			} else {
				return nil, fmt.Errorf("unexpected config type for one-time maintenance window: %T", c)
			}
		}
		return json.Marshal(onceCfg)
	})
	toDBConfigMap.Register(maintenance.WindowTypeManual, func(c maintenance.MaintenanceWindowConfig) ([]byte, error) {
		manualCfg, ok := c.(maintenance.ManualMaintenanceWindowConfig)
		if !ok {
			if manualCfgPtr, ok := c.(*maintenance.ManualMaintenanceWindowConfig); ok && manualCfgPtr != nil {
				manualCfg = *manualCfgPtr
			} else {
				return nil, fmt.Errorf("unexpected config type for manual maintenance window: %T", c)
			}
		}
		return json.Marshal(manualCfg)
	})

	toDomainTypeMap := mapper.New[sqlcgen.MaintenanceType, maintenance.WindowType, sqlcgen.MaintenanceType]()
	toDomainTypeMap.Register(sqlcgen.MaintenanceTypeONCE, func(_ sqlcgen.MaintenanceType) (maintenance.WindowType, error) {
		return maintenance.WindowTypeOneTime, nil
	})
	toDomainTypeMap.Register(sqlcgen.MaintenanceTypeMANUAL, func(_ sqlcgen.MaintenanceType) (maintenance.WindowType, error) {
		return maintenance.WindowTypeManual, nil
	})

	toDomainConfigMap := mapper.New[[]byte, maintenance.MaintenanceWindowConfig, maintenance.WindowType]()
	toDomainConfigMap.Register(maintenance.WindowTypeOneTime, func(payload []byte) (maintenance.MaintenanceWindowConfig, error) {
		var cfg maintenance.OneTimeMaintenanceWindowConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})
	toDomainConfigMap.Register(maintenance.WindowTypeManual, func(payload []byte) (maintenance.MaintenanceWindowConfig, error) {
		var cfg maintenance.ManualMaintenanceWindowConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})

	return &maintenanceWindowTypeMapper{
		log:                    log,
		toDBTypeRegistry:       toDBTypeMap,
		toDBConfigRegistry:     toDBConfigMap,
		toDomainTypeRegistry:   toDomainTypeMap,
		toDomainConfigRegistry: toDomainConfigMap,
	}
}

func (m maintenanceWindowTypeMapper) ToDBMaintenanceWindowType(t maintenance.WindowType) (sqlcgen.MaintenanceType, error) {
	return m.toDBTypeRegistry.Convert(t, t)
}

func (m maintenanceWindowTypeMapper) ToDomainMaintenanceWindowType(t sqlcgen.MaintenanceType) (maintenance.WindowType, error) {
	return m.toDomainTypeRegistry.Convert(t, t)
}

func (m maintenanceWindowTypeMapper) ToDBMaintenanceWindowConfig(config maintenance.MaintenanceWindowConfig) ([]byte, error) {
	return m.toDBConfigRegistry.Convert(config.Type(), config)
}

func (m maintenanceWindowTypeMapper) ToDomainMaintenanceWindowConfig(t maintenance.WindowType, payload []byte) (maintenance.MaintenanceWindowConfig, error) {
	return m.toDomainConfigRegistry.Convert(t, payload)
}
