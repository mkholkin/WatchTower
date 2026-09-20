package handler

import (
	"context"
	"encoding/json"
	"fmt"

	apiv2gen "WatchTower/internal/api/http/v2/gen"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	maintenancesvc "WatchTower/internal/service/maintenance"

	"github.com/google/uuid"
)

func mapDomainMaintenanceWindow(window *maintenance.MaintenanceWindow, monitorIDs []uuid.UUID) (apiv2gen.MaintenanceWindow, error) {
	result := apiv2gen.MaintenanceWindow{
		Id:         window.ID,
		Title:      window.Title,
		MonitorIds: monitorIDs,
	}
	if window.Description != "" {
		description := window.Description
		result.Description = &description
	}

	switch config := window.Config.(type) {
	case maintenance.OneTimeMaintenanceWindowConfig:
		if err := result.Config.FromOneTimeMaintenanceConfig(apiv2gen.OneTimeMaintenanceConfig{
			Type: apiv2gen.OneTime, StartsAt: config.StartTime, EndsAt: config.EndTime,
		}); err != nil {
			return apiv2gen.MaintenanceWindow{}, err
		}
	case *maintenance.OneTimeMaintenanceWindowConfig:
		if config != nil {
			if err := result.Config.FromOneTimeMaintenanceConfig(apiv2gen.OneTimeMaintenanceConfig{
				Type: apiv2gen.OneTime, StartsAt: config.StartTime, EndsAt: config.EndTime,
			}); err != nil {
				return apiv2gen.MaintenanceWindow{}, err
			}
		}
	case maintenance.ManualMaintenanceWindowConfig:
		if err := result.Config.FromManualMaintenanceConfig(apiv2gen.ManualMaintenanceConfig{
			Type: apiv2gen.Manual, IsActive: config.Active,
		}); err != nil {
			return apiv2gen.MaintenanceWindow{}, err
		}
	case *maintenance.ManualMaintenanceWindowConfig:
		if config != nil {
			if err := result.Config.FromManualMaintenanceConfig(apiv2gen.ManualMaintenanceConfig{
				Type: apiv2gen.Manual, IsActive: config.Active,
			}); err != nil {
				return apiv2gen.MaintenanceWindow{}, err
			}
		}
	default:
		return apiv2gen.MaintenanceWindow{}, fmt.Errorf("unsupported maintenance config %T", window.Config)
	}

	return result, nil
}

func monitorIDsByWindow(monitors []*monitor.Monitor) map[uuid.UUID][]uuid.UUID {
	result := make(map[uuid.UUID][]uuid.UUID)
	for _, mon := range monitors {
		for i := range mon.MaintenanceWindows {
			windowID := mon.MaintenanceWindows[i].ID
			result[windowID] = append(result[windowID], mon.ID)
		}
	}
	return result
}

