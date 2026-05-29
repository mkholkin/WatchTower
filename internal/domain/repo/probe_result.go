package repo

import (
	"context"

	"github.com/google/uuid"

	"WatchTower/internal/domain/entity/probe"
)

type ProbeResultRepository interface {
	Create(probeResult *probe.Result) error
	// FetchUnprocessed retrieves up to `limit` probe results with processing status "new",
	// ordered by probe_time ASC (oldest first).
	FetchUnprocessed(ctx context.Context, limit int) ([]*probe.Result, error)
	// BulkUpdateStatus sets the processing status for the given probe result IDs in a single batch.
	BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status probe.ProcessingStatus) error
}
