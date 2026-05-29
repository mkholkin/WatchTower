package mongodb

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
)

type targetRepositoryMG struct {
	coll *mongo.Collection
	log  *slog.Logger
}

func NewTargetRepository(db *mongo.Database, logger *slog.Logger) repo.TargetRepository {
	return &targetRepositoryMG{
		coll: db.Collection(collTargets),
		log:  logger.With("component", "mongodb_target_repository"),
	}
}

func (r *targetRepositoryMG) Create(ctx context.Context, tgt *target.Target) error {
	cfgJSON, err := json.Marshal(tgt.Config)
	if err != nil {
		return err
	}
	_, err = r.coll.InsertOne(ctx, bson.M{
		"_id":               tgt.ID.String(),
		"endpoint":          tgt.Endpoint,
		"protocol":          string(tgt.Config.Protocol()),
		"network_config":    string(cfgJSON),
		"is_active":         tgt.IsActive,
		"probe_interval_sec": tgt.ProbeIntervalSec,
		"config_hash":       tgt.ConfigHash,
	})
	return mapMongoError(err)
}

func (r *targetRepositoryMG) GetByID(ctx context.Context, id uuid.UUID) (*target.Target, error) {
	return r.findOne(ctx, bson.M{"_id": id.String()})
}

func (r *targetRepositoryMG) Update(ctx context.Context, tgt *target.Target) error {
	cfgJSON, err := json.Marshal(tgt.Config)
	if err != nil {
		return err
	}
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": tgt.ID.String()}, bson.M{"$set": bson.M{
		"endpoint":          tgt.Endpoint,
		"protocol":          string(tgt.Config.Protocol()),
		"network_config":    string(cfgJSON),
		"is_active":         tgt.IsActive,
		"probe_interval_sec": tgt.ProbeIntervalSec,
		"config_hash":       tgt.ConfigHash,
	}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *targetRepositoryMG) DeleteByID(ctx context.Context, id uuid.UUID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id.String()})
	if err != nil {
		return mapMongoError(err)
	}
	if res.DeletedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *targetRepositoryMG) UpdateProbeInterval(ctx context.Context, id uuid.UUID, probeIntervalSec int32) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id.String()}, bson.M{"$set": bson.M{
		"probe_interval_sec": probeIntervalSec,
	}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *targetRepositoryMG) GetByHash(ctx context.Context, hash string) (*target.Target, error) {
	return r.findOne(ctx, bson.M{"config_hash": hash})
}

func (r *targetRepositoryMG) GetAllActive(ctx context.Context) ([]target.Target, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer cursor.Close(ctx)

	var docs []targetDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make([]target.Target, len(docs))
	for i, d := range docs {
		tgt, err := docToTarget(d)
		if err != nil {
			return nil, err
		}
		result[i] = *tgt
	}
	return result, nil
}

func (r *targetRepositoryMG) Disable(ctx context.Context, id uuid.UUID) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id.String()}, bson.M{"$set": bson.M{"is_active": false}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *targetRepositoryMG) Enable(ctx context.Context, id uuid.UUID) error {
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id.String()}, bson.M{"$set": bson.M{"is_active": true}})
	if err != nil {
		return mapMongoError(err)
	}
	if res.MatchedCount == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *targetRepositoryMG) findOne(ctx context.Context, filter bson.M) (*target.Target, error) {
	var doc targetDoc
	if err := r.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		return nil, mapMongoError(err)
	}
	return docToTarget(doc)
}

type targetDoc struct {
	ID               string `bson:"_id"`
	Endpoint         string `bson:"endpoint"`
	Protocol         string `bson:"protocol"`
	NetworkConfig    string `bson:"network_config"`
	IsActive         bool   `bson:"is_active"`
	ProbeIntervalSec int32  `bson:"probe_interval_sec"`
	ConfigHash       string `bson:"config_hash"`
}

func docToTarget(d targetDoc) (*target.Target, error) {
	cfg, err := unmarshalNetworkConfig(target.Protocol(d.Protocol), []byte(d.NetworkConfig))
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(d.ID)
	if err != nil {
		return nil, err
	}
	return &target.Target{
		ID:               id,
		Endpoint:         d.Endpoint,
		Config:           cfg,
		IsActive:         d.IsActive,
		ProbeIntervalSec: d.ProbeIntervalSec,
		ConfigHash:       d.ConfigHash,
	}, nil
}
