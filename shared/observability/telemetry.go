package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const traceIDContextKey contextKey = "trace_id"
const TraceIDHeader = "X-Trace-ID"

type Telemetry struct {
	serviceName string
	tracer      trace.Tracer
	requestCnt  metric.Int64Counter
	errorCnt    metric.Int64Counter
	httpLatency metric.Float64Histogram
	procLatency metric.Float64Histogram
}

func New(serviceName string) *Telemetry {
	if strings.TrimSpace(serviceName) == "" {
		serviceName = "unknown-service"
	}

	meter := otel.Meter("job-finder/" + serviceName)
	requestCnt, _ := meter.Int64Counter("http.requests")
	errorCnt, _ := meter.Int64Counter("http.errors")
	httpLatency, _ := meter.Float64Histogram("http.latency.seconds")
	procLatency, _ := meter.Float64Histogram("processing.latency.seconds")

	return &Telemetry{
		serviceName: serviceName,
		tracer:      otel.Tracer("job-finder/" + serviceName),
		requestCnt:  requestCnt,
		errorCnt:    errorCnt,
		httpLatency: httpLatency,
		procLatency: procLatency,
	}
}

func (t *Telemetry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := strings.TrimSpace(r.Header.Get(TraceIDHeader))
		if traceID == "" {
			traceID = newTraceID()
		}

		ctx := context.WithValue(r.Context(), traceIDContextKey, traceID)
		ctx, span := t.tracer.Start(ctx, r.Method+" "+r.URL.Path)
		defer span.End()

		span.SetAttributes(
			attribute.String("service.name", t.serviceName),
			attribute.String("trace_id", traceID),
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
		)

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		recorder.Header().Set(TraceIDHeader, traceID)

		start := time.Now()
		next.ServeHTTP(recorder, r.WithContext(ctx))
		latency := time.Since(start)

		attrs := []attribute.KeyValue{
			attribute.String("service.name", t.serviceName),
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
			attribute.Int("http.status_code", recorder.status),
		}
		t.requestCnt.Add(ctx, 1, metric.WithAttributes(attrs...))
		t.httpLatency.Record(ctx, latency.Seconds(), metric.WithAttributes(attrs...))
		if recorder.status >= http.StatusBadRequest {
			t.errorCnt.Add(ctx, 1, metric.WithAttributes(attrs...))
		}

		span.SetAttributes(
			attribute.Int("http.status_code", recorder.status),
			attribute.Float64("http.latency_ms", float64(latency.Milliseconds())),
		)
		if recorder.status >= http.StatusInternalServerError {
			span.RecordError(fmt.Errorf("http status %d", recorder.status))
		}

		log.Printf("trace_id=%s service=%s method=%s path=%s status=%d latency_ms=%d",
			traceID, t.serviceName, r.Method, r.URL.Path, recorder.status, latency.Milliseconds())
	})
}

func (t *Telemetry) RecordProcessing(ctx context.Context, operation string, startedAt time.Time, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("service.name", t.serviceName),
		attribute.String("operation", operation),
	}
	t.procLatency.Record(ctx, time.Since(startedAt).Seconds(), metric.WithAttributes(attrs...))
	if err != nil {
		t.errorCnt.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, _ := ctx.Value(traceIDContextKey).(string)
	return strings.TrimSpace(traceID)
}

func EnsureTraceID(ctx context.Context, candidate string) (context.Context, string) {
	traceID := strings.TrimSpace(candidate)
	if traceID == "" {
		traceID = TraceIDFromContext(ctx)
	}
	if traceID == "" {
		traceID = newTraceID()
	}
	return context.WithValue(ctx, traceIDContextKey, traceID), traceID
}

func newTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}
