package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

var testTime = time.Date(2026, 2, 12, 15, 4, 5, 0, time.UTC)

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

func TestToTodoResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		todo   todo.Todo
		verify func(t *testing.T, got dto.TodoResponse)
	}{
		{
			name: "maps all fields correctly",
			todo: validTodo(),
			verify: func(t *testing.T, got dto.TodoResponse) {
				t.Helper()
				if got.ID != 1 {
					t.Errorf("ID = %d, want 1", got.ID)
				}
				if got.Title != "Buy groceries" {
					t.Errorf("Title = %q, want %q", got.Title, "Buy groceries")
				}
				if got.Description != "Milk, eggs, bread" {
					t.Errorf("Description = %q, want %q", got.Description, "Milk, eggs, bread")
				}
				if got.ProgressPercent != 0 {
					t.Errorf("ProgressPercent = %d, want 0", got.ProgressPercent)
				}
			},
		},
		{
			name: "status converts to string",
			todo: func() todo.Todo {
				td := validTodo()
				td.Status = todo.StatusInProgress
				return td
			}(),
			verify: func(t *testing.T, got dto.TodoResponse) {
				t.Helper()
				if got.Status != "in_progress" {
					t.Errorf("Status = %q, want %q", got.Status, "in_progress")
				}
			},
		},
		{
			name: "category converts to string",
			todo: func() todo.Todo {
				td := validTodo()
				td.Category = todo.CategoryWork
				return td
			}(),
			verify: func(t *testing.T, got dto.TodoResponse) {
				t.Helper()
				if got.Category != "work" {
					t.Errorf("Category = %q, want %q", got.Category, "work")
				}
			},
		},
		{
			name: "timestamps formatted as RFC3339",
			todo: validTodo(),
			verify: func(t *testing.T, got dto.TodoResponse) {
				t.Helper()
				want := "2026-02-12T15:04:05Z"
				if got.CreatedAt != want {
					t.Errorf("CreatedAt = %q, want %q", got.CreatedAt, want)
				}
				if got.UpdatedAt != want {
					t.Errorf("UpdatedAt = %q, want %q", got.UpdatedAt, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := dto.ToTodoResponse(&tt.todo)
			tt.verify(t, got)
		})
	}
}

func TestTodoResponse_JSONSerialization(t *testing.T) {
	t.Parallel()

	resp := dto.ToTodoResponse(&todo.Todo{
		ID:              42,
		Title:           "Test",
		Description:     "Desc",
		Status:          todo.StatusDone,
		Category:        todo.CategoryOther,
		ProgressPercent: 100,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	})

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	requiredKeys := []string{
		"id", "title", "description", "status", "category",
		"progress_percent", "created_at", "updated_at",
	}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q, got keys: %v", key, keys(m))
		}
	}
}

func keys(m map[string]any) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
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

func TestToProjectResponse(t *testing.T) {
	t.Parallel()

	t.Run("maps all fields correctly", func(t *testing.T) {
		t.Parallel()
		p := validProject()
		got := dto.ToProjectResponse(&p)
		if got.ID != 1 {
			t.Errorf("ID = %d, want 1", got.ID)
		}
		if got.Name != "Sprint 1" {
			t.Errorf("Name = %q, want %q", got.Name, "Sprint 1")
		}
		if got.Description != "First sprint tasks" {
			t.Errorf("Description = %q, want %q", got.Description, "First sprint tasks")
		}
		if got.CreatedAt != "2026-02-12T15:04:05Z" {
			t.Errorf("CreatedAt = %q, want %q", got.CreatedAt, "2026-02-12T15:04:05Z")
		}
	})

	t.Run("includes todos when populated", func(t *testing.T) {
		t.Parallel()
		p := validProject()
		p.Todos = []todo.Todo{validTodo(), validTodo()}
		got := dto.ToProjectResponse(&p)
		if len(got.Todos) != 2 {
			t.Errorf("len(Todos) = %d, want 2", len(got.Todos))
		}
	})

	t.Run("omits todos when empty", func(t *testing.T) {
		t.Parallel()
		p := validProject()
		got := dto.ToProjectResponse(&p)
		if got.Todos != nil {
			t.Errorf("Todos = %v, want nil (omitted)", got.Todos)
		}
	})
}

func TestToProjectListResponse(t *testing.T) {
	t.Parallel()

	t.Run("converts multiple projects", func(t *testing.T) {
		t.Parallel()
		projects := []project.Project{validProject(), validProject()}
		got := dto.ToProjectListResponse(projects)
		if got.Count != 2 {
			t.Errorf("Count = %d, want 2", got.Count)
		}
		if len(got.Projects) != 2 {
			t.Errorf("len(Projects) = %d, want 2", len(got.Projects))
		}
	})

	t.Run("empty slice returns empty list", func(t *testing.T) {
		t.Parallel()
		got := dto.ToProjectListResponse([]project.Project{})
		if got.Count != 0 {
			t.Errorf("Count = %d, want 0", got.Count)
		}
	})

	t.Run("nil slice returns empty list", func(t *testing.T) {
		t.Parallel()
		got := dto.ToProjectListResponse(nil)
		if got.Count != 0 {
			t.Errorf("Count = %d, want 0", got.Count)
		}
	})
}

func TestProjectResponse_JSONSerialization(t *testing.T) {
	t.Parallel()

	p := validProject()
	resp := dto.ToProjectResponse(&p)

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	requiredKeys := []string{"id", "name", "description", "created_at", "updated_at"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON missing key %q, got keys: %v", key, keys(m))
		}
	}

	// Todos should be omitted when empty
	if _, ok := m["todos"]; ok {
		t.Error("JSON contains 'todos' key, want omitted for empty project")
	}
}
