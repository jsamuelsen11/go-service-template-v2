package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/handlers"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
	"github.com/jsamuelsen11/go-service-template-v2/mocks"
)

func newTodoHandler(t *testing.T) (*handlers.TodoHandler, *mocks.MockTodoClient) {
	t.Helper()
	client := mocks.NewMockTodoClient(t)
	return handlers.NewTodoHandler(client), client
}

// --- ListTodos ---

func TestListTodos_Success(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	todos := []todo.Todo{validTodo()}
	client.EXPECT().ListTodos(mock.Anything, todo.Filter{}).Return(todos, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	h.ListTodos(rec, req)

	requireStatus(t, rec, http.StatusOK)
	resp := decodeJSON[dto.TodoListResponse](t, rec)
	if resp.Count != 1 {
		t.Errorf("Count = %d, want 1", resp.Count)
	}
}

func TestListTodos_WithFilters(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	todos := []todo.Todo{validTodo()}
	client.EXPECT().ListTodos(mock.Anything, todo.Filter{
		Status:   todo.StatusPending,
		Category: todo.CategoryWork,
	}).Return(todos, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos?status=pending&category=work", nil)
	h.ListTodos(rec, req)

	requireStatus(t, rec, http.StatusOK)
}

func TestListTodos_InvalidStatusFilter(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos?status=bad", nil)
	h.ListTodos(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestListTodos_InvalidCategoryFilter(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos?category=bad", nil)
	h.ListTodos(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestListTodos_InvalidProjectIDFilter(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos?project_id=abc", nil)
	h.ListTodos(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestListTodos_ServiceError(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	client.EXPECT().ListTodos(mock.Anything, todo.Filter{}).Return(nil, domain.ErrUnavailable)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	h.ListTodos(rec, req)

	requireStatus(t, rec, http.StatusBadGateway)
}

// --- CreateTodo ---

func TestCreateTodo_Success(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	created := validTodo()
	client.EXPECT().CreateTodo(mock.Anything, mock.AnythingOfType("*todo.Todo")).
		Return(&created, nil)

	body := jsonBody(t, dto.CreateTodoRequest{Title: "Buy groceries", Description: "Milk, eggs, bread"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", body)
	req.Header.Set("Content-Type", "application/json")
	h.CreateTodo(rec, req)

	requireStatus(t, rec, http.StatusCreated)
	resp := decodeJSON[dto.TodoResponse](t, rec)
	if resp.Title != "Buy groceries" {
		t.Errorf("Title = %q, want %q", resp.Title, "Buy groceries")
	}
}

func TestCreateTodo_InvalidJSON(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	h.CreateTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestCreateTodo_ValidationError(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	body := jsonBody(t, dto.CreateTodoRequest{Title: "", Description: ""})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", body)
	req.Header.Set("Content-Type", "application/json")
	h.CreateTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

// --- GetTodo ---

func TestGetTodo_Success(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	td := validTodo()
	client.EXPECT().GetTodo(mock.Anything, int64(1)).Return(&td, nil)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodGet, "/api/v1/todos/1", nil), map[string]string{"id": "1"})
	h.GetTodo(rec, req)

	requireStatus(t, rec, http.StatusOK)
	resp := decodeJSON[dto.TodoResponse](t, rec)
	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
}

func TestGetTodo_InvalidID(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodGet, "/api/v1/todos/abc", nil), map[string]string{"id": "abc"})
	h.GetTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestGetTodo_NotFound(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	client.EXPECT().GetTodo(mock.Anything, int64(999)).Return(nil, domain.ErrNotFound)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodGet, "/api/v1/todos/999", nil), map[string]string{"id": "999"})
	h.GetTodo(rec, req)

	requireStatus(t, rec, http.StatusNotFound)
}

// --- UpdateTodo ---

func TestUpdateTodo_Success(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	updated := validTodo()
	updated.Title = testUpdatedValue
	client.EXPECT().UpdateTodo(mock.Anything, int64(1), mock.AnythingOfType("*todo.Todo")).
		Return(&updated, nil)

	title := testUpdatedValue
	body := jsonBody(t, dto.UpdateTodoRequest{Title: &title})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/1", body)
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "1"})
	h.UpdateTodo(rec, req)

	requireStatus(t, rec, http.StatusOK)
	resp := decodeJSON[dto.TodoResponse](t, rec)
	if resp.Title != testUpdatedValue {
		t.Errorf("Title = %q, want %q", resp.Title, testUpdatedValue)
	}
}

func TestUpdateTodo_InvalidID(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/abc", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "abc"})
	h.UpdateTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestUpdateTodo_InvalidJSON(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos/1", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "1"})
	h.UpdateTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

// --- DeleteTodo ---

func TestDeleteTodo_Success(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	client.EXPECT().DeleteTodo(mock.Anything, int64(1)).Return(nil)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/todos/1", nil), map[string]string{"id": "1"})
	h.DeleteTodo(rec, req)

	requireStatus(t, rec, http.StatusNoContent)
}

func TestDeleteTodo_InvalidID(t *testing.T) {
	t.Parallel()
	h, _ := newTodoHandler(t)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/todos/abc", nil), map[string]string{"id": "abc"})
	h.DeleteTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestDeleteTodo_NotFound(t *testing.T) {
	t.Parallel()
	h, client := newTodoHandler(t)

	client.EXPECT().DeleteTodo(mock.Anything, int64(999)).Return(domain.ErrNotFound)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/todos/999", nil), map[string]string{"id": "999"})
	h.DeleteTodo(rec, req)

	requireStatus(t, rec, http.StatusNotFound)
}
