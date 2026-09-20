package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1handler "WatchTower/internal/api/http/v1/handler"
	v2gen "WatchTower/internal/api/http/v2/gen"
	v2handler "WatchTower/internal/api/http/v2/handler"
)

func TestSetupRouterRegistersVersionedAPIsAndSwagger(t *testing.T) {
	router := SetupRouter(
		v1handler.NewApiHandler(nil, nil, nil, nil, nil),
		v2handler.NewApiHandler(nil, nil, nil, nil, nil),
	)

	wantedRoutes := map[string]bool{
		"POST /api/v1/auth/login":  false,
		"GET /api/v1/monitors":     false,
		"POST /api/v2/auth/tokens": false,
		"GET /api/v2/monitors":     false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := wantedRoutes[key]; ok {
			wantedRoutes[key] = true
		}
	}
	for route, found := range wantedRoutes {
		if !found {
			t.Errorf("route %s is not registered", route)
		}
	}

	for _, path := range []string{"/swagger", "/openapi.yaml"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, response.Code)
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `url: "/openapi.yaml"`) {
		t.Fatal("Swagger UI does not load the canonical OpenAPI document")
	}
}

func TestV2RejectsMissingTokenAndMalformedJSONWithContractErrors(t *testing.T) {
	router := SetupRouter(
		v1handler.NewApiHandler(nil, nil, nil, nil, nil),
		v2handler.NewApiHandler(nil, nil, nil, nil, nil),
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v2/monitors", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/v2/monitors status = %d, want 401", response.Code)
	}
	if response.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q, want Bearer", response.Header().Get("WWW-Authenticate"))
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v2/users", strings.NewReader(`{"login":`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/v2/users status = %d, want 400", response.Code)
	}
	var body v2gen.Error
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != "invalid_request" {
		t.Fatalf("error code = %q, want invalid_request", body.Code)
	}
}

func TestV2RejectsUnknownJSONProperties(t *testing.T) {
	router := SetupRouter(
		v1handler.NewApiHandler(nil, nil, nil, nil, nil),
		v2handler.NewApiHandler(nil, nil, nil, nil, nil),
	)
	request := httptest.NewRequest(http.MethodPost, "/api/v2/users", strings.NewReader(`{"login":"alice","password":"correct-horse","admin":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/v2/users status = %d, want 400", response.Code)
	}
}
