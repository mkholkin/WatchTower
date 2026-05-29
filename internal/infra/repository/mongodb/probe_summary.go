package mongodb

import (
	"context"
	"log/slog"
	"time"

	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/repo"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type probeSummaryRepositoryMG struct {
	coll         *mongo.Collection
	resultsColl  *mongo.Collection
	monitorsColl *mongo.Collection
	log          *slog.Logger
}

func NewProbeSummaryRepository(db *mongo.Database, logger *slog.Logger) repo.ProbeSummaryRepository {
	return &probeSummaryRepositoryMG{
		coll:         db.Collection(collProbeSummaries),
		resultsColl:  db.Collection(collProbeResults),
		monitorsColl: db.Collection(collMonitors),
		log:          logger.With("component", "mongodb_probe_summary_repository"),
	}
}

func (r *probeSummaryRepositoryMG) GetMonitorLatestSummaries(ctx context.Context, monitorID uuid.UUID, limit int) ([]*probe.Summary, error) {
	if limit <= 0 {
		return []*probe.Summary{}, nil
	}

	cursor, err := r.coll.Find(ctx, bson.M{"monitor_id": monitorID.String()})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer cursor.Close(ctx)

	summaries, err := r.decodeCursor(ctx, cursor)
	if err != nil {
		return nil, err
	}

	// Sort manually
	for i := 0; i < len(summaries); i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[i].ProbeTime.Before(summaries[j].ProbeTime) {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}
	if len(summaries) > limit {
		summaries = summaries[:limit]
	}
	return summaries, nil
}

func (r *probeSummaryRepositoryMG) GetMonitorSummariesForPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*probe.Summary, error) {
	if to.Before(from) {
		from, to = to, from
	}

	cursor, err := r.coll.Find(ctx, bson.M{"monitor_id": monitorID.String()})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer cursor.Close(ctx)

	summaries, err := r.decodeCursor(ctx, cursor)
	if err != nil {
		return nil, err
	}

	filtered := make([]*probe.Summary, 0)
	for _, s := range summaries {
		if !s.ProbeTime.Before(from) && !s.ProbeTime.After(to) {
			filtered = append(filtered, s)
		}
	}

	// Sort manually
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].ProbeTime.Before(filtered[j].ProbeTime) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	return filtered, nil
}

func (r *probeSummaryRepositoryMG) Create(ctx context.Context, summary *probe.Summary) error {
	doc, err := r.resolveSummaryDoc(ctx, summary)
	if err != nil {
		return err
	}
	_, err = r.coll.InsertOne(ctx, doc)
	return mapMongoError(err)
}

func (r *probeSummaryRepositoryMG) BulkCreate(ctx context.Context, summaries []*probe.Summary) error {
	if len(summaries) == 0 {
		return nil
	}
	docs := make([]bson.M, 0, len(summaries))
	for _, s := range summaries {
		doc, err := r.resolveSummaryDoc(ctx, s)
		if err == nil {
			docs = append(docs, doc)
		} else {
			r.log.Warn("could not resolve probe result for summary", "monitor_id", s.MonitorID, "error", err)
		}
	}
	if len(docs) == 0 {
		return nil
	}
	_, err := r.coll.InsertMany(ctx, docs)
	return mapMongoError(err)
}

func (r *probeSummaryRepositoryMG) resolveSummaryDoc(ctx context.Context, s *probe.Summary) (bson.M, error) {
	// Look up monitor to find target_id
	var monDoc monitorDoc
	err := r.monitorsColl.FindOne(ctx, bson.M{"_id": s.MonitorID.String()}).Decode(&monDoc)
	if err != nil {
		return nil, mapMongoError(err)
	}

	// Look up probe_result by target_id, probe_time
	var resDoc probeResultDoc
	err = r.resultsColl.FindOne(ctx, bson.M{
		"target_id":  monDoc.TargetID,
		"probe_time": s.ProbeTime,
	}).Decode(&resDoc)
	if err != nil {
		return nil, mapMongoError(err)
	}

	return bson.M{
		"result_id":      resDoc.ID,
		"monitor_id":     s.MonitorID.String(),
		"monitor_status": string(s.MonitorStatus),
	}, nil
}

func (r *probeSummaryRepositoryMG) decodeCursor(ctx context.Context, cursor *mongo.Cursor) ([]*probe.Summary, error) {
	var docs []probeSummaryDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}

	if len(docs) == 0 {
		return []*probe.Summary{}, nil
	}

	resultIDs := make([]string, len(docs))
	for i, d := range docs {
		resultIDs[i] = d.ResultID
	}

	resultsCursor, err := r.resultsColl.Find(ctx, bson.M{"_id": bson.M{"$in": resultIDs}})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer resultsCursor.Close(ctx)

	var resultDocs []probeResultDoc
	if err := resultsCursor.All(ctx, &resultDocs); err != nil {
		return nil, mapMongoError(err)
	}

	resultMap := make(map[string]probeResultDoc)
	for _, rd := range resultDocs {
		resultMap[rd.ID] = rd
	}

	result := make([]*probe.Summary, 0, len(docs))
	for _, d := range docs {
		mid, err := uuid.Parse(d.MonitorID)
		if err != nil {
			return nil, err
		}

		rd, ok := resultMap[d.ResultID]
		if !ok {
			r.log.Warn("missing probe result for summary", "result_id", d.ResultID)
			continue
		}

		s := &probe.Summary{
			MonitorID:      mid,
			LatencyMs:      rd.LatencyMs,
			ProbeTime:      rd.ProbeTime,
			MonitorStatus:  monitor.Status(d.MonitorStatus),
			NetworkFailure: rd.NetworkFailure,
		}
		if rd.StatusCode != nil {
			s.StatusCode = *rd.StatusCode
		}
		if rd.ErrorMessage != nil {
			s.FailureReason = *rd.ErrorMessage
		}

		result = append(result, s)
	}
	return result, nil
}

type probeSummaryDoc struct {
	ResultID      string `bson:"result_id"`
	MonitorID     string `bson:"monitor_id"`
	MonitorStatus string `bson:"monitor_status"`
}
