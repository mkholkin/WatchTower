package mongodb

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
)

type monitorRepositoryMG struct {
	db              *mongo.Database
	monitorsColl    *mongo.Collection
	targetsColl     *mongo.Collection
	contactsColl    *mongo.Collection
	mwColl          *mongo.Collection
	usersColl       *mongo.Collection
	log             *slog.Logger
}

func NewMonitorRepository(db *mongo.Database, logger *slog.Logger) repo.MonitorRepository {
	return &monitorRepositoryMG{
		db:           db,
		monitorsColl: db.Collection(collMonitors),
		targetsColl:  db.Collection(collTargets),
		contactsColl: db.Collection(collAlertContacts),
		mwColl:       db.Collection(collMaintenanceWindows),
		usersColl:    db.Collection(collUsers),
		log:          logger.With("component", "mongodb_monitor_repository"),
	}
}

func (r *monitorRepositoryMG) Create(ctx context.Context, mon *monitor.Monitor) error {
	expJSON, err := json.Marshal(mon.Expectations)
	if err != nil {
		return err
	}

	contactIDs := make([]string, len(mon.AlertContacts))
	for i, c := range mon.AlertContacts {
		contactIDs[i] = c.ID.String()
	}
	mwIDs := make([]string, len(mon.MaintenanceWindows))
	for i, mw := range mon.MaintenanceWindows {
		mwIDs[i] = mw.ID.String()
	}

	_, err = r.monitorsColl.InsertOne(ctx, bson.M{
		"_id":                  mon.ID.String(),
		"label":                mon.Label,
		"target_id":            mon.Target.ID.String(),
		"user_login":           mon.User.Login,
		"is_active":            mon.IsActive,
		"probe_interval_sec":   mon.ProbeIntervalSec,
		"expectations":         string(expJSON),
		"expectations_protocol": string(mon.Expectations.Protocol()),
		"current_status":       string(mon.CurrentStatus),
		"last_evaluated_at":    mon.LastEvaluatedAt,
		"created_at":           mon.CreatedAt,
		"alert_contact_ids":    contactIDs,
		"maintenance_window_ids": mwIDs,
	})
	return mapMongoError(err)
}

func (r *monitorRepositoryMG) GetByID(ctx context.Context, id uuid.UUID) (*monitor.Monitor, error) {
	return r.resolveOne(ctx, bson.M{"_id": id.String()})
}

func (r *monitorRepositoryMG) Update(ctx context.Context, mon *monitor.Monitor) error {
	expJSON, err := json.Marshal(mon.Expectations)
	if err != nil {
		return err
	}

	contactIDs := make([]string, len(mon.AlertContacts))
	for i, c := range mon.AlertContacts {
		contactIDs[i] = c.ID.String()
	}
	mwIDs := make([]string, len(mon.MaintenanceWindows))
	for i, mw := range mon.MaintenanceWindows {
		mwIDs[i] = mw.ID.String()
	}

	res, err := r.monitorsColl.UpdateOne(ctx, bson.M{"_id": mon.ID.String()}, bson.M{"$set": bson.M{
		"label":                mon.Label,
		"target_id":            mon.Target.ID.String(),
		"user_login":           mon.User.Login,
		"is_active":            mon.IsActive,
		"probe_interval_sec":   mon.ProbeIntervalSec,
		"expectations":         string(expJSON),
		"expectations_protocol": string(mon.Expectations.Protocol()),
		"current_status":       string(mon.CurrentStatus),
		"last_evaluated_at":    mon.LastEvaluatedAt,
		"alert_contact_ids":    contactIDs,
		"maintenance_window_ids": mwIDs,
	}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *monitorRepositoryMG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	res, err := r.monitorsColl.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return mapMongoError(err)
	}
	if res.DeletedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *monitorRepositoryMG) GetAllByUser(ctx context.Context, usr *user.User) ([]*monitor.Monitor, error) {
	return r.resolveMany(ctx, bson.M{"user_login": usr.Login})
}

func (r *monitorRepositoryMG) GetAllByTargetID(ctx context.Context, targetID uuid.UUID) ([]*monitor.Monitor, error) {
	return r.resolveMany(ctx, bson.M{"target_id": targetID.String()})
}

func (r *monitorRepositoryMG) GetMonitorsToEvaluate(ctx context.Context, targetIDs []uuid.UUID) (map[uuid.UUID][]*monitor.Monitor, error) {
	if len(targetIDs) == 0 {
		return map[uuid.UUID][]*monitor.Monitor{}, nil
	}

	strIDs := make([]string, len(targetIDs))
	for i, id := range targetIDs {
		strIDs[i] = id.String()
	}

	docs, err := r.findMonitorDocs(ctx, bson.M{
		"target_id": bson.M{"$in": strIDs},
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}

	// Filter by evaluation time in Go
	now := time.Now()
	var eligible []monitorDoc
	for _, d := range docs {
		if d.LastEvaluatedAt.IsZero() || d.LastEvaluatedAt.Add(time.Duration(d.ProbeIntervalSec)*time.Second).Before(now) {
			eligible = append(eligible, d)
		}
	}

	monitors, err := r.populateMonitors(ctx, eligible)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]*monitor.Monitor)
	for _, m := range monitors {
		targetID := m.Target.ID
		result[targetID] = append(result[targetID], m)
	}
	return result, nil
}