func validateMaintenanceConfig(config apiv2gen.MaintenanceConfig) string {
	payload, err := json.Marshal(config)
	if err != nil {
		return "invalid maintenance config"
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil || fields == nil {
		return "invalid maintenance config"
	}
	var discriminator string
	if err := json.Unmarshal(fields["type"], &discriminator); err != nil {
		return "maintenance config type is required"
	}
	switch discriminator {
	case "manual":
		if len(fields) != 2 {
			return "manual maintenance config contains unsupported fields"
		}
		var active bool
		if err := json.Unmarshal(fields["is_active"], &active); err != nil {
			return "manual maintenance config requires is_active"
		}
	case "one_time":
		if len(fields) != 3 {
			return "one-time maintenance config contains unsupported fields"
		}
		if _, ok := fields["starts_at"]; !ok {
			return "one-time maintenance config requires starts_at"
		}
		if _, ok := fields["ends_at"]; !ok {
			return "one-time maintenance config requires ends_at"
		}
		if _, err := config.AsOneTimeMaintenanceConfig(); err != nil {
			return "invalid one-time maintenance config"
		}
	default:
		return "unsupported maintenance config"
	}
	return ""
}

func (a *APIHandler) maintenanceMonitorIDs(ctx context.Context) (map[uuid.UUID][]uuid.UUID, error) {
	if a.monitoringSvc == nil {
		return map[uuid.UUID][]uuid.UUID{}, nil
	}
	monitors, err := a.monitoringSvc.GetAllMonitors(ctx)
	if err != nil {
		return nil, err
	}
	return monitorIDsByWindow(monitors), nil
}

func (a *APIHandler) ListMaintenanceWindows(ctx context.Context, _ apiv2gen.ListMaintenanceWindowsRequestObject) (apiv2gen.ListMaintenanceWindowsResponseObject, error) {
	windows, err := a.maintenanceSvc.GetAllMaintenanceWindows(ctx)
	if err != nil {
		return listMaintenanceWindowsError(err), nil
	}
	monitorIDs, err := a.maintenanceMonitorIDs(ctx)
	if err != nil {
		return listMaintenanceWindowsError(err), nil
	}

	response := make(apiv2gen.ListMaintenanceWindows200JSONResponse, 0, len(windows))
	for i := range windows {
		mapped, err := mapDomainMaintenanceWindow(&windows[i], monitorIDs[windows[i].ID])
		if err != nil {
			return listMaintenanceWindowsError(err), nil
		}
		response = append(response, mapped)
	}
	return response, nil
}

func listMaintenanceWindowsError(err error) apiv2gen.ListMaintenanceWindowsResponseObject {
	status, body := errorFrom(err)
	if status == 401 {
		return apiv2gen.ListMaintenanceWindows401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	}
	return apiv2gen.ListMaintenanceWindows500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
}

func (a *APIHandler) CreateMaintenanceWindow(ctx context.Context, request apiv2gen.CreateMaintenanceWindowRequestObject) (apiv2gen.CreateMaintenanceWindowResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	if message := validateRequiredText(request.Body.Title, "title", 120); message != "" {
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}
	if message := validateOptionalText(request.Body.Description, "description", 1000); message != "" {
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}
	if request.Body.MonitorIds != nil && hasDuplicateUUIDs(*request.Body.MonitorIds) {
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("monitor_ids must contain unique values")}, nil
	}
	if message := validateMaintenanceConfig(request.Body.Config); message != "" {
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}
	description := ""
	if request.Body.Description != nil {
		description = *request.Body.Description
	}
	monitorIDs := []uuid.UUID{}
	if request.Body.MonitorIds != nil {
		monitorIDs = *request.Body.MonitorIds
	}
	for _, monitorID := range monitorIDs {
		if _, err := a.monitoringSvc.GetMonitor(ctx, monitorID); err != nil {
			return createMaintenanceWindowError(err), nil
		}
	}

	discriminator, err := request.Body.Config.Discriminator()
	if err != nil {
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("invalid maintenance config")}, nil
	}
	var window *maintenance.MaintenanceWindow
	switch discriminator {
	case "one_time":
		config, err := request.Body.Config.AsOneTimeMaintenanceConfig()
		if err != nil {
			return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("invalid one-time config")}, nil
		}
		window, err = a.maintenanceSvc.CreateOneTimeMaintenanceWindow(ctx, maintenancesvc.CreateOneTimeMaintenanceWindowDTO{
			Title: request.Body.Title, Description: description, StartTime: config.StartsAt, EndTime: config.EndsAt,
		})
	case "manual":
		config, configErr := request.Body.Config.AsManualMaintenanceConfig()
		if configErr != nil {
			return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("invalid manual config")}, nil
		}
		window, err = a.maintenanceSvc.CreateManualMaintenanceWindow(ctx, maintenancesvc.CreateManualMaintenanceWindowDTO{
			Title: request.Body.Title, Description: description, Active: config.IsActive,
		})
	default:
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("unsupported maintenance config")}, nil
	}
	if err != nil {
		return createMaintenanceWindowError(err), nil
	}
	for _, monitorID := range monitorIDs {
		if err := a.maintenanceSvc.AddMonitorToMaintenanceWindow(ctx, monitorID, window.ID); err != nil {
			if cleanupErr := a.maintenanceSvc.DeleteMaintenanceWindow(ctx, window.ID); cleanupErr != nil {
				return createMaintenanceWindowError(fmt.Errorf("maintenance window link failed and cleanup failed: %v; %v", err, cleanupErr)), nil
			}
			return createMaintenanceWindowError(err), nil
		}
	}

	mapped, err := mapDomainMaintenanceWindow(window, monitorIDs)
	if err != nil {
		return createMaintenanceWindowError(err), nil
	}
	location := fmt.Sprintf("/api/v2/maintenance-windows/%s", window.ID)
	return apiv2gen.CreateMaintenanceWindow201JSONResponse{
		Body:    mapped,
		Headers: apiv2gen.CreateMaintenanceWindow201ResponseHeaders{Location: &location},
	}, nil
}

