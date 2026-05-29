package postgres

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
	"WatchTower/pkg/mapper"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type alertContactRepositoryPG struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	log     *slog.Logger
	mpr     *typeMapper
}

func NewAlertContactRepository(pool *pgxpool.Pool, logger *slog.Logger) repo.AlertContactRepository {
	return &alertContactRepositoryPG{
		pool:    pool,
		queries: sqlcgen.New(pool),
		log:     logger,
		mpr:     newTypeMapper(logger),
	}
}

func (r *alertContactRepositoryPG) Create(ctx context.Context, contact *alert.Contact) error {
	dbType, err := r.mpr.ToDBContactType(contact.Type)
	if err != nil {
		r.log.Error("failed to convert contact type to DB type", "contact_type", contact.Type, "error", err)
		return repo.ErrInternal
	}

	dbConfig, err := r.mpr.ToDBContactConfig(contact.Config)
	if err != nil {
		r.log.Error("failed to convert contact config to DB config", "contact_type", contact.Type, "error", err)
		return repo.ErrInternal
	}

	params := sqlcgen.CreateAlertContactParams{
		ID:        pgtype.UUID{Bytes: contact.ID, Valid: true},
		UserLogin: contact.User.Login,
		Type:      dbType,
		Label:     contact.Name,
		Config:    dbConfig,
		IsActive:  contact.IsActive,
	}

	if err := r.queries.CreateAlertContact(ctx, params); err != nil {
		return repo.ErrDB
	}

	return nil
}

func (r *alertContactRepositoryPG) GetByID(ctx context.Context, id uuid.UUID) (*alert.Contact, error) {
	row, err := r.queries.GetAlertContactByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	domainType, err := r.mpr.ToDomainContactType(row.Type)
	if err != nil {
		r.log.Error("failed to convert contact type to domain contact", "error", err)
		return nil, repo.ErrInternal
	}

	domainConfig, err := r.mpr.ToDomainContactConfig(domainType, row.Config)
	if err != nil {
		r.log.Error("failed to convert contact config to domain config", "error", err)
		return nil, repo.ErrDB
	}

	return &alert.Contact{
		ID: row.ID.Bytes,
		User: &user.User{
			Login:        row.Login,
			PasswordHash: row.PasswordHash,
		},
		Type:     domainType,
		Name:     row.Label,
		Config:   domainConfig,
		IsActive: row.IsActive,
	}, nil
}

func (r *alertContactRepositoryPG) Update(ctx context.Context, contact *alert.Contact) error {
	dbType, err := r.mpr.ToDBContactType(contact.Type)
	if err != nil {
		r.log.Error("failed to convert contact type to DB type", "contact_type", contact.Type, "error", err)
		return repo.ErrInternal
	}

	dbConfig, err := r.mpr.ToDBContactConfig(contact.Config)
	if err != nil {
		r.log.Error("failed to convert contact config to DB config", "error", err)
		return repo.ErrInternal
	}

	params := sqlcgen.UpdateAlertContactParams{
		ID:        pgtype.UUID{Bytes: contact.ID, Valid: true},
		UserLogin: contact.User.Login,
		Type:      dbType,
		Label:     contact.Name,
		Config:    dbConfig,
	}

	if err := r.queries.UpdateAlertContact(ctx, params); err != nil {
		return repo.ErrDB
	}

	return nil
}

func (r *alertContactRepositoryPG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteAlertContactByID(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return repo.ErrDB
	}
	return nil
}

func (r *alertContactRepositoryPG) GetByIDBulk(ctx context.Context, ids []uuid.UUID) ([]alert.Contact, error) {
	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	rows, err := r.queries.GetAlertContactsByIDBulk(ctx, pgIDs)
	if err != nil {
		return nil, repo.ErrDB
	}

	contacts := make([]alert.Contact, len(rows))
	for i, row := range rows {
		domainType, err := r.mpr.ToDomainContactType(row.Type)
		if err != nil {
			r.log.Error("failed to convert contact type to domain contact", "error", err)
			return nil, repo.ErrInternal
		}

		domainConfig, err := r.mpr.ToDomainContactConfig(domainType, row.Config)
		if err != nil {
			r.log.Error("failed to convert contact config to domain config", "error", err)
			return nil, repo.ErrDB
		}

		contacts[i] = alert.Contact{
			ID: row.ID.Bytes,
			User: &user.User{
				Login:        row.Login,
				PasswordHash: row.PasswordHash,
			},
			Type:     domainType,
			Name:     row.Label,
			Config:   domainConfig,
			IsActive: row.IsActive,
		}
	}
	return contacts, nil
}

