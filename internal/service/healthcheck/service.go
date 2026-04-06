package service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	// DefaultWorkerCount is the default number of concurrent probe workers.
	DefaultWorkerCount = 100
	// DefaultTaskQueueSize is the buffer size of the channel between Scheduler and WorkerPool.
	DefaultTaskQueueSize = 1000
)

// HealthCheckerConfig holds the configuration for a HealthChecker instance.
type HealthCheckerConfig struct {
	WorkerCount   int
	TaskQueueSize int
}

// HealthChecker is a high-throughput executor that periodically probes active targets
// and persists raw ProbeResults. It does NOT evaluate status (UP/DOWN).
//
// Internally it consists of two async stages connected by a Go channel:
//
//	Scheduler ──(taskQueue)──▶ WorkerPool
type HealthChecker interface {
	Run(ctx context.Context) error
}

type healthChecker struct {
	scheduler  *Scheduler
	workerPool *WorkerPool
	taskQueue  chan target.Target
	log        *slog.Logger
}

// NewHealthChecker wires together the Scheduler, WorkerPool and shared task queue.
func NewHealthChecker(
	targetRepo repo.TargetRepository,
	probeResultRepo repo.ProbeResultRepository,
	subscriber message.Subscriber,
	proberRegistry *ProberRegistry,
	cfg HealthCheckerConfig,
	logger *slog.Logger,
) HealthChecker {
	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = DefaultWorkerCount
	}
	queueSize := cfg.TaskQueueSize
	if queueSize <= 0 {
		queueSize = DefaultTaskQueueSize
	}

	hcLog := logger.With("service", "healthcheck")
	taskQueue := make(chan target.Target, queueSize)

	return &healthChecker{
		taskQueue:  taskQueue,
		scheduler:  NewScheduler(targetRepo, subscriber, taskQueue, nil, hcLog),
		workerPool: NewWorkerPool(proberRegistry, probeResultRepo, taskQueue, workerCount, hcLog),
		log:        hcLog,
	}
}

// Run starts the health-check pipeline. It launches the worker pool in a background
// goroutine and runs the scheduler in the foreground. Blocks until ctx is cancelled.
func (hc *healthChecker) Run(ctx context.Context) error {
	// Workers run in the background; they will drain when the task queue is closed.
	go hc.workerPool.Run(ctx)

	hc.log.Info("starting scheduler")

	// Scheduler blocks until ctx is done.
	return hc.scheduler.Run(ctx)
}
