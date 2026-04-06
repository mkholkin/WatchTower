package repo

import (
	"context"

	"github.com/google/uuid"

	"WatchTower/internal/domain/entity/probe"
)

type ProbeSummaryRepository interface {
	// Create persists a single ProbeSummary and returns its ID.
	Create(ctx context.Context, summary *probe.Summary) (uuid.UUID, error)
	// BulkCreate persists multiple ProbeSummary records in a single batch.
	BulkCreate(ctx context.Context, summaries []probe.Summary) error
}
