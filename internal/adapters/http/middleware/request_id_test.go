package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestRequestID_GeneratesID(t *testing.T) {
	t.Parallel()

	var gotID string
	handler := middleware.RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID = middleware.RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if gotID == "" {
		t.Fatal("RequestIDFromContext returned empty string, want generated ID")
	}
	if !uuidPattern.MatchString(gotID) {
		t.Errorf("generated ID %q does not match UUID v4 pattern", gotID)
	}
	if respID := rec.Header().Get("X-Request-ID"); respID != gotID {
		t.Errorf("response X-Request-ID = %q, want %q", respID, gotID)
	}
}

func TestRequestID_ExtractsFromHeader(t *testing.T) {
	t.Parallel()

	var gotID string
	handler := middleware.RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID = middleware.RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Request-ID", "incoming-123")
	handler.ServeHTTP(rec, req)

	if gotID != "incoming-123" {
		t.Errorf("RequestIDFromContext = %q, want %q", gotID, "incoming-123")
	}
	if respID := rec.Header().Get("X-Request-ID"); respID != "incoming-123" {
		t.Errorf("response X-Request-ID = %q, want %q", respID, "incoming-123")
	}
}

func TestRequestID_UniquenessAcrossRequests(t *testing.T) {
	t.Parallel()

	ids := make(map[string]bool)
	handler := middleware.RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ids[middleware.RequestIDFromContext(r.Context())] = true
	}))

	for range 100 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		handler.ServeHTTP(rec, req)
	}

	if len(ids) != 100 {
		t.Errorf("unique IDs = %d, want 100", len(ids))
	}
}

func TestRequestIDFromContext_NotFound(t *testing.T) {
	t.Parallel()

	id := middleware.RequestIDFromContext(context.Background())
	if id != "" {
		t.Errorf("RequestIDFromContext = %q, want empty string", id)
	}
}

func TestWithRequestID_StoresInContext(t *testing.T) {
	t.Parallel()

	ctx := middleware.WithRequestID(context.Background(), "test-id")
	got := middleware.RequestIDFromContext(ctx)

	if got != "test-id" {
		t.Errorf("RequestIDFromContext = %q, want %q", got, "test-id")
	}
}
