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
	"github.com/stretchr/testify/require"
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
	require.NoError(t, recorder.Close())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(data), "private")
	require.NotContains(t, string(data), "accounts")
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2)
	var child, parent Span
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &child))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &parent))
	require.Equal(t, parent.SpanID, child.ParentID)
	require.Equal(t, parent.TraceID, child.TraceID)
	require.Equal(t, "GET /items/:id", parent.Name)
	require.Equal(t, "postgres SELECT", child.Name)
	require.GreaterOrEqual(t, parent.DurationNS, child.DurationNS)
	require.Equal(t, 200, parent.Status)
	require.Equal(t, parent.TraceID, response.Header().Get("X-Trace-ID"))
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
