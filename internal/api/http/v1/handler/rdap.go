package handler

import (
	v1 "WatchTower/internal/api/http/v1"
	"WatchTower/internal/infra/rdap"
	"WatchTower/internal/service"
	authsvc "WatchTower/internal/service/auth"
	monitorsvc "WatchTower/internal/service/monitoring_management"
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DomainLookup interface {
	Lookup(context.Context, string) (rdap.Registration, error)
}
type rdapHandler struct {
	auth     authsvc.AuthService
	monitors monitorsvc.MonitoringManagementService
	client   DomainLookup
}

func RegisterRDAPRoutes(router gin.IRouter, auth authsvc.AuthService, monitors monitorsvc.MonitoringManagementService, client DomainLookup) {
	h := rdapHandler{auth: auth, monitors: monitors, client: client}
	router.GET("/monitors/:monitorId/registration", h.registration)
}

func (h rdapHandler) authenticate(c *gin.Context) (context.Context, bool) {
	scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
	if !ok || scheme != "Bearer" || token == "" {
		rdapServiceError(c, service.ErrUnauthorized)
		return nil, false
	}
	login, err := h.auth.ParseToken(token)
	if err != nil || login == "" {
		rdapServiceError(c, service.ErrUnauthorized)
		return nil, false
	}
	return authsvc.ContextWithUser(c.Request.Context(), login), true
}

func (h rdapHandler) registration(c *gin.Context) {
	ctx, ok := h.authenticate(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("monitorId"))
	if err != nil {
		rdapServiceError(c, service.ErrInvalidData)
		return
	}
	mon, err := h.monitors.GetMonitor(ctx, id)
	if err != nil {
		rdapServiceError(c, err)
		return
	}
	if mon == nil || mon.Target == nil {
		rdapServiceError(c, service.ErrNotFound)
		return
	}
	h.lookup(c, ctx, mon.Target.Endpoint)
}

func (h rdapHandler) lookup(c *gin.Context, ctx context.Context, endpoint string) {
	if h.client == nil {
		c.JSON(503, errorResponse("RDAP_DISABLED", "registration lookup disabled"))
		return
	}
	result, err := h.client.Lookup(ctx, endpoint)
	if err != nil {
		rdapLookupError(c, err)
		return
	}
	c.JSON(200, result)
}

func rdapServiceError(c *gin.Context, err error) {
	response := v1.ResponseFromError(err)
	if response.Code == 500 {
		response.Body = errorResponse("INTERNAL_SERVER_ERROR", "monitor unavailable")
	}
	c.JSON(response.Code, response.Body)
}

func rdapLookupError(c *gin.Context, err error) {
	status := 502
	switch {
	case errors.Is(err, rdap.ErrInvalidDomain):
		status = 400
	case errors.Is(err, rdap.ErrNotFound):
		status = 404
	case errors.Is(err, rdap.ErrRateLimited):
		status = 503
	case errors.Is(err, rdap.ErrTimeout):
		status = 504
	}
	c.JSON(status, errorResponse("RDAP_ERROR", rdapMessage(err)))
}

func rdapMessage(err error) string {
	for _, known := range []error{rdap.ErrInvalidDomain, rdap.ErrNotFound, rdap.ErrRateLimited, rdap.ErrTimeout} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return rdap.ErrUnavailable.Error()
}
