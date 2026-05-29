package mongodb

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
)

type probeResultRepositoryMG struct {
	coll *mongo.Collection
	log  *slog.Logger
}

func NewProbeResultRepository(db *mongo.Database, logger *slog.Logger) repo.ProbeResultRepository {
	return &probeResultRepositoryMG{
		coll: db.Collection(collProbeResults),
		log:  logger.With("component", "mongodb_probe_result_repository"),
	}
}

func (r *probeResultRepositoryMG) Create(pr *probe.Result) error {
	if pr == nil || pr.Target == nil {
		r.log.Error("invalid probe result: nil value")
		return repo.ErrInternal
	}

	doc := bson.M{
		"_id":              pr.ID.String(),
		"target_id":        pr.Target.ID.String(),
		"probe_time":       pr.ProbeTime,
		"latency_ms":       pr.LatencyMs,
		"network_failure":  pr.NetworkFailure,
		"meta":             pr.Meta,
		"processing_status": string(pr.ProcessingStatus),
	}
	if pr.ErrorMessage != nil {
		doc["error_message"] = *pr.ErrorMessage
	}
	if pr.StatusCode.Valid {
		doc["status_code"] = pr.StatusCode.Int32
	}

	if _, err := r.coll.InsertOne(context.Background(), doc); err != nil {
		return mapMongoError(err)
	}
	return nil
}

func (r *probeResultRepositoryMG) FetchUnprocessed(ctx context.Context, limit int) ([]*probe.Result, error) {
	cursor, err := r.coll.Find(ctx, bson.M{
		"processing_status": string(probe.ProcessingStatusNew),
	}, options.Find().SetSort(bson.M{"probe_time": 1}).SetLimit(int64(limit)))
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer cursor.Close(ctx)

	var docs []probeResultDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}

	result := make([]*probe.Result, len(docs))
	for i, d := range docs {
		id, err := uuid.Parse(d.ID)
		if err != nil {
			return nil, err
		}
		tgtID, err := uuid.Parse(d.TargetID)
		if err != nil {
			return nil, err
		}
		pr := &probe.Result{
			ID:               id,
			LatencyMs:        d.LatencyMs,
			Meta:             d.Meta,
			NetworkFailure:   d.NetworkFailure,
			Target:           &target.Target{ID: tgtID},
			ProbeTime:        d.ProbeTime,
			ProcessingStatus: probe.ProcessingStatus(d.ProcessingStatus),
		}
		if d.ErrorMessage != nil {
			pr.ErrorMessage = d.ErrorMessage
		}
		if d.StatusCode != nil {
			pr.StatusCode = sql.NullInt32{Int32: *d.StatusCode, Valid: true}
		}
		result[i] = pr
	}
	return result, nil
}

func (r *probeResultRepositoryMG) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status probe.ProcessingStatus) error {
	if len(ids) == 0 {
		return nil
	}
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}
	_, err := r.coll.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": strIDs}}, bson.M{
		"$set": bson.M{"processing_status": string(status)},
	})
	return mapMongoError(err)
}

type probeResultDoc struct {
	ID               string  `bson:"_id"`
	TargetID         string  `bson:"target_id"`
	ProbeTime        time.Time `bson:"probe_time"`
	LatencyMs        int32   `bson:"latency_ms"`
	StatusCode       *int32  `bson:"status_code"`
	NetworkFailure   bool    `bson:"network_failure"`
	ErrorMessage     *string `bson:"error_message"`
	Meta             []byte  `bson:"meta"`
	ProcessingStatus string  `bson:"processing_status"`
}
