package notification

import (
	"WatchTower/internal/domain/repo"
	analyzationsvc "WatchTower/internal/service/analyze"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

const DefaultWorkerCount = 30 // TODO: config
const DefaultTaskQueueSize = 256

type NotificationService interface {
	Run(ctx context.Context) error
}

type Option func(service *notificationService)

func WithWorkerCount(count int) Option {
	return func(service *notificationService) {
		service.workerCount = count
	}
}

func WithTaskQueueSize(size int) Option {
	return func(service *notificationService) {
		service.taskQueue = make(chan Task, size)
	}
}

type notificationService struct {
	registry    NotificationProviderRegistry
	monitorRepo repo.MonitorRepository
	log         *slog.Logger
	taskQueue   chan Task
	workerCount int
	subscriber  message.Subscriber
}

func NewNotificationService(
	registry NotificationProviderRegistry,
	subscriber message.Subscriber,
	monitorRepo repo.MonitorRepository,
	logger *slog.Logger,
	opt ...Option,
) NotificationService {
	service := &notificationService{
		registry:    registry,
		monitorRepo: monitorRepo,
		subscriber:  subscriber,
		log:         logger.With("service", "notification"),
		taskQueue:   make(chan Task, DefaultTaskQueueSize),
		workerCount: DefaultWorkerCount,
	}

	for _, o := range opt {
		o(service)
	}

	return service
}

func (s *notificationService) Run(ctx context.Context) error {
	wp := WorkerPool{
		registry:    s.registry,
		log:         s.log,
		workerCount: s.workerCount,
		taskQueue:   s.taskQueue,
	}

	go wp.StartLoop(ctx)

	msgCh, err := s.subscriber.Subscribe(ctx, analyzationsvc.TopicMonitorStatusChanged)
	if err != nil {
		return fmt.Errorf("subscribe to monitor status topic: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgCh:
			if !ok {
				return nil
			}
			s.handleStatusChangedMessage(ctx, msg)
		}
	}
}

func (s *notificationService) handleStatusChangedMessage(ctx context.Context, msg *message.Message) {
	defer msg.Ack()

	var evt analyzationsvc.MonitorStatusChangedEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		s.log.Error("failed to unmarshal monitor status event", "error", err)
		return
	}

	mon, err := s.monitorRepo.GetByID(ctx, evt.MonitorID)
	if err != nil {
		s.log.Error("failed to load monitor for notification", "monitor_id", evt.MonitorID, "error", err)
		return
	}

	text := fmt.Sprintf(
		"Monitor '%s' status changed: %s -> %s (%s)",
		mon.Label,
		evt.OldStatus,
		evt.NewStatus,
		evt.OccurredAt.Format(time.RFC3339),
	)

	for i := range mon.AlertContacts {
		contact := mon.AlertContacts[i]
		if !contact.IsActive {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case s.taskQueue <- Task{contact: &contact, message: text}:
		default:
			s.log.Warn("notification queue is full, dropping task", "monitor_id", mon.ID, "contact_id", contact.ID)
		}
	}

}
