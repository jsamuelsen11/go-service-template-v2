package handlers

import (
	"net/http"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"
)

// TodoHandler handles HTTP requests for flat todo CRUD operations.
type TodoHandler struct {
	client ports.TodoClient
}

// NewTodoHandler creates a new TodoHandler with the given client port.
func NewTodoHandler(client ports.TodoClient) *TodoHandler {
	return &TodoHandler{client: client}
}

// ListTodos handles GET /api/v1/todos.
func (h *TodoHandler) ListTodos(w http.ResponseWriter, r *http.Request) {
	filter, err := parseTodoFilter(r)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	todos, err := h.client.ListTodos(r.Context(), filter)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToTodoListResponse(todos))
}

// CreateTodo handles POST /api/v1/todos.
func (h *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	t := decodeTodoCreate(w, r)
	if t == nil {
		return
	}

	created, err := h.client.CreateTodo(r.Context(), t)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToTodoResponse(created))
}

// GetTodo handles GET /api/v1/todos/{id}.
func (h *TodoHandler) GetTodo(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	t, err := h.client.GetTodo(r.Context(), id)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToTodoResponse(t))
}

// UpdateTodo handles PATCH /api/v1/todos/{id}.
func (h *TodoHandler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	t := decodeTodoUpdate(w, r)
	if t == nil {
		return
	}

	updated, err := h.client.UpdateTodo(r.Context(), id, t)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToTodoResponse(updated))
}

// DeleteTodo handles DELETE /api/v1/todos/{id}.
func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	if err := h.client.DeleteTodo(r.Context(), id); err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
