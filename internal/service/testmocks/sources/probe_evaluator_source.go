package sources

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"context"
)

// ProbeEvaluator mirrors analyze.ProbeEvaluator for stable mock generation.
type ProbeEvaluator interface {
	Evaluate(ctx context.Context, probeResult *probe.Result, mon *monitor.Monitor) (monitor.Status, error)
}


