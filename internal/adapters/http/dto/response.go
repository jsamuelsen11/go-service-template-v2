// Package dto provides HTTP request/response data transfer objects and
// RFC 9457 Problem Details error responses for the inbound HTTP adapter layer.
package dto

import (
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

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
func ToTodoResponse(t *domain.Todo) TodoResponse {
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
func ToTodoListResponse(todos []domain.Todo) TodoListResponse {
	items := make([]TodoResponse, len(todos))
	for i := range todos {
		items[i] = ToTodoResponse(&todos[i])
	}
	return TodoListResponse{
		Todos: items,
		Count: len(items),
	}
}
