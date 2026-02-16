package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
)

func TestChain_Empty(t *testing.T) {
	t.Parallel()

	handler := middleware.Chain()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bare"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "bare" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "bare")
	}
}

func TestChain_Order(t *testing.T) {
	t.Parallel()

	var order []string

	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(w, r)
				order = append(order, name+":after")
			})
		}
	}

	handler := middleware.Chain(
		mw("first"),
		mw("second"),
		mw("third"),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	expected := []string{
		"first:before", "second:before", "third:before",
		"handler",
		"third:after", "second:after", "first:after",
	}

	if len(order) != len(expected) {
		t.Fatalf("execution order length = %d, want %d: %v", len(order), len(expected), order)
	}
	for i, got := range order {
		if got != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, got, expected[i])
		}
	}
}

func TestChain_FullPipeline(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := testLogger(&buf)

	handler := middleware.Chain(
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.CorrelationID(),
		middleware.Logging(logger),
		middleware.Timeout(5*time.Second),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.RequestIDFromContext(r.Context())
		corrID := middleware.CorrelationIDFromContext(r.Context())
		if reqID == "" {
			t.Error("request ID not in context")
		}
		if corrID == "" {
			t.Error("correlation ID not in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pipeline", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("response missing X-Request-ID header")
	}
	if rec.Header().Get("X-Correlation-ID") == "" {
		t.Error("response missing X-Correlation-ID header")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "request started") {
		t.Error("log output missing 'request started'")
	}
	if !strings.Contains(logOutput, "request completed") {
		t.Error("log output missing 'request completed'")
	}
}
