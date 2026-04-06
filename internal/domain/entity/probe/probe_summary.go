package probe

import (
	"WatchTower/internal/domain/entity/monitor"
	"time"

	"github.com/google/uuid"
)

type Summary struct {
	MonitorID      uuid.UUID      `json:"monitor_id"`
	LatencyMs      int32          `json:"latency_ms"`
	ProbeTime      time.Time      `json:"probe_time"`
	MonitorStatus  monitor.Status `json:"monitor_status"`
	StatusCode     int32          `json:"status_code"`
	NetworkFailure bool           `json:"network_failure"`
	FailureReason  string         `json:"failure_reason"`
}

// ProbeSummaryRepository provides persistence for ProbeSummary records.
