package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
)

func TestTimeout_HandlerCompletesBeforeDeadline(t *testing.T) {
	t.Parallel()

	handler := middleware.Timeout(1 * time.Second)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
	if rec.Header().Get("X-Custom") != "value" {
		t.Errorf("X-Custom header = %q, want %q", rec.Header().Get("X-Custom"), "value")
	}
}

func TestTimeout_HandlerExceedsDeadline(t *testing.T) {
	t.Parallel()

	handler := middleware.Timeout(50 * time.Millisecond)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// Block until context is canceled.
		<-r.Context().Done()
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusGatewayTimeout)
	}
}

func TestTimeout_ContextCarriesDeadline(t *testing.T) {
	t.Parallel()

	var hasDeadline bool
	handler := middleware.Timeout(1 * time.Second)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, hasDeadline = r.Context().Deadline()
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if !hasDeadline {
		t.Error("context has no deadline, want deadline set by timeout middleware")
	}
}

func TestTimeout_DefaultStatusOnImplicitWrite(t *testing.T) {
	t.Parallel()

	handler := middleware.Timeout(1 * time.Second)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("no explicit status"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "no explicit status" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "no explicit status")
	}
}
