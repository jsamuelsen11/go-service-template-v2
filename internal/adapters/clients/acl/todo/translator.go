package todo

import (
	"time"

	domtodo "github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

// ToDomainTodo converts a downstream TodoDTO to a domain Todo entity.
// Maps GroupID to ProjectID and parses RFC3339 timestamps.
func ToDomainTodo(dto *TodoDTO) domtodo.Todo {
	createdAt, _ := time.Parse(time.RFC3339, dto.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, dto.UpdatedAt)

	return domtodo.Todo{
		ID:              dto.ID,
		Title:           dto.Title,
		Description:     dto.Description,
		Status:          domtodo.Status(dto.Status),
		Category:        domtodo.Category(dto.Category),
		ProgressPercent: int(dto.ProgressPercent),
		ProjectID:       dto.GroupID,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

// ToDomainTodoList converts a downstream TodoListResponseDTO to a slice of
// domain Todo entities.
func ToDomainTodoList(dto TodoListResponseDTO) []domtodo.Todo {
	todos := make([]domtodo.Todo, len(dto.Todos))
	for i := range dto.Todos {
		todos[i] = ToDomainTodo(&dto.Todos[i])
	}
	return todos
}

// ToCreateTodoRequest converts a domain Todo entity to a downstream
// CreateTodoRequestDTO. Maps ProjectID to GroupID.
func ToCreateTodoRequest(todo *domtodo.Todo) CreateTodoRequestDTO {
	return CreateTodoRequestDTO{
		Title:           todo.Title,
		Description:     todo.Description,
		Status:          todo.Status.String(),
		Category:        todo.Category.String(),
		ProgressPercent: int64(todo.ProgressPercent),
		GroupID:         todo.ProjectID,
	}
}

// ToUpdateTodoRequest converts a domain Todo entity to a downstream
// UpdateTodoRequestDTO. All fields are set (full replacement semantics).
func ToUpdateTodoRequest(todo *domtodo.Todo) UpdateTodoRequestDTO {
	status := todo.Status.String()
	category := todo.Category.String()
	progress := int64(todo.ProgressPercent)

	return UpdateTodoRequestDTO{
		Title:           &todo.Title,
		Description:     &todo.Description,
		Status:          &status,
		Category:        &category,
		ProgressPercent: &progress,
		GroupID:         todo.ProjectID,
	}
}
