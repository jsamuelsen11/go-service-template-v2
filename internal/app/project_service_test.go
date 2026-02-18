package app

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
	"github.com/jsamuelsen11/go-service-template-v2/mocks"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func int64Ptr(v int64) *int64 { return &v }

func validProject() project.Project {
	return project.Project{
		ID:          1,
		Name:        "Sprint 1",
		Description: "First sprint tasks",
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
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
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// --- ListProjects ---

func TestProjectService_ListProjects(t *testing.T) {
	t.Parallel()

	t.Run("returns projects on success", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		want := []project.Project{
			{ID: 1, Name: "Project A", Description: "Desc A"},
			{ID: 2, Name: "Project B", Description: "Desc B"},
		}
		mockClient.EXPECT().ListProjects(mock.Anything).Return(want, nil)

		got, err := svc.ListProjects(context.Background())
		if err != nil {
			t.Fatalf("ListProjects() error = %v, want nil", err)
		}
		if len(got) != 2 {
			t.Errorf("ListProjects() len = %d, want 2", len(got))
		}
		if got[0].Name != "Project A" {
			t.Errorf("ListProjects()[0].Name = %q, want %q", got[0].Name, "Project A")
		}
	})

	t.Run("returns error when client fails", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		mockClient.EXPECT().ListProjects(mock.Anything).Return(nil, domain.ErrUnavailable)

		_, err := svc.ListProjects(context.Background())
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("ListProjects() error = %v, want ErrUnavailable", err)
		}
	})
}

// --- GetProject ---

func TestProjectService_GetProject(t *testing.T) {
	t.Parallel()

	t.Run("returns project with todos populated", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		todos := []todo.Todo{
			{ID: 10, Title: "Todo A", Description: "Desc A", Status: todo.StatusPending, Category: todo.CategoryWork},
			{ID: 11, Title: "Todo B", Description: "Desc B", Status: todo.StatusDone, Category: todo.CategoryPersonal},
		}

		mockClient.EXPECT().GetProject(mock.Anything, int64(1)).Return(&proj, nil)
		mockClient.EXPECT().GetProjectTodos(mock.Anything, int64(1), todo.Filter{}).Return(todos, nil)

		got, err := svc.GetProject(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetProject() error = %v, want nil", err)
		}
		if got.ID != 1 {
			t.Errorf("GetProject().ID = %d, want 1", got.ID)
		}
		if len(got.Todos) != 2 {
			t.Errorf("GetProject().Todos len = %d, want 2", len(got.Todos))
		}
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		mockClient.EXPECT().GetProject(mock.Anything, int64(99)).Return(nil, domain.ErrNotFound)

		_, err := svc.GetProject(context.Background(), 99)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("GetProject() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("returns error when fetching todos fails", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		mockClient.EXPECT().GetProject(mock.Anything, int64(1)).Return(&proj, nil)
		mockClient.EXPECT().GetProjectTodos(mock.Anything, int64(1), todo.Filter{}).Return(nil, domain.ErrUnavailable)

		_, err := svc.GetProject(context.Background(), 1)
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("GetProject() error = %v, want ErrUnavailable", err)
		}
	})
}

// --- CreateProject ---

func TestProjectService_CreateProject(t *testing.T) {
	t.Parallel()

	t.Run("creates valid project", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		input := &project.Project{Name: "New Project", Description: "A new project"}
		created := &project.Project{ID: 5, Name: "New Project", Description: "A new project"}

		mockClient.EXPECT().CreateProject(mock.Anything, input).Return(created, nil)

		got, err := svc.CreateProject(context.Background(), input)
		if err != nil {
			t.Fatalf("CreateProject() error = %v, want nil", err)
		}
		if got.ID != 5 {
			t.Errorf("CreateProject().ID = %d, want 5", got.ID)
		}
	})

	t.Run("returns validation error for invalid project", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		invalid := &project.Project{Name: "", Description: ""}

		_, err := svc.CreateProject(context.Background(), invalid)
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("CreateProject() error = %v, want ErrValidation", err)
		}
	})

	t.Run("returns error when client fails", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		input := &project.Project{Name: "Project", Description: "Desc"}
		mockClient.EXPECT().CreateProject(mock.Anything, input).Return(nil, domain.ErrUnavailable)

		_, err := svc.CreateProject(context.Background(), input)
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("CreateProject() error = %v, want ErrUnavailable", err)
		}
	})
}

