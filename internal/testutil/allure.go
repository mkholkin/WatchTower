package testutil

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
)

// Case adds lab metadata to an ordinary Go test using the official Allure adapter.
// Each caller owns its fixtures and Arrange/Act/Assert steps.
func Case(t *testing.T, component, method, technique, style string, body func(*testing.T, *allure.Context)) {
	t.Helper()
	pc, _, _, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc).Name()
	packageEnd := strings.LastIndex(caller, "/") + 1
	packageEnd += strings.Index(caller[packageEnd:], ".")
	packageName := strings.TrimSuffix(caller[:packageEnd], "_test")
	allure.Wrap(t, func(a *allure.Context) {
		// gotest.Wrap finalizes on return, before this test's Cleanup callbacks.
		// A native child test runs all fixture cleanup before Allure finalizes,
		// including gomock verification failures and resource cleanup failures.
		a.Step("Scenario (including fixture cleanup)", func(*allure.Context) {
			t.Run("body", func(child *testing.T) { body(child, a) })
		})
	}, allure.WithParentSuite("WatchTower — ЛР №1"), allure.WithSuite(component),
		allure.WithParameterOptions("pid", strconv.Itoa(os.Getpid()), allure.ParameterOptions{Excluded: true}),
		allure.WithParameterOptions("seed", os.Getenv("LAB1_SEED"), allure.ParameterOptions{Excluded: true}),
		allure.WithPackage(packageName), allure.WithTestCaseID(packageName+"/"+t.Name()),
		allure.WithSubSuite(method), allure.WithLabel("technique", technique),
		allure.WithLabel("style", style), allure.WithTag(technique), allure.WithTag(style))
}
