package handler

import (
	"context"
	"fmt"
	"time"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/entity/probe"
	baseservice "WatchTower/internal/service"
)

func mapProbeSummary(summary *probe.Summary) apiv2gen.MonitorCheck {
	result := apiv2gen.MonitorCheck{
		MonitorId:      summary.MonitorID,
		CheckedAt:      summary.ProbeTime,
		Status:         apiv2gen.MonitorStatus(summary.MonitorStatus),
		LatencyMs:      int(summary.LatencyMs),
		NetworkFailure: summary.NetworkFailure,
	}
	if summary.StatusCode > 0 {
		statusCode := int(summary.StatusCode)
		result.StatusCode = &statusCode
	}
	if summary.FailureReason != "" {
		reason := summary.FailureReason
		result.FailureReason = &reason
	}
	return result
}

func (a *APIHandler) ListMonitorChecks(ctx context.Context, request apiv2gen.ListMonitorChecksRequestObject) (apiv2gen.ListMonitorChecksResponseObject, error) {
	limit := 100
	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
	}
	if limit < 1 || limit > 1000 {
		return listMonitorChecksError(fmt.Errorf("%w: limit must be between 1 and 1000", baseservice.ErrInvalidData)), nil
	}

	var (
		summaries []*probe.Summary
		err       error
	)
	if request.Params.From != nil || request.Params.To != nil {
		from, to, periodErr := requestedPeriod(request.Params.From, request.Params.To)
		if periodErr != nil {
			return listMonitorChecksError(periodErr), nil
		}
		summaries, err = a.metricsSvc.GetSummariesForPeriod(ctx, request.MonitorId, from, to)
		if len(summaries) > limit {
			summaries = summaries[:limit]
		}
	} else {
		summaries, err = a.metricsSvc.GetLastSummaries(ctx, request.MonitorId, limit)
	}
	if err != nil {
		return listMonitorChecksError(err), nil
	}

	response := make(apiv2gen.ListMonitorChecks200JSONResponse, 0, len(summaries))
	for _, summary := range summaries {
		response = append(response, mapProbeSummary(summary))
	}
	return response, nil
}

func requestedPeriod(fromValue *apiv2gen.From, toValue *apiv2gen.To) (time.Time, time.Time, error) {
	from := time.Unix(0, 0).UTC()
	to := time.Now().UTC()
	if fromValue != nil {
		from = *fromValue
	}
	if toValue != nil {
		to = *toValue
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: from must not be after to", baseservice.ErrInvalidData)
	}
	return from, to, nil
}

func listMonitorChecksError(err error) apiv2gen.ListMonitorChecksResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 400:
		return apiv2gen.ListMonitorChecks400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}
	case 401:
		return apiv2gen.ListMonitorChecks401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.ListMonitorChecks403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.ListMonitorChecks404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.ListMonitorChecks500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) GetMonitorSLA(ctx context.Context, request apiv2gen.GetMonitorSLARequestObject) (apiv2gen.GetMonitorSLAResponseObject, error) {
	from, to, err := requestedPeriod(request.Params.From, request.Params.To)
	if err != nil {
		return getMonitorSLAError(err), nil
	}
	stats, err := a.metricsSvc.GetSLA(ctx, request.MonitorId, from, to)
	if err != nil {
		return getMonitorSLAError(err), nil
	}
	return apiv2gen.GetMonitorSLA200JSONResponse{
		MonitorId:           stats.MonitorID,
		From:                stats.PeriodStart,
		To:                  stats.PeriodEnd,
		UptimePercentage:    float32(stats.UptimePercent),
		DowntimeDurationSec: stats.TotalDowntimeSec,
	}, nil
}

func getMonitorSLAError(err error) apiv2gen.GetMonitorSLAResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 400:
		return apiv2gen.GetMonitorSLA400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}
	case 401:
		return apiv2gen.GetMonitorSLA401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.GetMonitorSLA403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.GetMonitorSLA404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.GetMonitorSLA500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) ListMonitorStatusHistory(ctx context.Context, request apiv2gen.ListMonitorStatusHistoryRequestObject) (apiv2gen.ListMonitorStatusHistoryResponseObject, error) {
	from, to, err := requestedPeriod(request.Params.From, request.Params.To)
	if err != nil {
		return listMonitorStatusHistoryError(err), nil
	}
	events, err := a.metricsSvc.GetStatusHistory(ctx, request.MonitorId, from, to)
	if err != nil {
		return listMonitorStatusHistoryError(err), nil
	}

	response := make(apiv2gen.ListMonitorStatusHistory200JSONResponse, 0, len(events))
	for _, event := range events {
		item := apiv2gen.MonitorStatusInterval{
			MonitorId: request.MonitorId,
			Status:    apiv2gen.MonitorStatus(event.Status),
			StartedAt: event.StartTime,
		}
		if !event.EndTime.IsZero() {
			endedAt := event.EndTime
			item.EndedAt = &endedAt
		}
		response = append(response, item)
	}
	return response, nil
}

func listMonitorStatusHistoryError(err error) apiv2gen.ListMonitorStatusHistoryResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 400:
		return apiv2gen.ListMonitorStatusHistory400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}
	case 401:
		return apiv2gen.ListMonitorStatusHistory401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.ListMonitorStatusHistory403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.ListMonitorStatusHistory404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.ListMonitorStatusHistory500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}
