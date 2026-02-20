package ports

import (
	"context"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

// ProjectService defines the service port for project aggregate operations.
// Implemented by the application layer; called by inbound adapters (handlers).
// A project is a named collection of todos that maps to the downstream "group"
// concept through the anti-corruption layer.
type ProjectService interface {
	// ListProjects returns all projects without populating their todos.
	ListProjects(ctx context.Context) ([]project.Project, error)

	// GetProject returns a single project by ID with its todos populated.
	// Returns domain.ErrNotFound if the project does not exist.
	GetProject(ctx context.Context, id int64) (*project.Project, error)

	// CreateProject creates a new project and returns the created entity
	// with server-assigned fields (ID, timestamps).
	// Returns domain.ErrValidation if the project fails validation.
	CreateProject(ctx context.Context, project *project.Project) (*project.Project, error)

	// UpdateProject updates an existing project's metadata and returns
	// the updated entity.
	// Returns domain.ErrNotFound if the project does not exist.
	UpdateProject(ctx context.Context, id int64, project *project.Project) (*project.Project, error)

	// DeleteProject deletes a project. Todos in the project become ungrouped.
	// Returns domain.ErrNotFound if the project does not exist.
	DeleteProject(ctx context.Context, id int64) error

	// AddTodo creates a new todo within the specified project.
	// Returns domain.ErrNotFound if the project does not exist.
	// Returns domain.ErrValidation if the todo fails validation.
	AddTodo(ctx context.Context, projectID int64, todo *todo.Todo) (*todo.Todo, error)

	// UpdateTodo updates an existing todo within the specified project.
	// Returns domain.ErrNotFound if the project or todo does not exist.
	UpdateTodo(ctx context.Context, projectID, todoID int64, todo *todo.Todo) (*todo.Todo, error)

	// RemoveTodo deletes a todo from the specified project.
	// Returns domain.ErrNotFound if the project or todo does not exist.
	RemoveTodo(ctx context.Context, projectID, todoID int64) error

	// BulkUpdateTodos updates multiple todos within the specified project
	// concurrently. Uses partial success semantics: each update succeeds or
	// fails independently. Returns a hard error only for request-level
	// failures (project not found, validation). Individual update failures
	// are collected in BulkUpdateResult.Errors.
	BulkUpdateTodos(ctx context.Context, projectID int64, updates []TodoUpdate) (*BulkUpdateResult, error)
}

// TodoUpdate pairs a todo ID with the updated todo data for bulk operations.
type TodoUpdate struct {
	TodoID int64
	Todo   *todo.Todo
}

// BulkUpdateError records a single failed todo update within a bulk operation.
type BulkUpdateError struct {
	TodoID int64
	Err    error
}

// BulkUpdateResult holds the outcomes of a bulk update operation.
// Updated contains successfully updated todos; Errors contains per-item failures.
type BulkUpdateResult struct {
	Updated []todo.Todo
	Errors  []BulkUpdateError
}
