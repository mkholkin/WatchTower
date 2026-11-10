package observability

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func TestRequestAndDatabaseTraceAreCorrelatedAndRedacted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	recorder, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.ContextWithFallback = true
	router.Use(Middleware(recorder, slog.New(slog.NewJSONHandler(io.Discard, nil)), true))
	router.GET("/items/:id", func(c *gin.Context) {
		tracer := QueryTracer{Recorder: recorder}
		ctx := tracer.TraceQueryStart(c, nil, pgx.TraceQueryStartData{SQL: "SELECT secret FROM accounts WHERE token='private-value'", Args: []any{"private-password"}})
		tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		c.Status(200)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/items/private-id?token=private-query", nil))
	if err := recorder.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private") || strings.Contains(string(data), "accounts") {
		t.Fatal("trace leaked input")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("spans=%d want2", len(lines))
	}
	var child, parent Span
	if err := json.Unmarshal([]byte(lines[0]), &child); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &parent); err != nil {
		t.Fatal(err)
	}
	if child.ParentID != parent.SpanID || child.TraceID != parent.TraceID || parent.Name != "GET /items/:id" || child.Name != "postgres SELECT" {
		t.Fatal("incorrect trace structure")
	}
	if parent.DurationNS < child.DurationNS || parent.Status != 200 || response.Header().Get("X-Trace-ID") != parent.TraceID {
		t.Fatal("incorrect timing/status/correlation")
	}
}
func TestDisabledTracingCreatesNoSpans(t *testing.T) {
	var recorder *Recorder
	ctx, span := recorder.Start(context.Background(), "disabled")
	if span != nil || ctx == nil {
		t.Fatal("disabled tracer allocated span")
	}
	recorder.Finish(span, "ignored", 200)
}

func TestExtendedLogsAreOptionalAndExcludeRequestSecrets(t *testing.T) {
	for _, extended := range []bool{false, true} {
		var output strings.Builder
		logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
		router := gin.New()
		router.Use(Middleware(nil, logger, extended))
		router.POST("/login", func(c *gin.Context) { c.Status(401) })
		request := httptest.NewRequest("POST", "/login?token=private-token", strings.NewReader("private-password"))
		request.Header.Set("Authorization", "Bearer private-token")
		router.ServeHTTP(httptest.NewRecorder(), request)
		text := output.String()
		if extended && (!strings.Contains(text, "HTTP request completed") || !strings.Contains(text, "401")) {
			t.Fatal("missing diagnostic log")
		}
		if !extended && text != "" {
			t.Fatal("default mode emitted extended log")
		}
		if strings.Contains(text, "private") {
			t.Fatal("request secrets leaked")
		}
	}
}