func createMaintenanceWindowError(err error) apiv2gen.CreateMaintenanceWindowResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 400:
		return apiv2gen.CreateMaintenanceWindow400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}
	case 401:
		return apiv2gen.CreateMaintenanceWindow401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.CreateMaintenanceWindow403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.CreateMaintenanceWindow404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.CreateMaintenanceWindow500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) GetMaintenanceWindow(ctx context.Context, request apiv2gen.GetMaintenanceWindowRequestObject) (apiv2gen.GetMaintenanceWindowResponseObject, error) {
	window, err := a.maintenanceSvc.GetMaintenanceWindow(ctx, request.WindowId)
	if err != nil {
		return getMaintenanceWindowError(err), nil
	}
	monitorIDs, err := a.maintenanceMonitorIDs(ctx)
	if err != nil {
		return getMaintenanceWindowError(err), nil
	}
	mapped, err := mapDomainMaintenanceWindow(window, monitorIDs[window.ID])
	if err != nil {
		return getMaintenanceWindowError(err), nil
	}
	return apiv2gen.GetMaintenanceWindow200JSONResponse(mapped), nil
}

func getMaintenanceWindowError(err error) apiv2gen.GetMaintenanceWindowResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 401:
		return apiv2gen.GetMaintenanceWindow401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.GetMaintenanceWindow403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.GetMaintenanceWindow404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.GetMaintenanceWindow500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) PatchMaintenanceWindow(ctx context.Context, request apiv2gen.PatchMaintenanceWindowRequestObject) (apiv2gen.PatchMaintenanceWindowResponseObject, error) {
	if request.Body == nil {
		return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("request body is required")}, nil
	}
	body := request.Body
	if body.Title == nil && body.Description == nil && body.Config == nil {
		return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("at least one field is required")}, nil
	}
	if body.Title != nil {
		if message := validateRequiredText(*body.Title, "title", 120); message != "" {
			return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
		}
	}
	if message := validateOptionalText(body.Description, "description", 1000); message != "" {
		return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
	}
	if body.Config != nil {
		if message := validateMaintenanceConfig(*body.Config); message != "" {
			return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest(message)}, nil
		}
	}

	update := maintenancesvc.UpdateMaintenanceWindowDTO{WindowID: request.WindowId, Title: body.Title, Description: body.Description}
	if body.Config != nil {
		discriminator, err := body.Config.Discriminator()
		if err != nil {
			return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("invalid maintenance config")}, nil
		}
		switch discriminator {
		case "one_time":
			config, err := body.Config.AsOneTimeMaintenanceConfig()
			if err != nil {
				return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("invalid one-time config")}, nil
			}
			update.ConfigUpdate = maintenance.OneTimeConfigUpdate{StartTime: &config.StartsAt, EndTime: &config.EndsAt}
		case "manual":
			config, err := body.Config.AsManualMaintenanceConfig()
			if err != nil {
				return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("invalid manual config")}, nil
			}
			update.ConfigUpdate = maintenance.ManualConfigUpdate{Active: &config.IsActive}
		default:
			return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: badRequest("unsupported maintenance config")}, nil
		}
	}

	if err := a.maintenanceSvc.UpdateMaintenanceWindow(ctx, update); err != nil {
		return patchMaintenanceWindowError(err), nil
	}
	window, err := a.maintenanceSvc.GetMaintenanceWindow(ctx, request.WindowId)
	if err != nil {
		return patchMaintenanceWindowError(err), nil
	}
	monitorIDs, err := a.maintenanceMonitorIDs(ctx)
	if err != nil {
		return patchMaintenanceWindowError(err), nil
	}
	mapped, err := mapDomainMaintenanceWindow(window, monitorIDs[window.ID])
	if err != nil {
		return patchMaintenanceWindowError(err), nil
	}
	return apiv2gen.PatchMaintenanceWindow200JSONResponse(mapped), nil
}

