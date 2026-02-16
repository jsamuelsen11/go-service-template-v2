package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/telemetry"
)

// OpenTelemetry returns middleware that creates a trace span for each incoming
// request and records server request metrics. It extracts W3C Trace Context
// from incoming headers so that distributed traces are connected.
//
// If metrics is nil, metric recording is skipped (safe nil check).
func OpenTelemetry(metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			tracer := otel.GetTracerProvider().Tracer("middleware")
			spanName := fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
				),
			)
			defer span.End()

			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r.WithContext(ctx))

			status := rw.statusCode
			span.SetAttributes(attribute.Int("http.status_code", status))
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}

			recordServerMetrics(ctx, metrics, r.Method, start, status)
		})
	}
}

// recordServerMetrics records server request duration and count metrics.
// Safe to call with nil metrics.
func recordServerMetrics(ctx context.Context, metrics *telemetry.Metrics, method string, start time.Time, status int) {
	if metrics == nil {
		return
	}

	duration := time.Since(start).Seconds()

	result := "success"
	if status >= http.StatusBadRequest {
		result = "error"
	}

	attrs := metric.WithAttributes(
		telemetry.AttrHTTPMethod.String(method),
		telemetry.AttrHTTPStatus.Int(status),
		telemetry.AttrResult.String(result),
	)

	metrics.ServerRequestDuration.Record(ctx, duration, attrs)
	metrics.ServerRequestTotal.Add(ctx, 1, attrs)
}
