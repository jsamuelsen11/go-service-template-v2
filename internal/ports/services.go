package ports

import (
	"context"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// ProjectService defines the service port for project aggregate operations.
// Implemented by the application layer; called by inbound adapters (handlers).
// A project is a named collection of todos that maps to the downstream "group"
// concept through the anti-corruption layer.
type ProjectService interface {
	// ListProjects returns all projects without populating their todos.
	ListProjects(ctx context.Context) ([]domain.Project, error)

	// GetProject returns a single project by ID with its todos populated.
	// Returns domain.ErrNotFound if the project does not exist.
	GetProject(ctx context.Context, id int64) (*domain.Project, error)

	// CreateProject creates a new project and returns the created entity
	// with server-assigned fields (ID, timestamps).
	// Returns domain.ErrValidation if the project fails validation.
	CreateProject(ctx context.Context, project *domain.Project) (*domain.Project, error)

	// UpdateProject updates an existing project's metadata and returns
	// the updated entity.
	// Returns domain.ErrNotFound if the project does not exist.
	UpdateProject(ctx context.Context, id int64, project *domain.Project) (*domain.Project, error)

	// DeleteProject deletes a project. Todos in the project become ungrouped.
	// Returns domain.ErrNotFound if the project does not exist.
	DeleteProject(ctx context.Context, id int64) error

	// AddTodo creates a new todo within the specified project.
	// Returns domain.ErrNotFound if the project does not exist.
	// Returns domain.ErrValidation if the todo fails validation.
	AddTodo(ctx context.Context, projectID int64, todo *domain.Todo) (*domain.Todo, error)

	// UpdateTodo updates an existing todo within the specified project.
	// Returns domain.ErrNotFound if the project or todo does not exist.
	UpdateTodo(ctx context.Context, projectID, todoID int64, todo *domain.Todo) (*domain.Todo, error)

	// RemoveTodo deletes a todo from the specified project.
	// Returns domain.ErrNotFound if the project or todo does not exist.
	RemoveTodo(ctx context.Context, projectID, todoID int64) error
}
