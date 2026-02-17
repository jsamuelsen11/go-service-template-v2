package acl

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/httpclient"
)

const msgRequired = "is required"

// newTestClient creates an httpclient.Client pointing at the given test server
// with circuit breaker and retry configured for fast test execution.
func newTestClient(t *testing.T, baseURL string) *httpclient.Client {
	t.Helper()

	cfg := &config.ClientConfig{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
		Retry: config.RetryConfig{
			MaxAttempts:     1,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
			Multiplier:      1,
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			MaxFailures:   5,
			Timeout:       30 * time.Second,
			HalfOpenLimit: 1,
		},
	}
	logger := slog.Default()

	return httpclient.New(cfg, "todo-api-test", nil, logger)
}

// writeJSON encodes v as JSON to the response writer, failing the test on error.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()

	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
}

// --- Todo CRUD tests ---

func TestTodoClient_ListTodos(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/todos" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"todos": []map[string]any{{
				"id": 1, "title": "Buy milk", "description": "2% milk",
				"status": "pending", "category": "personal",
				"progress_percent": 0,
				"created_at":       "2025-01-01T00:00:00Z",
				"updated_at":       "2025-01-01T00:00:00Z",
			}},
			"count": 1,
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	todos, err := client.ListTodos(context.Background(), domain.TodoFilter{})
	if err != nil {
		t.Fatalf("ListTodos() error = %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("len(todos) = %d, want 1", len(todos))
	}
	if todos[0].Title != "Buy milk" {
		t.Errorf("Title = %q, want %q", todos[0].Title, "Buy milk")
	}
}

func TestTodoClient_ListTodos_WithFilter(t *testing.T) {
	t.Parallel()

	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{"todos": []any{}, "count": 0})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	_, err := client.ListTodos(context.Background(), domain.TodoFilter{
		Status:   domain.StatusPending,
		Category: domain.CategoryWork,
	})
	if err != nil {
		t.Fatalf("ListTodos() error = %v", err)
	}
	if gotQuery == "" {
		t.Fatal("expected query parameters, got empty string")
	}
}

func TestTodoClient_GetTodo(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/todos/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"id": 42, "title": "Test todo", "description": "A test",
			"status": "done", "category": "work",
			"progress_percent": 100,
			"created_at":       "2025-01-01T00:00:00Z",
			"updated_at":       "2025-01-02T00:00:00Z",
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	todo, err := client.GetTodo(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetTodo() error = %v", err)
	}
	if todo.ID != 42 {
		t.Errorf("ID = %d, want 42", todo.ID)
	}
	if todo.Status != domain.StatusDone {
		t.Errorf("Status = %q, want %q", todo.Status, domain.StatusDone)
	}
}

func TestTodoClient_GetTodo_NotFound(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		writeJSON(t, w, map[string]any{
			"detail": "todo 999 not found",
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	_, err := client.GetTodo(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetTodo() error = %v, want ErrNotFound", err)
	}
}

func TestTodoClient_CreateTodo(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		writeJSON(t, w, map[string]any{
			"id": 10, "title": "New todo", "description": "Fresh",
			"status": "pending", "category": "personal",
			"progress_percent": 0,
			"created_at":       "2025-06-01T00:00:00Z",
			"updated_at":       "2025-06-01T00:00:00Z",
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	input := &domain.Todo{
		Title:       "New todo",
		Description: "Fresh",
		Status:      domain.StatusPending,
		Category:    domain.CategoryPersonal,
	}
	created, err := client.CreateTodo(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateTodo() error = %v", err)
	}
	if created.ID != 10 {
		t.Errorf("ID = %d, want 10", created.ID)
	}
}

func TestTodoClient_UpdateTodo(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/todos/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"id": 5, "title": "Updated", "description": "Changed",
			"status": "in_progress", "category": "work",
			"progress_percent": 50,
			"created_at":       "2025-01-01T00:00:00Z",
			"updated_at":       "2025-06-01T00:00:00Z",
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	input := &domain.Todo{
		Title:           "Updated",
		Description:     "Changed",
		Status:          domain.StatusInProgress,
		Category:        domain.CategoryWork,
		ProgressPercent: 50,
	}
	updated, err := client.UpdateTodo(context.Background(), 5, input)
	if err != nil {
		t.Fatalf("UpdateTodo() error = %v", err)
	}
	if updated.ProgressPercent != 50 {
		t.Errorf("ProgressPercent = %d, want 50", updated.ProgressPercent)
	}
}

func TestTodoClient_DeleteTodo(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/todos/7" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	if err := client.DeleteTodo(context.Background(), 7); err != nil {
		t.Fatalf("DeleteTodo() error = %v", err)
	}
}

func TestTodoClient_DeleteTodo_NotFound(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		writeJSON(t, w, map[string]any{"detail": "not found"})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	err := client.DeleteTodo(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("DeleteTodo() error = %v, want ErrNotFound", err)
	}
}

// --- Project CRUD tests (downstream "groups") ---

func TestTodoClient_ListProjects(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/groups" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"groups": []map[string]any{{
				"id": 1, "name": "Work", "description": "Work tasks",
				"created_at": "2025-01-01T00:00:00Z",
				"updated_at": "2025-01-01T00:00:00Z",
			}},
			"count": 1,
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}
	if projects[0].Name != "Work" {
		t.Errorf("Name = %q, want %q", projects[0].Name, "Work")
	}
}

