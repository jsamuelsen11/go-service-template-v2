package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

const testUpdatedValue = "Updated"

var testTime = time.Date(2026, 2, 12, 15, 4, 5, 0, time.UTC)

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func validProject() project.Project {
	return project.Project{
		ID:          1,
		Name:        "Sprint 1",
		Description: "First sprint tasks",
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
	}
}

func validTodo() todo.Todo {
	return todo.Todo{
		ID:              1,
		Title:           "Buy groceries",
		Description:     "Milk, eggs, bread",
		Status:          todo.StatusPending,
		Category:        todo.CategoryPersonal,
		ProgressPercent: 0,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		t.Fatalf("failed to encode JSON body: %v", err)
	}
	return buf
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	return result
}

func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}
