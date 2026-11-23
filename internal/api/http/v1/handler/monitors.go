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
		return apigen.GetMonitorsList401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMonitorsListResponseObject {
		return apigen.GetMonitorsList500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var createMonitorResponseFactory = v1.ResponseFactory[apigen.CreateMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.CreateMonitorResponseObject {
		return apigen.CreateMonitor400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.CreateMonitorResponseObject {
		return apigen.CreateMonitor401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.CreateMonitorResponseObject {
		return apigen.CreateMonitor500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var deleteMonitorResponseFactory = v1.ResponseFactory[apigen.DeleteMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.DeleteMonitorResponseObject {
		return apigen.DeleteMonitor500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var getMonitorDetailsResponseFactory = v1.ResponseFactory[apigen.GetMonitorDetailsResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.GetMonitorDetailsResponseObject {
		return apigen.GetMonitorDetails500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var updateMonitorResponseFactory = v1.ResponseFactory[apigen.UpdateMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.UpdateMonitorResponseObject {
		return apigen.UpdateMonitor500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var removeAlertContactFromMonitorResponseFactory = v1.ResponseFactory[apigen.RemoveAlertContactFromMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.RemoveAlertContactFromMonitorResponseObject {
		return apigen.RemoveAlertContactFromMonitor500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var addAlertContactToMonitorResponseFactory = v1.ResponseFactory[apigen.AddAlertContactToMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.AddAlertContactToMonitorResponseObject {
		return apigen.AddAlertContactToMonitor500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var disableMonitorResponseFactory = v1.ResponseFactory[apigen.DisableMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.DisableMonitorResponseObject {
		return apigen.DisableMonitor500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
	},
}

var enableMonitorResponseFactory = v1.ResponseFactory[apigen.EnableMonitorResponseObject]{
	400: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor400JSONResponse{N400JSONResponse: apigen.N400JSONResponse(er)}
	},
	401: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor401JSONResponse{N401JSONResponse: apigen.N401JSONResponse(er)}
	},
	403: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor403JSONResponse{N403JSONResponse: apigen.N403JSONResponse(er)}
	},
	404: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor404JSONResponse{N404JSONResponse: apigen.N404JSONResponse(er)}
	},
	500: func(er apigen.ErrorResponse) apigen.EnableMonitorResponseObject {
		return apigen.EnableMonitor500JSONResponse{N500JSONResponse: apigen.N500JSONResponse(er)}
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

	responseList := apigen.GetMonitorsList200JSONResponse{}
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
		networkConfig = mapHTTPMonitorNetworkConfig(apiNetConf)
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
	if hasNetworkConfig != hasExpectations {
		return updateMonitorResponseFactory[400](errorResponse("INVALID_DATA", "network_config and expectations are required together")), nil
	}
	var networkConfig *dto.MonitorNetworkConfig
	var expectations *dto.MonitorExpectations
	if hasNetworkConfig {
		var response apigen.UpdateMonitorResponseObject
		networkConfig, expectations, response = decodeMonitorUpdateConfig(request)
		if response != nil {
			return response, nil
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

func mapHTTPMonitorNetworkConfig(config apigen.HTTPConfig) dto.MonitorNetworkConfig {
	headers := make(map[string]string)
	if config.Headers != nil {
		headers = *config.Headers
	}
	body := ""
	if config.Body != nil {
		body = *config.Body
	}
	return dto.HTTPMonitorNetworkConfig{Method: string(config.Method), Headers: headers, Body: body, FollowRedirects: config.FollowRedirects}
}

func decodeMonitorUpdateConfig(request apigen.UpdateMonitorRequestObject) (*dto.MonitorNetworkConfig, *dto.MonitorExpectations, apigen.UpdateMonitorResponseObject) {
	var networkConfig *dto.MonitorNetworkConfig
	var expectations *dto.MonitorExpectations
	expDisc, err := request.Body.Expectations.Discriminator()
	if err != nil {
		return nil, nil, v1.ResponseFromFactory(updateMonitorResponseFactory, err)
	}
	cfgDisc, err := request.Body.NetworkConfig.Discriminator()
	if err != nil {
		return nil, nil, v1.ResponseFromFactory(updateMonitorResponseFactory, err)
	}
	if expDisc != cfgDisc {
		return nil, nil, updateMonitorResponseFactory[400](errorResponse("INVALID_DATA", "expectations and network config types are mismatching"))
	}

	tgtProtocol := cfgDisc

	switch tgtProtocol {
	case string(apigen.HTTP):
		apiNetConf, err := request.Body.NetworkConfig.AsHTTPConfig()
		if err != nil {
			return nil, nil, v1.ResponseFromFactory(updateMonitorResponseFactory, err)
		}
		config := mapHTTPMonitorNetworkConfig(apiNetConf)
		networkConfig = &config

		apiExp, err := request.Body.Expectations.AsHTTPExpectations()
		if err != nil {
			return nil, nil, v1.ResponseFromFactory(updateMonitorResponseFactory, err)
		}
		expectations = new(dto.MonitorExpectations(dto.HTTPMonitorExpectations{
			StatusCodes:  apiExp.ExpectedStatusCodes,
			MaxLatencyMs: apiExp.ExpectedResponseTimeMs,
		}))
	default:
		return nil, nil, updateMonitorResponseFactory[400](errorResponse("INVALID_DATA", "unsupported monitor protocol"))
	}
	return networkConfig, expectations, nil
}
