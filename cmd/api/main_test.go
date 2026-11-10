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
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var child, parent observability.Span
	if err := decoder.Decode(&child); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&parent); err != nil {
		t.Fatal(err)
	}
	if child.Name != "postgres SELECT" || parent.Name != "GET /trace-regression" {
		t.Fatalf("unexpected spans: child=%q parent=%q", child.Name, parent.Name)
	}
	if child.TraceID != parent.TraceID || child.ParentID != parent.SpanID {
		t.Fatalf("database span is disconnected from HTTP span: child=%+v parent=%+v", child, parent)
	}
	if response.Header().Get("X-Trace-ID") != parent.TraceID {
		t.Fatal("response trace ID differs from HTTP span")
	}
}
