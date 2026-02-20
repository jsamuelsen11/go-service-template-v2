// Package dto provides HTTP request/response data transfer objects and
// RFC 9457 Problem Details error responses for the inbound HTTP adapter layer.
package dto

import (
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"
)

// ProjectResponse represents a single project in HTTP responses.
type ProjectResponse struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Todos       []TodoResponse `json:"todos,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// ProjectListResponse represents a list of projects in HTTP responses.
type ProjectListResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Count    int               `json:"count"`
}

// ToProjectResponse converts a domain Project entity to an HTTP response DTO.
// Todos are included only if the project has them populated.
func ToProjectResponse(p *project.Project) ProjectResponse {
	resp := ProjectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}

	if len(p.Todos) > 0 {
		resp.Todos = make([]TodoResponse, len(p.Todos))
		for i := range p.Todos {
			resp.Todos[i] = ToTodoResponse(&p.Todos[i])
		}
	}

	return resp
}

// ToProjectListResponse converts a slice of domain Project entities to an
// HTTP list response DTO.
func ToProjectListResponse(projects []project.Project) ProjectListResponse {
	items := make([]ProjectResponse, len(projects))
	for i := range projects {
		items[i] = ToProjectResponse(&projects[i])
	}
	return ProjectListResponse{
		Projects: items,
		Count:    len(items),
	}
}

// TodoResponse represents a single TODO item in HTTP responses.
type TodoResponse struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	Category        string `json:"category"`
	ProgressPercent int    `json:"progress_percent"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ToTodoResponse converts a domain Todo entity to an HTTP response DTO.
func ToTodoResponse(t *todo.Todo) TodoResponse {
	return TodoResponse{
		ID:              t.ID,
		Title:           t.Title,
		Description:     t.Description,
		Status:          t.Status.String(),
		Category:        t.Category.String(),
		ProgressPercent: t.ProgressPercent,
		CreatedAt:       t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       t.UpdatedAt.Format(time.RFC3339),
	}
}

// BulkUpdateTodosResponse represents the result of a bulk update operation.
// It includes both successful updates and per-item errors.
type BulkUpdateTodosResponse struct {
	Updated   []TodoResponse        `json:"updated"`
	Errors    []BulkUpdateErrorItem `json:"errors"`
	Total     int                   `json:"total"`
	Succeeded int                   `json:"succeeded"`
	Failed    int                   `json:"failed"`
}

// BulkUpdateErrorItem represents a single failed update within a bulk operation.
type BulkUpdateErrorItem struct {
	TodoID  int64  `json:"todo_id"`
	Message string `json:"message"`
}

// ToBulkUpdateResponse converts a ports.BulkUpdateResult to an HTTP response DTO.
func ToBulkUpdateResponse(result *ports.BulkUpdateResult) BulkUpdateTodosResponse {
	updated := make([]TodoResponse, len(result.Updated))
	for i := range result.Updated {
		updated[i] = ToTodoResponse(&result.Updated[i])
	}

	errs := make([]BulkUpdateErrorItem, len(result.Errors))
	for i, e := range result.Errors {
		errs[i] = BulkUpdateErrorItem{
			TodoID:  e.TodoID,
			Message: e.Err.Error(),
		}
	}

	total := len(result.Updated) + len(result.Errors)
	return BulkUpdateTodosResponse{
		Updated:   updated,
		Errors:    errs,
		Total:     total,
		Succeeded: len(result.Updated),
		Failed:    len(result.Errors),
	}
}
