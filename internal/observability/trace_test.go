package observability

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestRequestAndDatabaseTraceAreCorrelatedAndRedacted(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	recorder := newRecorder(exporter, resource.Empty())
	router := gin.New()
	router.ContextWithFallback = true
	router.Use(Middleware(recorder, slog.New(slog.NewJSONHandler(io.Discard, nil)), false))
	router.GET("/items/:id", func(c *gin.Context) {
		tracer := QueryTracer{Recorder: recorder}
		ctx := tracer.TraceQueryStart(c, nil, pgx.TraceQueryStartData{SQL: "-- name: PrivateAccount :one\nSELECT secret FROM accounts WHERE token='private-value'", Args: []any{"private-password"}})
		tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errors.New("private-error")})
		c.Status(503)
	})
	request := httptest.NewRequest("GET", "/items/private-id?token=private-query", strings.NewReader("private-body"))
	request.Header.Set("Authorization", "Bearer private-header")
	request.Header.Set("tracestate", "vendor=private-header-value")
	request.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-00")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.NoError(t, recorder.provider.ForceFlush(context.Background()))
	spans := exporter.GetSpans()
	require.Len(t, spans, 2)
	child, parent := spans[0], spans[1]
	require.Equal(t, "postgres SELECT", child.Name)
	require.Equal(t, "GET /items/:id", parent.Name)
	require.Equal(t, parent.SpanContext.SpanID(), child.Parent.SpanID())
	require.Equal(t, parent.SpanContext.TraceID(), child.SpanContext.TraceID())
	require.Equal(t, "0123456789abcdef0123456789abcdef", parent.SpanContext.TraceID().String())
	require.Equal(t, "0123456789abcdef", parent.Parent.SpanID().String())
	require.True(t, parent.SpanContext.IsSampled())
	require.Equal(t, parent.SpanContext.TraceID().String(), response.Header().Get("X-Trace-ID"))
	require.Contains(t, parent.Attributes, attribute.Int("http.response.status_code", 503))
	require.Contains(t, child.Attributes, attribute.String("db.operation.name", "SELECT"))
	require.Contains(t, child.Attributes, attribute.String("db.system.name", "postgresql"))
	serialized, err := json.Marshal(spans)
	require.NoError(t, err)
	require.NotContains(t, string(serialized), "private-header-value")
	for _, span := range spans {
		require.Empty(t, span.SpanContext.TraceState().String())
		require.Empty(t, span.Parent.TraceState().String())
		require.NotContains(t, span.Name, "private")
		require.Empty(t, span.Events)
		for _, attr := range span.Attributes {
			require.NotContains(t, attr.Value.Emit(), "private")
		}
	}
	require.NoError(t, recorder.Shutdown(context.Background()))
	require.Equal(t, Summary{EndedSpans: 2, ExportedSpans: 2}, recorder.Summary())
}

func TestOTLPHTTPShutdownFlushesEverySpanAndResource(t *testing.T) {
	var mu sync.Mutex
	var requests []*collector.ExportTraceServiceRequest
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/traces", r.URL.Path)
		require.Equal(t, "application/x-protobuf", r.Header.Get("Content-Type"))
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		message := new(collector.ExportTraceServiceRequest)
		require.NoError(t, proto.Unmarshal(data, message))
		mu.Lock()
		requests = append(requests, message)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer receiver.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", receiver.URL+"/v1/traces")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "run.id=run1,profile=search,variant=trace,repetition=2")
	recorder, err := New(context.Background())
	require.NoError(t, err)
	for range 2200 {
		_, span := recorder.tracer.Start(context.Background(), "flush")
		span.End()
	}
	require.NoError(t, recorder.Shutdown(context.Background()))
	mu.Lock()
	defer mu.Unlock()
	count := 0
	for _, request := range requests {
		for _, batch := range request.ResourceSpans {
			attrs := make(map[string]string)
			for _, attr := range batch.Resource.Attributes {
				attrs[attr.Key] = attr.Value.GetStringValue()
			}
			require.Equal(t, "watchtower-lab6", attrs["service.name"])
			require.Equal(t, "run1", attrs["run.id"])
			for _, scope := range batch.ScopeSpans {
				count += len(scope.Spans)
			}
		}
	}
	require.Equal(t, 2200, count)
	require.Equal(t, Summary{EndedSpans: 2200, ExportedSpans: 2200}, recorder.Summary())
}

