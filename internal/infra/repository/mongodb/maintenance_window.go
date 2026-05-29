package mongodb

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/user"
	"WatchTower/internal/domain/repo"
)

type maintenanceWindowRepositoryMG struct {
	coll *mongo.Collection
	log  *slog.Logger
}

func NewMaintenanceWindowRepository(db *mongo.Database, logger *slog.Logger) repo.MaintenanceWindowRepository {
	return &maintenanceWindowRepositoryMG{
		coll: db.Collection(collMaintenanceWindows),
		log:  logger.With("component", "mongodb_maintenance_window_repository"),
	}
}

func (r *maintenanceWindowRepositoryMG) Create(ctx context.Context, mw *maintenance.MaintenanceWindow) error {
	cfgJSON, err := json.Marshal(mw.Config)
	if err != nil {
		return err
	}
	_, err = r.coll.InsertOne(ctx, bson.M{
		"_id":         mw.ID.String(),
		"user_login":  mw.User.Login,
		"title":       mw.Title,
		"description": mw.Description,
		"type":        string(mw.Type),
		"config":      string(cfgJSON),
		"monitor_ids": []string{},
	})
	return mapMongoError(err)
}

func (r *maintenanceWindowRepositoryMG) GetByID(ctx context.Context, id uuid.UUID) (*maintenance.MaintenanceWindow, error) {
	return r.findOne(ctx, bson.M{"_id": id.String()})
}

func (r *maintenanceWindowRepositoryMG) Update(ctx context.Context, mw *maintenance.MaintenanceWindow) error {
	cfgJSON, err := json.Marshal(mw.Config)
	if err != nil {
		return err
	}
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": mw.ID.String()}, bson.M{"$set": bson.M{
		"user_login":  mw.User.Login,
		"title":       mw.Title,
		"description": mw.Description,
		"type":        string(mw.Type),
		"config":      string(cfgJSON),
	}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *maintenanceWindowRepositoryMG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return mapMongoError(err)
	}
	if res.DeletedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *maintenanceWindowRepositoryMG) GetByIDBulk(ctx context.Context, ids []uuid.UUID) ([]maintenance.MaintenanceWindow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}
	return r.findMany(ctx, bson.M{"_id": bson.M{"$in": strIDs}})
}

func (r *maintenanceWindowRepositoryMG) GetByUserLogin(ctx context.Context, userLogin string) ([]maintenance.MaintenanceWindow, error) {
	return r.findMany(ctx, bson.M{"user_login": userLogin})
}

func (r *maintenanceWindowRepositoryMG) LinkMonitor(ctx context.Context, window *maintenance.MaintenanceWindow, monitorID uuid.UUID) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": window.ID.String()}, bson.M{
		"$addToSet": bson.M{"monitor_ids": monitorID.String()},
	})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *maintenanceWindowRepositoryMG) UnlinkMonitor(ctx context.Context, window *maintenance.MaintenanceWindow, monitorID uuid.UUID) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": window.ID.String()}, bson.M{
		"$pull": bson.M{"monitor_ids": monitorID.String()},
	})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *maintenanceWindowRepositoryMG) findOne(ctx context.Context, filter bson.M) (*maintenance.MaintenanceWindow, error) {
	var doc maintenanceWindowDoc
	if err := r.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, mapMongoError(err)
	}
	return docToMaintenanceWindow(doc)
}

func (r *maintenanceWindowRepositoryMG) findMany(ctx context.Context, filter bson.M) ([]maintenance.MaintenanceWindow, error) {
	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer cursor.Close(ctx)

	var docs []maintenanceWindowDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make([]maintenance.MaintenanceWindow, len(docs))
	for i, d := range docs {
		mw, err := docToMaintenanceWindow(d)
		if err != nil {
			return nil, err
		}
		result[i] = *mw
	}
	return result, nil
}

type maintenanceWindowDoc struct {
	ID          string   `bson:"_id"`
	UserLogin   string   `bson:"user_login"`
	Title       string   `bson:"title"`
	Description string   `bson:"description"`
	Type        string   `bson:"type"`
	Config      string   `bson:"config"`
	MonitorIDs  []string `bson:"monitor_ids"`
}

func docToMaintenanceWindow(d maintenanceWindowDoc) (*maintenance.MaintenanceWindow, error) {
	windowType := maintenance.WindowType(d.Type)
	cfg, err := unmarshalMaintenanceWindowConfig(windowType, []byte(d.Config))
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &maintenance.MaintenanceWindow{
		ID:          id,
		User:        &user.User{Login: d.UserLogin},
		Title:       d.Title,
		Description: d.Description,
		Type:        windowType,
		Config:      cfg,
	}, nil
}
