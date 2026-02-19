package acl

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	aclproject "github.com/jsamuelsen11/go-service-template-v2/internal/adapters/clients/acl/project"
	acltodo "github.com/jsamuelsen11/go-service-template-v2/internal/adapters/clients/acl/todo"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/httpclient"
	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"
)

// Compile-time interface check.
var _ ports.TodoClient = (*TodoClient)(nil)

// TodoClient is the outbound adapter for the downstream TODO API. It
// implements [ports.TodoClient] (11 CRUD methods for todos and projects).
//
// All methods translate between our domain types and the downstream API's
// representations via the ACL translators in sub-packages [acltodo] and
// [aclproject]. HTTP errors are mapped to domain errors (ErrNotFound,
// ErrValidation, etc.) by [TranslateHTTPError].
//
// The underlying [httpclient.Client] provides circuit breaking, retry with
// exponential backoff, OpenTelemetry tracing, and health checking
// ([ports.HealthChecker]) for every outbound call.
type TodoClient struct {
	req *Requester
}

// NewTodoClient creates a TodoClient that sends requests through the given
// [httpclient.Client]. The client's BaseURL should point to the downstream
// TODO API root (e.g. "https://todo-api.example.com"). The logger is used
// for error-level diagnostics on failed or unexpected responses.
func NewTodoClient(client *httpclient.Client, logger *slog.Logger) *TodoClient {
	return &TodoClient{
		req: NewRequester(client, logger),
	}
}

// --- Todo operations ---

// ListTodos fetches todos from GET /api/v1/todos, optionally filtered by
// status, category, and project (mapped to group_id). A zero-value
// [todo.Filter] returns all todos. Returns the translated domain
// slice or a domain error on failure.
func (c *TodoClient) ListTodos(ctx context.Context, filter todo.Filter) ([]todo.Todo, error) {
	path := "/api/v1/todos" + filterQuery(filter)

	var dto acltodo.TodoListResponseDTO
	if err := c.req.Do(ctx, http.MethodGet, path, nil, &dto); err != nil {
		return nil, err
	}
	return acltodo.ToDomainTodoList(dto), nil
}

// GetTodo fetches a single todo by ID from GET /api/v1/todos/{id}.
// Returns [domain.ErrNotFound] if the downstream API returns 404.
func (c *TodoClient) GetTodo(ctx context.Context, id int64) (*todo.Todo, error) {
	path := fmt.Sprintf("/api/v1/todos/%d", id)

	var dto acltodo.TodoDTO
	if err := c.req.Do(ctx, http.MethodGet, path, nil, &dto); err != nil {
		return nil, err
	}
	result := acltodo.ToDomainTodo(&dto)
	return &result, nil
}

// CreateTodo sends a POST /api/v1/todos with the translated request body
// and returns the created todo as a domain entity. Returns
// [domain.ErrValidation] if the downstream rejects the payload.
func (c *TodoClient) CreateTodo(ctx context.Context, t *todo.Todo) (*todo.Todo, error) {
	reqDTO := acltodo.ToCreateTodoRequest(t)

	var respDTO acltodo.TodoDTO
	if err := c.req.Do(ctx, http.MethodPost, "/api/v1/todos", reqDTO, &respDTO); err != nil {
		return nil, err
	}
	result := acltodo.ToDomainTodo(&respDTO)
	return &result, nil
}

// UpdateTodo sends a PUT /api/v1/todos/{id} with the translated request
// body and returns the updated todo. Returns [domain.ErrNotFound] if the
// todo does not exist or [domain.ErrValidation] if the payload is rejected.
func (c *TodoClient) UpdateTodo(ctx context.Context, id int64, t *todo.Todo) (*todo.Todo, error) {
	path := fmt.Sprintf("/api/v1/todos/%d", id)
	reqDTO := acltodo.ToUpdateTodoRequest(t)

	var respDTO acltodo.TodoDTO
	if err := c.req.Do(ctx, http.MethodPut, path, reqDTO, &respDTO); err != nil {
		return nil, err
	}
	result := acltodo.ToDomainTodo(&respDTO)
	return &result, nil
}

