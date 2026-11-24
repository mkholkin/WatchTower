package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"WatchTower/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	collector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestSetupRouterCorrelatesDatabaseSpanThroughGinContext(t *testing.T) {
	var spans []*tracepb.Span
	var mu sync.Mutex
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NotContains(t, string(data), "private-header-value")
		request := new(collector.ExportTraceServiceRequest)
		require.NoError(t, proto.Unmarshal(data, request))
		mu.Lock()
		for _, batch := range request.ResourceSpans {
			for _, scope := range batch.ScopeSpans {
				spans = append(spans, scope.Spans...)
			}
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer receiver.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", receiver.URL+"/v1/traces")
	recorder, err := observability.New(context.Background())
	require.NoError(t, err)
	router := SetupRouter(nil, observability.Middleware(recorder, slog.New(slog.NewJSONHandler(io.Discard, nil)), false))
	router.GET("/trace-regression", func(c *gin.Context) {
		tracer := observability.QueryTracer{Recorder: recorder}
		// Generated strict handlers pass *gin.Context to application services.
		ctx := tracer.TraceQueryStart(c, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
		tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		c.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/trace-regression", nil)
	request.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	request.Header.Set("tracestate", "vendor=private-header-value")
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code)
	require.NoError(t, recorder.Shutdown(context.Background()))
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, spans, 2)
	child, parent := spans[0], spans[1]
	require.Empty(t, parent.TraceState)
	require.Empty(t, child.TraceState)
	require.Equal(t, "postgres SELECT", child.Name)
	require.Equal(t, "GET /trace-regression", parent.Name)
	require.Equal(t, parent.TraceId, child.TraceId)
	require.Equal(t, parent.SpanId, child.ParentSpanId)
	require.Equal(t, int64(2), recorder.Summary().ExportedSpans)
}

func TestServeDrainsActiveRequestOnCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	entered, release := make(chan struct{}), make(chan struct{})
	server := &http.Server{Addr: addr, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopped := make(chan error, 1)
	go func() { stopped <- serve(ctx, server) }()
	clientDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		var response *http.Response
		var requestErr error
		for range 100 {
			response, requestErr = client.Get("http://" + addr)
			if requestErr == nil {
				response.Body.Close()
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		clientDone <- requestErr
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not accept request")
	}
	cancel()
	select {
	case err := <-stopped:
		t.Fatalf("server stopped before request ended: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-clientDone)
	require.NoError(t, <-stopped)
}

func TestTracingFlagDefaultsToDisabled(t *testing.T) {
	t.Setenv("WT_TRACING_ENABLED", "false")
	recorder, err := setupTracing(context.Background())
	require.NoError(t, err)
	require.Nil(t, recorder)
}
