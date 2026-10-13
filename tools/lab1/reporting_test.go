package lab1_test

import (
	"WatchTower/internal/testutil"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A fixture cleanup failure must never appear as a passing Allure case.
func TestAllureIncludesFixtureCleanupFailures(t *testing.T) {
	if os.Getenv("WATCHTOWER_REPORT_PROBE") == "1" {
		testutil.Case(t, "reporting-probe", "Cleanup", "equivalence", "classic", func(t *testing.T, _ *allure.Context) {
			t.Cleanup(func() { t.Error("deliberate fixture cleanup failure") })
		})
		return
	}
	dir := t.TempDir()
	binary, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(binary, "-test.run=^TestAllureIncludesFixtureCleanupFailures$", "-test.count=1")
	cmd.Env = append(os.Environ(), "WATCHTOWER_REPORT_PROBE=1", "ALLURE_RESULTS_DIR="+dir)
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, string(output))
	require.Equal(t, 1, exit.ExitCode(), string(output))
	paths, err := filepath.Glob(filepath.Join(dir, "*-result.json"))
	require.NoError(t, err)
	require.Len(t, paths, 1)
	data, err := os.ReadFile(paths[0])
	require.NoError(t, err)
	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	require.Equal(t, "failed", result.Status)
}
