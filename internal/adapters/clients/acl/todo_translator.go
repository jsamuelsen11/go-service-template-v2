package acl

import (
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// toDomainTodo converts a downstream todoDTO to a domain Todo entity.
// Maps GroupID to ProjectID and parses RFC3339 timestamps.
func toDomainTodo(dto *todoDTO) domain.Todo {
	createdAt, _ := time.Parse(time.RFC3339, dto.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, dto.UpdatedAt)

	return domain.Todo{
		ID:              dto.ID,
		Title:           dto.Title,
		Description:     dto.Description,
		Status:          domain.TodoStatus(dto.Status),
		Category:        domain.TodoCategory(dto.Category),
		ProgressPercent: int(dto.ProgressPercent),
		ProjectID:       dto.GroupID,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

// toDomainTodoList converts a downstream todoListResponseDTO to a slice of
// domain Todo entities.
func toDomainTodoList(dto todoListResponseDTO) []domain.Todo {
	todos := make([]domain.Todo, len(dto.Todos))
	for i := range dto.Todos {
		todos[i] = toDomainTodo(&dto.Todos[i])
	}
	return todos
}

// toCreateTodoRequest converts a domain Todo entity to a downstream
// createTodoRequestDTO. Maps ProjectID to GroupID.
func toCreateTodoRequest(todo *domain.Todo) createTodoRequestDTO {
	return createTodoRequestDTO{
		Title:           todo.Title,
		Description:     todo.Description,
		Status:          todo.Status.String(),
		Category:        todo.Category.String(),
		ProgressPercent: int64(todo.ProgressPercent),
		GroupID:         todo.ProjectID,
	}
}

// toUpdateTodoRequest converts a domain Todo entity to a downstream
// updateTodoRequestDTO. All fields are set (full replacement semantics).
func toUpdateTodoRequest(todo *domain.Todo) updateTodoRequestDTO {
	status := todo.Status.String()
	category := todo.Category.String()
	progress := int64(todo.ProgressPercent)

	return updateTodoRequestDTO{
		Title:           &todo.Title,
		Description:     &todo.Description,
		Status:          &status,
		Category:        &category,
		ProgressPercent: &progress,
		GroupID:         todo.ProjectID,
	}
}
