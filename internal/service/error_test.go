package service_test

import (
	service "WatchTower/internal/service"
	"WatchTower/internal/testutil"
	"errors"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

// AAA: each Case arranges fixtures, performs one production operation, then asserts observable behavior.

func TestErrorPreservesServiceAndInnerErrors(t *testing.T) {
	testutil.Case(t, "service", "NewError/Error", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
		var inner, err error
		a.Step("Arrange two independent causes", func(*allure.Context) {
			inner = errors.New("database unavailable")
		})
		a.Step("Act: create the service error", func(*allure.Context) {
			err = service.NewError(service.ErrNotFound, inner)
		})
		a.Step("Assert: both causes remain observable", func(*allure.Context) {
			if !errors.Is(err, service.ErrNotFound) || !errors.Is(err, inner) {
				t.Fatalf("errors.Is cannot reach wrapped errors: %v", err)
			}
			if text := err.Error(); !strings.Contains(text, service.ErrNotFound.Error()) || !strings.Contains(text, inner.Error()) {
				t.Fatalf("Error() = %q, want both causes", text)
			}
			causes := err.(interface{ Unwrap() []error }).Unwrap()
			if len(causes) != 2 || causes[0] == nil || causes[1] == nil {
				t.Fatalf("Unwrap() = %#v, want two non-nil causes", causes)
			}
		})
	})
}

func TestErrorSupportsSingleCause(t *testing.T) {
	testutil.Case(t, "service", "NewError/Error", "boundary", "classic", func(t *testing.T, a *allure.Context) {
		var serviceCause, err error
		a.Step("Arrange a single service cause", func(*allure.Context) {
			serviceCause = service.ErrInvalidData
		})
		a.Step("Act: create the service error", func(*allure.Context) {
			err = service.NewError(serviceCause, nil)
		})
		a.Step("Assert: the service cause remains observable", func(*allure.Context) {
			if !errors.Is(err, service.ErrInvalidData) || err.Error() != service.ErrInvalidData.Error() {
				t.Fatalf("single-cause error = %q", err)
			}
			causes := err.(interface{ Unwrap() []error }).Unwrap()
			if len(causes) != 1 || causes[0] == nil {
				t.Fatalf("Unwrap() = %#v, want one non-nil cause", causes)
			}
		})
	})
}

func TestErrorSupportsZeroCauses(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "zero value", err: service.Error{}},
		{name: "constructor with nil causes", err: service.NewError(nil, nil)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "service", "NewError/Error", "boundary", "classic", func(t *testing.T, a *allure.Context) {
				var text string
				a.Step("Arrange a service error without causes", func(*allure.Context) {})
				a.Step("Act: render the error", func(*allure.Context) { text = tc.err.Error() })
				a.Step("Assert: rendering and unwrapping are zero-safe", func(*allure.Context) {
					if text != "" {
						t.Fatalf("Error() = %q, want empty string", text)
					}
					causes := tc.err.(interface{ Unwrap() []error }).Unwrap()
					if len(causes) != 0 {
						t.Fatalf("Unwrap() = %#v, want no causes", causes)
					}
				})
			})
		})
	}
}
