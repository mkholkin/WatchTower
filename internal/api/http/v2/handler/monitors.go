package handler

import (
	"context"
	"fmt"
	"net/url"
	"unicode/utf8"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	monitordto "WatchTower/internal/service/monitoring_management/dto"
)

func mapHTTPConfig(config apiv2gen.HTTPConfig) monitordto.HTTPMonitorNetworkConfig {
	headers := map[string]string{}
	if config.Headers != nil {
		headers = *config.Headers
	}
	body := ""
	if config.Body != nil {
		body = *config.Body
	}
	return monitordto.HTTPMonitorNetworkConfig{
		Method:          string(config.Method),
		Headers:         headers,
		Body:            body,
		FollowRedirects: config.FollowRedirects,
	}
}

func mapHTTPExpectations(expectations apiv2gen.HTTPExpectations) monitordto.HTTPMonitorExpectations {
	return monitordto.HTTPMonitorExpectations{
		StatusCodes:  expectations.ExpectedStatusCodes,
		MaxLatencyMs: expectations.MaxResponseTimeMs,
	}
}

func mapDomainMonitor(mon *monitor.Monitor) apiv2gen.Monitor {
	result := apiv2gen.Monitor{
		Id:                   mon.ID,
		Label:                mon.Label,
		ProbeIntervalSec:     int(mon.ProbeIntervalSec),
		Protocol:             apiv2gen.HTTP,
		IsEnabled:            mon.IsActive,
		Status:               apiv2gen.MonitorStatus(mon.CurrentStatus),
		CreatedAt:            mon.CreatedAt,
		AlertContactIds:      make([]apiv2gen.UUID, 0, len(mon.AlertContacts)),
		MaintenanceWindowIds: make([]apiv2gen.UUID, 0, len(mon.MaintenanceWindows)),
	}

	if mon.Target != nil {
		result.Endpoint = mon.Target.Endpoint
		switch config := mon.Target.Config.(type) {
		case target.HTTPConfig:
			result.NetworkConfig = mapDomainHTTPConfig(config)
		case *target.HTTPConfig:
			if config != nil {
				result.NetworkConfig = mapDomainHTTPConfig(*config)
			}
		}
	}

	switch expectations := mon.Expectations.(type) {
	case monitor.HTTPExpectations:
		result.Expectations = mapDomainHTTPExpectations(expectations)
	case *monitor.HTTPExpectations:
		if expectations != nil {
			result.Expectations = mapDomainHTTPExpectations(*expectations)
		}
	}

	for i := range mon.AlertContacts {
		result.AlertContactIds = append(result.AlertContactIds, mon.AlertContacts[i].ID)
	}
	for i := range mon.MaintenanceWindows {
		result.MaintenanceWindowIds = append(result.MaintenanceWindowIds, mon.MaintenanceWindows[i].ID)
	}
	return result
}

func mapDomainHTTPConfig(config target.HTTPConfig) apiv2gen.HTTPConfig {
	result := apiv2gen.HTTPConfig{
		Method:          apiv2gen.HTTPMethod(config.Method),
		FollowRedirects: config.FollowRedirects,
	}
	if len(config.Headers) != 0 {
		headers := config.Headers
		result.Headers = &headers
	}
	if config.Body != "" {
		body := config.Body
		result.Body = &body
	}
	return result
}

func mapDomainHTTPExpectations(expectations monitor.HTTPExpectations) apiv2gen.HTTPExpectations {
	return apiv2gen.HTTPExpectations{
		ExpectedStatusCodes: expectations.StatusCodes,
		MaxResponseTimeMs:   expectations.MaxLatencyMs,
	}
}

func (a *APIHandler) ListMonitors(ctx context.Context, _ apiv2gen.ListMonitorsRequestObject) (apiv2gen.ListMonitorsResponseObject, error) {
	monitors, err := a.monitoringSvc.GetAllMonitors(ctx)
	if err != nil {
		status, body := errorFrom(err)
		if status == 401 {
			return apiv2gen.ListMonitors401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		}
		return apiv2gen.ListMonitors500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
	}

	response := make(apiv2gen.ListMonitors200JSONResponse, 0, len(monitors))
	for _, mon := range monitors {
		response = append(response, mapDomainMonitor(mon))
	}
	return response, nil
}

