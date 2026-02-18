// Package app provides application services that orchestrate use cases by
// coordinating between domain logic and infrastructure through port interfaces.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"
)

// Compile-time check that ProjectService implements ports.ProjectService.
var _ ports.ProjectService = (*ProjectService)(nil)

// ProjectService implements ports.ProjectService by orchestrating calls to the
// downstream TODO API through the TodoClient port. It handles validation,
// structured logging, and multi-step coordination but contains no business logic.
type ProjectService struct {
	todoClient ports.TodoClient
	logger     *slog.Logger
}

// NewProjectService creates a ProjectService. The client port provides access
// to the downstream TODO API for project and todo operations. The logger is
// used for structured request/error logging.
func NewProjectService(client ports.TodoClient, logger *slog.Logger) *ProjectService {
	return &ProjectService{
		todoClient: client,
		logger:     logger,
	}
}

// ListProjects returns all projects without populating their todos.
func (s *ProjectService) ListProjects(ctx context.Context) ([]project.Project, error) {
	s.logger.InfoContext(ctx, "listing projects")

	projects, err := s.todoClient.ListProjects(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list projects",
			slog.String("operation", "ListProjects"),
			slog.Any("error", err),
		)
		return nil, err
	}

	return projects, nil
}

// GetProject returns a single project by ID with its todos populated.
func (s *ProjectService) GetProject(ctx context.Context, id int64) (*project.Project, error) {
	s.logger.InfoContext(ctx, "fetching project", slog.Int64("id", id))

	proj, err := s.todoClient.GetProject(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch project",
			slog.String("operation", "GetProject"),
			slog.Int64("id", id),
			slog.Any("error", err),
		)
		return nil, err
	}

	todos, err := s.todoClient.GetProjectTodos(ctx, id, todo.Filter{})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch project todos",
			slog.String("operation", "GetProject"),
			slog.Int64("project_id", id),
			slog.Any("error", err),
		)
		return nil, err
	}

	proj.Todos = todos
	return proj, nil
}

// CreateProject validates and creates a new project, returning the created
// entity with server-assigned fields (ID, timestamps).
func (s *ProjectService) CreateProject(ctx context.Context, p *project.Project) (*project.Project, error) {
	s.logger.InfoContext(ctx, "creating project", slog.String("name", p.Name))

	if err := p.Validate(); err != nil {
		return nil, err
	}

	created, err := s.todoClient.CreateProject(ctx, p)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create project",
			slog.String("operation", "CreateProject"),
			slog.Any("error", err),
		)
		return nil, err
	}

	return created, nil
}

// UpdateProject validates and updates an existing project's metadata.
func (s *ProjectService) UpdateProject(ctx context.Context, id int64, p *project.Project) (*project.Project, error) {
	s.logger.InfoContext(ctx, "updating project", slog.Int64("id", id))

	if err := p.Validate(); err != nil {
		return nil, err
	}

	updated, err := s.todoClient.UpdateProject(ctx, id, p)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update project",
			slog.String("operation", "UpdateProject"),
			slog.Int64("id", id),
			slog.Any("error", err),
		)
		return nil, err
	}

	return updated, nil
}

// DeleteProject deletes a project. Todos in the project become ungrouped.
func (s *ProjectService) DeleteProject(ctx context.Context, id int64) error {
	s.logger.InfoContext(ctx, "deleting project", slog.Int64("id", id))

	if err := s.todoClient.DeleteProject(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete project",
			slog.String("operation", "DeleteProject"),
			slog.Int64("id", id),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// AddTodo creates a new todo within the specified project.
func (s *ProjectService) AddTodo(ctx context.Context, projectID int64, td *todo.Todo) (*todo.Todo, error) {
	s.logger.InfoContext(ctx, "adding todo to project", slog.Int64("project_id", projectID))

	if err := td.Validate(); err != nil {
		return nil, err
	}

	if _, err := s.todoClient.GetProject(ctx, projectID); err != nil {
		s.logger.ErrorContext(ctx, "failed to verify project",
			slog.String("operation", "AddTodo"),
			slog.Int64("project_id", projectID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("verifying project: %w", err)
	}

	td.ProjectID = &projectID

	created, err := s.todoClient.CreateTodo(ctx, td)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create todo",
			slog.String("operation", "AddTodo"),
			slog.Int64("project_id", projectID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("creating todo: %w", err)
	}

	return created, nil
}

// UpdateTodo updates an existing todo within the specified project.
func (s *ProjectService) UpdateTodo(ctx context.Context, projectID, todoID int64, td *todo.Todo) (*todo.Todo, error) {
	s.logger.InfoContext(ctx, "updating todo in project",
		slog.Int64("project_id", projectID),
		slog.Int64("todo_id", todoID),
	)

	if err := td.Validate(); err != nil {
		return nil, err
	}

	if _, err := s.todoClient.GetProject(ctx, projectID); err != nil {
		s.logger.ErrorContext(ctx, "failed to verify project",
			slog.String("operation", "UpdateTodo"),
			slog.Int64("project_id", projectID),
			slog.Int64("todo_id", todoID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("verifying project: %w", err)
	}

	td.ProjectID = &projectID

	updated, err := s.todoClient.UpdateTodo(ctx, todoID, td)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update todo",
			slog.String("operation", "UpdateTodo"),
			slog.Int64("project_id", projectID),
			slog.Int64("todo_id", todoID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("updating todo: %w", err)
	}

	return updated, nil
}

// RemoveTodo deletes a todo from the specified project.
func (s *ProjectService) RemoveTodo(ctx context.Context, projectID, todoID int64) error {
	s.logger.InfoContext(ctx, "removing todo from project",
		slog.Int64("project_id", projectID),
		slog.Int64("todo_id", todoID),
	)

	if _, err := s.todoClient.GetProject(ctx, projectID); err != nil {
		s.logger.ErrorContext(ctx, "failed to verify project",
			slog.String("operation", "RemoveTodo"),
			slog.Int64("project_id", projectID),
			slog.Int64("todo_id", todoID),
			slog.Any("error", err),
		)
		return fmt.Errorf("verifying project: %w", err)
	}

	if err := s.todoClient.DeleteTodo(ctx, todoID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete todo",
			slog.String("operation", "RemoveTodo"),
			slog.Int64("project_id", projectID),
			slog.Int64("todo_id", todoID),
			slog.Any("error", err),
		)
		return fmt.Errorf("deleting todo: %w", err)
	}

	return nil
}
