// Package todo implements the Anti-Corruption Layer translators for the
// downstream TODO API's todo resources.
package todo

// TodoDTO matches the downstream Todo schema.
// Fields use int64 to match the OpenAPI spec's format: int64 annotation.
type TodoDTO struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	Category        string `json:"category"`
	ProgressPercent int64  `json:"progress_percent"`
	GroupID         *int64 `json:"group_id,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// CreateTodoRequestDTO matches the downstream CreateTodoRequest schema.
type CreateTodoRequestDTO struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Status          string `json:"status,omitempty"`
	Category        string `json:"category,omitempty"`
	ProgressPercent int64  `json:"progress_percent,omitempty"`
	GroupID         *int64 `json:"group_id,omitempty"`
}

// UpdateTodoRequestDTO matches the downstream UpdateTodoRequest schema.
// All fields are optional; nil means "do not change this field.".
type UpdateTodoRequestDTO struct {
	Title           *string `json:"title,omitempty"`
	Description     *string `json:"description,omitempty"`
	Status          *string `json:"status,omitempty"`
	Category        *string `json:"category,omitempty"`
	ProgressPercent *int64  `json:"progress_percent,omitempty"`
	GroupID         *int64  `json:"group_id,omitempty"`
}

// TodoListResponseDTO matches the downstream TodoListResponse schema.
type TodoListResponseDTO struct {
	Todos []TodoDTO `json:"todos"`
	Count int64     `json:"count"`
}
