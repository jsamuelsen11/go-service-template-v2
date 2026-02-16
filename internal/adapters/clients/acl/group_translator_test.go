package acl

import (
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

func TestToDomainProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dto    groupDTO
		verify func(t *testing.T, got domain.Project)
	}{
		{
			name: "maps all fields",
			dto: groupDTO{
				ID:          10,
				Name:        "Sprint 1",
				Description: "First sprint tasks",
				CreatedAt:   "2026-02-12T15:04:05Z",
				UpdatedAt:   "2026-02-12T16:04:05Z",
			},
			verify: func(t *testing.T, got domain.Project) {
				t.Helper()
				if got.ID != 10 {
					t.Errorf("ID = %d, want 10", got.ID)
				}
				if got.Name != "Sprint 1" {
					t.Errorf("Name = %q, want %q", got.Name, "Sprint 1")
				}
				if got.Description != "First sprint tasks" {
					t.Errorf("Description = %q, want %q", got.Description, "First sprint tasks")
				}
			},
		},
		{
			name: "parses RFC3339 timestamps",
			dto: groupDTO{
				CreatedAt: "2026-02-12T15:04:05Z",
				UpdatedAt: "2026-02-12T16:04:05Z",
			},
			verify: func(t *testing.T, got domain.Project) {
				t.Helper()
				wantCreated := time.Date(2026, 2, 12, 15, 4, 5, 0, time.UTC)
				wantUpdated := time.Date(2026, 2, 12, 16, 4, 5, 0, time.UTC)
				if !got.CreatedAt.Equal(wantCreated) {
					t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, wantCreated)
				}
				if !got.UpdatedAt.Equal(wantUpdated) {
					t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, wantUpdated)
				}
			},
		},
		{
			name: "invalid timestamp defaults to zero time",
			dto: groupDTO{
				CreatedAt: "bad",
				UpdatedAt: "",
			},
			verify: func(t *testing.T, got domain.Project) {
				t.Helper()
				if !got.CreatedAt.IsZero() {
					t.Errorf("CreatedAt = %v, want zero time", got.CreatedAt)
				}
				if !got.UpdatedAt.IsZero() {
					t.Errorf("UpdatedAt = %v, want zero time", got.UpdatedAt)
				}
			},
		},
		{
			name: "Todos slice is nil by default",
			dto: groupDTO{
				CreatedAt: "2026-02-12T15:04:05Z",
				UpdatedAt: "2026-02-12T15:04:05Z",
			},
			verify: func(t *testing.T, got domain.Project) {
				t.Helper()
				if got.Todos != nil {
					t.Errorf("Todos = %v, want nil", got.Todos)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toDomainProject(tt.dto)
			tt.verify(t, got)
		})
	}
}

func TestToDomainProjectList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dto       groupListResponseDTO
		wantLen   int
		wantFirst int64
	}{
		{
			name: "converts multiple groups to projects",
			dto: groupListResponseDTO{
				Groups: []groupDTO{
					{ID: 1, Name: "Sprint 1", CreatedAt: "2026-02-12T15:04:05Z", UpdatedAt: "2026-02-12T15:04:05Z"},
					{ID: 2, Name: "Sprint 2", CreatedAt: "2026-02-12T15:04:05Z", UpdatedAt: "2026-02-12T15:04:05Z"},
				},
				Count: 2,
			},
			wantLen:   2,
			wantFirst: 1,
		},
		{
			name: "empty list",
			dto: groupListResponseDTO{
				Groups: []groupDTO{},
				Count:  0,
			},
			wantLen: 0,
		},
		{
			name: "nil groups slice",
			dto: groupListResponseDTO{
				Groups: nil,
				Count:  0,
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toDomainProjectList(tt.dto)
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantLen > 0 && got[0].ID != tt.wantFirst {
				t.Errorf("first ID = %d, want %d", got[0].ID, tt.wantFirst)
			}
		})
	}
}

func TestToCreateGroupRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		project *domain.Project
		verify  func(t *testing.T, got createGroupRequestDTO)
	}{
		{
			name: "maps name and description",
			project: &domain.Project{
				Name:        "Sprint 1",
				Description: "First sprint tasks",
			},
			verify: func(t *testing.T, got createGroupRequestDTO) {
				t.Helper()
				if got.Name != "Sprint 1" {
					t.Errorf("Name = %q, want %q", got.Name, "Sprint 1")
				}
				if got.Description != "First sprint tasks" {
					t.Errorf("Description = %q, want %q", got.Description, "First sprint tasks")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toCreateGroupRequest(tt.project)
			tt.verify(t, got)
		})
	}
}

func TestToUpdateGroupRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		project *domain.Project
		verify  func(t *testing.T, got updateGroupRequestDTO)
	}{
		{
			name: "sets all fields as pointers",
			project: &domain.Project{
				Name:        "Updated Sprint",
				Description: "Updated description",
			},
			verify: func(t *testing.T, got updateGroupRequestDTO) {
				t.Helper()
				if got.Name == nil || *got.Name != "Updated Sprint" {
					t.Errorf("Name = %v, want %q", got.Name, "Updated Sprint")
				}
				if got.Description == nil || *got.Description != "Updated description" {
					t.Errorf("Description = %v, want %q", got.Description, "Updated description")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toUpdateGroupRequest(tt.project)
			tt.verify(t, got)
		})
	}
}