func TestTodoClient_GetProject(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/groups/3" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"id": 3, "name": "Personal", "description": "Personal project",
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z",
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	project, err := client.GetProject(context.Background(), 3)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if project.ID != 3 {
		t.Errorf("ID = %d, want 3", project.ID)
	}
}

func TestTodoClient_CreateProject(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/groups" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		writeJSON(t, w, map[string]any{
			"id": 5, "name": "New project", "description": "A project",
			"created_at": "2025-06-01T00:00:00Z",
			"updated_at": "2025-06-01T00:00:00Z",
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	input := &domain.Project{Name: "New project", Description: "A project"}
	created, err := client.CreateProject(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if created.ID != 5 {
		t.Errorf("ID = %d, want 5", created.ID)
	}
}

func TestTodoClient_UpdateProject(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/groups/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"id": 5, "name": "Renamed", "description": "Updated desc",
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-06-01T00:00:00Z",
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	input := &domain.Project{Name: "Renamed", Description: "Updated desc"}
	updated, err := client.UpdateProject(context.Background(), 5, input)
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updated.Name != "Renamed" {
		t.Errorf("Name = %q, want %q", updated.Name, "Renamed")
	}
}

func TestTodoClient_DeleteProject(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/groups/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	if err := client.DeleteProject(context.Background(), 5); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
}

func TestTodoClient_GetProjectTodos(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/groups/2/todos" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"todos": []map[string]any{{
				"id": 1, "title": "Grouped todo", "description": "In a group",
				"status": "pending", "category": "work",
				"progress_percent": 0, "group_id": 2,
				"created_at": "2025-01-01T00:00:00Z",
				"updated_at": "2025-01-01T00:00:00Z",
			}},
			"count": 1,
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	todos, err := client.GetProjectTodos(context.Background(), 2, domain.TodoFilter{})
	if err != nil {
		t.Fatalf("GetProjectTodos() error = %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("len(todos) = %d, want 1", len(todos))
	}
	projectID := int64(2)
	if todos[0].ProjectID == nil || *todos[0].ProjectID != projectID {
		t.Errorf("ProjectID = %v, want %d", todos[0].ProjectID, projectID)
	}
}

// --- Validation error test ---

func TestTodoClient_CreateTodo_ValidationError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(t, w, map[string]any{
			"detail": "validation failed",
			"errors": []map[string]any{
				{"location": "body.title", "message": msgRequired},
			},
		})
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	_, err := client.CreateTodo(context.Background(), &domain.Todo{})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("CreateTodo() error = %v, want ErrValidation", err)
	}

	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error is not *ValidationError: %v", err)
	}
	if verr.Fields["title"] != msgRequired {
		t.Errorf("Fields[title] = %q, want %q", verr.Fields["title"], msgRequired)
	}
}

// --- Server error test ---

func TestTodoClient_GetTodo_ServerError(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewTodoClient(newTestClient(t, ts.URL), slog.Default())
	_, err := client.GetTodo(context.Background(), 1)
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("GetTodo() error = %v, want ErrUnavailable", err)
	}
}

// --- filterQuery tests ---

func TestFilterQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter domain.TodoFilter
		want   string
	}{
		{
			name:   "empty filter produces empty string",
			filter: domain.TodoFilter{},
			want:   "",
		},
		{
			name:   "status only",
			filter: domain.TodoFilter{Status: domain.StatusPending},
			want:   "?status=pending",
		},
		{
			name:   "category only",
			filter: domain.TodoFilter{Category: domain.CategoryWork},
			want:   "?category=work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := filterQuery(tt.filter)
			if got != tt.want {
				t.Errorf("filterQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}
