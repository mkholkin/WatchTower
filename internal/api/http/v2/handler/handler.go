package handler

import (
	apiv2gen "WatchTower/internal/api/http/v2/gen"
	authsvc "WatchTower/internal/service/auth"
	contactssvc "WatchTower/internal/service/contacts"
	maintenancesvc "WatchTower/internal/service/maintenance"
	metricssvc "WatchTower/internal/service/metrics"
	monitorsvc "WatchTower/internal/service/monitoring_management"
)

var _ apiv2gen.StrictServerInterface = (*APIHandler)(nil)

type APIHandler struct {
	authSvc          authsvc.AuthService
	monitoringSvc    monitorsvc.MonitoringManagementService
	alertContactsSvc contactssvc.ContactService
	maintenanceSvc   maintenancesvc.MaintenanceService
	metricsSvc       metricssvc.MetricQueryService
}

func NewApiHandler(
	authSvc authsvc.AuthService,
	monitoringSvc monitorsvc.MonitoringManagementService,
	alertContactsSvc contactssvc.ContactService,
	maintenanceSvc maintenancesvc.MaintenanceService,
	metricsSvc metricssvc.MetricQueryService,
) *APIHandler {
	return &APIHandler{
		authSvc:          authSvc,
		monitoringSvc:    monitoringSvc,
		alertContactsSvc: alertContactsSvc,
		maintenanceSvc:   maintenanceSvc,
		metricsSvc:       metricsSvc,
	}
}
