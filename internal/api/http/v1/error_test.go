package v1

import (
	"WatchTower/internal/domain/repo"
	"WatchTower/internal/service"
	"WatchTower/internal/testutil"
	"errors"
	"fmt"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

func TestResponseFromError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "bare unauthorized", err: service.ErrUnauthorized, status: 401, code: "UNAUTHORIZED"},
		{name: "wrapped unauthorized", err: fmt.Errorf("authenticate: %w", service.ErrUnauthorized), status: 401, code: "UNAUTHORIZED"},
		{name: "service unauthorized", err: service.NewError(service.ErrUnauthorized, nil), status: 401, code: "UNAUTHORIZED"},
		{name: "joined permission denied", err: errors.Join(service.ErrPermissionDenied, errors.New("foreign owner")), status: 403, code: "PERMISSION_DENIED"},
		{name: "bare not found", err: service.ErrNotFound, status: 404, code: "NOT_FOUND"},
		{name: "repository not found", err: repo.ErrNotFound, status: 404, code: "NOT_FOUND"},
		{name: "bare invalid data", err: service.ErrInvalidData, status: 400, code: "INVALID_DATA"},
		{name: "unknown failure", err: errors.New("storage failed"), status: 500, code: "INTERNAL_SERVER_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "http", "ResponseFromError", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
				got := ResponseFromError(tc.err)
				if got.Code != tc.status || got.Body.Code != tc.code {
					t.Fatalf("ResponseFromError()=%#v, want status=%d code=%s", got, tc.status, tc.code)
				}
			})
		})
	}
}