func patchMaintenanceWindowError(err error) apiv2gen.PatchMaintenanceWindowResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 400:
		return apiv2gen.PatchMaintenanceWindow400JSONResponse{BadRequestJSONResponse: apiv2gen.BadRequestJSONResponse(body)}
	case 401:
		return apiv2gen.PatchMaintenanceWindow401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.PatchMaintenanceWindow403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.PatchMaintenanceWindow404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.PatchMaintenanceWindow500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) DeleteMaintenanceWindow(ctx context.Context, request apiv2gen.DeleteMaintenanceWindowRequestObject) (apiv2gen.DeleteMaintenanceWindowResponseObject, error) {
	if err := a.maintenanceSvc.DeleteMaintenanceWindow(ctx, request.WindowId); err != nil {
		status, body := errorFrom(err)
		switch status {
		case 401:
			return apiv2gen.DeleteMaintenanceWindow401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 403:
			return apiv2gen.DeleteMaintenanceWindow403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}, nil
		case 404:
			return apiv2gen.DeleteMaintenanceWindow404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}, nil
		default:
			return apiv2gen.DeleteMaintenanceWindow500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}
	return apiv2gen.DeleteMaintenanceWindow204Response{}, nil
}

func (a *APIHandler) AttachMonitorToMaintenanceWindow(ctx context.Context, request apiv2gen.AttachMonitorToMaintenanceWindowRequestObject) (apiv2gen.AttachMonitorToMaintenanceWindowResponseObject, error) {
	if err := a.maintenanceSvc.AddMonitorToMaintenanceWindow(ctx, request.MonitorId, request.WindowId); err != nil {
		return attachMonitorError(err), nil
	}
	return apiv2gen.AttachMonitorToMaintenanceWindow204Response{}, nil
}

func attachMonitorError(err error) apiv2gen.AttachMonitorToMaintenanceWindowResponseObject {
	status, body := errorFrom(err)
	switch status {
	case 401:
		return apiv2gen.AttachMonitorToMaintenanceWindow401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}
	case 403:
		return apiv2gen.AttachMonitorToMaintenanceWindow403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}
	case 404:
		return apiv2gen.AttachMonitorToMaintenanceWindow404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}
	default:
		return apiv2gen.AttachMonitorToMaintenanceWindow500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}
	}
}

func (a *APIHandler) DetachMonitorFromMaintenanceWindow(ctx context.Context, request apiv2gen.DetachMonitorFromMaintenanceWindowRequestObject) (apiv2gen.DetachMonitorFromMaintenanceWindowResponseObject, error) {
	if err := a.maintenanceSvc.RemoveMonitorFromMaintenanceWindow(ctx, request.MonitorId, request.WindowId); err != nil {
		status, body := errorFrom(err)
		switch status {
		case 401:
			return apiv2gen.DetachMonitorFromMaintenanceWindow401JSONResponse{UnauthorizedJSONResponse: unauthorized(body)}, nil
		case 403:
			return apiv2gen.DetachMonitorFromMaintenanceWindow403JSONResponse{ForbiddenJSONResponse: apiv2gen.ForbiddenJSONResponse(body)}, nil
		case 404:
			return apiv2gen.DetachMonitorFromMaintenanceWindow404JSONResponse{NotFoundJSONResponse: apiv2gen.NotFoundJSONResponse(body)}, nil
		default:
			return apiv2gen.DetachMonitorFromMaintenanceWindow500JSONResponse{InternalErrorJSONResponse: apiv2gen.InternalErrorJSONResponse(body)}, nil
		}
	}
	return apiv2gen.DetachMonitorFromMaintenanceWindow204Response{}, nil
}
