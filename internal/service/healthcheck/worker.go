package healtcheck_service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

const (
	// ProbeTimeout is the hard system timeout applied to every probe call.
	ProbeTimeout = 30 * time.Second
)

// WorkerPool reads targets from the task queue, probes them using the appropriate
// protocol-specific Prober and saves the raw result to the repository.
type WorkerPool struct {
	proberRegistry  ProberRegistry
	probeResultRepo repo.ProbeResultRepository
	taskQueue       <-chan target.Target
	workerCount     int
	log             *slog.Logger
}

// NewWorkerPool creates a new WorkerPool.
func NewWorkerPool(
	proberRegistry ProberRegistry,
	probeResultRepo repo.ProbeResultRepository,
	taskQueue <-chan target.Target,
	workerCount int,
	logger *slog.Logger,
) *WorkerPool {
	if workerCount <= 0 {
		workerCount = DefaultWorkerCount
	}
	return &WorkerPool{
		proberRegistry:  proberRegistry,
		probeResultRepo: probeResultRepo,
		taskQueue:       taskQueue,
		workerCount:     workerCount,
		log:             logger.With("component", "worker_pool"),
	}
}

// Run starts workerCount goroutines that consume from taskQueue.
// Blocks until ctx is cancelled and all workers have drained.
func (wp *WorkerPool) Run(ctx context.Context) {
	wp.log.Info("starting worker pool", "worker_count", wp.workerCount)

	var wg sync.WaitGroup

	for i := 0; i < wp.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wp.loop(ctx)
		}()
	}

	wg.Wait()
	wp.log.Info("worker pool stopped")
}

// loop is the main loop for a single worker goroutine.
func (wp *WorkerPool) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tgt, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			wp.probeAndSave(ctx, tgt)
		}
	}
}

// probeAndSave performs a single probe against the target and persists the raw result.
func (wp *WorkerPool) probeAndSave(ctx context.Context, target target.Target) {
	probeTime := time.Now()
	prober, err := wp.proberRegistry.Get(target.Config.Protocol())
	if err != nil {
		wp.log.Error("no prober for protocol", "protocol", target.Config.Protocol(), "target_id", target.ID, "error", err)
		return
	}

	// Apply a small random jitter to avoid thundering herd when multiple workers receive target at the same time.
	jitter := time.Duration(rand.Int63n(int64(time.Duration(MinProbeIntervalSec) * time.Second / 2)))
	time.Sleep(jitter)

	probeCtx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()

	wp.log.Debug("probing target", "target_id", target.ID, "endpoint", target.Endpoint, "protocol", target.Config.Protocol())

	result, err := prober.Probe(probeCtx, &target)
	if err != nil {
		wp.log.Error("probe failed", "target_id", target.ID, "endpoint", target.Endpoint, "error", err)
		return
	}
	result.ProbeTime = probeTime

	if err := wp.probeResultRepo.Create(result); err != nil {
		wp.log.Error("failed to save probe result", "target_id", target.ID, "error", err)
	}
}
