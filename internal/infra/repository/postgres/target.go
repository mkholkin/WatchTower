package postgres

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
	"WatchTower/pkg/mapper"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type targetRepositoryPG struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	log     *slog.Logger
	mpr     *targetTypeMapper
}

func NewTargetRepository(pool *pgxpool.Pool, logger *slog.Logger) repo.TargetRepository {
	return &targetRepositoryPG{
		pool:    pool,
		queries: sqlcgen.New(pool),
		log:     logger,
		mpr:     newTargetTypeMapper(logger),
	}
}

func (r *targetRepositoryPG) Create(ctx context.Context, tgt *target.Target) error {
	dbProtocol, err := r.mpr.ToDBProtocolType(tgt.Config.Protocol())
	if err != nil {
		r.log.Error("failed to convert protocol to DB type", "protocol", tgt.Config.Protocol(), "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	dbConfig, err := r.mpr.ToDBNetworkConfig(tgt.Config)
	if err != nil {
		r.log.Error("failed to convert network config to DB config", "protocol", tgt.Config.Protocol(), "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	params := sqlcgen.CreateTargetParams{
		ID:               pgtype.UUID{Bytes: tgt.ID, Valid: true},
		SignatureHash:    tgt.ConfigHash,
		Protocol:         dbProtocol,
		Endpoint:         tgt.Endpoint,
		NetworkConfig:    dbConfig,
		IsActive:         tgt.IsActive,
		ProbeIntervalSec: tgt.ProbeIntervalSec,
	}

	if err := r.queries.CreateTarget(ctx, params); err != nil {
		return errors.Join(repo.ErrDB, err)
	}

	return nil
}

func (r *targetRepositoryPG) GetByID(ctx context.Context, id uuid.UUID) (*target.Target, error) {
	row, err := r.queries.GetTargetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	return r.mapTargetToDomain(row)
}

func (r *targetRepositoryPG) Update(ctx context.Context, tgt *target.Target) error {
	dbProtocol, err := r.mpr.ToDBProtocolType(tgt.Config.Protocol())
	if err != nil {
		r.log.Error("failed to convert protocol to DB type", "protocol", tgt.Config.Protocol(), "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	dbConfig, err := r.mpr.ToDBNetworkConfig(tgt.Config)
	if err != nil {
		r.log.Error("failed to convert network config to DB config", "protocol", tgt.Config.Protocol(), "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	params := sqlcgen.UpdateTargetParams{
		ID:               pgtype.UUID{Bytes: tgt.ID, Valid: true},
		SignatureHash:    tgt.ConfigHash,
		Protocol:         dbProtocol,
		Endpoint:         tgt.Endpoint,
		NetworkConfig:    dbConfig,
		IsActive:         tgt.IsActive,
		ProbeIntervalSec: tgt.ProbeIntervalSec,
	}

	if err := r.queries.UpdateTarget(ctx, params); err != nil {
		return errors.Join(repo.ErrDB, err)
	}

	return nil
}

func (r *targetRepositoryPG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteTargetByID(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return errors.Join(repo.ErrDB, err)
	}
	return nil
}

func (r *targetRepositoryPG) UpdateProbeInterval(ctx context.Context, id uuid.UUID, probeIntervalSec int32) error {
	params := sqlcgen.UpdateTargetProbeIntervalParams{
		ID:               pgtype.UUID{Bytes: id, Valid: true},
		ProbeIntervalSec: probeIntervalSec,
	}

	if err := r.queries.UpdateTargetProbeInterval(ctx, params); err != nil {
		return errors.Join(repo.ErrDB, err)
	}
	return nil
}

func (r *targetRepositoryPG) GetByHash(ctx context.Context, hash string) (*target.Target, error) {
	row, err := r.queries.GetTargetByHash(ctx, hash)
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	return r.mapTargetToDomain(row)
}

func (r *targetRepositoryPG) GetAllActive(ctx context.Context) ([]target.Target, error) {
	rows, err := r.queries.GetAllActiveTargets(ctx)
	if err != nil {
		return nil, errors.Join(repo.ErrDB, err)
	}

	targets := make([]target.Target, len(rows))
	for i, row := range rows {
		tgt, mapErr := r.mapTargetToDomain(row)
		if mapErr != nil {
			return nil, mapErr
		}
		targets[i] = *tgt
	}

	return targets, nil
}

func (r *targetRepositoryPG) Disable(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DisableTarget(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return errors.Join(repo.ErrDB, err)
	}
	return nil
}

func (r *targetRepositoryPG) Enable(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.EnableTarget(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return errors.Join(repo.ErrDB, err)
	}
	return nil
}

func (r *targetRepositoryPG) mapTargetToDomain(row sqlcgen.Target) (*target.Target, error) {
	domainProtocol, err := r.mpr.ToDomainProtocolType(row.Protocol)
	if err != nil {
		r.log.Error("failed to convert protocol to domain type", "protocol", row.Protocol, "error", err)
		return nil, errors.Join(repo.ErrInternal, err)
	}

	domainConfig, err := r.mpr.ToDomainNetworkConfig(domainProtocol, row.NetworkConfig)
	if err != nil {
		r.log.Error("failed to convert network config to domain config", "protocol", row.Protocol, "error", err)
		return nil, errors.Join(repo.ErrDB, err)
	}

	return &target.Target{
		ID:               row.ID.Bytes,
		Endpoint:         row.Endpoint,
		Config:           domainConfig,
		IsActive:         row.IsActive,
		ProbeIntervalSec: row.ProbeIntervalSec,
		ConfigHash:       row.SignatureHash,
	}, nil
}

type targetTypeMapper struct {
	log                    *slog.Logger
	toDBProtocolRegistry   mapper.Mapper[target.Protocol, sqlcgen.ProtocolType, target.Protocol]
	toDBConfigRegistry     mapper.Mapper[target.NetworkConfig, []byte, target.Protocol]
	toDomainProtocolReg    mapper.Mapper[sqlcgen.ProtocolType, target.Protocol, sqlcgen.ProtocolType]
	toDomainConfigRegistry mapper.Mapper[[]byte, target.NetworkConfig, target.Protocol]
}

func newTargetTypeMapper(log *slog.Logger) *targetTypeMapper {
	toDBProtocolMap := mapper.New[target.Protocol, sqlcgen.ProtocolType, target.Protocol]()
	toDBProtocolMap.Register(target.ProtocolHTTP, func(_ target.Protocol) (sqlcgen.ProtocolType, error) {
		return sqlcgen.ProtocolTypeHTTP, nil
	})
	toDBProtocolMap.Register(target.ProtocolTCP, func(_ target.Protocol) (sqlcgen.ProtocolType, error) {
		return sqlcgen.ProtocolTypeTCP, nil
	})
	toDBProtocolMap.Register(target.ProtocolICMP, func(_ target.Protocol) (sqlcgen.ProtocolType, error) {
		return sqlcgen.ProtocolTypeICMP, nil
	})

	toDBConfigMap := mapper.New[target.NetworkConfig, []byte, target.Protocol]()
	toDBConfigMap.Register(target.ProtocolHTTP, func(c target.NetworkConfig) ([]byte, error) {
		httpCfg, ok := c.(target.HTTPConfig)
		if !ok {
			if httpCfgPtr, ok := c.(*target.HTTPConfig); ok && httpCfgPtr != nil {
				httpCfg = *httpCfgPtr
			} else {
				return nil, fmt.Errorf("unexpected config type for HTTP: %T", c)
			}
		}
		return json.Marshal(httpCfg)
	})
	toDBConfigMap.Register(target.ProtocolTCP, func(c target.NetworkConfig) ([]byte, error) {
		tcpCfg, ok := c.(target.TCPConfig)
		if !ok {
			if tcpCfgPtr, ok := c.(*target.TCPConfig); ok && tcpCfgPtr != nil {
				tcpCfg = *tcpCfgPtr
			} else {
				return nil, fmt.Errorf("unexpected config type for TCP: %T", c)
			}
		}
		return json.Marshal(tcpCfg)
	})
	toDBConfigMap.Register(target.ProtocolICMP, func(c target.NetworkConfig) ([]byte, error) {
		icmpCfg, ok := c.(target.ICMPConfig)
		if !ok {
			if icmpCfgPtr, ok := c.(*target.ICMPConfig); ok && icmpCfgPtr != nil {
				icmpCfg = *icmpCfgPtr
			} else {
				return nil, fmt.Errorf("unexpected config type for ICMP: %T", c)
			}
		}
		return json.Marshal(icmpCfg)
	})

	toDomainProtocolMap := mapper.New[sqlcgen.ProtocolType, target.Protocol, sqlcgen.ProtocolType]()
	toDomainProtocolMap.Register(sqlcgen.ProtocolTypeHTTP, func(_ sqlcgen.ProtocolType) (target.Protocol, error) {
		return target.ProtocolHTTP, nil
	})
	toDomainProtocolMap.Register(sqlcgen.ProtocolTypeTCP, func(_ sqlcgen.ProtocolType) (target.Protocol, error) {
		return target.ProtocolTCP, nil
	})
	toDomainProtocolMap.Register(sqlcgen.ProtocolTypeICMP, func(_ sqlcgen.ProtocolType) (target.Protocol, error) {
		return target.ProtocolICMP, nil
	})

	toDomainConfigMap := mapper.New[[]byte, target.NetworkConfig, target.Protocol]()
	toDomainConfigMap.Register(target.ProtocolHTTP, func(payload []byte) (target.NetworkConfig, error) {
		var cfg target.HTTPConfig
		if err := json.Unmarshal(payload, &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	})
	toDomainConfigMap.Register(target.ProtocolTCP, func(payload []byte) (target.NetworkConfig, error) {
		var cfg target.TCPConfig
		if err := json.Unmarshal(payload, &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	})
	toDomainConfigMap.Register(target.ProtocolICMP, func(payload []byte) (target.NetworkConfig, error) {
		var cfg target.ICMPConfig
		if err := json.Unmarshal(payload, &cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	})

	return &targetTypeMapper{
		log:                    log,
		toDBProtocolRegistry:   toDBProtocolMap,
		toDBConfigRegistry:     toDBConfigMap,
		toDomainProtocolReg:    toDomainProtocolMap,
		toDomainConfigRegistry: toDomainConfigMap,
	}
}

func (m targetTypeMapper) ToDBProtocolType(p target.Protocol) (sqlcgen.ProtocolType, error) {
	return m.toDBProtocolRegistry.Convert(p, p)
}

func (m targetTypeMapper) ToDomainProtocolType(p sqlcgen.ProtocolType) (target.Protocol, error) {
	return m.toDomainProtocolReg.Convert(p, p)
}

func (m targetTypeMapper) ToDBNetworkConfig(config target.NetworkConfig) ([]byte, error) {
	return m.toDBConfigRegistry.Convert(config.Protocol(), config)
}

func (m targetTypeMapper) ToDomainNetworkConfig(protocol target.Protocol, payload []byte) (target.NetworkConfig, error) {
	return m.toDomainConfigRegistry.Convert(protocol, payload)
}