// DeleteTodo sends a DELETE /api/v1/todos/{id}. Returns
// [domain.ErrNotFound] if the todo does not exist.
func (c *TodoClient) DeleteTodo(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/v1/todos/%d", id)
	return c.req.Do(ctx, http.MethodDelete, path, nil, nil)
}

// --- Project operations (downstream "groups") ---

// ListProjects fetches all projects from GET /api/v1/groups. Projects are
// returned without their todos populated. The downstream "group" concept
// is translated to our domain "project" concept.
func (c *TodoClient) ListProjects(ctx context.Context) ([]project.Project, error) {
	var dto aclproject.GroupListResponseDTO
	if err := c.req.Do(ctx, http.MethodGet, "/api/v1/groups", nil, &dto); err != nil {
		return nil, err
	}
	return aclproject.ToDomainProjectList(dto), nil
}

// GetProject fetches a single project by ID from GET /api/v1/groups/{id}.
// Returns [domain.ErrNotFound] if the downstream API returns 404.
func (c *TodoClient) GetProject(ctx context.Context, id int64) (*project.Project, error) {
	path := fmt.Sprintf("/api/v1/groups/%d", id)

	var dto aclproject.GroupDTO
	if err := c.req.Do(ctx, http.MethodGet, path, nil, &dto); err != nil {
		return nil, err
	}
	result := aclproject.ToDomainProject(dto)
	return &result, nil
}

// CreateProject sends a POST /api/v1/groups with the translated request
// body and returns the created project. Returns [domain.ErrValidation]
// if the downstream rejects the payload.
func (c *TodoClient) CreateProject(ctx context.Context, p *project.Project) (*project.Project, error) {
	reqDTO := aclproject.ToCreateGroupRequest(p)

	var respDTO aclproject.GroupDTO
	if err := c.req.Do(ctx, http.MethodPost, "/api/v1/groups", reqDTO, &respDTO); err != nil {
		return nil, err
	}
	result := aclproject.ToDomainProject(respDTO)
	return &result, nil
}

// UpdateProject sends a PUT /api/v1/groups/{id} with the translated request
// body and returns the updated project. Returns [domain.ErrNotFound] if the
// project does not exist.
func (c *TodoClient) UpdateProject(ctx context.Context, id int64, p *project.Project) (*project.Project, error) {
	path := fmt.Sprintf("/api/v1/groups/%d", id)
	reqDTO := aclproject.ToUpdateGroupRequest(p)

	var respDTO aclproject.GroupDTO
	if err := c.req.Do(ctx, http.MethodPut, path, reqDTO, &respDTO); err != nil {
		return nil, err
	}
	result := aclproject.ToDomainProject(respDTO)
	return &result, nil
}

// DeleteProject sends a DELETE /api/v1/groups/{id}. Todos belonging to the
// project become ungrouped. Returns [domain.ErrNotFound] if the project
// does not exist.
func (c *TodoClient) DeleteProject(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/v1/groups/%d", id)
	return c.req.Do(ctx, http.MethodDelete, path, nil, nil)
}

// GetProjectTodos fetches todos belonging to a specific project from
// GET /api/v1/groups/{id}/todos. The filter's ProjectID field is ignored
// (the project is identified by the URL path). Status and category filters
// are forwarded as query parameters. Returns [domain.ErrNotFound] if the
// project does not exist.
func (c *TodoClient) GetProjectTodos(ctx context.Context, projectID int64, filter todo.Filter) ([]todo.Todo, error) {
	// Zero out ProjectID -- it's encoded in the URL path.
	filter.ProjectID = nil
	path := fmt.Sprintf("/api/v1/groups/%d/todos", projectID) + filterQuery(filter)

	var dto acltodo.TodoListResponseDTO
	if err := c.req.Do(ctx, http.MethodGet, path, nil, &dto); err != nil {
		return nil, err
	}
	return acltodo.ToDomainTodoList(dto), nil
}

// filterQuery converts a [todo.Filter] to a URL query string (including
// the leading "?"). Returns an empty string if no filters are set.
func filterQuery(f todo.Filter) string {
	v := url.Values{}
	if f.Status != "" {
		v.Set("status", f.Status.String())
	}
	if f.Category != "" {
		v.Set("category", f.Category.String())
	}
	if f.ProjectID != nil {
		v.Set("group_id", fmt.Sprintf("%d", *f.ProjectID))
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}
