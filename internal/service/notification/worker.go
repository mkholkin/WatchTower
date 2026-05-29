package notification

import (
	alert "WatchTower/internal/domain/entity/alert_contact"
	"context"
	"log/slog"
	"sync"
)

type Task struct {
	contact *alert.Contact
	message string
}

type WorkerPool struct {
	registry    NotificationProviderRegistry
	workerCount int
	taskQueue   <-chan Task
	log         *slog.Logger
}

func NewWorkerPool(
	registry NotificationProviderRegistry,
	workerCount int,
	taskQueue <-chan Task,
	log *slog.Logger,
) *WorkerPool {
	return &WorkerPool{
		registry:    registry,
		workerCount: workerCount,
		taskQueue:   taskQueue,
		log:         log.With("service", "notification", "component", "worker_pool"),
	}
}

func (wp *WorkerPool) StartLoop(ctx context.Context) {
	wp.log.Info("starting notification worker pool", "worker_count", wp.workerCount)
	var wg sync.WaitGroup

	for i := 0; i < wp.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wp.loop(ctx)
		}()
	}

	wg.Wait()
	wp.log.Info("notification worker pool stopped")
}

func (wp *WorkerPool) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}
			wp.sendNotification(task)
		}
	}
}

func (wp *WorkerPool) sendNotification(task Task) {
	notificationProvider, err := wp.registry.Get(task.contact.Type)
	if err != nil {
		slog.Error("failed to get notification provider", "contact_type", task.contact.Type, "error", err)
		return
	}

	if err := notificationProvider.SendNotification(task.contact, task.message); err != nil {
		slog.Error("failed to send notification", "contact_type", task.contact.Type, "error", err)
		return
	}
}
