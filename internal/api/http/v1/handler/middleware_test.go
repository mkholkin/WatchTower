package handler

import (
	apigen "WatchTower/internal/api/http/v1/gen"
	authsvc "WatchTower/internal/service/auth"
	"WatchTower/internal/service/common/provider"
	monitorsvc "WatchTower/internal/service/monitoring_management"
	"WatchTower/internal/testutil"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/gin-gonic/gin"
)

func TestProtectedMonitorRouteRejectsInvalidAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
	}{
		{name: "missing header"},
		{name: "invalid scheme", header: "Basic credentials"},
		{name: "invalid token", header: "Bearer invalid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "http", "AuthStrictMiddleware", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				monitors := monitorsvc.NewMonitoringManagementService(nil, nil, nil, nil, provider.NewUserProvider(nil), nil, testutil.NoopLogger())
				handler := NewApiHandler(authsvc.NewService(nil, "test-secret", time.Hour), monitors, nil, nil, nil)
				router := gin.New()
				apigen.RegisterHandlers(router, apigen.NewStrictHandler(handler, []apigen.StrictMiddlewareFunc{handler.AuthStrictMiddleware}))
				request := httptest.NewRequest(http.MethodGet, "/monitors", nil)
				if tc.header != "" {
					request.Header.Set("Authorization", tc.header)
				}
				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				if response.Code != http.StatusUnauthorized {
					t.Fatalf("GET /monitors status=%d body=%s, want 401", response.Code, response.Body.String())
				}
				var body apigen.ErrorResponse
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Code != "UNAUTHORIZED" {
					t.Fatalf("error body=%#v decode error=%v, want UNAUTHORIZED", body, err)
				}
			})
		})
	}
}
