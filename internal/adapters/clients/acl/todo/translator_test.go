package todo

import (
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

func ptrInt64(v int64) *int64 { return &v }

func TestToDomainTodo_FieldMapping(t *testing.T) {
	t.Parallel()

	dto := &todoDTO{
		ID:              42,
		Title:           "Buy groceries",
		Description:     "Milk, eggs, bread",
		Status:          "pending",
		Category:        "personal",
		ProgressPercent: 25,
		GroupID:         ptrInt64(7),
		CreatedAt:       "2026-02-12T15:04:05Z",
		UpdatedAt:       "2026-02-12T15:04:05Z",
	}

	got := ToDomainTodo(dto)

	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if got.Title != "Buy groceries" {
		t.Errorf("Title = %q, want %q", got.Title, "Buy groceries")
	}
	if got.Description != "Milk, eggs, bread" {
		t.Errorf("Description = %q, want %q", got.Description, "Milk, eggs, bread")
	}
	if got.Status != domain.StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusPending)
	}
	if got.Category != domain.CategoryPersonal {
		t.Errorf("Category = %q, want %q", got.Category, domain.CategoryPersonal)
	}
	if got.ProgressPercent != 25 {
		t.Errorf("ProgressPercent = %d, want 25", got.ProgressPercent)
	}
}

func TestToDomainTodo_GroupIDMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		groupID   *int64
		wantNil   bool
		wantValue int64
	}{
		{
			name:      "GroupID maps to ProjectID",
			groupID:   ptrInt64(7),
			wantValue: 7,
		},
		{
			name:    "nil GroupID maps to nil ProjectID",
			groupID: nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToDomainTodo(&todoDTO{
				GroupID:   tt.groupID,
				CreatedAt: "2026-02-12T15:04:05Z",
				UpdatedAt: "2026-02-12T15:04:05Z",
			})
			if tt.wantNil && got.ProjectID != nil {
				t.Errorf("ProjectID = %d, want nil", *got.ProjectID)
			}
			if !tt.wantNil {
				if got.ProjectID == nil {
					t.Fatal("ProjectID is nil, want non-nil")
				}
				if *got.ProjectID != tt.wantValue {
					t.Errorf("ProjectID = %d, want %d", *got.ProjectID, tt.wantValue)
				}
			}
		})
	}
}

func TestToDomainTodo_Timestamps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		createdAt   string
		updatedAt   string
		wantCreated time.Time
		wantUpdated time.Time
	}{
		{
			name:        "parses RFC3339 timestamps",
			createdAt:   "2026-02-12T15:04:05Z",
			updatedAt:   "2026-02-12T16:04:05Z",
			wantCreated: time.Date(2026, 2, 12, 15, 4, 5, 0, time.UTC),
			wantUpdated: time.Date(2026, 2, 12, 16, 4, 5, 0, time.UTC),
		},
		{
			name:      "invalid timestamp defaults to zero time",
			createdAt: "not-a-date",
			updatedAt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToDomainTodo(&todoDTO{
				CreatedAt: tt.createdAt,
				UpdatedAt: tt.updatedAt,
			})
			if !got.CreatedAt.Equal(tt.wantCreated) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, tt.wantCreated)
			}
			if !got.UpdatedAt.Equal(tt.wantUpdated) {
				t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, tt.wantUpdated)
			}
		})
	}
}

func TestToDomainTodo_ProgressPercent(t *testing.T) {
	t.Parallel()

	got := ToDomainTodo(&todoDTO{
		ProgressPercent: 100,
		CreatedAt:       "2026-02-12T15:04:05Z",
		UpdatedAt:       "2026-02-12T15:04:05Z",
	})

	if got.ProgressPercent != 100 {
		t.Errorf("ProgressPercent = %d, want 100", got.ProgressPercent)
	}
}

func TestToDomainTodoList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dto       todoListResponseDTO
		wantLen   int
		wantFirst int64
	}{
		{
			name: "converts multiple todos",
			dto: todoListResponseDTO{
				Todos: []todoDTO{
					{ID: 1, CreatedAt: "2026-02-12T15:04:05Z", UpdatedAt: "2026-02-12T15:04:05Z"},
					{ID: 2, CreatedAt: "2026-02-12T15:04:05Z", UpdatedAt: "2026-02-12T15:04:05Z"},
					{ID: 3, CreatedAt: "2026-02-12T15:04:05Z", UpdatedAt: "2026-02-12T15:04:05Z"},
				},
				Count: 3,
			},
			wantLen:   3,
			wantFirst: 1,
		},
		{
			name: "empty list",
			dto: todoListResponseDTO{
				Todos: []todoDTO{},
				Count: 0,
			},
			wantLen: 0,
		},
		{
			name: "nil todos slice",
			dto: todoListResponseDTO{
				Todos: nil,
				Count: 0,
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToDomainTodoList(tt.dto)
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantLen > 0 && got[0].ID != tt.wantFirst {
				t.Errorf("first ID = %d, want %d", got[0].ID, tt.wantFirst)
			}
		})
	}
}

