package handler

import (
	apigen "WatchTower/internal/api/http/v1/gen"
	authsvc "WatchTower/internal/service/auth"
	contactssvc "WatchTower/internal/service/contacts"
	maintenancesvc "WatchTower/internal/service/maintenance"
	metricssvc "WatchTower/internal/service/metrics"
	monitorsvc "WatchTower/internal/service/monitoring_management"
	"log/slog"
)

var _ apigen.StrictServerInterface = (*ApiHandler)(nil)

type ApiHandler struct {
	authSvc          authsvc.AuthService
	monitoringSvc    monitorsvc.MonitoringManagementService
	alertContactsSvc contactssvc.ContactService
	maintenanceSvc   maintenancesvc.MaintenanceService
	metricsSvc       metricssvc.MetricQueryService
	log              *slog.Logger
}

func NewApiHandler(
	authSvc authsvc.AuthService,
	monitoringSvc monitorsvc.MonitoringManagementService,
	alertContactsSvc contactssvc.ContactService,
	maintenanceSvc maintenancesvc.MaintenanceService,
	metricsSvc metricssvc.MetricQueryService,
) *ApiHandler {
	return &ApiHandler{
		authSvc:          authSvc,
		monitoringSvc:    monitoringSvc,
		alertContactsSvc: alertContactsSvc,
		maintenanceSvc:   maintenanceSvc,
		metricsSvc:       metricsSvc,
	}
}

func errorResponse(code, message string) apigen.ErrorResponse {
	return apigen.ErrorResponse{Code: code, Message: message}
}
