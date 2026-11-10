// Package observability records correlated HTTP and PostgreSQL spans as JSONL.
// Input values, SQL text, headers, bodies and error messages are never exported.
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Span struct {
	TraceID    string `json:"trace_id"`
	SpanID     string `json:"span_id"`
	ParentID   string `json:"parent_id,omitempty"`
	Name       string `json:"name"`
	StartedNS  int64  `json:"started_ns"`
	DurationNS int64  `json:"duration_ns"`
	Status     int    `json:"status"`
	started    time.Time
}
type Recorder struct {
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
	err     error
}
type spanKey struct{}

var current atomic.Pointer[Recorder]

func Current() *Recorder     { return current.Load() }
func SetCurrent(r *Recorder) { current.Store(r) }
func New(path string) (*Recorder, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	return &Recorder{file: f, encoder: json.NewEncoder(f)}, nil
}
func identifier() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("trace identifier unavailable")
	}
	return hex.EncodeToString(b[:])
}
func (r *Recorder) Start(ctx context.Context, name string) (context.Context, *Span) {
	if r == nil {
		return ctx, nil
	}
	now := time.Now()
	s := &Span{TraceID: identifier(), SpanID: identifier(), Name: name, StartedNS: now.UnixNano(), started: now}
	if parent, ok := ctx.Value(spanKey{}).(*Span); ok {
		s.TraceID = parent.TraceID
		s.ParentID = parent.SpanID
	}
	return context.WithValue(ctx, spanKey{}, s), s
}
func (r *Recorder) Finish(s *Span, name string, status int) {
	if r == nil || s == nil {
		return
	}
	s.Name = name
	s.Status = status
	s.DurationNS = time.Since(s.started).Nanoseconds()
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.encoder.Encode(s); err != nil && r.err == nil {
		r.err = errors.New("trace export failed")
		slog.Error("trace export failed")
	}
}
func (r *Recorder) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return errors.Join(r.err, r.file.Close())
}
func Middleware(r *Recorder, logger *slog.Logger, extended bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx, span := r.Start(c.Request.Context(), "HTTP")
		c.Request = c.Request.WithContext(ctx)
		if span != nil {
			c.Header("X-Trace-ID", span.TraceID)
		}
		defer func() {
			route := c.FullPath()
			if route == "" {
				route = "unmatched"
			}
			method := c.Request.Method
			switch method {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
			default:
				method = "OTHER"
			}
			r.Finish(span, method+" "+route, c.Writer.Status())
			if extended {
				fields := []any{"method", method, "route", route, "status", c.Writer.Status(), "duration_ns", time.Since(start).Nanoseconds(), "response_bytes", c.Writer.Size()}
				if span != nil {
					fields = append(fields, "trace_id", span.TraceID)
				}
				logger.DebugContext(ctx, "HTTP request completed", fields...)
			}
		}()
		c.Next()
	}
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
	ctx, _ = t.Recorder.Start(ctx, "postgres "+operation(data.SQL))
	return ctx
}
func (t QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span, _ := ctx.Value(spanKey{}).(*Span)
	if span == nil {
		return
	}
	status := 200
	if data.Err != nil {
		status = 500
	}
	t.Recorder.Finish(span, span.Name, status)
}
