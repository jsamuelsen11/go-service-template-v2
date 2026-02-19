package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
	appctx "github.com/jsamuelsen11/go-service-template-v2/internal/app/context"
)

func TestAppContext_InjectsRequestContext(t *testing.T) {
	t.Parallel()

	var gotRC *appctx.RequestContext
	handler := middleware.AppContext()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRC = appctx.FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if gotRC == nil {
		t.Fatal("AppContext middleware did not inject RequestContext into context")
	}
}

func TestAppContext_EachRequestGetsUniqueContext(t *testing.T) {
	t.Parallel()

	var contexts []*appctx.RequestContext
	handler := middleware.AppContext()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		contexts = append(contexts, appctx.FromContext(r.Context()))
	}))

	for range 3 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		handler.ServeHTTP(rec, req)
	}

	if len(contexts) != 3 {
		t.Fatalf("expected 3 contexts, got %d", len(contexts))
	}

	// Each request should get a distinct RequestContext instance.
	if contexts[0] == contexts[1] || contexts[1] == contexts[2] {
		t.Error("expected each request to get a unique RequestContext")
	}
}

func TestFromContext_ReturnsNilWithoutMiddleware(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		rc := appctx.FromContext(r.Context())
		if rc != nil {
			t.Error("expected nil RequestContext without middleware, got non-nil")
		}
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)
}