func TestExporterFailureIsReturnedAtShutdown(t *testing.T) {
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
	defer receiver.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", receiver.URL+"/v1/traces")
	recorder, err := New(context.Background())
	require.NoError(t, err)
	_, span := recorder.tracer.Start(context.Background(), "failed")
	span.End()
	require.Error(t, recorder.provider.ForceFlush(context.Background()))
	require.ErrorContains(t, recorder.Shutdown(context.Background()), "trace export failed")
	require.Equal(t, Summary{EndedSpans: 1, ExportErrors: 1}, recorder.Summary())
}

func TestDisabledTracingCreatesNoSpans(t *testing.T) {
	var recorder *Recorder
	require.NoError(t, recorder.Shutdown(context.Background()))
	router := gin.New()
	router.Use(Middleware(recorder, slog.Default(), false))
	router.GET("/test", func(c *gin.Context) { c.Status(200) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/test", nil))
	require.Empty(t, response.Header().Get("X-Trace-ID"))
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
		if extended {
			require.Contains(t, output.String(), "HTTP request completed")
		} else {
			require.Empty(t, output.String())
		}
		require.NotContains(t, output.String(), "private")
	}
}

func TestStandardServiceNameEnvironmentOverridesDefault(t *testing.T) {
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer receiver.Close()
	t.Setenv("OTEL_SERVICE_NAME", "watchtower-test")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", receiver.URL+"/v1/traces")
	recorder, err := New(context.Background())
	require.NoError(t, err)
	_, span := recorder.tracer.Start(context.Background(), "resource")
	require.Contains(t, span.(sdktrace.ReadWriteSpan).Resource().Attributes(), attribute.String("service.name", "watchtower-test"))
	span.End()
	require.NoError(t, recorder.Shutdown(context.Background()))
}

func TestShutdownRejectsUnfinishedSpans(t *testing.T) {
	recorder := newRecorder(tracetest.NewInMemoryExporter(), resource.Empty())
	_, span := recorder.tracer.Start(context.Background(), "unfinished")
	require.ErrorContains(t, recorder.Shutdown(context.Background()), "trace export failed")
	span.End()
}

func TestQueryTracerClassifiesLeadingSQLComments(t *testing.T) {
	cases := []struct{ name, sql, operation string }{
		{"generated", "-- name: PrivateAccount :one\nSELECT secret FROM private_accounts", "SELECT"},
		{"multiple lines", " \t-- private annotation\n-- private second annotation\r\n select secret", "SELECT"},
		{"block", " /* private annotation */ UPDATE private_accounts SET secret = 1", "UPDATE"},
		{"nested blocks", "/* private outer /* private inner */ outer */ INSERT INTO private_accounts VALUES (1)", "INSERT"},
		{"mixed", "-- private annotation\n/* private block */ -- private second\n DELETE FROM private_accounts", "DELETE"},
		{"unterminated block", "/* private annotation SELECT secret", "OTHER"},
		{"unterminated nested block", "/* outer /* private inner */ SELECT secret", "OTHER"},
		{"line without newline", "-- private annotation SELECT secret", "OTHER"},
		{"comments only", "/* private block */ -- private annotation\n", "OTHER"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exporter := tracetest.NewInMemoryExporter()
			recorder := newRecorder(exporter, resource.Empty())
			tracer := QueryTracer{Recorder: recorder}
			ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: tc.sql})
			tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
			require.NoError(t, recorder.provider.ForceFlush(context.Background()))
			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			require.Equal(t, "postgres "+tc.operation, spans[0].Name)
			require.Contains(t, spans[0].Attributes, attribute.String("db.operation.name", tc.operation))
			serialized, err := json.Marshal(spans)
			require.NoError(t, err)
			require.NotContains(t, string(serialized), "private")
			require.NoError(t, recorder.Shutdown(context.Background()))
		})
	}
}
