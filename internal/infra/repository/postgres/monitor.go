package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
)

type monitorRepositoryPG struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
	log     *slog.Logger
	mpr     *monitorTypeMapper
}

func NewMonitorRepository(pool *pgxpool.Pool, logger *slog.Logger) repo.MonitorRepository {
	return &monitorRepositoryPG{
		pool:    pool,
		queries: sqlcgen.New(pool),
		log:     logger,
		mpr:     newMonitorTypeMapper(),
	}
}

func (r *monitorRepositoryPG) Create(ctx context.Context, mon *monitor.Monitor) error {
	dbStatus, err := r.mpr.ToDBStatusType(mon.CurrentStatus)
	if err != nil {
		r.log.Error("failed to convert monitor status to DB status", "status", mon.CurrentStatus, "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	dbExpectations, err := r.mpr.ToDBExpectations(mon.Expectations)
	if err != nil {
		r.log.Error("failed to convert monitor expectations to DB payload", "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	params := sqlcgen.CreateMonitorParams{
		ID:               pgtype.UUID{Bytes: mon.ID, Valid: true},
		TargetID:         pgtype.UUID{Bytes: mon.Target.ID, Valid: true},
		UserLogin:        mon.User.Login,
		Label:            mon.Label,
		IsActive:         mon.IsActive,
		ProbeIntervalSec: mon.ProbeIntervalSec,
		Expectations:     dbExpectations,
		CurrentStatus:    dbStatus,
		CreatedAt:        pgtype.Timestamptz{Time: mon.CreatedAt, Valid: true},
	}

	if err := r.queries.CreateMonitor(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *monitorRepositoryPG) GetByID(ctx context.Context, id uuid.UUID) (*monitor.Monitor, error) {
	row, err := r.queries.GetMonitorByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	mon, err := mapMonitorRowToDomain(
		r.mpr,
		row.Monitor,
		row.Target,
		row.User,
		row.AlertContacts,
		row.MaintenanceWindows,
	)
	if err != nil {
		r.log.Error("failed to map monitor row to domain", "monitor_id", row.Monitor.ID, "error", err)
		return nil, errors.Join(repo.ErrInternal, err)
	}

	return mon, nil
}

func (r *monitorRepositoryPG) Update(ctx context.Context, mon *monitor.Monitor) error {
	dbStatus, err := r.mpr.ToDBStatusType(mon.CurrentStatus)
	if err != nil {
		r.log.Error("failed to convert monitor status to DB status", "status", mon.CurrentStatus, "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	dbExpectations, err := r.mpr.ToDBExpectations(mon.Expectations)
	if err != nil {
		r.log.Error("failed to convert monitor expectations to DB payload", "error", err)
		return errors.Join(repo.ErrInternal, err)
	}

	params := sqlcgen.UpdateMonitorParams{
		ID:               pgtype.UUID{Bytes: mon.ID, Valid: true},
		TargetID:         pgtype.UUID{Bytes: mon.Target.ID, Valid: true},
		UserLogin:        mon.User.Login,
		Label:            mon.Label,
		IsActive:         mon.IsActive,
		ProbeIntervalSec: mon.ProbeIntervalSec,
		Expectations:     dbExpectations,
		CurrentStatus:    dbStatus,
		LastEvaluatedAt:  pgtype.Timestamptz{Time: mon.LastEvaluatedAt, Valid: true},
	}

	if err := r.queries.UpdateMonitor(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *monitorRepositoryPG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteMonitorByID(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return mapPGXErrorToRepo(err)
	}
	return nil
}

func (r *monitorRepositoryPG) GetAllByUser(ctx context.Context, usr *user.User) ([]*monitor.Monitor, error) {
	rows, err := r.queries.GetAllMonitorsByUser(ctx, usr.Login)
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	var res []*monitor.Monitor
	for _, row := range rows {
		mon, err := mapMonitorRowToDomain(
			r.mpr,
			row.Monitor,
			row.Target,
			row.User,
			row.AlertContacts,
			row.MaintenanceWindows,
		)
		if err != nil {
			r.log.Error("failed to map monitor row to domain", "monitor_id", row.Monitor.ID, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}
		res = append(res, mon)
	}
	return res, nil
}

func (r *monitorRepositoryPG) GetAllByTargetID(ctx context.Context, targetID uuid.UUID) ([]*monitor.Monitor, error) {
	rows, err := r.queries.GetAllMonitorsByTargetID(ctx, pgtype.UUID{Bytes: targetID, Valid: true})
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	var res []*monitor.Monitor
	for _, row := range rows {
		mon, err := mapMonitorRowToDomain(
			r.mpr,
			row.Monitor,
			row.Target,
			row.User,
			row.AlertContacts,
			row.MaintenanceWindows,
		)
		if err != nil {
			r.log.Error("failed to map monitor row to domain", "monitor_id", row.Monitor.ID, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}
		res = append(res, mon)
	}
	return res, nil
}

func (r *monitorRepositoryPG) GetMonitorsToEvaluate(ctx context.Context, targetIDs []uuid.UUID) (map[uuid.UUID][]*monitor.Monitor, error) {
	pgIDs := make([]pgtype.UUID, len(targetIDs))
	for i, id := range targetIDs {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	rows, err := r.queries.GetMonitorsToEvaluate(ctx, pgIDs)
	if err != nil {
		return nil, mapPGXErrorToRepo(err)
	}

	res := make(map[uuid.UUID][]*monitor.Monitor)
	for _, row := range rows {
		mon, err := mapMonitorRowToDomain(
			r.mpr,
			row.Monitor,
			row.Target,
			row.User,
			row.AlertContacts,
			row.MaintenanceWindows,
		)
		if err != nil {
			r.log.Error("failed to map monitor row to domain", "monitor_id", row.Monitor.ID, "error", err)
			return nil, errors.Join(repo.ErrInternal, err)
		}
		targetID := mon.Target.ID
		res[targetID] = append(res[targetID], mon)
	}

	return res, nil
}

func (r *monitorRepositoryPG) BulkUpdateEvaluation(ctx context.Context, monitors []*monitor.Monitor) error {
	if len(monitors) == 0 {
		return nil
	}

	ids := make([]pgtype.UUID, len(monitors))
	statuses := make([]string, len(monitors))
	evalAts := make([]pgtype.Timestamp, len(monitors))

	for i, mon := range monitors {
		dbStatus, err := r.mpr.ToDBStatusType(mon.CurrentStatus)
		if err != nil {
			r.log.Error("failed to convert monitor status to DB status", "monitor_id", mon.ID, "status", mon.CurrentStatus, "error", err)
			return errors.Join(repo.ErrInternal, err)
		}

		ids[i] = pgtype.UUID{Bytes: mon.ID, Valid: true}
		statuses[i] = string(dbStatus)
		evalAts[i] = pgtype.Timestamp{Time: mon.LastEvaluatedAt, Valid: true}
	}

	params := sqlcgen.BulkUpdateEvaluationParams{
		Ids:          ids,
		Statuses:     statuses,
		EvaluatedAts: evalAts,
	}

	if err := r.queries.BulkUpdateEvaluation(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *monitorRepositoryPG) AddAlertContact(ctx context.Context, mon *monitor.Monitor, contact *alert.Contact) error {
	params := sqlcgen.AddAlertContactToMonitorParams{
		MonitorID: pgtype.UUID{Bytes: mon.ID, Valid: true},
		ContactID: pgtype.UUID{Bytes: contact.ID, Valid: true},
	}
	if err := r.queries.AddAlertContactToMonitor(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *monitorRepositoryPG) RemoveAlertContact(ctx context.Context, mon *monitor.Monitor, contact *alert.Contact) error {
	params := sqlcgen.RemoveAlertContactFromMonitorParams{
		MonitorID: pgtype.UUID{Bytes: mon.ID, Valid: true},
		ContactID: pgtype.UUID{Bytes: contact.ID, Valid: true},
	}
	if err := r.queries.RemoveAlertContactFromMonitor(ctx, params); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *monitorRepositoryPG) Enable(ctx context.Context, monitorID uuid.UUID) error {
	if err := r.queries.EnableMonitor(ctx, pgtype.UUID{Bytes: monitorID, Valid: true}); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

func (r *monitorRepositoryPG) Disable(ctx context.Context, monitorID uuid.UUID) error {
	if err := r.queries.DisableMonitor(ctx, pgtype.UUID{Bytes: monitorID, Valid: true}); err != nil {
		return mapPGXErrorToRepo(err)
	}

	return nil
}

// ----------------- Helpers -----------------

// JSON rows are used for json_agg/jsonb_agg mapping because config is a JSON object,
// while sqlc-generated models represent it as []byte for direct table scans.
type monitorAlertContactJSONRow struct {
	ID        uuid.UUID           `json:"id"`
	UserLogin string              `json:"user_login"`
	Type      sqlcgen.ContactType `json:"type"`
	Label     string              `json:"label"`
	Config    json.RawMessage     `json:"config"`
	IsActive  bool                `json:"is_active"`
}

type monitorMaintenanceWindowJSONRow struct {
	ID          uuid.UUID               `json:"id"`
	UserLogin   string                  `json:"user_login"`
	Title       string                  `json:"title"`
	Description *string                 `json:"description"`
	Type        sqlcgen.MaintenanceType `json:"type"`
	Config      json.RawMessage         `json:"config"`
}

func mapMonitorRowToDomain(
	mpr *monitorTypeMapper,
	dbMonitor sqlcgen.Monitor,
	dbTarget sqlcgen.Target,
	dbUser sqlcgen.User,
	alertContactsRaw, maintenanceWindowsRaw interface{},
) (*monitor.Monitor, error) {
	var dbContacts []monitorAlertContactJSONRow
	if err := parseJSONArray(alertContactsRaw, &dbContacts); err != nil {
		return nil, err
	}
	var mappedContacts []alert.Contact
	for _, c := range dbContacts {
		domainType, err := mpr.ToDomainContactType(c.Type)
		if err != nil {
			return nil, err
		}

		domainConfig, err := mpr.ToDomainContactConfig(domainType, []byte(c.Config))
		if err != nil {
			return nil, err
		}

		mappedContacts = append(mappedContacts, alert.Contact{
			ID:       c.ID,
			User:     &user.User{Login: c.UserLogin},
			Name:     c.Label,
			Type:     domainType,
			IsActive: c.IsActive,
			Config:   domainConfig,
		})
	}

	var dbWindows []monitorMaintenanceWindowJSONRow
	if err := parseJSONArray(maintenanceWindowsRaw, &dbWindows); err != nil {
		return nil, err
	}
	var mappedWindows []maintenance.MaintenanceWindow
	for _, w := range dbWindows {
		domainType, err := mpr.ToDomainMaintenanceWindowType(w.Type)
		if err != nil {
			return nil, err
		}

		domainConfig, err := mpr.ToDomainMaintenanceWindowConfig(domainType, []byte(w.Config))
		if err != nil {
			return nil, err
		}

		description := ""
		if w.Description != nil {
			description = *w.Description
		}

		mappedWindows = append(mappedWindows, maintenance.MaintenanceWindow{
			ID:          w.ID,
			User:        &user.User{Login: w.UserLogin},
			Title:       w.Title,
			Description: description,
			Type:        domainType,
			Config:      domainConfig,
		})
	}

	domainProtocol, err := mpr.ToDomainTargetProtocol(dbTarget.Protocol)
	if err != nil {
		return nil, err
	}

	expectations, err := mpr.ToDomainExpectations(domainProtocol, dbMonitor.Expectations)
	if err != nil {
		return nil, err
	}

	networkConfig, err := mpr.ToDomainTargetNetworkConfig(domainProtocol, dbTarget.NetworkConfig)
	if err != nil {
		return nil, err
	}

	domainStatus, err := mpr.ToDomainStatusType(dbMonitor.CurrentStatus)
	if err != nil {
		return nil, err
	}

	mon := &monitor.Monitor{
		ID:                 dbMonitor.ID.Bytes,
		Label:              dbMonitor.Label,
		AlertContacts:      mappedContacts,
		MaintenanceWindows: mappedWindows,
		CurrentStatus:      domainStatus,
		LastEvaluatedAt:    dbMonitor.LastEvaluatedAt.Time,
		ProbeIntervalSec:   dbMonitor.ProbeIntervalSec,
		IsActive:           dbMonitor.IsActive,
		CreatedAt:          dbMonitor.CreatedAt.Time,
		Expectations:       expectations,
		Target: &target.Target{
			ID:               dbTarget.ID.Bytes,
			Endpoint:         dbTarget.Endpoint,
			Config:           networkConfig,
			IsActive:         dbTarget.IsActive,
			ProbeIntervalSec: dbTarget.ProbeIntervalSec,
			ConfigHash:       dbTarget.SignatureHash,
		},
		User: &user.User{
			Login:        dbUser.Login,
			PasswordHash: dbUser.PasswordHash,
		},
	}

	return mon, nil
}
