// Package observability exports correlated HTTP and PostgreSQL spans through OTLP.
package observability

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Summary struct {
	EndedSpans    int64 `json:"ended_spans"`
	ExportedSpans int64 `json:"exported_spans"`
	ExportErrors  int64 `json:"export_errors"`
}

type Recorder struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	started  atomic.Int64
	ended    atomic.Int64
	exported atomic.Int64
	failures atomic.Int64
}

var current atomic.Pointer[Recorder]

func Current() *Recorder     { return current.Load() }
func SetCurrent(r *Recorder) { current.Store(r) }

func New(ctx context.Context) (*Recorder, error) {
	res, err := resource.New(ctx, resource.WithAttributes(attribute.String("service.name", "watchtower-lab6")), resource.WithFromEnv(), resource.WithTelemetrySDK())
	if err != nil {
		return nil, err
	}
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	return newRecorder(exporter, res), nil
}

func newRecorder(exporter sdktrace.SpanExporter, res *resource.Resource) *Recorder {
	r := &Recorder{}
	r.provider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(spanCounter{recorder: r}),
		sdktrace.WithBatcher(countingExporter{SpanExporter: exporter, recorder: r}, sdktrace.WithBlocking()),
	)
	r.tracer = r.provider.Tracer("WatchTower/internal/observability")
	return r
}

func (r *Recorder) Summary() Summary {
	if r == nil {
		return Summary{}
	}
	return Summary{EndedSpans: r.ended.Load(), ExportedSpans: r.exported.Load(), ExportErrors: r.failures.Load()}
}

func (r *Recorder) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	err := r.provider.Shutdown(ctx)
	summary := r.Summary()
	if summary.ExportErrors > 0 || summary.EndedSpans != summary.ExportedSpans || r.started.Load() != summary.EndedSpans {
		return errors.Join(err, errors.New("trace export failed: incomplete span delivery"))
	}
	return err
}

type spanCounter struct{ recorder *Recorder }

func (s spanCounter) OnStart(context.Context, sdktrace.ReadWriteSpan) { s.recorder.started.Add(1) }
func (s spanCounter) OnEnd(sdktrace.ReadOnlySpan)                     { s.recorder.ended.Add(1) }
func (s spanCounter) Shutdown(context.Context) error                  { return nil }
func (s spanCounter) ForceFlush(context.Context) error                { return nil }

type countingExporter struct {
	sdktrace.SpanExporter
	recorder *Recorder
}

func (e countingExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if err := e.SpanExporter.ExportSpans(ctx, spans); err != nil {
		e.recorder.failures.Add(1)
		return errors.New("trace export failed")
	}
	e.recorder.exported.Add(int64(len(spans)))
	return nil
}

func Middleware(r *Recorder, logger *slog.Logger, extended bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		ctx, span := startHTTP(r, c)
		c.Request = c.Request.WithContext(ctx)
		defer finishHTTP(c, span, logger, extended, started)
		c.Next()
	}
}

func startHTTP(r *Recorder, c *gin.Context) (context.Context, trace.Span) {
	ctx := c.Request.Context()
	if r == nil {
		return ctx, nil
	}
	ctx = propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(c.Request.Header))
	parent := trace.SpanContextFromContext(ctx).WithTraceState(trace.TraceState{})
	ctx = trace.ContextWithSpanContext(ctx, parent)
	ctx, span := r.tracer.Start(ctx, "HTTP", trace.WithSpanKind(trace.SpanKindServer))
	c.Header("X-Trace-ID", span.SpanContext().TraceID().String())
	return ctx, span
}

func finishHTTP(c *gin.Context, span trace.Span, logger *slog.Logger, extended bool, started time.Time) {
	route := c.FullPath()
	if route == "" {
		route = "unmatched"
	}
	method := safeMethod(c.Request.Method)
	status := c.Writer.Status()
	if span != nil {
		span.SetName(method + " " + route)
		span.SetAttributes(attribute.String("http.request.method", method), attribute.String("http.route", route), attribute.Int("http.response.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, "")
		}
		span.End()
	}
	if extended {
		logHTTP(c, span, logger, method, route, started)
	}
}

func safeMethod(method string) string {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return method
	default:
		return "OTHER"
	}
}

func logHTTP(c *gin.Context, span trace.Span, logger *slog.Logger, method, route string, started time.Time) {
	fields := []any{"method", method, "route", route, "status", c.Writer.Status(), "duration_ns", time.Since(started).Nanoseconds(), "response_bytes", c.Writer.Size()}
	if span != nil {
		fields = append(fields, "trace_id", span.SpanContext().TraceID().String())
	}
	logger.DebugContext(c.Request.Context(), "HTTP request completed", fields...)
}

type QueryTracer struct{ Recorder *Recorder }

func operation(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "OTHER"
	}
	switch op := strings.ToUpper(fields[0]); op {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "BEGIN", "COMMIT", "ROLLBACK", "CREATE", "ALTER", "DROP", "GRANT", "DO":
		return op
	}
	return "OTHER"
}

func (t QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if t.Recorder == nil {
		return ctx
	}
	op := operation(data.SQL)
	ctx, _ = t.Recorder.tracer.Start(ctx, "postgres "+op, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attribute.String("db.operation.name", op), attribute.String("db.system.name", "postgresql")))
	return ctx
}

func (t QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if t.Recorder == nil {
		return
	}
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.SetStatus(codes.Error, "")
	}
	span.End()
}
