package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"WatchTower/internal/observability"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestSetupRouterCorrelatesDatabaseSpanThroughGinContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	recorder, err := observability.New(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := recorder.Close(); err != nil {
			t.Error(err)
		}
	})
	router := SetupRouter(nil, observability.Middleware(recorder, slog.New(slog.NewJSONHandler(io.Discard, nil)), false))
	router.GET("/trace-regression", func(c *gin.Context) {
		tracer := observability.QueryTracer{Recorder: recorder}
		// The generated strict handlers pass *gin.Context to application services.
		ctx := tracer.TraceQueryStart(c, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
		tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		c.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/trace-regression", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	decoder := json.NewDecoder(file)
	var child, parent observability.Span
	require.NoError(t, decoder.Decode(&child))
	require.NoError(t, decoder.Decode(&parent))
	require.Equal(t, "postgres SELECT", child.Name)
	require.Equal(t, "GET /trace-regression", parent.Name)
	require.Equal(t, parent.TraceID, child.TraceID)
	require.Equal(t, parent.SpanID, child.ParentID)
	require.Equal(t, parent.TraceID, response.Header().Get("X-Trace-ID"))
}
