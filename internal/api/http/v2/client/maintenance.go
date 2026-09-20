package client

import (
	"context"
	"fmt"
	"net/http"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/entity/maintenance"
	maintenancesvc "WatchTower/internal/service/maintenance"

	"github.com/google/uuid"
)

func mapAPIMaintenanceWindow(value apiv2gen.MaintenanceWindow) (*maintenance.MaintenanceWindow, error) {
	description := ""
	if value.Description != nil {
		description = *value.Description
	}
	discriminator, err := value.Config.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("read maintenance config: %w", err)
	}
	result := &maintenance.MaintenanceWindow{ID: value.Id, Title: value.Title, Description: description}
	switch discriminator {
	case "one_time":
		config, err := value.Config.AsOneTimeMaintenanceConfig()
		if err != nil {
			return nil, err
		}
		result.Type = maintenance.WindowTypeOneTime
		result.Config = maintenance.OneTimeMaintenanceWindowConfig{StartTime: config.StartsAt, EndTime: config.EndsAt}
	case "manual":
		config, err := value.Config.AsManualMaintenanceConfig()
		if err != nil {
			return nil, err
		}
		result.Type = maintenance.WindowTypeManual
		result.Config = maintenance.ManualMaintenanceWindowConfig{Active: config.IsActive}
	default:
		return nil, fmt.Errorf("unsupported maintenance config %q", discriminator)
	}
	return result, nil
}

func (c *Client) GetAllMaintenanceWindows(ctx context.Context) ([]maintenance.MaintenanceWindow, error) {
	response, err := c.api.ListMaintenanceWindowsWithResponse(ctx, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("list maintenance windows: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	result := make([]maintenance.MaintenanceWindow, 0, len(*response.JSON200))
	for _, value := range *response.JSON200 {
		window, err := mapAPIMaintenanceWindow(value)
		if err != nil {
			return nil, err
		}
		result = append(result, *window)
	}
	return result, nil
}

func (c *Client) GetMaintenanceWindow(ctx context.Context, windowID uuid.UUID) (*maintenance.MaintenanceWindow, error) {
	response, err := c.api.GetMaintenanceWindowWithResponse(ctx, windowID, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("get maintenance window: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	return mapAPIMaintenanceWindow(*response.JSON200)
}

func (c *Client) CreateOneTimeMaintenanceWindow(ctx context.Context, dto maintenancesvc.CreateOneTimeMaintenanceWindowDTO) (*maintenance.MaintenanceWindow, error) {
	var config apiv2gen.MaintenanceConfig
	if err := config.FromOneTimeMaintenanceConfig(apiv2gen.OneTimeMaintenanceConfig{
		Type: apiv2gen.OneTime, StartsAt: dto.StartTime, EndsAt: dto.EndTime,
	}); err != nil {
		return nil, err
	}
	return c.createMaintenanceWindow(ctx, dto.Title, dto.Description, config)
}

func (c *Client) CreateManualMaintenanceWindow(ctx context.Context, dto maintenancesvc.CreateManualMaintenanceWindowDTO) (*maintenance.MaintenanceWindow, error) {
	var config apiv2gen.MaintenanceConfig
	if err := config.FromManualMaintenanceConfig(apiv2gen.ManualMaintenanceConfig{Type: apiv2gen.Manual, IsActive: false}); err != nil {
		return nil, err
	}
	return c.createMaintenanceWindow(ctx, dto.Title, dto.Description, config)
}

func (c *Client) createMaintenanceWindow(ctx context.Context, title, description string, config apiv2gen.MaintenanceConfig) (*maintenance.MaintenanceWindow, error) {
	body := apiv2gen.CreateMaintenanceWindowJSONRequestBody{Title: title, Config: config}
	if description != "" {
		body.Description = &description
	}
	response, err := c.api.CreateMaintenanceWindowWithResponse(ctx, body, c.withAuth)
	if err != nil {
		return nil, fmt.Errorf("create maintenance window: %w", err)
	}
	if response.StatusCode() != http.StatusCreated || response.JSON201 == nil {
		return nil, responseError(response.StatusCode(), response.Body)
	}
	return mapAPIMaintenanceWindow(*response.JSON201)
}

func (c *Client) UpdateMaintenanceWindow(ctx context.Context, dto maintenancesvc.UpdateMaintenanceWindowDTO) error {
	body := apiv2gen.PatchMaintenanceWindowJSONRequestBody{Title: dto.Title, Description: dto.Description}
	if dto.ConfigUpdate != nil {
		var config apiv2gen.MaintenanceConfig
		switch typed := dto.ConfigUpdate.(type) {
		case maintenance.OneTimeConfigUpdate:
			current, err := c.GetMaintenanceWindow(ctx, dto.WindowID)
			if err != nil {
				return err
			}
			existing, ok := current.Config.(maintenance.OneTimeMaintenanceWindowConfig)
			if !ok {
				return fmt.Errorf("one-time update is not applicable to %s", current.Type)
			}
			if typed.StartTime != nil {
				existing.StartTime = *typed.StartTime
			}
			if typed.EndTime != nil {
				existing.EndTime = *typed.EndTime
			}
			if err := config.FromOneTimeMaintenanceConfig(apiv2gen.OneTimeMaintenanceConfig{
				Type: apiv2gen.OneTime, StartsAt: existing.StartTime, EndsAt: existing.EndTime,
			}); err != nil {
				return err
			}
		case maintenance.ManualConfigUpdate:
			active := false
			if typed.Active != nil {
				active = *typed.Active
			}
			if err := config.FromManualMaintenanceConfig(apiv2gen.ManualMaintenanceConfig{Type: apiv2gen.Manual, IsActive: active}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported maintenance config update %T", dto.ConfigUpdate)
		}
		body.Config = &config
	}
	response, err := c.api.PatchMaintenanceWindowWithResponse(ctx, dto.WindowID, body, c.withAuth)
	if err != nil {
		return fmt.Errorf("patch maintenance window: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) AddMonitorToMaintenanceWindow(ctx context.Context, monitorID uuid.UUID, windowID uuid.UUID) error {
	response, err := c.api.AttachMonitorToMaintenanceWindowWithResponse(ctx, windowID, monitorID, c.withAuth)
	if err != nil {
		return fmt.Errorf("attach monitor to maintenance window: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) RemoveMonitorFromMaintenanceWindow(ctx context.Context, monitorID uuid.UUID, windowID uuid.UUID) error {
	response, err := c.api.DetachMonitorFromMaintenanceWindowWithResponse(ctx, windowID, monitorID, c.withAuth)
	if err != nil {
		return fmt.Errorf("detach monitor from maintenance window: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}

func (c *Client) DeleteMaintenanceWindow(ctx context.Context, windowID uuid.UUID) error {
	response, err := c.api.DeleteMaintenanceWindowWithResponse(ctx, windowID, c.withAuth)
	if err != nil {
		return fmt.Errorf("delete maintenance window: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.Body)
	}
	return nil
}
