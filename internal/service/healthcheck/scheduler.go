package service

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/domain/repo"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
)

const MinProbeIntervalSec = 1
const SchedulerTickInterval = MinProbeIntervalSec*time.Second + 10*time.Millisecond

// Topics for watermill events.
const (
	TopicTargetCreated = "target.created"
	TopicTargetUpdated = "target.updated"
	TopicTargetDeleted = "target.deleted"
)

// scheduleEntry holds the in-memory state for a single target.
type scheduleEntry struct {
	Target   target.Target
	NextTick time.Time
}

// Scheduler maintains the in-memory schedule of active targets.
// Every second it checks which targets are due for probing and pushes them into the task queue.
// It synchronises its state through watermill events after the initial load from the repository.
type Scheduler struct {
	targetRepo repo.TargetRepository
	subscriber message.Subscriber
	taskQueue  chan<- target.Target
	clock      clockwork.Clock
	log        *slog.Logger

	mu       sync.RWMutex
	schedule map[uuid.UUID]*scheduleEntry // targetID → entry
}

// NewScheduler creates a new Scheduler.
func NewScheduler(
	targetRepo repo.TargetRepository,
	subscriber message.Subscriber,
	taskQueue chan<- target.Target,
	clock clockwork.Clock,
	logger *slog.Logger,
) *Scheduler {
	if clock == nil {
		clock = clockwork.NewRealClock()
	}

	return &Scheduler{
		targetRepo: targetRepo,
		subscriber: subscriber,
		taskQueue:  taskQueue,
		clock:      clock,
		log:        logger.With("component", "scheduler"),
		schedule:   make(map[uuid.UUID]*scheduleEntry),
	}
}

// Run starts the scheduler. It loads active targets, subscribes to watermill events and
// begins ticking every second. Blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	if err := s.loadActiveTargets(ctx); err != nil {
		return err
	}

	go s.listenEvents(ctx)
	s.tickLoop(ctx)

	return nil
}

// loadActiveTargets fetches all active targets from the repository and populates the schedule.
func (s *Scheduler) loadActiveTargets(ctx context.Context) error {
	targets, err := s.targetRepo.GetAllActive(ctx)
	if err != nil {
		return err
	}

	for _, t := range targets {
		s.addTargetToSchedule(t)
	}

	s.log.Info("loaded active targets", "count", len(targets))
	return nil
}

// addTargetToSchedule adds a target to the in-memory schedule.
func (s *Scheduler) addTargetToSchedule(t target.Target) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now()
	startIn := time.Duration(t.ProbeIntervalSec) * time.Second
	s.schedule[t.ID] = &scheduleEntry{
		Target:   t,
		NextTick: now.Add(startIn),
	}
}

// tickLoop runs every second and dispatches due targets into the task queue.
func (s *Scheduler) tickLoop(ctx context.Context) {
	ticker := s.clock.NewTicker(SchedulerTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.Chan():
			s.dispatchDue(ctx, now)
		}
	}
}

// dispatchDue finds all entries whose NextTick is in the past and sends them into taskQueue.
func (s *Scheduler) dispatchDue(ctx context.Context, now time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, entry := range s.schedule {
		if now.Before(entry.NextTick) {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case s.taskQueue <- entry.Target:
			entry.NextTick = now.Add(time.Duration(entry.Target.ProbeIntervalSec) * time.Second)
		default:
			s.log.Error("task queue full, skipping target", "target_id", entry.Target.ID)
		}
	}
}

// --- Watermill event handling ---

// TargetEvent is the JSON payload expected for target lifecycle events.
type TargetEvent struct {
	ID uuid.UUID `json:"id"`
}

// listenEvents subscribes to target lifecycle topics and updates the in-memory schedule.
func (s *Scheduler) listenEvents(ctx context.Context) {
	topics := []string{TopicTargetCreated, TopicTargetUpdated, TopicTargetDeleted}

	for _, topic := range topics {
		msgCh, err := s.subscriber.Subscribe(ctx, topic)
		if err != nil {
			s.log.Error("failed to subscribe to topic", "topic", topic, "error", err)
			continue
		}

		go s.consumeTopic(ctx, topic, msgCh)
	}
}

// consumeTopic reads messages from a single watermill topic channel and dispatches them.
func (s *Scheduler) consumeTopic(ctx context.Context, topic string, msgCh <-chan *message.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			s.handleEvent(topic, msg)
		}
	}
}

// handleEvent parses a watermill message and updates the schedule accordingly.
func (s *Scheduler) handleEvent(topic string, msg *message.Message) {
	defer msg.Ack()

	var evt TargetEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		s.log.Error("failed to unmarshal event", "topic", topic, "error", err)
		return
	}

	switch topic {
	case TopicTargetCreated:
		s.onTargetCreated(evt)
	case TopicTargetUpdated:
		s.onTargetUpdated(evt)
	case TopicTargetDeleted:
		s.onTargetDeleted(evt)
	}
}

// onTargetCreated adds a new entry to the schedule with a random jitter.
func (s *Scheduler) onTargetCreated(evt TargetEvent) {
	// TODO: context?
	tgt, err := s.targetRepo.GetByID(context.TODO(), evt.ID)
	if err != nil {
		s.log.Error("failed to fetch target", "target_id", evt.ID, "error", err)
		return
	}

	s.addTargetToSchedule(*tgt)
	s.log.Info("target added", "target_id", evt.ID)
}

// onTargetUpdated replaces the schedule entry, keeping the existing NextTick to avoid
// resetting the probe cadence on every config change.
func (s *Scheduler) onTargetUpdated(evt TargetEvent) {
	// TODO: context?
	tgt, err := s.targetRepo.GetByID(context.TODO(), evt.ID)
	if err != nil {
		s.log.Error("failed to fetch target for update", "target_id", evt.ID, "error", err)
		return
	}

	if !tgt.IsActive {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.schedule[evt.ID]
	if !ok {
		s.log.Warn("failed to find updated target in scheduler cache", "target_id", evt.ID)
		return
	}

	nextTick := s.clock.Now().Add(time.Duration(tgt.ProbeIntervalSec) * time.Second)
	if existing.NextTick.Before(nextTick) {
		nextTick = existing.NextTick
	}

	existing.Target = *tgt
	existing.NextTick = nextTick

	s.log.Info("target updated", "target_id", evt.ID)
}

// onTargetDeleted removes the entry from the schedule.
func (s *Scheduler) onTargetDeleted(evt TargetEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.schedule, evt.ID)
	s.log.Info("target deleted", "target_id", evt.ID)
}
