package handler

import (
	apigen "WatchTower/internal/api/http/v1/gen"
	authsvc "WatchTower/internal/service/auth"
	"WatchTower/internal/service/mfa"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type disabledRegistration struct{ authsvc.AuthService }

func (disabledRegistration) Register(context.Context, string, string) error {
	return authsvc.ErrInvalidCredentials
}

func TestDisabledLegacyRegistrationIsClientError(t *testing.T) {
	h := NewApiHandler(disabledRegistration{}, nil, nil, nil, nil)
	response, err := h.RegisterUser(context.Background(), apigen.RegisterUserRequestObject{
		Body: &apigen.RegisterUserJSONRequestBody{Login: "alice", Password: "synthetic-password"},
	})
	if err != nil {
		t.Fatal(err)
	}
	writer := httptest.NewRecorder()
	if err := response.VisitRegisterUserResponse(writer); err != nil {
		t.Fatal(err)
	}
	if writer.Code != http.StatusBadRequest {
		t.Fatalf("disabled registration status=%d; want400", writer.Code)
	}
}

type mfaHTTPStub struct {
	MFAService
	err   error
	calls int
}

func (s *mfaHTTPStub) BeginLogin(_ context.Context, login, password string) (string, error) {
	s.calls++
	return "opaque-challenge", s.err
}

func TestMFALoginHTTPContract(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		err           error
		status, calls int
	}{
		{"challenge only", `{"login":"alice","password":"not-a-real-password"}`, nil, 202, 1},
		{"locked", `{"login":"alice","password":"not-a-real-password"}`, mfa.ErrLocked, 423, 1},
		{"unknown field", `{"login":"alice","password":"x","admin":true}`, nil, 400, 0},
		{"extra JSON", `{"login":"alice","password":"x"}{}`, nil, 400, 0},
		{"empty body", `{}`, nil, 400, 0},
		{"oversized body", strings.Repeat("x", 9000), nil, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &mfaHTTPStub{err: tc.err}
			router := gin.New()
			RegisterMFARoutes(router, service)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/mfa/login", strings.NewReader(tc.body)))
			if response.Code != tc.status || service.calls != tc.calls {
				t.Fatalf("status=%d calls=%d; want %d/%d", response.Code, service.calls, tc.status, tc.calls)
			}
			if tc.status == 202 && (!strings.Contains(response.Body.String(), "challenge_id") || strings.Contains(response.Body.String(), "access_token")) {
				t.Fatal("first factor must return a challenge, never an access token")
			}
		})
	}
}
