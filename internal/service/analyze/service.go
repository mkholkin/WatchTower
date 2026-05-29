package service

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	"WatchTower/internal/domain/repo"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

const (
	// DefaultFetchLimit is the maximum number of unprocessed probe results to fetch per cycle.
	DefaultFetchLimit = 1000
	// DefaultLoadSheddingThreshold is the number of results to keep for processing;
	// older results beyond this threshold are marked as canceled.
	DefaultLoadSheddingThreshold = 500

	// TopicMonitorStatusChanged is the watermill topic for monitor status change events.
	TopicMonitorStatusChanged = "monitor.status_changed"
)

// MonitorStatusChangedEvent is the payload published when a monitor transitions between statuses.
type MonitorStatusChangedEvent struct {
	MonitorID  uuid.UUID      `json:"monitor_id"`
	OldStatus  monitor.Status `json:"old_status"`
	NewStatus  monitor.Status `json:"new_status"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// ProbeEvaluator performs the core logic of comparing a ProbeResult against a Monitor's
// Expectations and determining the resulting MonitorStatus.
type ProbeEvaluator interface {
	Evaluate(ctx context.Context, probeResult *probe.Result, monitor *monitor.Monitor) (monitor.Status, error)
}

// ProbeAnalyzationServiceConfig holds tunables for ProbeAnalyzationService.
type ProbeAnalyzationServiceConfig struct {
	FetchLimit            int
	LoadSheddingThreshold int
}

// ProbeAnalyzationService is the "brain" of the system.
// It transforms raw ProbeResults into meaningful business states (UP / DOWN / MAINTENANCE),
// persists ProbeSummary records and publishes status-change events via watermill.
type ProbeAnalyzationService struct {
	monitorRepo      repo.MonitorRepository
	probeRepo        repo.ProbeResultRepository
	probeSummaryRepo repo.ProbeSummaryRepository
	evaluator        ProbeEvaluator
	publisher        message.Publisher
	cfg              ProbeAnalyzationServiceConfig
	log              *slog.Logger
}

// NewProbeAnalyzationService creates a new ProbeAnalyzationService.
func NewProbeAnalyzationService(
	monitorRepo repo.MonitorRepository,
	probeRepo repo.ProbeResultRepository,
	probeSummaryRepo repo.ProbeSummaryRepository,
	evaluator ProbeEvaluator,
	publisher message.Publisher,
	cfg ProbeAnalyzationServiceConfig,
	logger *slog.Logger,
) *ProbeAnalyzationService {
	if cfg.FetchLimit <= 0 {
		cfg.FetchLimit = DefaultFetchLimit
	}
	if cfg.LoadSheddingThreshold <= 0 {
		cfg.LoadSheddingThreshold = DefaultLoadSheddingThreshold
	}

	return &ProbeAnalyzationService{
		monitorRepo:      monitorRepo,
		probeRepo:        probeRepo,
		probeSummaryRepo: probeSummaryRepo,
		evaluator:        evaluator,
		publisher:        publisher,
		cfg:              cfg,
		log:              logger.With("service", "probe_analyzation"),
	}
}

// Run executes a single analysis cycle: fetch → load-shed → enrich → evaluate → commit.
// It is designed to be called periodically (e.g. by a ticker or cron-like scheduler).
func (s *ProbeAnalyzationService) Run(ctx context.Context) error {
	// 1. Fetch unprocessed probe results.
	results, err := s.fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch unprocessed results: %w", err)
	}
	if len(results) == 0 {
		s.log.Debug("no unprocessed probe results found")
		return nil
	}

	s.log.Info("fetched unprocessed probe results", "count", len(results))

	// 2. Load Shedding – cancel stale results that exceed the threshold.
	active, canceled := s.loadShed(results)
	if len(canceled) > 0 {
		s.log.Info("load shedding: canceling stale results", "canceled", len(canceled), "kept", len(active))
		if err := s.markCanceled(ctx, canceled); err != nil {
			return fmt.Errorf("mark canceled: %w", err)
		}
	}

	// 3. Enrich – resolve unique target IDs and fetch monitors that are due for evaluation.
	targetIDs := s.extractTargetIDs(active)
	monitors, err := s.enrich(ctx, targetIDs)
	if err != nil {
		return fmt.Errorf("enrich monitors: %w", err)
	}

	s.log.Debug("enriched monitors for evaluation", "target_count", len(targetIDs), "monitor_count", len(monitors))

	// 4. Evaluate each (ProbeResult, Monitor) pair.
	summaries, updatedMonitors, processedIDs, err := s.evaluate(ctx, active, monitors)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}

	// 5. Commit – persist summaries, update probe result statuses and monitor evaluations.
	if err := s.commit(ctx, summaries, updatedMonitors, processedIDs); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	s.log.Info("analysis cycle completed", "summaries", len(summaries), "monitors_updated", len(updatedMonitors))
	return nil
}

// fetch retrieves a batch of unprocessed probe results from the repository.
func (s *ProbeAnalyzationService) fetch(ctx context.Context) ([]probe.Result, error) {
	return s.probeRepo.FetchUnprocessed(ctx, s.cfg.FetchLimit)
}

// loadShed splits results into active (to be processed) and canceled (to be discarded).
// If the total exceeds LoadSheddingThreshold, only the newest results (by position) are kept.
func (s *ProbeAnalyzationService) loadShed(results []probe.Result) (active, canceled []probe.Result) {
	if len(results) <= s.cfg.LoadSheddingThreshold {
		return results, nil
	}

	// Results are sorted by probe_time ASC (oldest first).
	// Keep the tail (newest), cancel the head (oldest) to avoid accumulating lag.
	cutoff := len(results) - s.cfg.LoadSheddingThreshold
	return results[cutoff:], results[:cutoff]
}

// markCanceled sets the processing status of the given results to "canceled".
func (s *ProbeAnalyzationService) markCanceled(ctx context.Context, canceled []probe.Result) error {
	ids := make([]uuid.UUID, len(canceled))
	for i, r := range canceled {
		ids[i] = r.ID
	}
	return s.probeRepo.BulkUpdateStatus(ctx, ids, probe.ProcessingStatusCanceled)
}

// extractTargetIDs returns a deduplicated slice of target IDs from the given probe results.
func (s *ProbeAnalyzationService) extractTargetIDs(results []probe.Result) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(results))
	ids := make([]uuid.UUID, 0, len(results))

	for _, r := range results {
		if r.Target == nil {
			continue
		}
		tid := r.Target.ID
		if _, ok := seen[tid]; !ok {
			seen[tid] = struct{}{}
			ids = append(ids, tid)
		}
	}

	return ids
}

// enrich fetches monitors that are ready for evaluation for the given target IDs.
func (s *ProbeAnalyzationService) enrich(ctx context.Context, targetIDs []uuid.UUID) (map[uuid.UUID]*monitor.Monitor, error) {
	if len(targetIDs) == 0 {
		return nil, nil
	}
	return s.monitorRepo.GetMonitorsToEvaluate(ctx, targetIDs)
}

// evaluate iterates over active probe results, matches them with their monitors,
// runs the evaluator, generates ProbeSummary records, and detects status changes.
// Returns the summaries to persist, monitors whose status changed, and processed result IDs.
func (s *ProbeAnalyzationService) evaluate(
	ctx context.Context,
	results []probe.Result,
	monitors map[uuid.UUID]*monitor.Monitor,
) ([]probe.Summary, []*monitor.Monitor, []uuid.UUID, error) {
	var (
		summaries       []probe.Summary
		updatedMonitors []*monitor.Monitor
		processedIDs    []uuid.UUID
		// Track which monitors we've already recorded as "updated" to avoid duplicates.
		monitorSeen = make(map[uuid.UUID]struct{})
	)

	for i := range results {
		r := &results[i]
		if r.Target == nil {
			s.log.Error("probe result has nil target, skipping", "probe_result_id", r.ID)
			processedIDs = append(processedIDs, r.ID)
			continue
		}

		mon, ok := monitors[r.Target.ID]
		if !ok {
			// No monitor is due for evaluation for this target – still mark probe as processed.
			s.log.Debug("no monitor to evaluate for target", "target_id", r.Target.ID, "probe_result_id", r.ID)
			processedIDs = append(processedIDs, r.ID)
			continue
		}

		// Determine monitor status: maintenance takes precedence.
		var newStatus monitor.Status
		if mon.OnMaintenance() {
			newStatus = monitor.StatusMaintenance
		} else {
			evaluated, err := s.evaluator.Evaluate(ctx, r, mon)
			if err != nil {
				s.log.Error("evaluation failed", "probe_result_id", r.ID, "monitor_id", mon.ID, "error", err)
				processedIDs = append(processedIDs, r.ID)
				continue
			}
			newStatus = evaluated
		}

		// Build ProbeSummary.
		summary := probe.Summary{
			MonitorID:      mon.ID,
			LatencyMs:      r.LatencyMs,
			ProbeTime:      r.ProbeTime,
			MonitorStatus:  newStatus,
			NetworkFailure: r.NetworkFailure,
		}
		if r.StatusCode.Valid {
			summary.StatusCode = r.StatusCode.Int32
		}
		summaries = append(summaries, summary)

		// Detect status transition.
		oldStatus := mon.CurrentStatus
		if newStatus != oldStatus {
			s.log.Info("monitor status changed",
				"monitor_id", mon.ID,
				"old_status", oldStatus,
				"new_status", newStatus,
			)

			if err := s.publishStatusChanged(mon.ID, oldStatus, newStatus); err != nil {
				s.log.Error("failed to publish status change event", "monitor_id", mon.ID, "error", err)
				// Non-fatal: continue processing.
			}
		}

		// Update in-memory monitor state for bulk persistence.
		mon.CurrentStatus = newStatus
		mon.LastEvaluatedAt = time.Now()
		if _, ok := monitorSeen[mon.ID]; !ok {
			monitorSeen[mon.ID] = struct{}{}
			updatedMonitors = append(updatedMonitors, mon)
		}

		processedIDs = append(processedIDs, r.ID)
	}

	return summaries, updatedMonitors, processedIDs, nil
}

// publishStatusChanged sends a MonitorStatusChangedEvent to watermill.
func (s *ProbeAnalyzationService) publishStatusChanged(
	monitorID uuid.UUID,
	oldStatus, newStatus monitor.Status,
) error {
	evt := MonitorStatusChangedEvent{
		MonitorID:  monitorID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		OccurredAt: time.Now(),
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal status changed event: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	return s.publisher.Publish(TopicMonitorStatusChanged, msg)
}

// commit persists all artefacts produced by the evaluation step in a single logical batch.
func (s *ProbeAnalyzationService) commit(
	ctx context.Context,
	summaries []probe.Summary,
	updatedMonitors []*monitor.Monitor,
	processedIDs []uuid.UUID,
) error {
	// Persist probe summaries.
	if len(summaries) > 0 {
		if err := s.probeSummaryRepo.BulkCreate(ctx, summaries); err != nil {
			s.log.Error("failed to bulk-create probe summaries", "error", err)
			return fmt.Errorf("bulk create summaries: %w", err)
		}
	}

	// Mark probe results as processed.
	if len(processedIDs) > 0 {
		if err := s.probeRepo.BulkUpdateStatus(ctx, processedIDs, probe.ProcessingStatusProcessed); err != nil {
			s.log.Error("failed to bulk-update probe result status", "error", err)
			return fmt.Errorf("bulk update probe results: %w", err)
		}
	}

	// Bulk-update monitor evaluations.
	if len(updatedMonitors) > 0 {
		if err := s.monitorRepo.BulkUpdateEvaluation(ctx, updatedMonitors); err != nil {
			s.log.Error("failed to bulk-update monitor evaluations", "error", err)
			return fmt.Errorf("bulk update monitors: %w", err)
		}
	}

	return nil
}
