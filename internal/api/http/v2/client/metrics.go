package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/probe"
	metricssvc "WatchTower/internal/service/metrics"

	"github.com/google/uuid"
)

func mapAPIMonitorCheck(value apiv2gen.MonitorCheck) *probe.Summary {
	statusCode := int32(0)
	if value.StatusCode != nil {
		statusCode = int32(*value.StatusCode)
	}
	failureReason := ""
	if value.FailureReason != nil {
		failureReason = *value.FailureReason
	}
	return &probe.Summary{
		MonitorID:      value.MonitorId,
		ProbeTime:      value.CheckedAt,
		MonitorStatus:  monitor.Status(value.Status),
		LatencyMs:      int32(value.LatencyMs),
		StatusCode:     statusCode,
		NetworkFailure: value.NetworkFailure,
		FailureReason:  failureReason,
	}
}

func (c *Client) GetLastSummaries(ctx context.Context, monitorID uuid.UUID, limit int) ([]*probe.Summary, error) {
	apiLimit := apiv2gen.Limit(limit)
	return c.getSummaries(ctx, monitorID, &apiv2gen.ListMonitorChecksParams{Limit: &apiLimit})
}

func (c *Client) GetSummariesForPeriod(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]*probe.Summary, error) {
	apiFrom, apiTo := apiv2gen.From(from), apiv2gen.To(to)
	return c.getSummaries(ctx, monitorID, &apiv2gen.ListMonitorChecksParams{From: &apiFrom, To: &apiTo})
}

func (c *Client) getSummaries(ctx context.Context, monitorID uuid.UUID, params *apiv2gen.ListMonitorChecksParams) ([]*probe.Summary, error) {
	response, err := c.api.ListMonitorChecksWithResponse(ctx, monitorID, params, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("list monitor checks: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	result := make([]*probe.Summary, 0, len(*response.JSON200))
	for _, value := range *response.JSON200 {
		result = append(result, mapAPIMonitorCheck(value))
	}
	return result, nil
}

func (c *Client) GetSLA(ctx context.Context, monitorID uuid.UUID, from, to time.Time) (metricssvc.SLAStats, error) {
	apiFrom, apiTo := apiv2gen.From(from), apiv2gen.To(to)
	response, err := c.api.GetMonitorSLAWithResponse(ctx, monitorID, &apiv2gen.GetMonitorSLAParams{From: &apiFrom, To: &apiTo}, c.withAuth)
	if err != nil {
		return metricssvc.SLAStats{}, fmt.Errorf("get monitor SLA: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return metricssvc.SLAStats{}, responseError(response.StatusCode(), response.Body)
	}
	return metricssvc.SLAStats{
		MonitorID:        response.JSON200.MonitorId,
		UptimePercent:    float64(response.JSON200.UptimePercentage),
		TotalDowntimeSec: response.JSON200.DowntimeDurationSec,
		PeriodStart:      response.JSON200.From,
		PeriodEnd:        response.JSON200.To,
	}, nil
}

func (c *Client) GetStatusHistory(ctx context.Context, monitorID uuid.UUID, from, to time.Time) ([]metricssvc.StatusEvent, error) {
	apiFrom, apiTo := apiv2gen.From(from), apiv2gen.To(to)
	response, err := c.api.ListMonitorStatusHistoryWithResponse(ctx, monitorID, &apiv2gen.ListMonitorStatusHistoryParams{From: &apiFrom, To: &apiTo}, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("list monitor status history: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	result := make([]metricssvc.StatusEvent, 0, len(*response.JSON200))
	for _, value := range *response.JSON200 {
		event := metricssvc.StatusEvent{Status: monitor.Status(value.Status), StartTime: value.StartedAt}
		if value.EndedAt != nil {
			event.EndTime = *value.EndedAt
		}
		result = append(result, event)
	}
	return result, nil
}
