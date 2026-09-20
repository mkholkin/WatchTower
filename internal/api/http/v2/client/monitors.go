package client

import (
	"context"
	"fmt"
	"net/http"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	monitordto "WatchTower/internal/service/monitoring_management/dto"

	"github.com/google/uuid"
)

func mapAPIMonitor(value apiv2gen.Monitor) *monitor.Monitor {
	headers := map[string]string{}
	if value.NetworkConfig.Headers != nil {
		headers = *value.NetworkConfig.Headers
	}
	body := ""
	if value.NetworkConfig.Body != nil {
		body = *value.NetworkConfig.Body
	}
	contacts := make([]alert.Contact, 0, len(value.AlertContactIds))
	for _, id := range value.AlertContactIds {
		contacts = append(contacts, alert.Contact{ID: id})
	}
	windows := make([]maintenance.MaintenanceWindow, 0, len(value.MaintenanceWindowIds))
	for _, id := range value.MaintenanceWindowIds {
		windows = append(windows, maintenance.MaintenanceWindow{ID: id})
	}
	return &monitor.Monitor{
		ID:    value.Id,
		Label: value.Label,
		Target: &target.Target{
			Endpoint: value.Endpoint,
			Config: target.HTTPConfig{
				Method: string(value.NetworkConfig.Method), Headers: headers, Body: body,
				FollowRedirects: value.NetworkConfig.FollowRedirects,
			},
		},
		AlertContacts:      contacts,
		MaintenanceWindows: windows,
		CurrentStatus:      monitor.Status(value.Status),
		ProbeIntervalSec:   int32(value.ProbeIntervalSec),
		IsActive:           value.IsEnabled,
		CreatedAt:          value.CreatedAt,
		Expectations: monitor.HTTPExpectations{
			StatusCodes: value.Expectations.ExpectedStatusCodes, MaxLatencyMs: value.Expectations.MaxResponseTimeMs,
		},
	}
}

func apiMonitorConfig(value monitordto.MonitorNetworkConfig) (apiv2gen.HTTPConfig, error) {
	var config monitordto.HTTPMonitorNetworkConfig
	switch typed := value.(type) {
	case monitordto.HTTPMonitorNetworkConfig:
		config = typed
	case *monitordto.HTTPMonitorNetworkConfig:
		if typed == nil {
			return apiv2gen.HTTPConfig{}, fmt.Errorf("network config is nil")
		}
		config = *typed
	default:
		return apiv2gen.HTTPConfig{}, fmt.Errorf("v2 API supports HTTP monitors, got %T", value)
	}
	result := apiv2gen.HTTPConfig{Method: apiv2gen.HTTPMethod(config.Method), FollowRedirects: config.FollowRedirects}
	if len(config.Headers) != 0 {
		headers := config.Headers
		result.Headers = &headers
	}
	if config.Body != "" {
		body := config.Body
		result.Body = &body
	}
	return result, nil
}

func apiMonitorExpectations(value monitordto.MonitorExpectations) (apiv2gen.HTTPExpectations, error) {
	var expectations monitordto.HTTPMonitorExpectations
	switch typed := value.(type) {
	case monitordto.HTTPMonitorExpectations:
		expectations = typed
	case *monitordto.HTTPMonitorExpectations:
		if typed == nil {
			return apiv2gen.HTTPExpectations{}, fmt.Errorf("expectations are nil")
		}
		expectations = *typed
	default:
		return apiv2gen.HTTPExpectations{}, fmt.Errorf("v2 API supports HTTP monitors, got %T", value)
	}
	return apiv2gen.HTTPExpectations{
		ExpectedStatusCodes: expectations.StatusCodes,
		MaxResponseTimeMs:   expectations.MaxLatencyMs,
	}, nil
}