// --- UpdateProject ---

func TestProjectService_UpdateProject(t *testing.T) {
	t.Parallel()

	t.Run("updates valid project", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		input := &project.Project{Name: "Updated", Description: "Updated desc"}
		updated := &project.Project{ID: 1, Name: "Updated", Description: "Updated desc"}

		mockClient.EXPECT().UpdateProject(mock.Anything, int64(1), input).Return(updated, nil)

		got, err := svc.UpdateProject(context.Background(), 1, input)
		if err != nil {
			t.Fatalf("UpdateProject() error = %v, want nil", err)
		}
		if got.Name != "Updated" {
			t.Errorf("UpdateProject().Name = %q, want %q", got.Name, "Updated")
		}
	})

	t.Run("returns validation error for invalid project", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		invalid := &project.Project{Name: "", Description: ""}

		_, err := svc.UpdateProject(context.Background(), 1, invalid)
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("UpdateProject() error = %v, want ErrValidation", err)
		}
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		input := &project.Project{Name: "Project", Description: "Desc"}
		mockClient.EXPECT().UpdateProject(mock.Anything, int64(99), input).Return(nil, domain.ErrNotFound)

		_, err := svc.UpdateProject(context.Background(), 99, input)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("UpdateProject() error = %v, want ErrNotFound", err)
		}
	})
}

// --- DeleteProject ---

func TestProjectService_DeleteProject(t *testing.T) {
	t.Parallel()

	t.Run("deletes project successfully", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		mockClient.EXPECT().DeleteProject(mock.Anything, int64(1)).Return(nil)

		err := svc.DeleteProject(context.Background(), 1)
		if err != nil {
			t.Errorf("DeleteProject() error = %v, want nil", err)
		}
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		mockClient.EXPECT().DeleteProject(mock.Anything, int64(99)).Return(domain.ErrNotFound)

		err := svc.DeleteProject(context.Background(), 99)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteProject() error = %v, want ErrNotFound", err)
		}
	})
}

// --- AddTodo ---

func TestProjectService_AddTodo(t *testing.T) {
	t.Parallel()

	t.Run("adds todo to project with ProjectID set", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		proj.ID = 5
		mockClient.EXPECT().GetProject(mock.Anything, int64(5)).Return(&proj, nil)

		td := validTodo()
		created := validTodo()
		created.ID = 42
		created.ProjectID = int64Ptr(5)

		mockClient.EXPECT().CreateTodo(mock.Anything, &td).Return(&created, nil)

		got, err := svc.AddTodo(context.Background(), 5, &td)
		if err != nil {
			t.Fatalf("AddTodo() error = %v, want nil", err)
		}
		if got.ID != 42 {
			t.Errorf("AddTodo().ID = %d, want 42", got.ID)
		}
		if td.ProjectID == nil || *td.ProjectID != 5 {
			t.Errorf("todo.ProjectID = %v, want 5", td.ProjectID)
		}
	})

	t.Run("returns validation error for invalid todo", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		invalid := &todo.Todo{Title: "", Description: "", Status: "bad", Category: "bad"}

		_, err := svc.AddTodo(context.Background(), 1, invalid)
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("AddTodo() error = %v, want ErrValidation", err)
		}
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		mockClient.EXPECT().GetProject(mock.Anything, int64(99)).Return(nil, domain.ErrNotFound)

		td := validTodo()
		_, err := svc.AddTodo(context.Background(), 99, &td)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("AddTodo() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("returns error when create fails", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		mockClient.EXPECT().GetProject(mock.Anything, int64(1)).Return(&proj, nil)
		mockClient.EXPECT().CreateTodo(mock.Anything, mock.Anything).Return(nil, domain.ErrUnavailable)

		td := validTodo()
		_, err := svc.AddTodo(context.Background(), 1, &td)
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("AddTodo() error = %v, want ErrUnavailable", err)
		}
	})
}

