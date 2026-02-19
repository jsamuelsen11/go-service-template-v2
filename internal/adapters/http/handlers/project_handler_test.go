package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/handlers"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/mocks"
)

func newProjectHandler(t *testing.T) (*handlers.ProjectHandler, *mocks.MockProjectService) {
	t.Helper()
	svc := mocks.NewMockProjectService(t)
	return handlers.NewProjectHandler(svc), svc
}

// --- ListProjects ---

func TestListProjects_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	projects := []project.Project{validProject()}
	svc.EXPECT().ListProjects(mock.Anything).Return(projects, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	h.ListProjects(rec, req)

	requireStatus(t, rec, http.StatusOK)
	resp := decodeJSON[dto.ProjectListResponse](t, rec)
	if resp.Count != 1 {
		t.Errorf("Count = %d, want 1", resp.Count)
	}
}

func TestListProjects_ServiceError(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	svc.EXPECT().ListProjects(mock.Anything).Return(nil, domain.ErrUnavailable)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	h.ListProjects(rec, req)

	requireStatus(t, rec, http.StatusBadGateway)
}

// --- CreateProject ---

func TestCreateProject_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	created := validProject()
	svc.EXPECT().CreateProject(mock.Anything, mock.AnythingOfType("*project.Project")).
		Return(&created, nil)

	body := jsonBody(t, dto.CreateProjectRequest{Name: "Sprint 1", Description: "First sprint tasks"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body)
	req.Header.Set("Content-Type", "application/json")
	h.CreateProject(rec, req)

	requireStatus(t, rec, http.StatusCreated)
	resp := decodeJSON[dto.ProjectResponse](t, rec)
	if resp.Name != "Sprint 1" {
		t.Errorf("Name = %q, want %q", resp.Name, "Sprint 1")
	}
}

func TestCreateProject_InvalidJSON(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	h.CreateProject(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestCreateProject_ValidationError(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	body := jsonBody(t, dto.CreateProjectRequest{Name: "", Description: ""})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body)
	req.Header.Set("Content-Type", "application/json")
	h.CreateProject(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestCreateProject_ServiceError(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	svc.EXPECT().CreateProject(mock.Anything, mock.AnythingOfType("*project.Project")).
		Return(nil, domain.ErrUnavailable)

	body := jsonBody(t, dto.CreateProjectRequest{Name: "Sprint 1", Description: "Desc"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", body)
	req.Header.Set("Content-Type", "application/json")
	h.CreateProject(rec, req)

	requireStatus(t, rec, http.StatusBadGateway)
}

// --- GetProject ---

func TestGetProject_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	p := validProject()
	svc.EXPECT().GetProject(mock.Anything, int64(1)).Return(&p, nil)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodGet, "/api/v1/projects/1", nil), map[string]string{"id": "1"})
	h.GetProject(rec, req)

	requireStatus(t, rec, http.StatusOK)
	resp := decodeJSON[dto.ProjectResponse](t, rec)
	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
}

func TestGetProject_InvalidID(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodGet, "/api/v1/projects/abc", nil), map[string]string{"id": "abc"})
	h.GetProject(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestGetProject_NotFound(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	svc.EXPECT().GetProject(mock.Anything, int64(999)).Return(nil, domain.ErrNotFound)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodGet, "/api/v1/projects/999", nil), map[string]string{"id": "999"})
	h.GetProject(rec, req)

	requireStatus(t, rec, http.StatusNotFound)
}

// --- UpdateProject ---

func TestUpdateProject_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	updated := validProject()
	updated.Name = testUpdatedValue
	svc.EXPECT().UpdateProject(mock.Anything, int64(1), mock.AnythingOfType("*project.Project")).
		Return(&updated, nil)

	name := testUpdatedValue
	body := jsonBody(t, dto.UpdateProjectRequest{Name: &name})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/1", body)
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "1"})
	h.UpdateProject(rec, req)

	requireStatus(t, rec, http.StatusOK)
	resp := decodeJSON[dto.ProjectResponse](t, rec)
	if resp.Name != testUpdatedValue {
		t.Errorf("Name = %q, want %q", resp.Name, testUpdatedValue)
	}
}

