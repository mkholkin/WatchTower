package probe

import (
	"WatchTower/internal/domain/entity/target"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ProcessingStatus string

const (
	ProcessingStatusProcessed ProcessingStatus = "processed"
	ProcessingStatusNew       ProcessingStatus = "new"
	ProcessingStatusCanceled  ProcessingStatus = "canceled"
)

type Result struct {
	ID               uuid.UUID
	LatencyMs        int32
	Meta             string
	NetworkFailure   bool
	StatusCode       sql.NullInt32
	Target           *target.Target
	ProbeTime        time.Time
	ProcessingStatus ProcessingStatus
}

