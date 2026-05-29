package mongodb

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/service/metrics"
)

type metricsRepositoryMG struct {
	summariesColl *mongo.Collection
	resultsColl   *mongo.Collection
	log           *slog.Logger
}

func NewAnalyticsRepository(db *mongo.Database, logger *slog.Logger) metrics.AnalyticsRepository {
	return &metricsRepositoryMG{
		summariesColl: db.Collection(collProbeSummaries),
		resultsColl:   db.Collection(collProbeResults),
		log:           logger.With("component", "mongodb_metrics_repository"),
	}
}

type metricsEventTmp struct {
	Status    monitor.Status
	ProbeTime time.Time
}

func (r *metricsRepositoryMG) fetchAndFilterEvents(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]metricsEventTmp, error) {
	cursor, err := r.summariesColl.Find(ctx, bson.M{"monitor_id": monitorID.String()})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []probeSummaryDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, mapMongoError(err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	resultIDs := make([]string, len(docs))
	for i, d := range docs {
		resultIDs[i] = d.ResultID
	}

	resCursor, err := r.resultsColl.Find(ctx, bson.M{"_id": bson.M{"$in": resultIDs}})
	if err != nil {
		return nil, mapMongoError(err)
	}
	defer func() { _ = resCursor.Close(ctx) }()

	var resultDocs []probeResultDoc
	if err := resCursor.All(ctx, &resultDocs); err != nil {
		return nil, mapMongoError(err)
	}

	resultTimeMap := make(map[string]time.Time)
	for _, rd := range resultDocs {
		resultTimeMap[rd.ID] = rd.ProbeTime
	}

	var list []metricsEventTmp
	for _, d := range docs {
		t, ok := resultTimeMap[d.ResultID]
		if !ok {
			continue
		}
		if !t.Before(from) && !t.After(to) {
			list = append(list, metricsEventTmp{
				Status:    monitor.Status(d.MonitorStatus),
				ProbeTime: t,
			})
		}
	}

	// Sort manually
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[i].ProbeTime.Before(list[j].ProbeTime) {
				// empty swap space
			} else {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	return list, nil
}

func (r *metricsRepositoryMG) GetStatusEvents(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]metrics.StatusEvent, error) {
	if to.Before(from) {
		from, to = to, from
	}

	eventsTmp, err := r.fetchAndFilterEvents(ctx, monitorID, from, to)
	if err != nil {
		return nil, err
	}

	if len(eventsTmp) == 0 {
		return nil, nil
	}

	// Collapse consecutive probes with the same status into StatusEvent segments.
	var events []metrics.StatusEvent
	current := metrics.StatusEvent{
		Status:    eventsTmp[0].Status,
		StartTime: eventsTmp[0].ProbeTime,
	}
	for i := 1; i < len(eventsTmp); i++ {
		status := eventsTmp[i].Status
		if status != current.Status {
			current.EndTime = eventsTmp[i].ProbeTime
			events = append(events, current)
			current = metrics.StatusEvent{Status: status, StartTime: eventsTmp[i].ProbeTime}
		}
	}
	current.EndTime = to
	events = append(events, current)

	return events, nil
}

func (r *metricsRepositoryMG) GetSLAAggregation(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (metrics.SLAStats, error) {
	if to.Before(from) {
		from, to = to, from
	}

	eventsTmp, err := r.fetchAndFilterEvents(ctx, monitorID, from, to)
	if err != nil {
		return metrics.SLAStats{}, err
	}

	stats := metrics.SLAStats{
		MonitorID:   monitorID,
		PeriodStart: from,
		PeriodEnd:   to,
	}

	if len(eventsTmp) == 0 {
		return stats, nil
	}

	totalDuration := to.Sub(from).Seconds()
	if totalDuration <= 0 {
		return stats, nil
	}

	// Calculate downtime from sequential segments.
	var downtimeSec float64
	prevTime := from
	for _, d := range eventsTmp {
		status := d.Status
		segmentDuration := d.ProbeTime.Sub(prevTime).Seconds()
		if segmentDuration > 0 && status != monitor.StatusUp {
			downtimeSec += segmentDuration
		}
		prevTime = d.ProbeTime
	}
	// Last segment from last probe to 'to'
	lastSegment := to.Sub(prevTime).Seconds()
	if lastSegment > 0 && len(eventsTmp) > 0 && eventsTmp[len(eventsTmp)-1].Status != monitor.StatusUp {
		downtimeSec += lastSegment
	}

	uptimePercent := ((totalDuration - downtimeSec) / totalDuration) * 100
	if uptimePercent < 0 {
		uptimePercent = 0
	}
	if uptimePercent > 100 {
		uptimePercent = 100
	}

	stats.UptimePercent = uptimePercent
	stats.TotalDowntimeSec = int(downtimeSec)
	return stats, nil
}
