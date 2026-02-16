package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
)

func TestLogging_LogsStartAndCompletion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := testLogger(&buf)

	handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items", http.NoBody)
	handler.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "request started") {
		t.Error("log output missing 'request started'")
	}
	if !strings.Contains(output, "request completed") {
		t.Error("log output missing 'request completed'")
	}
	if !strings.Contains(output, "POST") {
		t.Error("log output missing method")
	}
	if !strings.Contains(output, "/items") {
		t.Error("log output missing path")
	}
}

func TestLogging_EnrichesLoggerWithIDs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := testLogger(&buf)

	// Chain: RequestID → CorrelationID → Logging → handler
	handler := middleware.RequestID()(
		middleware.CorrelationID()(
			middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})),
		),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Request-ID", "req-log-test")
	req.Header.Set("X-Correlation-ID", "corr-log-test")
	handler.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "req-log-test") {
		t.Error("log output missing request_id")
	}
	if !strings.Contains(output, "corr-log-test") {
		t.Error("log output missing correlation_id")
	}
}

func TestLogging_StoresEnrichedLoggerInContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := testLogger(&buf)

	var contextLoggerFound bool
	handler := middleware.RequestID()(
		middleware.Logging(logger)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			ctxLogger := logging.FromContext(r.Context())
			// The context logger should be the enriched one, not slog.Default().
			contextLoggerFound = ctxLogger != nil
			ctxLogger.Info("handler log")
		})),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Request-ID", "ctx-logger-test")
	handler.ServeHTTP(rec, req)

	if !contextLoggerFound {
		t.Error("logging.FromContext returned nil, want enriched logger")
	}

	output := buf.String()
	if !strings.Contains(output, "handler log") {
		t.Error("handler log not captured, enriched logger may not be stored in context")
	}
	if !strings.Contains(output, "ctx-logger-test") {
		t.Error("handler log missing request_id from enriched logger")
	}
}

func TestLogging_IncludesDuration(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := testLogger(&buf)

	handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "duration") {
		t.Error("log output missing duration")
	}
}

func TestLogging_IncludesStatusCode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := testLogger(&buf)

	handler := middleware.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	handler.ServeHTTP(rec, req)

	output := buf.String()
	if !strings.Contains(output, "status=404") {
		t.Errorf("log output missing status=404, got: %s", output)
	}
}
