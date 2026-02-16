package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
)

func TestCorrelationID_ExtractsFromHeader(t *testing.T) {
	t.Parallel()

	var gotID string
	handler := middleware.CorrelationID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID = middleware.CorrelationIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Correlation-ID", "corr-abc")
	handler.ServeHTTP(rec, req)

	if gotID != "corr-abc" {
		t.Errorf("CorrelationIDFromContext = %q, want %q", gotID, "corr-abc")
	}
	if respID := rec.Header().Get("X-Correlation-ID"); respID != "corr-abc" {
		t.Errorf("response X-Correlation-ID = %q, want %q", respID, "corr-abc")
	}
}

func TestCorrelationID_DefaultsToRequestID(t *testing.T) {
	t.Parallel()

	var gotID string
	// Chain: RequestID → CorrelationID → handler
	handler := middleware.RequestID()(
		middleware.CorrelationID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotID = middleware.CorrelationIDFromContext(r.Context())
		})),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	reqID := rec.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Fatal("X-Request-ID response header is empty")
	}
	if gotID != reqID {
		t.Errorf("CorrelationIDFromContext = %q, want request ID %q", gotID, reqID)
	}
}

func TestCorrelationID_SetsResponseHeader(t *testing.T) {
	t.Parallel()

	handler := middleware.CorrelationID()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Correlation-ID", "corr-xyz")
	handler.ServeHTTP(rec, req)

	if respID := rec.Header().Get("X-Correlation-ID"); respID != "corr-xyz" {
		t.Errorf("response X-Correlation-ID = %q, want %q", respID, "corr-xyz")
	}
}

func TestCorrelationIDFromContext_NotFound(t *testing.T) {
	t.Parallel()

	id := middleware.CorrelationIDFromContext(context.Background())
	if id != "" {
		t.Errorf("CorrelationIDFromContext = %q, want empty string", id)
	}
}

func TestWithCorrelationID_StoresInContext(t *testing.T) {
	t.Parallel()

	ctx := middleware.WithCorrelationID(context.Background(), "test-corr")
	got := middleware.CorrelationIDFromContext(ctx)

	if got != "test-corr" {
		t.Errorf("CorrelationIDFromContext = %q, want %q", got, "test-corr")
	}
}