func (a *APIHandler) CreateMonitor(ctx context.Context, request apiv2gen.CreateMonitorRequestObject) (apiv2gen.CreateMonitorResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.CreateMonitor400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	if message := validateMonitorCreate(*request.Body); message != "" {
		return apiv2gen.CreateMonitor400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}

	contactIDs := []apiv2gen.UUID{}
	if request.Body.AlertContactIds != nil {
		contactIDs = *request.Body.AlertContactIds
	}
	windowIDs := []apiv2gen.UUID{}
	if request.Body.MaintenanceWindowIds != nil {
		windowIDs = *request.Body.MaintenanceWindowIds
	}

	mon, err := a.monitoringSvc.CreateMonitor(ctx, monitordto.CreateMonitorDTO{
		Label:                request.Body.Label,
		Endpoint:             request.Body.Endpoint,
		ProbeIntervalSec:     int32(request.Body.ProbeIntervalSec),
		AlertContactIDs:      contactIDs,
		MaintenanceWindowIDs: windowIDs,
		NetworkConfig:        mapHTTPConfig(request.Body.NetworkConfig),
		Expectations:         mapHTTPExpectations(request.Body.Expectations),
	})
	if err != nil {
		status, body := errorFrom(err)
		switch status {
		case 400:
			return apiv2gen.CreateMonitor400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}, nil
		case 401:
			return apiv2gen.CreateMonitor401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 403:
			return apiv2gen.CreateMonitor403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}, nil
		case 404:
			return apiv2gen.CreateMonitor404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}, nil
		default:
			return apiv2gen.CreateMonitor500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}

	location := fmt.Sprintf("/api/v2/monitors/%s", mon.ID)
	return apiv2gen.CreateMonitor201JSONResponse{
		Body:    mapDomainMonitor(mon),
		Headers: apiv2gen.CreateMonitor201ResponseHeaders{Location: &location},
	}, nil
}

func (a *APIHandler) GetMonitor(ctx context.Context, request apiv2gen.GetMonitorRequestObject) (apiv2gen.GetMonitorResponseObject, error) {
	mon, err := a.monitoringSvc.GetMonitor(ctx, request.MonitorId)
	if err != nil {
		status, body := errorFrom(err)
		switch status {
		case 401:
			return apiv2gen.GetMonitor401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 403:
			return apiv2gen.GetMonitor403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}, nil
		case 404:
			return apiv2gen.GetMonitor404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}, nil
		default:
			return apiv2gen.GetMonitor500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}
	return apiv2gen.GetMonitor200JSONResponse(mapDomainMonitor(mon)), nil
}

func (a *APIHandler) PatchMonitor(ctx context.Context, request apiv2gen.PatchMonitorRequestObject) (apiv2gen.PatchMonitorResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.PatchMonitor400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	body := request.Body
	if body.Label == nil && body.Endpoint == nil && body.ProbeIntervalSec == nil && body.NetworkConfig == nil && body.Expectations == nil && body.IsEnabled == nil {
		return apiv2gen.PatchMonitor400JSONResponse{BadRequestJSONResponse: badRequest("at least one field is required")}, nil
	}
	if message := validateMonitorPatch(*body); message != "" {
		return apiv2gen.PatchMonitor400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}

	update := monitordto.UpdateMonitorDTO{ID: request.MonitorId, Label: body.Label, Endpoint: body.Endpoint, IsEnabled: body.IsEnabled}
	if body.ProbeIntervalSec != nil {
		value := int32(*body.ProbeIntervalSec)
		update.ProbeIntervalSec = &value
	}
	if body.NetworkConfig != nil {
		config := monitordto.MonitorNetworkConfig(mapHTTPConfig(*body.NetworkConfig))
		update.NetworkConfig = &config
	}
	if body.Expectations != nil {
		expectations := monitordto.MonitorExpectations(mapHTTPExpectations(*body.Expectations))
		update.Expectations = &expectations
	}

	hasResourceUpdate := update.Label != nil || update.Endpoint != nil || update.ProbeIntervalSec != nil || update.NetworkConfig != nil || update.Expectations != nil || update.IsEnabled != nil
	if hasResourceUpdate {
		if err := a.monitoringSvc.UpdateMonitor(ctx, update); err != nil {
			return patchMonitorError(err), nil
		}
	}

	mon, err := a.monitoringSvc.GetMonitor(ctx, request.MonitorId)
	if err != nil {
		return patchMonitorError(err), nil
	}
	return apiv2gen.PatchMonitor200JSONResponse(mapDomainMonitor(mon)), nil
}

func patchMonitorError(err error) apiv2gen.PatchMonitorResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 400:
		return apiv2gen.PatchMonitor400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}
	case 401:
		return apiv2gen.PatchMonitor401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.PatchMonitor403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.PatchMonitor404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.PatchMonitor500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) DeleteMonitor(ctx context.Context, request apiv2gen.DeleteMonitorRequestObject) (apiv2gen.DeleteMonitorResponseObject, error) {
	if err := a.monitoringSvc.DeleteMonitor(ctx, request.MonitorId); err != nil {
		status, body := errorFrom(err)
		switch status {
		case 401:
			return apiv2gen.DeleteMonitor401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 403:
			return apiv2gen.DeleteMonitor403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}, nil
		case 404:
			return apiv2gen.DeleteMonitor404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}, nil
		default:
			return apiv2gen.DeleteMonitor500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}
	return apiv2gen.DeleteMonitor204Response{}, nil
}