func TestUpdateProject_InvalidID(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/abc", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "abc"})
	h.UpdateProject(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestUpdateProject_InvalidJSON(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/1", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "1"})
	h.UpdateProject(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestUpdateProject_ValidationError(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	empty := ""
	body := jsonBody(t, dto.UpdateProjectRequest{Name: &empty})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/1", body)
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"id": "1"})
	h.UpdateProject(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

// --- DeleteProject ---

func TestDeleteProject_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	svc.EXPECT().DeleteProject(mock.Anything, int64(1)).Return(nil)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/projects/1", nil), map[string]string{"id": "1"})
	h.DeleteProject(rec, req)

	requireStatus(t, rec, http.StatusNoContent)
}

func TestDeleteProject_InvalidID(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/projects/abc", nil), map[string]string{"id": "abc"})
	h.DeleteProject(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestDeleteProject_NotFound(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	svc.EXPECT().DeleteProject(mock.Anything, int64(999)).Return(domain.ErrNotFound)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/projects/999", nil), map[string]string{"id": "999"})
	h.DeleteProject(rec, req)

	requireStatus(t, rec, http.StatusNotFound)
}

// --- AddProjectTodo ---

func TestAddProjectTodo_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	created := validTodo()
	svc.EXPECT().AddTodo(mock.Anything, int64(1), mock.AnythingOfType("*todo.Todo")).
		Return(&created, nil)

	body := jsonBody(t, dto.CreateTodoRequest{Title: "Buy groceries", Description: "Milk, eggs, bread"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/1/todos", body)
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"projectId": "1"})
	h.AddProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusCreated)
	resp := decodeJSON[dto.TodoResponse](t, rec)
	if resp.Title != "Buy groceries" {
		t.Errorf("Title = %q, want %q", resp.Title, "Buy groceries")
	}
}

func TestAddProjectTodo_InvalidProjectID(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	body := jsonBody(t, dto.CreateTodoRequest{Title: "T", Description: "D"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/abc/todos", body)
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"projectId": "abc"})
	h.AddProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestAddProjectTodo_ValidationError(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	body := jsonBody(t, dto.CreateTodoRequest{Title: "", Description: ""})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/1/todos", body)
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"projectId": "1"})
	h.AddProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

// --- UpdateProjectTodo ---

func TestUpdateProjectTodo_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	updated := validTodo()
	updated.Title = testUpdatedValue
	svc.EXPECT().UpdateTodo(mock.Anything, int64(1), int64(2), mock.AnythingOfType("*todo.Todo")).
		Return(&updated, nil)

	title := testUpdatedValue
	body := jsonBody(t, dto.UpdateTodoRequest{Title: &title})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/1/todos/2", body)
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"projectId": "1", "todoId": "2"})
	h.UpdateProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusOK)
	resp := decodeJSON[dto.TodoResponse](t, rec)
	if resp.Title != testUpdatedValue {
		t.Errorf("Title = %q, want %q", resp.Title, testUpdatedValue)
	}
}

func TestUpdateProjectTodo_InvalidProjectID(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/abc/todos/1", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"projectId": "abc", "todoId": "1"})
	h.UpdateProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestUpdateProjectTodo_InvalidTodoID(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/1/todos/abc", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParams(req, map[string]string{"projectId": "1", "todoId": "abc"})
	h.UpdateProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

// --- RemoveProjectTodo ---

func TestRemoveProjectTodo_Success(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	svc.EXPECT().RemoveTodo(mock.Anything, int64(1), int64(2)).Return(nil)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/projects/1/todos/2", nil), map[string]string{"projectId": "1", "todoId": "2"})
	h.RemoveProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusNoContent)
}

func TestRemoveProjectTodo_InvalidProjectID(t *testing.T) {
	t.Parallel()
	h, _ := newProjectHandler(t)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/projects/abc/todos/1", nil), map[string]string{"projectId": "abc", "todoId": "1"})
	h.RemoveProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusBadRequest)
}

func TestRemoveProjectTodo_NotFound(t *testing.T) {
	t.Parallel()
	h, svc := newProjectHandler(t)

	svc.EXPECT().RemoveTodo(mock.Anything, int64(1), int64(999)).Return(domain.ErrNotFound)

	rec := httptest.NewRecorder()
	req := withChiParams(httptest.NewRequest(http.MethodDelete, "/api/v1/projects/1/todos/999", nil), map[string]string{"projectId": "1", "todoId": "999"})
	h.RemoveProjectTodo(rec, req)

	requireStatus(t, rec, http.StatusNotFound)
}

// --- Error propagation ---

func TestProjectHandler_ErrorPropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"validation", &domain.ValidationError{Fields: map[string]string{"x": "bad"}}, http.StatusBadRequest},
		{"conflict", domain.ErrConflict, http.StatusConflict},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden},
		{"unavailable", domain.ErrUnavailable, http.StatusBadGateway},
		{"unknown", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h, svc := newProjectHandler(t)

			svc.EXPECT().GetProject(mock.Anything, int64(1)).Return(nil, tt.err)

			rec := httptest.NewRecorder()
			req := withChiParams(httptest.NewRequest(http.MethodGet, "/api/v1/projects/1", nil), map[string]string{"id": "1"})
			h.GetProject(rec, req)

			requireStatus(t, rec, tt.wantStatus)
		})
	}
}
