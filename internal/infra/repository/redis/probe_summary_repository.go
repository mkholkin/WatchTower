package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/repo"
)

const defaultProbeSummaryQueueSize = 100
const defaultProbeSummaryQueueTTL = 30 * time.Minute

type probeSummaryRepositoryCache struct {
	client   *goredis.Client
	fallback repo.ProbeSummaryRepository
	queueMax int
	queueTTL time.Duration
	log      *slog.Logger
}

type ProbeSummaryRepositoryOption func(*probeSummaryRepositoryCache)

func WithQueueMax(queueMax int) ProbeSummaryRepositoryOption {
	return func(r *probeSummaryRepositoryCache) {
		if queueMax > 0 {
			r.queueMax = queueMax
		}
	}
}

func WithQueueTTL(queueTTL time.Duration) ProbeSummaryRepositoryOption {
	return func(r *probeSummaryRepositoryCache) {
		if queueTTL > 0 {
			r.queueTTL = queueTTL
		}
	}
}

func NewProbeSummaryRepository(
	client *goredis.Client,
	fallback repo.ProbeSummaryRepository,
	logger *slog.Logger,
	opts ...ProbeSummaryRepositoryOption,
) repo.ProbeSummaryRepository {
	r := &probeSummaryRepositoryCache{
		client:   client,
		fallback: fallback,
		queueMax: defaultProbeSummaryQueueSize,
		queueTTL: defaultProbeSummaryQueueTTL,
		log:      logger.With("component", "probe_summary_repository_cache"),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}

	return r
}

func (r *probeSummaryRepositoryCache) GetMonitorLatestSummaries(
	ctx context.Context,
	monitorID uuid.UUID,
	limit int,
) ([]*probe.Summary, error) {
	if limit <= 0 {
		return []*probe.Summary{}, nil
	}

	if limit > r.queueMax {
		summaries, err := r.fallback.GetMonitorLatestSummaries(ctx, monitorID, limit)
		if err != nil {
			return nil, err
		}
		return summaries, nil
	}

	key := queueKey(monitorID)
	cached, err := r.client.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err == nil && len(cached) > 0 {
		if ttlErr := r.touchQueueTTL(ctx, key); ttlErr != nil {
			r.log.Warn("failed to extend queue ttl on read", "monitor_id", monitorID, "error", ttlErr)
		}
		return decodeSummaries(cached)
	}
	if err != nil {
		r.log.Warn("failed to read probe summaries from redis, using fallback", "monitor_id", monitorID, "error", err)
	}

	summaries, err := r.fallback.GetMonitorLatestSummaries(ctx, monitorID, limit)
	if err != nil {
		return nil, err
	}

	if cacheErr := r.enqueueBatch(ctx, monitorID, summaries); cacheErr != nil {
		r.log.Warn("failed to cache probe summaries in redis", "monitor_id", monitorID, "error", cacheErr)
	} else {
		// TTL is controlled by read traffic only.
		if ttlErr := r.touchQueueTTL(ctx, key); ttlErr != nil {
			r.log.Warn("failed to set queue ttl after cache warmup", "monitor_id", monitorID, "error", ttlErr)
		}
	}

	return summaries, nil
}

func (r *probeSummaryRepositoryCache) GetMonitorSummariesForPeriod(
	ctx context.Context,
	monitorID uuid.UUID,
	from, to time.Time,
) ([]*probe.Summary, error) {
	return r.fallback.GetMonitorSummariesForPeriod(ctx, monitorID, from, to)
}

func (r *probeSummaryRepositoryCache) Create(ctx context.Context, summary *probe.Summary) error {
	if summary == nil {
		r.log.Error("attempted to create nil probe summary, skipping")
		return nil
	}

	updated, err := r.enqueueIfQueueAlive(ctx, summary)
	if err != nil {
		r.log.Error("failed to enqueue probe summary into redis", "monitor_id", summary.MonitorID, "error", err)
		return nil
	}
	if !updated {
		r.log.Debug("skip cache update on write: queue is absent or expired", "monitor_id", summary.MonitorID)
	}

	return nil
}

func (r *probeSummaryRepositoryCache) BulkCreate(ctx context.Context, summaries []*probe.Summary) error {
	for i := range summaries {
		if summaries[i] == nil {
			r.log.Error("attempted to create nil probe summary in bulk operation, skipping", "index", i)
			continue
		}

		updated, err := r.enqueueIfQueueAlive(ctx, summaries[i])
		if err != nil {
			r.log.Error("failed to enqueue probe summary into redis", "monitor_id", summaries[i].MonitorID, "error", err)
			continue
		}
		if !updated {
			r.log.Debug("skip cache update on write: queue is absent or expired", "monitor_id", summaries[i].MonitorID)
		}
	}

	return nil
}

func (r *probeSummaryRepositoryCache) enqueueIfQueueAlive(
	ctx context.Context,
	summary *probe.Summary,
) (bool, error) {
	if summary == nil {
		return false, nil
	}

	key := queueKey(summary.MonitorID)
	alive, err := r.isQueueAlive(ctx, key)
	if err != nil {
		return false, err
	}
	if !alive {
		return false, nil
	}

	if err := r.enqueue(ctx, summary); err != nil {
		return false, err
	}

	return true, nil
}

func (r *probeSummaryRepositoryCache) isQueueAlive(ctx context.Context, key string) (bool, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// Redis semantics:
	//  -2 => key does not exist
	//  -1 => key exists but has no expire
	//  >0 => key exists and has remaining TTL
	return ttl > 0, nil
}

func (r *probeSummaryRepositoryCache) enqueueBatch(
	ctx context.Context,
	monitorID uuid.UUID,
	summaries []*probe.Summary,
) error {
	for i := len(summaries) - 1; i >= 0; i-- {
		if summaries[i].MonitorID != monitorID {
			continue
		}
		if err := r.enqueue(ctx, summaries[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *probeSummaryRepositoryCache) enqueue(ctx context.Context, summary *probe.Summary) error {
	payload, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	key := queueKey(summary.MonitorID)
	pipe := r.client.Pipeline()
	pipe.LPush(ctx, key, payload)
	pipe.LTrim(ctx, key, 0, int64(r.queueMax-1))

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *probeSummaryRepositoryCache) touchQueueTTL(ctx context.Context, key string) error {
	if r.queueTTL <= 0 {
		return nil
	}

	if err := r.client.Expire(ctx, key, r.queueTTL).Err(); err != nil {
		return err
	}

	return nil
}

func decodeSummaries(raw []string) ([]*probe.Summary, error) {
	result := make([]*probe.Summary, len(raw))
	for i := range raw {
		if err := json.Unmarshal([]byte(raw[i]), &result[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func queueKey(monitorID uuid.UUID) string {
	return fmt.Sprintf("probe_summary:monitor:%s", monitorID.String())
}