func (r *monitorRepositoryMG) BulkUpdateEvaluation(ctx context.Context, monitors []*monitor.Monitor) error {
	if len(monitors) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, len(monitors))
	for i, mon := range monitors {
		models[i] = mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": mon.ID.String()}).
			SetUpdate(bson.M{"$set": bson.M{
				"current_status":    string(mon.CurrentStatus),
				"last_evaluated_at": mon.LastEvaluatedAt,
			}})
	}

	_, err := r.monitorsColl.BulkWrite(ctx, models)
	return mapMongoError(err)
}

func (r *monitorRepositoryMG) AddAlertContact(ctx context.Context, mon *monitor.Monitor, contact *alert.Contact) error {
	res, err := r.monitorsColl.UpdateOne(ctx, bson.M{"_id": mon.ID.String()}, bson.M{
		"$addToSet": bson.M{"alert_contact_ids": contact.ID.String()},
	})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *monitorRepositoryMG) RemoveAlertContact(ctx context.Context, mon *monitor.Monitor, contact *alert.Contact) error {
	res, err := r.monitorsColl.UpdateOne(ctx, bson.M{"_id": mon.ID.String()}, bson.M{
		"$pull": bson.M{"alert_contact_ids": contact.ID.String()},
	})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *monitorRepositoryMG) Enable(ctx context.Context, monitorID uuid.UUID) error {
	res, err := r.monitorsColl.UpdateOne(ctx, bson.M{"_id": monitorID.String()}, bson.M{
		"$set": bson.M{"is_active": true},
	})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *monitorRepositoryMG) Disable(ctx context.Context, monitorID uuid.UUID) error {
	res, err := r.monitorsColl.UpdateOne(ctx, bson.M{"_id": monitorID.String()}, bson.M{
		"$set": bson.M{"is_active": false, "current_status": string(monitor.StatusUnknown)},
	})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// --- document types ---

type monitorDoc struct {
	ID                  string    `bson:"_id"`
	Label               string    `bson:"label"`
	TargetID            string    `bson:"target_id"`
	UserLogin           string    `bson:"user_login"`
	IsActive            bool      `bson:"is_active"`
	ProbeIntervalSec    int32     `bson:"probe_interval_sec"`
	Expectations        string    `bson:"expectations"`
	ExpectationsProtocol string   `bson:"expectations_protocol"`
	CurrentStatus       string    `bson:"current_status"`
	LastEvaluatedAt     time.Time `bson:"last_evaluated_at"`
	CreatedAt           time.Time `bson:"created_at"`
	AlertContactIDs     []string  `bson:"alert_contact_ids"`
	MaintenanceWindowIDs []string `bson:"maintenance_window_ids"`
}

// --- resolution helpers ---

func (r *monitorRepositoryMG) findMonitorDocs(ctx context.Context, filter bson.M) ([]monitorDoc, error) {
	cursor, err := r.monitorsColl.Find(ctx, filter)
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []monitorDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}
	return docs, nil
}

func (r *monitorRepositoryMG) resolveOne(ctx context.Context, filter bson.M) (*monitor.Monitor, error) {
	var doc monitorDoc
	if err := r.monitorsColl.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, mapMongoError(err)
	}
	monitors, err := r.populateMonitors(ctx, []monitorDoc{doc})
	if err != nil {
		return nil, err
	}
	if len(monitors) == 0 {
		return nil, repo.ErrNotFound
	}
	return monitors[0], nil
}

func (r *monitorRepositoryMG) resolveMany(ctx context.Context, filter bson.M) ([]*monitor.Monitor, error) {
	docs, err := r.findMonitorDocs(ctx, filter)
	if err != nil {
		return nil, err
	}
	return r.populateMonitors(ctx, docs)
}

