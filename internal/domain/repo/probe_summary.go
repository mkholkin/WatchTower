package repo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"WatchTower/internal/domain/entity/probe"
)

type ProbeSummaryRepository interface {
	// GetMonitorLatestSummaries returns latest summaries for a monitor, limited by `limit`.
	GetMonitorLatestSummaries(ctx context.Context, monitorID uuid.UUID, limit int) ([]*probe.Summary, error)
	// GetMonitorSummariesForPeriod returns all summaries for monitor in [from, to].
	GetMonitorSummariesForPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*probe.Summary, error)
	// Create persists a single ProbeSummary and returns its ID.
	Create(ctx context.Context, summary *probe.Summary) error
	// BulkCreate persists multiple ProbeSummary records in a single batch.
	BulkCreate(ctx context.Context, summaries []*probe.Summary) error
}