func (c *Client) GetAllMonitors(ctx context.Context) ([]*monitor.Monitor, error) {
	response, err := c.api.ListMonitorsWithResponse(ctx, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	result := make([]*monitor.Monitor, 0, len(*response.JSON200))
	for _, value := range *response.JSON200 {
		result = append(result, mapAPIMonitor(value))
	}
	return result, nil
}

func (c *Client) GetMonitor(ctx context.Context, monitorID uuid.UUID) (*monitor.Monitor, error) {
	response, err := c.api.GetMonitorWithResponse(ctx, monitorID, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("get monitor: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	return mapAPIMonitor(*response.JSON200), nil
}

func (c *Client) CreateMonitor(ctx context.Context, dto monitordto.CreateMonitorDTO) (*monitor.Monitor, error) {
	config, err := apiMonitorConfig(dto.NetworkConfig)
	if err != nil {
		return nil, err
	}
	expectations, err := apiMonitorExpectations(dto.Expectations)
	if err != nil {
		return nil, err
	}
	contactIDs := append([]uuid.UUID(nil), dto.AlertContactIDs...)
	windowIDs := append([]uuid.UUID(nil), dto.MaintenanceWindowIDs...)
	response, err := c.api.CreateMonitorWithResponse(ctx, apiv2gen.CreateMonitorJSONRequestBody{
		Label: dto.Label, Endpoint: dto.Endpoint, ProbeIntervalSec: int(dto.ProbeIntervalSec),
		NetworkConfig: config, Expectations: expectations,
		AlertContactIds: &contactIDs, MaintenanceWindowIds: &windowIDs,
	}, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("create monitor: %w", err)
	}
	if response.StatusCode() != http.StatusCreated || response.JSON201 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	return mapAPIMonitor(*response.JSON201), nil
}

func (c *Client) UpdateMonitor(ctx context.Context, dto monitordto.UpdateMonitorDTO) error {
	body := apiv2gen.PatchMonitorJSONRequestBody{Label: dto.Label, Endpoint: dto.Endpoint, IsEnabled: dto.IsEnabled}
	if dto.ProbeIntervalSec != nil {
		value := int(*dto.ProbeIntervalSec)
		body.ProbeIntervalSec = &value
	}
	if dto.NetworkConfig != nil {
		config, err := apiMonitorConfig(*dto.NetworkConfig)
		if err != nil {
			return err
		}
		body.NetworkConfig = &config
	}
	if dto.Expectations != nil {
		expectations, err := apiMonitorExpectations(*dto.Expectations)
		if err != nil {
			return err
		}
		body.Expectations = &expectations
	}
	response, err := c.api.PatchMonitorWithResponse(ctx, dto.ID, body, c.withAuth)
	if err != nil {
		return fmt.Errorf("patch monitor: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) setMonitorEnabled(ctx context.Context, monitorID uuid.UUID, enabled bool) error {
	response, err := c.api.PatchMonitorWithResponse(ctx, monitorID, apiv2gen.PatchMonitorJSONRequestBody{IsEnabled: &enabled}, c.withAuth)
	if err != nil {
		return fmt.Errorf("set monitor enabled: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) EnableMonitor(ctx context.Context, monitorID uuid.UUID) error {
	return c.setMonitorEnabled(ctx, monitorID, true)
}

func (c *Client) DisableMonitor(ctx context.Context, monitorID uuid.UUID) error {
	return c.setMonitorEnabled(ctx, monitorID, false)
}

func (c *Client) DeleteMonitor(ctx context.Context, monitorID uuid.UUID) error {
	response, err := c.api.DeleteMonitorWithResponse(ctx, monitorID, c.withAuth)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) LinkAlertContact(ctx context.Context, monitorID uuid.UUID, contactID uuid.UUID) error {
	response, err := c.api.AttachAlertContactWithResponse(ctx, monitorID, contactID, c.withAuth)
	if err != nil {
		return fmt.Errorf("link alert contact: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) UnlinkAlertContact(ctx context.Context, monitorID uuid.UUID, contactID uuid.UUID) error {
	response, err := c.api.DetachAlertContactWithResponse(ctx, monitorID, contactID, c.withAuth)
	if err != nil {
		return fmt.Errorf("unlink alert contact: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}
