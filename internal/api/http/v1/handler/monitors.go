package handler

import (
	v1 "WatchTower/internal/api/http/v1"
	apigen "WatchTower/internal/api/http/v1/gen"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/monitoring_management/dto"
	"context"
)

var getMonitorsListResponseFactory = v1.ResponseFactory[apigen.GetMonitorsListResponseObject]{
	401: func(er apigen.ErrorResponse) apigen.GetMonitorsListResponseObject {
		return apigen.GetMonitorsList401JSONResponse{apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMonitorsListResponseObject {
		return apigen.GetMonitorsList500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var createMonitorResponseFactory = v1.ResponseFactory[apigen.CreateMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.CreateMonitorResponseObject {
		return apigen.CreateMonitor400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.CreateMonitorResponseObject {
		return apigen.CreateMonitor401JSONResponse{apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.CreateMonitorResponseObject {
		return apigen.CreateMonitor500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var deleteMonitorResponseFactory = v1.ResponseFactory[apigen.DeleteMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var getMonitorDetailsResponseFactory = v1.ResponseFactory[apigen.GetMonitorDetailsResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var updateMonitorResponseFactory = v1.ResponseFactory[apigen.UpdateMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var removeAlertContactFromMonitorResponseFactory = v1.ResponseFactory[apigen.RemoveAlertContactFromMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var addAlertContactToMonitorResponseFactory = v1.ResponseFactory[apigen.AddAlertContactToMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var disableMonitorResponseFactory = v1.ResponseFactory[apigen.DisableMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

var enableMonitorResponseFactory = v1.ResponseFactory[apigen.EnableMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor400JSONResponse{apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor401JSONResponse{apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor403JSONResponse{apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor404JSONResponse{apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor500JSONResponse{apigen.N500JSONResponse(er)}
	},
}

func mapDomainMonitorToAPI(m *monitor.Monitor) apigen.Monitor {
	apiMonitor := apigen.Monitor{
		Id:                 m.ID,
		Label:              m.Label,
		Endpoint:           m.Target.Endpoint,
		ProbeInterval:      int(m.ProbeIntervalSec),
		IsEnabled:          m.IsActive,
		Status:             apigen.MonitorStatus(m.CurrentStatus),
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.LastEvaluatedAt,
		AlertContacts:      make([]apigen.AlertContact, 0),
		MaintenanceWindows: make([]apigen.MaintenanceWindow, 0),
	}

	switch config := m.Target.Config.(type) {
	case target.HTTPConfig:
		httpConfig := apigen.HTTPConfig{
			Method:          apigen.HTTPConfigMethod(config.Method),
			FollowRedirects: config.FollowRedirects,
		}
		if len(config.Headers) > 0 {
			httpConfig.Headers = &config.Headers
		}
		if config.Body != "" {
			httpConfig.Body = &config.Body
		}
		apiMonitor.NetworkConfig.FromHTTPConfig(httpConfig)
	}

	// Map Expectations based on protocol
	switch exp := m.Expectations.(type) {
	case monitor.HTTPExpectations:
		httpExp := apigen.HTTPExpectations{
			ExpectedStatusCodes: exp.StatusCodes,
		}
		httpExp.ExpectedResponseTimeMs = exp.MaxLatencyMs
		apiMonitor.Expectations.FromHTTPExpectations(httpExp)
	}

	for i := range m.AlertContacts {
		apiContact, err := mapDomainAlertContactToAPI(&m.AlertContacts[i])
		if err != nil {
			continue
		}
		apiMonitor.AlertContacts = append(apiMonitor.AlertContacts, apiContact)
	}

	for i := range m.MaintenanceWindows {
		apiMaintenanceWindow, err := mapDomainMaintenanceWindowToAPI(&m.MaintenanceWindows[i])
		if err != nil {
			continue
		}
		apiMonitor.MaintenanceWindows = append(apiMonitor.MaintenanceWindows, apiMaintenanceWindow)
	}

	return apiMonitor
}

func (a *ApiHandler) GetMonitorsList(ctx context.Context, request apigen.GetMonitorsListRequestObject) (apigen.GetMonitorsListResponseObject, error) {
	_ = request
	monitors, err := a.monitoringSvc.GetAllMonitors(ctx)
	if err != nil {
		return v1.ResponseFromFactory(getMonitorsListResponseFactory, err), nil
	}

	var responseList apigen.GetMonitorsList200JSONResponse
	for _, m := range monitors {
		responseList = append(responseList, mapDomainMonitorToAPI(m))
	}

	return responseList, nil
}

func (a *ApiHandler) CreateMonitor(ctx context.Context, request apigen.CreateMonitorRequestObject) (apigen.CreateMonitorResponseObject, error) {
	if request.Body == nil {
		return createMonitorResponseFactory[400](errorResponse("INVALID_DATA", "request body is required")), nil
	}

	expDisc, err := request.Body.Expectations.Discriminator()
	if err != nil {
		return v1.ResponseFromFactory(createMonitorResponseFactory, err), nil
	}
	cfgDisc, err := request.Body.NetworkConfig.Discriminator()
	if err != nil {
		return v1.ResponseFromFactory(createMonitorResponseFactory, err), nil
	}
	if expDisc != cfgDisc {
		return createMonitorResponseFactory[400](errorResponse("INVALID_DATA", "expectations and network config types are mismatching")), nil
	}

	tgtProtocol := cfgDisc

	var networkConfig dto.MonitorNetworkConfig
	switch tgtProtocol {
	case string(apigen.HTTP):
		apiNetConf, err := request.Body.NetworkConfig.AsHTTPConfig()
		if err != nil {
			return v1.ResponseFromFactory(createMonitorResponseFactory, err), nil
		}
		headers := make(map[string]string)
		if apiNetConf.Headers != nil {
			headers = *apiNetConf.Headers
		}
		body := ""
		if apiNetConf.Body != nil {
			body = *apiNetConf.Body
		}
		networkConfig = dto.HTTPMonitorNetworkConfig{
			Method:          string(apiNetConf.Method),
			Headers:         headers,
			Body:            body,
			FollowRedirects: apiNetConf.FollowRedirects,
		}
	default:
		return createMonitorResponseFactory[400](errorResponse("INVALID_DATA", "unsupported monitor protocol")), nil
	}

	var expectations dto.MonitorExpectations
	switch tgtProtocol {
	case string(apigen.HTTP):
		apiExp, err := request.Body.Expectations.AsHTTPExpectations()
		if err != nil {
			return v1.ResponseFromFactory(createMonitorResponseFactory, err), nil
		}
		expectations = dto.HTTPMonitorExpectations{
			StatusCodes:  apiExp.ExpectedStatusCodes,
			MaxLatencyMs: apiExp.ExpectedResponseTimeMs,
		}
	default:
		return createMonitorResponseFactory[400](errorResponse("INVALID_DATA", "unsupported monitor protocol")), nil
	}

	mon, err := a.monitoringSvc.CreateMonitor(ctx, dto.CreateMonitorDTO{
		Label:                request.Body.Label,
		Endpoint:             request.Body.Endpoint,
		ProbeIntervalSec:     int32(request.Body.ProbeInterval),
		AlertContactIDs:      request.Body.AlertContactIds,
		MaintenanceWindowIDs: request.Body.MaintenanceWindowIds,
		Expectations:         expectations,
		NetworkConfig:        networkConfig,
	})

	if err != nil {
		return v1.ResponseFromFactory(createMonitorResponseFactory, err), nil
	}

	apiMonitor := mapDomainMonitorToAPI(mon)
	return apigen.CreateMonitor201JSONResponse(apiMonitor), nil
}

func (a *ApiHandler) DeleteMonitor(ctx context.Context, request apigen.DeleteMonitorRequestObject) (apigen.DeleteMonitorResponseObject, error) {
	err := a.monitoringSvc.DeleteMonitor(ctx, request.MonitorId)
	if err != nil {
		return v1.ResponseFromFactory(deleteMonitorResponseFactory, err), nil
	}

	return apigen.DeleteMonitor204Response{}, nil
}

func (a *ApiHandler) GetMonitorDetails(ctx context.Context, request apigen.GetMonitorDetailsRequestObject) (apigen.GetMonitorDetailsResponseObject, error) {
	mon, err := a.monitoringSvc.GetMonitor(ctx, request.MonitorId)
	if err != nil {
		return v1.ResponseFromFactory(getMonitorDetailsResponseFactory, err), nil
	}

	apiMonitor := mapDomainMonitorToAPI(mon)
	return apigen.GetMonitorDetails200JSONResponse(apiMonitor), nil
}

func (a *ApiHandler) UpdateMonitor(ctx context.Context, request apigen.UpdateMonitorRequestObject) (apigen.UpdateMonitorResponseObject, error) {
	if request.Body == nil {
		return updateMonitorResponseFactory[400](errorResponse("INVALID_DATA", "request body is required")), nil
	}

	hasNetworkConfig := request.Body.NetworkConfig != nil
	hasExpectations := request.Body.Expectations != nil
	if hasNetworkConfig || hasExpectations {
		if !(hasNetworkConfig && hasExpectations) {
			return updateMonitorResponseFactory[400](errorResponse("INVALID_DATA", "network_config and expectations are required together")), nil
		}
	}

	var networkConfig *dto.MonitorNetworkConfig
	var expectations *dto.MonitorExpectations
	if hasNetworkConfig && hasExpectations {
		expDisc, err := request.Body.Expectations.Discriminator()
		if err != nil {
			return v1.ResponseFromFactory(updateMonitorResponseFactory, err), nil
		}
		cfgDisc, err := request.Body.NetworkConfig.Discriminator()
		if err != nil {
			return v1.ResponseFromFactory(updateMonitorResponseFactory, err), nil
		}
		if expDisc != cfgDisc {
			return updateMonitorResponseFactory[400](errorResponse("INVALID_DATA", "expectations and network config types are mismatching")), nil
		}

		tgtProtocol := cfgDisc

		switch tgtProtocol {
		case string(apigen.HTTP):
			apiNetConf, err := request.Body.NetworkConfig.AsHTTPConfig()
			if err != nil {
				return v1.ResponseFromFactory(updateMonitorResponseFactory, err), nil
			}
			headers := make(map[string]string)
			if apiNetConf.Headers != nil {
				headers = *apiNetConf.Headers
			}
			body := ""
			if apiNetConf.Body != nil {
				body = *apiNetConf.Body
			}

			networkConfig = new(dto.MonitorNetworkConfig(dto.HTTPMonitorNetworkConfig{
				Method:          string(apiNetConf.Method),
				Headers:         headers,
				Body:            body,
				FollowRedirects: apiNetConf.FollowRedirects,
			}))

			apiExp, err := request.Body.Expectations.AsHTTPExpectations()
			if err != nil {
				return v1.ResponseFromFactory(updateMonitorResponseFactory, err), nil
			}
			expectations = new(dto.MonitorExpectations(dto.HTTPMonitorExpectations{
				StatusCodes:  apiExp.ExpectedStatusCodes,
				MaxLatencyMs: apiExp.ExpectedResponseTimeMs,
			}))
		default:
			return updateMonitorResponseFactory[400](errorResponse("INVALID_DATA", "unsupported monitor protocol")), nil
		}
	}

	var probeIntervalSec *int32
	if request.Body.ProbeInterval != nil {
		val := int32(*request.Body.ProbeInterval)
		probeIntervalSec = new(val)
	}

	err := a.monitoringSvc.UpdateMonitor(ctx, dto.UpdateMonitorDTO{
		ID:               request.MonitorId,
		Label:            request.Body.Label,
		Endpoint:         request.Body.Endpoint,
		ProbeIntervalSec: probeIntervalSec,
		NetworkConfig:    networkConfig,
		Expectations:     expectations,
	})
	if err != nil {
		return v1.ResponseFromFactory(updateMonitorResponseFactory, err), nil
	}

	mon, err := a.monitoringSvc.GetMonitor(ctx, request.MonitorId)
	if err != nil {
		return v1.ResponseFromFactory(updateMonitorResponseFactory, err), nil
	}

	apiMonitor := mapDomainMonitorToAPI(mon)
	return apigen.UpdateMonitor200JSONResponse(apiMonitor), nil
}

func (a *ApiHandler) RemoveAlertContactFromMonitor(ctx context.Context, request apigen.RemoveAlertContactFromMonitorRequestObject) (apigen.RemoveAlertContactFromMonitorResponseObject, error) {
	if request.Body == nil {
		return removeAlertContactFromMonitorResponseFactory[400](errorResponse("INVALID_DATA", "request body is required")), nil
	}

	if err := a.monitoringSvc.UnlinkAlertContact(ctx, request.MonitorId, request.Body.AlertContactId); err != nil {
		return v1.ResponseFromFactory(removeAlertContactFromMonitorResponseFactory, err), nil
	}

	return apigen.RemoveAlertContactFromMonitor200JSONResponse{}, nil
}

func (a *ApiHandler) AddAlertContactToMonitor(ctx context.Context, request apigen.AddAlertContactToMonitorRequestObject) (apigen.AddAlertContactToMonitorResponseObject, error) {
	if request.Body == nil {
		return addAlertContactToMonitorResponseFactory[400](errorResponse("INVALID_DATA", "request body is required")), nil
	}

	if err := a.monitoringSvc.LinkAlertContact(ctx, request.MonitorId, request.Body.AlertContactId); err != nil {
		return v1.ResponseFromFactory(addAlertContactToMonitorResponseFactory, err), nil
	}

	return apigen.AddAlertContactToMonitor200JSONResponse{}, nil
}

func (a *ApiHandler) DisableMonitor(ctx context.Context, request apigen.DisableMonitorRequestObject) (apigen.DisableMonitorResponseObject, error) {
	err := a.monitoringSvc.DisableMonitor(ctx, request.MonitorId)
	if err != nil {
		return v1.ResponseFromFactory(disableMonitorResponseFactory, err), nil
	}

	return apigen.DisableMonitor200JSONResponse{}, nil
}

func (a *ApiHandler) EnableMonitor(ctx context.Context, request apigen.EnableMonitorRequestObject) (apigen.EnableMonitorResponseObject, error) {
	err := a.monitoringSvc.EnableMonitor(ctx, request.MonitorId)
	if err != nil {
		return v1.ResponseFromFactory(enableMonitorResponseFactory, err), nil
	}

	return apigen.EnableMonitor200JSONResponse{}, nil
}