func (r *monitorRepositoryMG) populateMonitors(ctx context.Context, docs []monitorDoc) ([]*monitor.Monitor, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	// Collect all related IDs
	targetIDSet := make(map[string]bool)
	contactIDSet := make(map[string]bool)

	monitorIDs := make([]string, 0, len(docs))

	for _, d := range docs {
		targetIDSet[d.TargetID] = true
		monitorIDs = append(monitorIDs, d.ID)
		for _, cid := range d.AlertContactIDs {
			contactIDSet[cid] = true
		}
	}

	// Batch-lookup targets
	targetMap, err := r.loadTargets(ctx, targetIDSet)
	if err != nil {
		return nil, err
	}

	// Batch-lookup alert contacts
	contactMap, err := r.loadContacts(ctx, contactIDSet)
	if err != nil {
		return nil, err
	}

	// Batch-lookup maintenance windows
	mwMap, err := r.loadMaintenanceWindowsForMonitors(ctx, monitorIDs)
	if err != nil {
		return nil, err
	}

	result := make([]*monitor.Monitor, 0, len(docs))
	for _, d := range docs {
		id, err := uuid.Parse(d.ID)
		if err != nil {
			return nil, err
		}
		tgt := targetMap[d.TargetID]
		if tgt == nil {
			r.log.Warn("target not found for monitor", "monitor_id", d.ID, "target_id", d.TargetID)
			continue
		}

		expectations, err := unmarshalExpectations(target.Protocol(d.ExpectationsProtocol), []byte(d.Expectations))
		if err != nil {
			return nil, err
		}

		contacts := make([]alert.Contact, 0, len(d.AlertContactIDs))
		for _, cid := range d.AlertContactIDs {
			if c, ok := contactMap[cid]; ok {
				contacts = append(contacts, *c)
			}
		}

		windows := mwMap[d.ID]
		if windows == nil {
			windows = make([]maintenance.MaintenanceWindow, 0)
		}

		result = append(result, &monitor.Monitor{
			ID:                 id,
			Label:              d.Label,
			Target:             tgt,
			User:               &user.User{Login: d.UserLogin},
			AlertContacts:      contacts,
			MaintenanceWindows: windows,
			CurrentStatus:      monitor.Status(d.CurrentStatus),
			LastEvaluatedAt:    d.LastEvaluatedAt,
			ProbeIntervalSec:   d.ProbeIntervalSec,
			IsActive:           d.IsActive,
			CreatedAt:          d.CreatedAt,
			Expectations:       expectations,
		})
	}

	return result, nil
}

func (r *monitorRepositoryMG) loadTargets(ctx context.Context, idSet map[string]bool) (map[string]*target.Target, error) {
	if len(idSet) == 0 {
		return map[string]*target.Target{}, nil
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	cursor, err := r.targetsColl.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var tDocs []targetDoc
	if err := cursor.All(ctx, &tDocs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make(map[string]*target.Target, len(tDocs))
	for _, d := range tDocs {
		tgt, err := docToTarget(d)
		if err != nil {
			return nil, err
		}
		result[d.ID] = tgt
	}
	return result, nil
}

func (r *monitorRepositoryMG) loadContacts(ctx context.Context, idSet map[string]bool) (map[string]*alert.Contact, error) {
	if len(idSet) == 0 {
		return map[string]*alert.Contact{}, nil
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	cursor, err := r.contactsColl.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var cDocs []alertContactDoc
	if err := cursor.All(ctx, &cDocs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make(map[string]*alert.Contact, len(cDocs))
	for _, d := range cDocs {
		c, err := docToAlertContact(d)
		if err != nil {
			return nil, err
		}
		result[d.ID] = c
	}
	return result, nil
}

func (r *monitorRepositoryMG) loadMaintenanceWindowsForMonitors(ctx context.Context, monitorIDs []string) (map[string][]maintenance.MaintenanceWindow, error) {
	if len(monitorIDs) == 0 {
		return map[string][]maintenance.MaintenanceWindow{}, nil
	}

	cursor, err := r.mwColl.Find(ctx, bson.M{"monitor_ids": bson.M{"$in": monitorIDs}})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var mwDocs []maintenanceWindowDoc
	if err := cursor.All(ctx, &mwDocs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make(map[string][]maintenance.MaintenanceWindow)
	for _, d := range mwDocs {
		mw, err := docToMaintenanceWindow(d)
		if err != nil {
			return nil, err
		}
		for _, mID := range d.MonitorIDs {
			result[mID] = append(result[mID], *mw)
		}
	}
	return result, nil
}