func (a *APIHandler) AttachAlertContact(ctx context.Context, request apiv2gen.AttachAlertContactRequestObject) (apiv2gen.AttachAlertContactResponseObject, error) {
	if err := a.monitoringSvc.LinkAlertContact(ctx, request.MonitorId, request.ContactId); err != nil {
		return attachAlertContactError(err), nil
	}
	return apiv2gen.AttachAlertContact204Response{}, nil
}

func attachAlertContactError(err error) apiv2gen.AttachAlertContactResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 401:
		return apiv2gen.AttachAlertContact401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.AttachAlertContact403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.AttachAlertContact404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.AttachAlertContact500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) DetachAlertContact(ctx context.Context, request apiv2gen.DetachAlertContactRequestObject) (apiv2gen.DetachAlertContactResponseObject, error) {
	if err := a.monitoringSvc.UnlinkAlertContact(ctx, request.MonitorId, request.ContactId); err != nil {
		status, body := errorFrom(err)
		switch status {
		case 401:
			return apiv2gen.DetachAlertContact401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 403:
			return apiv2gen.DetachAlertContact403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}, nil
		case 404:
			return apiv2gen.DetachAlertContact404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}, nil
		default:
			return apiv2gen.DetachAlertContact500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}
	return apiv2gen.DetachAlertContact204Response{}, nil
}

func validateMonitorCreate(body apiv2gen.MonitorCreate) string {
	if message := validateMonitorLabel(body.Label); message != "" {
		return message
	}
	if message := validateMonitorEndpoint(body.Endpoint); message != "" {
		return message
	}
	if body.ProbeIntervalSec < 1 {
		return "probe_interval_sec must be at least 1"
	}
	if message := validateHTTPConfig(body.NetworkConfig); message != "" {
		return message
	}
	if message := validateHTTPExpectations(body.Expectations); message != "" {
		return message
	}
	if body.AlertContactIds != nil && hasDuplicateUUIDs(*body.AlertContactIds) {
		return "alert_contact_ids must contain unique values"
	}
	if body.MaintenanceWindowIds != nil && hasDuplicateUUIDs(*body.MaintenanceWindowIds) {
		return "maintenance_window_ids must contain unique values"
	}
	return ""
}

func validateMonitorPatch(body apiv2gen.MonitorPatch) string {
	if body.Label != nil {
		if message := validateMonitorLabel(*body.Label); message != "" {
			return message
		}
	}
	if body.Endpoint != nil {
		if message := validateMonitorEndpoint(*body.Endpoint); message != "" {
			return message
		}
	}
	if body.ProbeIntervalSec != nil && *body.ProbeIntervalSec < 1 {
		return "probe_interval_sec must be at least 1"
	}
	if body.NetworkConfig != nil {
		if message := validateHTTPConfig(*body.NetworkConfig); message != "" {
			return message
		}
	}
	if body.Expectations != nil {
		return validateHTTPExpectations(*body.Expectations)
	}
	return ""
}

func validateMonitorLabel(label string) string {
	length := utf8.RuneCountInString(label)
	if length < 1 || length > 120 {
		return "label length must be between 1 and 120 characters"
	}
	return ""
}

func validateMonitorEndpoint(endpoint string) string {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "endpoint must be an absolute HTTP or HTTPS URI"
	}
	return ""
}

func validateHTTPConfig(config apiv2gen.HTTPConfig) string {
	if !config.Method.Valid() {
		return "network_config.method is not supported"
	}
	return ""
}

func validateHTTPExpectations(expectations apiv2gen.HTTPExpectations) string {
	if len(expectations.ExpectedStatusCodes) == 0 {
		return "expectations.expected_status_codes must not be empty"
	}
	seen := make(map[int]struct{}, len(expectations.ExpectedStatusCodes))
	for _, statusCode := range expectations.ExpectedStatusCodes {
		if statusCode < 100 || statusCode > 599 {
			return "expectations.expected_status_codes must contain HTTP status codes"
		}
		if _, exists := seen[statusCode]; exists {
			return "expectations.expected_status_codes must contain unique values"
		}
		seen[statusCode] = struct{}{}
	}
	if expectations.MaxResponseTimeMs < 1 {
		return "expectations.max_response_time_ms must be at least 1"
	}
	return ""
}
