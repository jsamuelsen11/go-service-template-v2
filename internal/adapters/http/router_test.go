package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"

	adapthttp "github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http"
	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/handlers"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/mocks"
)

func newTestRouter(t *testing.T) (http.Handler, *mocks.MockProjectService) {
	t.Helper()
	svc := mocks.NewMockProjectService(t)
	registry := mocks.NewMockHealthRegistry(t)

	ph := handlers.NewProjectHandler(svc)
	hh := handlers.NewHealthHandler(registry)

	router := adapthttp.NewRouter(ph, hh)
	return router, svc
}

func TestRouter_AllRoutesRegistered(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)

	expectedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health/live"},
		{http.MethodGet, "/health/ready"},
		{http.MethodGet, "/api/v1/projects"},
		{http.MethodPost, "/api/v1/projects"},
		{http.MethodGet, "/api/v1/projects/{id}"},
		{http.MethodPatch, "/api/v1/projects/{id}"},
		{http.MethodDelete, "/api/v1/projects/{id}"},
		{http.MethodPost, "/api/v1/projects/{projectId}/todos"},
		{http.MethodPatch, "/api/v1/projects/{projectId}/todos/{todoId}"},
		{http.MethodDelete, "/api/v1/projects/{projectId}/todos/{todoId}"},
	}

	chiRouter, ok := router.(*chi.Mux)
	if !ok {
		t.Fatal("router is not *chi.Mux")
	}

	registered := make(map[string]bool)
	err := chi.Walk(chiRouter, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk error: %v", err)
	}

	for _, expected := range expectedRoutes {
		key := expected.method + " " + expected.path
		if !registered[key] {
			t.Errorf("route %s not registered", key)
		}
	}
}

func TestRouter_MiddlewareApplied(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockProjectService(t)
	registry := mocks.NewMockHealthRegistry(t)

	ph := handlers.NewProjectHandler(svc)
	hh := handlers.NewHealthHandler(registry)

	called := false
	testMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	router := adapthttp.NewRouter(ph, hh, testMW)

	registry.EXPECT().CheckAll(mock.Anything).Return(map[string]error{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(rec, req)

	if !called {
		t.Error("middleware was not called")
	}
}

func TestRouter_IntegrationListProjects(t *testing.T) {
	t.Parallel()

	router, svc := newTestRouter(t)

	svc.EXPECT().ListProjects(mock.Anything).Return([]project.Project{}, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRouter_NotFoundReturns404(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
