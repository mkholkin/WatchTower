package metrics

import (
	"WatchTower/internal/domain/entity/monitor"
	"time"

	"github.com/google/uuid"
)

type SLAStats struct {
	MonitorID        uuid.UUID `json:"monitor_id"`
	UptimePercent    float64   `json:"uptime_percent"`
	TotalDowntimeSec int
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
}

type StatusEvent struct {
	Status    monitor.Status `json:"status"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time,omitempty"`
	Reason    string         `json:"reason,omitempty"`
}
