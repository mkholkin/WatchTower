package handler

import (
	v1 "WatchTower/internal/api/http/v1"
	apigen "WatchTower/internal/api/http/v1/gen"
	"WatchTower/internal/domain/entity/maintenance"
	maintenancesvc "WatchTower/internal/service/maintenance"
	"context"
	"fmt"
)

var getMaintenanceWindowsListResponseFactory = v1.ResponseFactory[apigen.GetMaintenanceWindowsListResponseObject]{
	401: func(er apigen.ErrorResponse) apigen.GetMaintenanceWindowsListResponseObject {
		return apigen.GetMaintenanceWindowsList401JSONResponse{apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMaintenanceWindowsListResponseObject {
		return apigen.GetMaintenanceWindowsList500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var createMaintenanceWindowResponseFactory = v1.ResponseFactory[apigen.CreateMaintenanceWindowResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.CreateMaintenanceWindowResponseObject {
		return apigen.CreateMaintenanceWindow400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.CreateMaintenanceWindowResponseObject {
		return apigen.CreateMaintenanceWindow401JSONResponse{apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.CreateMaintenanceWindowResponseObject {
		return apigen.CreateMaintenanceWindow500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var deleteMaintenanceWindowResponseFactory = v1.ResponseFactory[apigen.DeleteMaintenanceWindowResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.DeleteMaintenanceWindowResponseObject {
		return apigen.DeleteMaintenanceWindow400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.DeleteMaintenanceWindowResponseObject {
		return apigen.DeleteMaintenanceWindow401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.DeleteMaintenanceWindowResponseObject {
		return apigen.DeleteMaintenanceWindow403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.DeleteMaintenanceWindowResponseObject {
		return apigen.DeleteMaintenanceWindow404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.DeleteMaintenanceWindowResponseObject {
		return apigen.DeleteMaintenanceWindow500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var getMaintenanceWindowDetailsResponseFactory = v1.ResponseFactory[apigen.GetMaintenanceWindowDetailsResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.GetMaintenanceWindowDetailsResponseObject {
		return apigen.GetMaintenanceWindowDetails400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.GetMaintenanceWindowDetailsResponseObject {
		return apigen.GetMaintenanceWindowDetails401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.GetMaintenanceWindowDetailsResponseObject {
		return apigen.GetMaintenanceWindowDetails403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.GetMaintenanceWindowDetailsResponseObject {
		return apigen.GetMaintenanceWindowDetails404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMaintenanceWindowDetailsResponseObject {
		return apigen.GetMaintenanceWindowDetails500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var addMonitorToMaintenanceWindowResponseFactory = v1.ResponseFactory[apigen.AddMonitorToMaintenanceWindowResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.AddMonitorToMaintenanceWindowResponseObject {
		return apigen.AddMonitorToMaintenanceWindow400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.AddMonitorToMaintenanceWindowResponseObject {
		return apigen.AddMonitorToMaintenanceWindow401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.AddMonitorToMaintenanceWindowResponseObject {
		return apigen.AddMonitorToMaintenanceWindow403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.AddMonitorToMaintenanceWindowResponseObject {
		return apigen.AddMonitorToMaintenanceWindow404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.AddMonitorToMaintenanceWindowResponseObject {
		return apigen.AddMonitorToMaintenanceWindow500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var removeMonitorFromMaintenanceWindowResponseFactory = v1.ResponseFactory[apigen.RemoveMonitorFromMaintenanceWindowResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.RemoveMonitorFromMaintenanceWindowResponseObject {
		return apigen.RemoveMonitorFromMaintenanceWindow400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.RemoveMonitorFromMaintenanceWindowResponseObject {
		return apigen.RemoveMonitorFromMaintenanceWindow401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.RemoveMonitorFromMaintenanceWindowResponseObject {
		return apigen.RemoveMonitorFromMaintenanceWindow403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.RemoveMonitorFromMaintenanceWindowResponseObject {
		return apigen.RemoveMonitorFromMaintenanceWindow404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.RemoveMonitorFromMaintenanceWindowResponseObject {
		return apigen.RemoveMonitorFromMaintenanceWindow500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var updateMaintenanceWindowResponseFactory = v1.ResponseFactory[apigen.UpdateMaintenanceWindowResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.UpdateMaintenanceWindowResponseObject {
		return apigen.UpdateMaintenanceWindow400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.UpdateMaintenanceWindowResponseObject {
		return apigen.UpdateMaintenanceWindow401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.UpdateMaintenanceWindowResponseObject {
		return apigen.UpdateMaintenanceWindow403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.UpdateMaintenanceWindowResponseObject {
		return apigen.UpdateMaintenanceWindow404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.UpdateMaintenanceWindowResponseObject {
		return apigen.UpdateMaintenanceWindow500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

func mapDomainMaintenanceWindowToAPI(mw *maintenance.MaintenanceWindow) (apigen.MaintenanceWindow, error) {
	apiWindow := apigen.MaintenanceWindow{
		Id:          mw.ID,
		Title:       mw.Title,
		Description: new(mw.Description),
	}

	switch cfg := mw.Config.(type) {
	case maintenance.OneTimeMaintenanceWindowConfig:
		apiCfg := apigen.OneTimeMaintenanceConfig{
			StartTime: cfg.StartTime,
			EndTime:   cfg.EndTime,
		}
		if err := apiWindow.Config.FromOneTimeMaintenanceConfig(apiCfg); err != nil {
			return apigen.MaintenanceWindow{}, err
		}
	case maintenance.ManualMaintenanceWindowConfig:
		apiCfg := apigen.ManualMaintenanceConfig{
			IsActive: cfg.Active,
		}
		if err := apiWindow.Config.FromManualMaintenanceConfig(apiCfg); err != nil {
			return apigen.MaintenanceWindow{}, err
		}
	default:
		return apigen.MaintenanceWindow{}, fmt.Errorf("unsupported maintenance window config type: %T", mw.Config)
	}

	return apiWindow, nil
}

func successResponse(message string) apigen.SuccessResponse {
	success := true
	if message == "" {
		return apigen.SuccessResponse{Success: &success}
	}
	return apigen.SuccessResponse{Success: &success, Message: &message}
}

func (a *ApiHandler) GetMaintenanceWindowsList(ctx context.Context, request apigen.GetMaintenanceWindowsListRequestObject) (apigen.GetMaintenanceWindowsListResponseObject, error) {
	_ = request
	windows, err := a.maintenanceSvc.GetAllMaintenanceWindows(ctx)
	if err != nil {
		return v1.ResponseFromFactory(getMaintenanceWindowsListResponseFactory, err), nil
	}

	windowsList := apigen.GetMaintenanceWindowsList200JSONResponse{}
	for i := range windows {
		apiWindow, mapErr := mapDomainMaintenanceWindowToAPI(&windows[i])
		if mapErr != nil {
			return v1.ResponseFromFactory(getMaintenanceWindowsListResponseFactory, mapErr), nil
		}
		windowsList = append(windowsList, apiWindow)
	}

	return windowsList, nil
}

func (a *ApiHandler) CreateMaintenanceWindow(ctx context.Context, request apigen.CreateMaintenanceWindowRequestObject) (apigen.CreateMaintenanceWindowResponseObject, error) {
	if request.Body == nil {
		return createMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", "request body is required")), nil
	}
	var mw *maintenance.MaintenanceWindow
	var createErr error

	description := ""
	if request.Body.Description != nil {
		description = *request.Body.Description
	}

	switch discriminator, discErr := request.Body.Config.Discriminator(); {
	case discErr != nil:
		return createMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", discErr.Error())), nil
	case discriminator == string(maintenance.WindowTypeOneTime):
		cfg, err := request.Body.Config.AsOneTimeMaintenanceConfig()
		if err != nil {
			return createMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", err.Error())), nil
		}
		mw, createErr = a.maintenanceSvc.CreateOneTimeMaintenanceWindow(ctx, maintenancesvc.CreateOneTimeMaintenanceWindowDTO{
			Title:       request.Body.Title,
			Description: description,
			StartTime:   cfg.StartTime,
			EndTime:     cfg.EndTime,
		})
	case discriminator == string(maintenance.WindowTypeManual):
		mw, createErr = a.maintenanceSvc.CreateManualMaintenanceWindow(ctx, maintenancesvc.CreateManualMaintenanceWindowDTO{
			Title:       request.Body.Title,
			Description: description,
		})

	default:
		return createMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", "unsupported maintenance window type")), nil
	}

	if createErr != nil {
		return v1.ResponseFromFactory(createMaintenanceWindowResponseFactory, createErr), nil
	}

	apiWindow, mapErr := mapDomainMaintenanceWindowToAPI(mw)
	if mapErr != nil {
		return v1.ResponseFromFactory(createMaintenanceWindowResponseFactory, mapErr), nil
	}

	return apigen.CreateMaintenanceWindow201JSONResponse(apiWindow), nil
}

func (a *ApiHandler) DeleteMaintenanceWindow(ctx context.Context, request apigen.DeleteMaintenanceWindowRequestObject) (apigen.DeleteMaintenanceWindowResponseObject, error) {
	if err := a.maintenanceSvc.DeleteMaintenanceWindow(ctx, request.WindowId); err != nil {
		return v1.ResponseFromFactory(deleteMaintenanceWindowResponseFactory, err), nil
	}

	return apigen.DeleteMaintenanceWindow204Response{}, nil
}

func (a *ApiHandler) GetMaintenanceWindowDetails(ctx context.Context, request apigen.GetMaintenanceWindowDetailsRequestObject) (apigen.GetMaintenanceWindowDetailsResponseObject, error) {
	mw, err := a.maintenanceSvc.GetMaintenanceWindow(ctx, request.WindowId)
	if err != nil {
		return v1.ResponseFromFactory(getMaintenanceWindowDetailsResponseFactory, err), nil
	}

	apiWindow, mapErr := mapDomainMaintenanceWindowToAPI(mw)
	if mapErr != nil {
		return v1.ResponseFromFactory(getMaintenanceWindowDetailsResponseFactory, mapErr), nil
	}

	return apigen.GetMaintenanceWindowDetails200JSONResponse(apiWindow), nil
}

func (a *ApiHandler) UpdateMaintenanceWindow(ctx context.Context, request apigen.UpdateMaintenanceWindowRequestObject) (apigen.UpdateMaintenanceWindowResponseObject, error) {
	if request.Body == nil {
		return updateMaintenanceWindowResponseFactory[500](errorResponse("INVALID_DATA", "request body is required")), nil
	}

	var cfgUpdate maintenance.MaintenanceWindowConfigUpdate

	switch windowType, discErr := request.Body.Config.Discriminator(); {
	case discErr != nil:
		return updateMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", discErr.Error())), nil
	case windowType == string(maintenance.WindowTypeOneTime):
		cfg, err := request.Body.Config.AsOneTimeMaintenanceConfig()
		if err != nil {
			return v1.ResponseFromFactory(updateMaintenanceWindowResponseFactory, err), nil
		}
		cfgUpdate = maintenance.OneTimeConfigUpdate{
			StartTime: new(cfg.StartTime),
			EndTime:   new(cfg.EndTime),
		}
	case windowType == string(maintenance.WindowTypeManual):
		cfg, err := request.Body.Config.AsManualMaintenanceConfig()
		if err != nil {
			return v1.ResponseFromFactory(updateMaintenanceWindowResponseFactory, err), nil
		}
		cfgUpdate = maintenance.ManualConfigUpdate{
			Active: new(cfg.IsActive),
		}
	default:
		return updateMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", "unsupported maintenance window type")), nil
	}

	if err := a.maintenanceSvc.UpdateMaintenanceWindow(ctx, maintenancesvc.UpdateMaintenanceWindowDTO{
		WindowID:     request.WindowId,
		Title:        request.Body.Title,
		Description:  request.Body.Description,
		ConfigUpdate: cfgUpdate,
	}); err != nil {
		return v1.ResponseFromFactory(updateMaintenanceWindowResponseFactory, err), nil
	}

	mw, err := a.maintenanceSvc.GetMaintenanceWindow(ctx, request.WindowId)
	if err != nil {
		return v1.ResponseFromFactory(updateMaintenanceWindowResponseFactory, err), nil
	}

	apiWindow, mapErr := mapDomainMaintenanceWindowToAPI(mw)
	if mapErr != nil {
		return v1.ResponseFromFactory(updateMaintenanceWindowResponseFactory, mapErr), nil
	}

	return apigen.UpdateMaintenanceWindow200JSONResponse(apiWindow), nil
}

func (a *ApiHandler) AddMonitorToMaintenanceWindow(ctx context.Context, request apigen.AddMonitorToMaintenanceWindowRequestObject) (apigen.AddMonitorToMaintenanceWindowResponseObject, error) {
	if request.Body == nil {
		return addMonitorToMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", "monitor_id is required")), nil
	}

	if err := a.maintenanceSvc.AddMonitorToMaintenanceWindow(ctx, request.Body.MonitorId, request.WindowId); err != nil {
		return v1.ResponseFromFactory(addMonitorToMaintenanceWindowResponseFactory, err), nil
	}

	return apigen.AddMonitorToMaintenanceWindow200JSONResponse(successResponse("monitor added to maintenance window")), nil
}

func (a *ApiHandler) RemoveMonitorFromMaintenanceWindow(ctx context.Context, request apigen.RemoveMonitorFromMaintenanceWindowRequestObject) (apigen.RemoveMonitorFromMaintenanceWindowResponseObject, error) {
	if request.Body == nil {
		return removeMonitorFromMaintenanceWindowResponseFactory[400](errorResponse("INVALID_DATA", "monitor_id is required")), nil
	}

	if err := a.maintenanceSvc.RemoveMonitorFromMaintenanceWindow(ctx, request.Body.MonitorId, request.WindowId); err != nil {
		return v1.ResponseFromFactory(removeMonitorFromMaintenanceWindowResponseFactory, err), nil
	}

	return apigen.RemoveMonitorFromMaintenanceWindow200JSONResponse(successResponse("monitor removed from maintenance window")), nil
}