// --- UpdateTodo ---

func TestProjectService_UpdateTodo(t *testing.T) {
	t.Parallel()

	t.Run("updates todo in project", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		mockClient.EXPECT().GetProject(mock.Anything, int64(1)).Return(&proj, nil)

		td := validTodo()
		updated := validTodo()
		updated.ID = 10
		updated.ProjectID = int64Ptr(1)
		updated.Title = "Updated title"

		// Verify todoID (10) is passed to client, not projectID (1)
		mockClient.EXPECT().UpdateTodo(mock.Anything, int64(10), &td).Return(&updated, nil)

		got, err := svc.UpdateTodo(context.Background(), 1, 10, &td)
		if err != nil {
			t.Fatalf("UpdateTodo() error = %v, want nil", err)
		}
		if got.ID != 10 {
			t.Errorf("UpdateTodo().ID = %d, want 10", got.ID)
		}
		if td.ProjectID == nil || *td.ProjectID != 1 {
			t.Errorf("todo.ProjectID = %v, want 1", td.ProjectID)
		}
	})

	t.Run("returns validation error for invalid todo", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		invalid := &todo.Todo{Title: "", Description: "", Status: "bad", Category: "bad"}

		_, err := svc.UpdateTodo(context.Background(), 1, 10, invalid)
		if !errors.Is(err, domain.ErrValidation) {
			t.Errorf("UpdateTodo() error = %v, want ErrValidation", err)
		}
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		mockClient.EXPECT().GetProject(mock.Anything, int64(99)).Return(nil, domain.ErrNotFound)

		td := validTodo()
		_, err := svc.UpdateTodo(context.Background(), 99, 10, &td)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("UpdateTodo() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("returns error when update fails", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		mockClient.EXPECT().GetProject(mock.Anything, int64(1)).Return(&proj, nil)
		mockClient.EXPECT().UpdateTodo(mock.Anything, int64(10), mock.Anything).Return(nil, domain.ErrNotFound)

		td := validTodo()
		_, err := svc.UpdateTodo(context.Background(), 1, 10, &td)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("UpdateTodo() error = %v, want ErrNotFound", err)
		}
	})
}

// --- RemoveTodo ---

func TestProjectService_RemoveTodo(t *testing.T) {
	t.Parallel()

	t.Run("removes todo from project", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		mockClient.EXPECT().GetProject(mock.Anything, int64(1)).Return(&proj, nil)
		mockClient.EXPECT().DeleteTodo(mock.Anything, int64(10)).Return(nil)

		err := svc.RemoveTodo(context.Background(), 1, 10)
		if err != nil {
			t.Errorf("RemoveTodo() error = %v, want nil", err)
		}
	})

	t.Run("returns error when project not found", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		mockClient.EXPECT().GetProject(mock.Anything, int64(99)).Return(nil, domain.ErrNotFound)

		err := svc.RemoveTodo(context.Background(), 99, 10)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("RemoveTodo() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("returns error when todo not found", func(t *testing.T) {
		t.Parallel()
		mockClient := mocks.NewMockTodoClient(t)
		svc := NewProjectService(mockClient, discardLogger())

		proj := validProject()
		mockClient.EXPECT().GetProject(mock.Anything, int64(1)).Return(&proj, nil)
		mockClient.EXPECT().DeleteTodo(mock.Anything, int64(99)).Return(domain.ErrNotFound)

		err := svc.RemoveTodo(context.Background(), 1, 99)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("RemoveTodo() error = %v, want ErrNotFound", err)
		}
	})
}
