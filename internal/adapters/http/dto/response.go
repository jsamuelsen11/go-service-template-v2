// Package dto provides HTTP request/response data transfer objects and
// RFC 9457 Problem Details error responses for the inbound HTTP adapter layer.
package dto

import (
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
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

// TodoListResponse represents a list of TODO items in HTTP responses.
type TodoListResponse struct {
	Todos []TodoResponse `json:"todos"`
	Count int            `json:"count"`
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

// ToTodoListResponse converts a slice of domain Todo entities to an HTTP
// list response DTO.
func ToTodoListResponse(todos []todo.Todo) TodoListResponse {
	items := make([]TodoResponse, len(todos))
	for i := range todos {
		items[i] = ToTodoResponse(&todos[i])
	}
	return TodoListResponse{
		Todos: items,
		Count: len(items),
	}
}
