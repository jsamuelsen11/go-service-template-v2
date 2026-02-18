package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/handlers"
	"github.com/jsamuelsen11/go-service-template-v2/mocks"
)

// --- Liveness ---

func TestLiveness_AlwaysOK(t *testing.T) {
	t.Parallel()

	registry := mocks.NewMockHealthRegistry(t)
	h := handlers.NewHealthHandler(registry)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	h.Liveness(rec, req)

	requireStatus(t, rec, http.StatusOK)

	resp := decodeJSON[map[string]string](t, rec)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

// --- Readiness ---

func TestReadiness_AllHealthy(t *testing.T) {
	t.Parallel()

	registry := mocks.NewMockHealthRegistry(t)
	registry.EXPECT().CheckAll(mock.Anything).Return(map[string]error{
		"todo-api": nil,
	})

	h := handlers.NewHealthHandler(registry)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	h.Readiness(rec, req)

	requireStatus(t, rec, http.StatusOK)

	resp := decodeJSON[map[string]any](t, rec)
	if resp["status"] != "ready" {
		t.Errorf("status = %q, want %q", resp["status"], "ready")
	}
	checks, ok := resp["checks"].(map[string]any)
	if !ok {
		t.Fatal("checks field not a map")
	}
	if checks["todo-api"] != "ok" {
		t.Errorf("todo-api check = %v, want %q", checks["todo-api"], "ok")
	}
}

func TestReadiness_Unhealthy(t *testing.T) {
	t.Parallel()

	registry := mocks.NewMockHealthRegistry(t)
	registry.EXPECT().CheckAll(mock.Anything).Return(map[string]error{
		"todo-api": errors.New("connection refused"),
		"database": nil,
	})

	h := handlers.NewHealthHandler(registry)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	h.Readiness(rec, req)

	requireStatus(t, rec, http.StatusServiceUnavailable)

	resp := decodeJSON[map[string]any](t, rec)
	if resp["status"] != "not_ready" {
		t.Errorf("status = %q, want %q", resp["status"], "not_ready")
	}
	checks, ok := resp["checks"].(map[string]any)
	if !ok {
		t.Fatal("checks field not a map")
	}
	if checks["todo-api"] != "connection refused" {
		t.Errorf("todo-api check = %v, want %q", checks["todo-api"], "connection refused")
	}
	if checks["database"] != "ok" {
		t.Errorf("database check = %v, want %q", checks["database"], "ok")
	}
}

func TestReadiness_NoCheckers(t *testing.T) {
	t.Parallel()

	registry := mocks.NewMockHealthRegistry(t)
	registry.EXPECT().CheckAll(mock.Anything).Return(map[string]error{})

	h := handlers.NewHealthHandler(registry)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	h.Readiness(rec, req)

	requireStatus(t, rec, http.StatusOK)
}
