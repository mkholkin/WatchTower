package handler

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/infra/rdap"
	"WatchTower/internal/service"
	authsvc "WatchTower/internal/service/auth"
	monitorsvc "WatchTower/internal/service/monitoring_management"
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type rdapAuthStub struct{ authsvc.AuthService }

func (rdapAuthStub) ParseToken(token string) (string, error) {
	if token != "valid" {
		return "", errors.New("invalid token")
	}
	return "alice", nil
}

type rdapMonitorStub struct {
	monitorsvc.MonitoringManagementService
	err   error
	calls int
}

func (s *rdapMonitorStub) GetMonitor(ctx context.Context, id uuid.UUID) (*monitor.Monitor, error) {
	s.calls++
	if login, ok := authsvc.UserFromContext(ctx); !ok || login != "alice" {
		return nil, service.ErrUnauthorized
	}
	return &monitor.Monitor{Target: &target.Target{Endpoint: "https://probe.example.com/health"}}, s.err
}

type rdapLookupStub struct {
	err   error
	calls int
}

func (s *rdapLookupStub) Lookup(ctx context.Context, endpoint string) (rdap.Registration, error) {
	s.calls++
	return rdap.Registration{Domain: "example.com"}, s.err
}

func TestRDAPRouteAuthenticationAndOwnership(t *testing.T) {
	cases := []struct {
		name, token, id                   string
		err                               error
		status, monitorCalls, lookupCalls int
	}{
		{"missing token", "", uuid.NewString(), nil, 401, 0, 0},
		{"invalid token", "Bearer invalid", uuid.NewString(), nil, 401, 0, 0},
		{"invalid id", "Bearer valid", "bad", nil, 400, 0, 0},
		{"forbidden", "Bearer valid", uuid.NewString(), service.ErrPermissionDenied, 403, 1, 0},
		{"missing monitor", "Bearer valid", uuid.NewString(), service.ErrNotFound, 404, 1, 0},
		{"success", "Bearer valid", uuid.NewString(), nil, 200, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			monitors := &rdapMonitorStub{err: tc.err}
			lookup := &rdapLookupStub{}
			router := gin.New()
			RegisterRDAPRoutes(router, rdapAuthStub{}, monitors, lookup)
			request := httptest.NewRequest("GET", "/monitors/"+tc.id+"/registration", nil)
			request.Header.Set("Authorization", tc.token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.status || monitors.calls != tc.monitorCalls || lookup.calls != tc.lookupCalls {
				t.Fatalf("status/calls %d/%d/%d; want %d/%d/%d body=%s", response.Code, monitors.calls, lookup.calls, tc.status, tc.monitorCalls, tc.lookupCalls, response.Body)
			}
		})
	}
}

func TestRDAPRouteUpstreamErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"invalid", rdap.ErrInvalidDomain, 400}, {"missing", rdap.ErrNotFound, 404}, {"limited", rdap.ErrRateLimited, 503}, {"unavailable", rdap.ErrUnavailable, 502}, {"timeout", rdap.ErrTimeout, 504},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			RegisterRDAPRoutes(router, rdapAuthStub{}, &rdapMonitorStub{}, &rdapLookupStub{err: tc.err})
			response := rdapRequest(router)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body)
			}
		})
	}
}

func TestRDAPRouteDisabled(t *testing.T) {
	router := gin.New()
	RegisterRDAPRoutes(router, rdapAuthStub{}, &rdapMonitorStub{}, nil)
	if response := rdapRequest(router); response.Code != 503 {
		t.Fatalf("got %d", response.Code)
	}
}

func rdapRequest(router *gin.Engine) *httptest.ResponseRecorder {
	request := httptest.NewRequest("GET", "/monitors/"+uuid.NewString()+"/registration", nil)
	request.Header.Set("Authorization", "Bearer valid")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
