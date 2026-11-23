package handler

import (
	v1 "WatchTower/internal/api/http/v1"
	apigen "WatchTower/internal/api/http/v1/gen"
	"context"
	"time"

	"WatchTower/internal/domain/entity/probe"

	"github.com/google/uuid"
)

var getMonitorsMonitorIdChecksResponseFactory = v1.ResponseFactory[apigen.GetMonitorsMonitorIdChecksResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdChecksResponseObject {
		return apigen.GetMonitorsMonitorIdChecks400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdChecksResponseObject {
		return apigen.GetMonitorsMonitorIdChecks401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdChecksResponseObject {
		return apigen.GetMonitorsMonitorIdChecks403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdChecksResponseObject {
		return apigen.GetMonitorsMonitorIdChecks404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdChecksResponseObject {
		return apigen.GetMonitorsMonitorIdChecks500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var getMonitorsMonitorIdSlaResponseFactory = v1.ResponseFactory[apigen.GetMonitorsMonitorIdSlaResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdSlaResponseObject {
		return apigen.GetMonitorsMonitorIdSla400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdSlaResponseObject {
		return apigen.GetMonitorsMonitorIdSla401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdSlaResponseObject {
		return apigen.GetMonitorsMonitorIdSla403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdSlaResponseObject {
		return apigen.GetMonitorsMonitorIdSla404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMonitorsMonitorIdSlaResponseObject {
		return apigen.GetMonitorsMonitorIdSla500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var GetMonitorStatusHistoryResponseFactory = v1.ResponseFactory[apigen.GetMonitorStatusHistoryResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.GetMonitorStatusHistoryResponseObject {
		return apigen.GetMonitorStatusHistory400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.GetMonitorStatusHistoryResponseObject {
		return apigen.GetMonitorStatusHistory401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.GetMonitorStatusHistoryResponseObject {
		return apigen.GetMonitorStatusHistory403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.GetMonitorStatusHistoryResponseObject {
		return apigen.GetMonitorStatusHistory404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMonitorStatusHistoryResponseObject {
		return apigen.GetMonitorStatusHistory500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

func (a *ApiHandler) GetMonitorsMonitorIdChecks(ctx context.Context, request apigen.GetMonitorsMonitorIdChecksRequestObject) (apigen.GetMonitorsMonitorIdChecksResponseObject, error) {
	var (
		limit = request.Params.Limit
		from  = request.Params.From
		to    = request.Params.To
	)

	if message := monitorChecksParameterError(limit, from, to); message != "" {
		return getMonitorsMonitorIdChecksResponseFactory[400](errorResponse("INVALID_DATA", message)), nil
	}

	var summaries []*probe.Summary
	var err error

	switch {
	case limit != nil:
		summaries, err = a.metricsSvc.GetLastSummaries(ctx, request.MonitorId, *limit)
	case from != nil && to != nil:
		summaries, err = a.metricsSvc.GetSummariesForPeriod(ctx, request.MonitorId, *from, *to)
	default:
		defaultLimit := 100
		summaries, err = a.metricsSvc.GetLastSummaries(ctx, request.MonitorId, defaultLimit)
	}

	if err != nil {
		return v1.ResponseFromFactory(getMonitorsMonitorIdChecksResponseFactory, err), nil
	}

	return mapMonitorChecks(request.MonitorId, summaries), nil
}

func mapMonitorChecks(monitorID uuid.UUID, summaries []*probe.Summary) apigen.GetMonitorsMonitorIdChecks200JSONResponse {
	response := apigen.GetMonitorsMonitorIdChecks200JSONResponse{}
	for _, s := range summaries {
		if s == nil {
			continue
		}

		var statusCode *int
		if s.StatusCode != 0 {
			code := int(s.StatusCode)
			if code != 0 {
				statusCode = new(code)
			}
		}

		var reason *string
		if s.FailureReason != "" {
			reason = &s.FailureReason
		}

		response = append(response, apigen.MonitorCheck{
			MonitorId:      monitorID,
			CheckTime:      s.ProbeTime,
			Status:         apigen.MonitorStatus(s.MonitorStatus),
			LatencyMs:      int(s.LatencyMs),
			StatusCode:     statusCode,
			NetworkFailure: s.NetworkFailure,
			FailureReason:  reason,
		})
	}

	return response
}

func (a *ApiHandler) GetMonitorsMonitorIdSla(ctx context.Context, request apigen.GetMonitorsMonitorIdSlaRequestObject) (apigen.GetMonitorsMonitorIdSlaResponseObject, error) {
	from := time.Unix(0, 0).UTC()
	if request.Params.From != nil {
		from = *request.Params.From
	}

	to := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if request.Params.To != nil {
		to = *request.Params.To
	}

	sla, err := a.metricsSvc.GetSLA(ctx, request.MonitorId, from, to)
	if err != nil {
		return v1.ResponseFromFactory(getMonitorsMonitorIdSlaResponseFactory, err), nil
	}

	apiSLA := apigen.MonitorSLA{
		MonitorId:           request.MonitorId,
		StartTime:           sla.PeriodStart,
		EndTime:             sla.PeriodEnd,
		UptimePercentage:    float32(sla.UptimePercent),
		DowntimeDurationSec: sla.TotalDowntimeSec,
	}

	return apigen.GetMonitorsMonitorIdSla200JSONResponse(apiSLA), nil
}

func (a *ApiHandler) GetMonitorStatusHistory(ctx context.Context, request apigen.GetMonitorStatusHistoryRequestObject) (apigen.GetMonitorStatusHistoryResponseObject, error) {
	from := time.Unix(0, 0).UTC()
	if request.Params.From != nil {
		from = *request.Params.From
	}

	to := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if request.Params.To != nil {
		to = *request.Params.To
	}

	events, err := a.metricsSvc.GetStatusHistory(ctx, request.MonitorId, from, to)
	if err != nil {
		return v1.ResponseFromFactory(GetMonitorStatusHistoryResponseFactory, err), nil
	}

	response := apigen.GetMonitorStatusHistory200JSONResponse{}
	for i := range events {
		var reason *string
		if events[i].Reason != "" {
			reason = &events[i].Reason
		}

		response = append(response, apigen.MonitorStatusEvent{
			MonitorId: request.MonitorId,
			Status:    apigen.MonitorStatus(events[i].Status),
			StartTime: events[i].StartTime,
			EndTime:   events[i].EndTime,
			Reason:    reason,
		})
	}

	return response, nil
}

func monitorChecksParameterError(limit *int, from, to *time.Time) string {
	if limit != nil && (from != nil || to != nil) {
		return "limit cannot be used with from/to"
	}
	if (from != nil && to == nil) || (from == nil && to != nil) {
		return "from and to must be provided together"
	}
	return ""
}
