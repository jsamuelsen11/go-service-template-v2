package ports

import (
	"context"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

// TodoClient defines the client port for downstream TODO API operations.
// Implemented by the ACL adapter; called by the application layer.
// Methods map 1:1 to downstream API endpoints using domain terminology.
// The ACL translates between our "Project" concept and the downstream "Group" concept.
type TodoClient interface {
	// ListTodos returns todos matching the given filter criteria.
	// Pass a zero-value Filter to list all todos.
	ListTodos(ctx context.Context, filter todo.Filter) ([]todo.Todo, error)

	// GetTodo returns a single todo by ID.
	// Returns domain.ErrNotFound if the todo does not exist.
	GetTodo(ctx context.Context, id int64) (*todo.Todo, error)

	// CreateTodo creates a new todo and returns the created entity.
	// The todo's ProjectID field maps to the downstream group_id.
	CreateTodo(ctx context.Context, todo *todo.Todo) (*todo.Todo, error)

	// UpdateTodo updates an existing todo and returns the updated entity.
	// Returns domain.ErrNotFound if the todo does not exist.
	UpdateTodo(ctx context.Context, id int64, todo *todo.Todo) (*todo.Todo, error)

	// DeleteTodo deletes a todo by ID.
	// Returns domain.ErrNotFound if the todo does not exist.
	DeleteTodo(ctx context.Context, id int64) error

	// ListProjects returns all projects (mapped from downstream groups).
	// Returned projects do not include their todos.
	ListProjects(ctx context.Context) ([]project.Project, error)

	// GetProject returns a single project by ID (mapped from downstream group).
	// Returns domain.ErrNotFound if the project does not exist.
	GetProject(ctx context.Context, id int64) (*project.Project, error)

	// CreateProject creates a new project and returns the created entity.
	CreateProject(ctx context.Context, project *project.Project) (*project.Project, error)

	// UpdateProject updates an existing project and returns the updated entity.
	// Returns domain.ErrNotFound if the project does not exist.
	UpdateProject(ctx context.Context, id int64, project *project.Project) (*project.Project, error)

	// DeleteProject deletes a project by ID. Todos become ungrouped.
	// Returns domain.ErrNotFound if the project does not exist.
	DeleteProject(ctx context.Context, id int64) error

	// GetProjectTodos returns todos belonging to a specific project,
	// optionally filtered by status and category.
	// Returns domain.ErrNotFound if the project does not exist.
	GetProjectTodos(ctx context.Context, projectID int64, filter todo.Filter) ([]todo.Todo, error)
}
