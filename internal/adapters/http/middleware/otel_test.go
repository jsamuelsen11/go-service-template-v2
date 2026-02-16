package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
)

// OTEL tests are NOT parallel because they modify the global TracerProvider.

func setupTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.Cleanup(func() {
		_ = tp.Shutdown(t.Context())
	})

	return exporter
}

func TestOpenTelemetry_CreatesSpan(t *testing.T) {
	exporter := setupTracer(t)

	handler := middleware.OpenTelemetry(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}

	span := spans[0]
	if span.Name != "HTTP GET /test" {
		t.Errorf("span name = %q, want %q", span.Name, "HTTP GET /test")
	}
}

func TestOpenTelemetry_SetsSpanAttributes(t *testing.T) {
	exporter := setupTracer(t)

	handler := middleware.OpenTelemetry(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items/42", http.NoBody)
	handler.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}

	attrs := make(map[string]any)
	for _, a := range spans[0].Attributes {
		attrs[string(a.Key)] = a.Value.AsInterface()
	}

	if method, ok := attrs["http.method"].(string); !ok || method != "POST" {
		t.Errorf("http.method attr = %v, want %q", attrs["http.method"], "POST")
	}
	if status, ok := attrs["http.status_code"].(int64); !ok || status != http.StatusNotFound {
		t.Errorf("http.status_code attr = %v, want %d", attrs["http.status_code"], http.StatusNotFound)
	}
}

func TestOpenTelemetry_SetsErrorStatusOn5xx(t *testing.T) {
	exporter := setupTracer(t)

	handler := middleware.OpenTelemetry(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/error", http.NoBody)
	handler.ServeHTTP(rec, req)

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans recorded")
	}

	if spans[0].Status.Code != codes.Error {
		t.Errorf("span status code = %d, want %d (Error)", spans[0].Status.Code, codes.Error)
	}
}

func TestOpenTelemetry_NilMetricsNoPanic(t *testing.T) {
	t.Parallel()

	handler := middleware.OpenTelemetry(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)

	// Should not panic with nil metrics.
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