func TestToCreateTodoRequest(t *testing.T) {
	t.Parallel()

	projectID := int64(5)

	tests := []struct {
		name   string
		todo   *domain.Todo
		verify func(t *testing.T, got createTodoRequestDTO)
	}{
		{
			name: "maps all fields",
			todo: &domain.Todo{
				Title:           "Buy groceries",
				Description:     "Milk, eggs, bread",
				Status:          domain.StatusPending,
				Category:        domain.CategoryPersonal,
				ProgressPercent: 50,
				ProjectID:       &projectID,
			},
			verify: func(t *testing.T, got createTodoRequestDTO) {
				t.Helper()
				if got.Title != "Buy groceries" {
					t.Errorf("Title = %q, want %q", got.Title, "Buy groceries")
				}
				if got.Description != "Milk, eggs, bread" {
					t.Errorf("Description = %q, want %q", got.Description, "Milk, eggs, bread")
				}
				if got.Status != "pending" {
					t.Errorf("Status = %q, want %q", got.Status, "pending")
				}
				if got.Category != "personal" {
					t.Errorf("Category = %q, want %q", got.Category, "personal")
				}
				if got.ProgressPercent != 50 {
					t.Errorf("ProgressPercent = %d, want 50", got.ProgressPercent)
				}
			},
		},
		{
			name: "ProjectID maps to GroupID",
			todo: &domain.Todo{
				Status:    domain.StatusPending,
				Category:  domain.CategoryWork,
				ProjectID: &projectID,
			},
			verify: func(t *testing.T, got createTodoRequestDTO) {
				t.Helper()
				if got.GroupID == nil {
					t.Fatal("GroupID is nil, want non-nil")
				}
				if *got.GroupID != 5 {
					t.Errorf("GroupID = %d, want 5", *got.GroupID)
				}
			},
		},
		{
			name: "nil ProjectID maps to nil GroupID",
			todo: &domain.Todo{
				Status:    domain.StatusPending,
				Category:  domain.CategoryWork,
				ProjectID: nil,
			},
			verify: func(t *testing.T, got createTodoRequestDTO) {
				t.Helper()
				if got.GroupID != nil {
					t.Errorf("GroupID = %d, want nil", *got.GroupID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToCreateTodoRequest(tt.todo)
			tt.verify(t, got)
		})
	}
}

// requirePtrEqual asserts a pointer field is non-nil and equals the expected value.
func requirePtrEqual[T comparable](t *testing.T, field string, got *T, want T) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %v", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %v, want %v", field, *got, want)
	}
}

func TestToUpdateTodoRequest(t *testing.T) {
	t.Parallel()

	projectID := int64(3)

	tests := []struct {
		name   string
		todo   *domain.Todo
		verify func(t *testing.T, got updateTodoRequestDTO)
	}{
		{
			name: "sets all fields as pointers",
			todo: &domain.Todo{
				Title:           "Updated title",
				Description:     "Updated desc",
				Status:          domain.StatusInProgress,
				Category:        domain.CategoryWork,
				ProgressPercent: 75,
				ProjectID:       &projectID,
			},
			verify: func(t *testing.T, got updateTodoRequestDTO) {
				t.Helper()
				requirePtrEqual(t, "Title", got.Title, "Updated title")
				requirePtrEqual(t, "Description", got.Description, "Updated desc")
				requirePtrEqual(t, "Status", got.Status, "in_progress")
				requirePtrEqual(t, "Category", got.Category, "work")
				requirePtrEqual(t, "ProgressPercent", got.ProgressPercent, int64(75))
				requirePtrEqual(t, "GroupID", got.GroupID, int64(3))
			},
		},
		{
			name: "nil ProjectID maps to nil GroupID",
			todo: &domain.Todo{
				Status:    domain.StatusDone,
				Category:  domain.CategoryOther,
				ProjectID: nil,
			},
			verify: func(t *testing.T, got updateTodoRequestDTO) {
				t.Helper()
				if got.GroupID != nil {
					t.Errorf("GroupID = %d, want nil", *got.GroupID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ToUpdateTodoRequest(tt.todo)
			tt.verify(t, got)
		})
	}
}
