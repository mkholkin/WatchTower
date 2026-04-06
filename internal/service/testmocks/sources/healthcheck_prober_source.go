package sources

import (
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/entity/target"
	"context"
)

// Prober mirrors healthcheck.Prober for stable mock generation.
type Prober interface {
	Probe(ctx context.Context, target *target.Target) (probe.Result, error)
}
