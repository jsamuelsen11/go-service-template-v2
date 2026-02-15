package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

var testTime = time.Date(2026, 2, 12, 15, 4, 5, 0, time.UTC)

func validTodo() domain.Todo {
	return domain.Todo{
		ID:              1,
		Title:           "Buy groceries",
		Description:     "Milk, eggs, bread",
		Status:          domain.StatusPending,
		Category:        domain.CategoryPersonal,
		ProgressPercent: 0,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}
}

func TestToTodoResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		todo   domain.Todo
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
			todo: func() domain.Todo {
				td := validTodo()
				td.Status = domain.StatusInProgress
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
			todo: func() domain.Todo {
				td := validTodo()
				td.Category = domain.CategoryWork
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

func TestToTodoListResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		todos     []domain.Todo
		wantCount int
		wantLen   int
	}{
		{
			name:      "converts multiple todos",
			todos:     []domain.Todo{validTodo(), validTodo()},
			wantCount: 2,
			wantLen:   2,
		},
		{
			name:      "empty slice returns empty list",
			todos:     []domain.Todo{},
			wantCount: 0,
			wantLen:   0,
		},
		{
			name:      "nil slice returns empty list",
			todos:     nil,
			wantCount: 0,
			wantLen:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := dto.ToTodoListResponse(tt.todos)
			if got.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", got.Count, tt.wantCount)
			}
			if len(got.Todos) != tt.wantLen {
				t.Errorf("len(Todos) = %d, want %d", len(got.Todos), tt.wantLen)
			}
		})
	}
}

func TestTodoResponse_JSONSerialization(t *testing.T) {
	t.Parallel()

	resp := dto.ToTodoResponse(&domain.Todo{
		ID:              42,
		Title:           "Test",
		Description:     "Desc",
		Status:          domain.StatusDone,
		Category:        domain.CategoryOther,
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