func (r *alertContactRepositoryPG) Enable(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.EnableAlertContact(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return repo.ErrDB
	}
	return nil
}

func (r *alertContactRepositoryPG) Disable(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DisableAlertContact(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return repo.ErrDB
	}
	return nil
}

func (r *alertContactRepositoryPG) GetByUserLogin(ctx context.Context, userLogin string) ([]alert.Contact, error) {
	rows, err := r.queries.GetAlertContactsByUserLogin(ctx, userLogin)
	if err != nil {
		return nil, repo.ErrDB
	}

	contacts := make([]alert.Contact, len(rows))
	for i, row := range rows {
		domainType, err := r.mpr.ToDomainContactType(row.Type)
		if err != nil {
			r.log.Error("failed to convert contact type to domain contact", "error", err)
			return nil, repo.ErrInternal
		}

		domainConfig, err := r.mpr.ToDomainContactConfig(domainType, row.Config)
		if err != nil {
			r.log.Error("failed to convert contact config to domain config", "error", err)
			return nil, repo.ErrDB
		}

		contacts[i] = alert.Contact{
			ID: row.ID.Bytes,
			User: &user.User{
				Login:        row.Login,
				PasswordHash: row.PasswordHash,
			},
			Type:     domainType,
			Name:     row.Label,
			Config:   domainConfig,
			IsActive: row.IsActive,
		}
	}
	return contacts, nil
}

type typeMapper struct {
	log                    *slog.Logger
	toDBTypeRegistry       mapper.Mapper[alert.ContactType, sqlcgen.ContactType, alert.ContactType]
	toDBConfigRegistry     mapper.Mapper[alert.ContactConfig, []byte, alert.ContactType]
	toDomainTypeRegistry   mapper.Mapper[sqlcgen.ContactType, alert.ContactType, sqlcgen.ContactType]
	toDomainConfigRegistry mapper.Mapper[[]byte, alert.ContactConfig, alert.ContactType]
}

func newTypeMapper(log *slog.Logger) *typeMapper {
	toDBTypeMap := mapper.New[alert.ContactType, sqlcgen.ContactType, alert.ContactType]()
	toDBTypeMap.Register(alert.ContactTypeTelegram, func(f alert.ContactType) (sqlcgen.ContactType, error) {
		return sqlcgen.ContactTypeTELEGRAM, nil
	})

	toDBConfigMap := mapper.New[alert.ContactConfig, []byte, alert.ContactType]()
	toDBConfigMap.Register(alert.ContactTypeTelegram, func(c alert.ContactConfig) ([]byte, error) {
		tg, ok := c.(alert.TelegramContactConfig)
		if !ok {
			if tgPtr, ok := c.(*alert.TelegramContactConfig); ok && tgPtr != nil {
				tg = *tgPtr
			} else {
				return nil, fmt.Errorf("unexpected config type for telegram: %T", c)
			}
		}
		return json.Marshal(tg)
	})

	toDomainTypeMap := mapper.New[sqlcgen.ContactType, alert.ContactType, sqlcgen.ContactType]()
	toDomainTypeMap.Register(sqlcgen.ContactTypeTELEGRAM, func(f sqlcgen.ContactType) (alert.ContactType, error) {
		return alert.ContactTypeTelegram, nil
	})

	toDomainConfigMap := mapper.New[[]byte, alert.ContactConfig, alert.ContactType]()
	toDomainConfigMap.Register(alert.ContactTypeTelegram, func(payload []byte) (alert.ContactConfig, error) {
		var tgConfig alert.TelegramContactConfig
		if err := json.Unmarshal(payload, &tgConfig); err != nil {
			return nil, err
		}
		return tgConfig, nil
	})

	return &typeMapper{
		log:                    log,
		toDBConfigRegistry:     toDBConfigMap,
		toDBTypeRegistry:       toDBTypeMap,
		toDomainTypeRegistry:   toDomainTypeMap,
		toDomainConfigRegistry: toDomainConfigMap,
	}
}

func (m typeMapper) ToDBContactType(t alert.ContactType) (sqlcgen.ContactType, error) {
	return m.toDBTypeRegistry.Convert(t, t)
}

func (m typeMapper) ToDomainContactType(t sqlcgen.ContactType) (alert.ContactType, error) {
	return m.toDomainTypeRegistry.Convert(t, t)
}

func (m typeMapper) ToDBContactConfig(config alert.ContactConfig) ([]byte, error) {
	return m.toDBConfigRegistry.Convert(config.Type(), config)
}

func (m typeMapper) ToDomainContactConfig(t alert.ContactType, json []byte) (alert.ContactConfig, error) {
	return m.toDomainConfigRegistry.Convert(t, json)
}
