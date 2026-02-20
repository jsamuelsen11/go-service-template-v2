// Package handlers provides HTTP request handlers for the service's API endpoints.
package handlers

import (
	"net/http"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"
)

// ProjectHandler handles HTTP requests for project CRUD and nested
// project-todo operations.
type ProjectHandler struct {
	svc ports.ProjectService
}

// NewProjectHandler creates a new ProjectHandler with the given service port.
func NewProjectHandler(svc ports.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// ListProjects handles GET /api/v1/projects.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.ListProjects(r.Context())
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToProjectListResponse(projects))
}

// CreateProject handles POST /api/v1/projects.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProjectRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	p := &project.Project{
		Name:        req.Name,
		Description: req.Description,
	}

	created, err := h.svc.CreateProject(r.Context(), p)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToProjectResponse(created))
}

// GetProject handles GET /api/v1/projects/{id}.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	p, err := h.svc.GetProject(r.Context(), id)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToProjectResponse(p))
}

// UpdateProject handles PATCH /api/v1/projects/{id}.
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	var req dto.UpdateProjectRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	p := &project.Project{}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}

	updated, err := h.svc.UpdateProject(r.Context(), id, p)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToProjectResponse(updated))
}

// DeleteProject handles DELETE /api/v1/projects/{id}.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	if err := h.svc.DeleteProject(r.Context(), id); err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddProjectTodo handles POST /api/v1/projects/{projectId}/todos.
func (h *ProjectHandler) AddProjectTodo(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r, "projectId")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	t := decodeTodoCreate(w, r)
	if t == nil {
		return
	}

	created, err := h.svc.AddTodo(r.Context(), projectID, t)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToTodoResponse(created))
}

// UpdateProjectTodo handles PATCH /api/v1/projects/{projectId}/todos/{todoId}.
func (h *ProjectHandler) UpdateProjectTodo(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r, "projectId")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	todoID, err := parseID(r, "todoId")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	t := decodeTodoUpdate(w, r)
	if t == nil {
		return
	}

	updated, err := h.svc.UpdateTodo(r.Context(), projectID, todoID, t)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToTodoResponse(updated))
}

// RemoveProjectTodo handles DELETE /api/v1/projects/{projectId}/todos/{todoId}.
func (h *ProjectHandler) RemoveProjectTodo(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r, "projectId")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	todoID, err := parseID(r, "todoId")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	if err := h.svc.RemoveTodo(r.Context(), projectID, todoID); err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkUpdateProjectTodos handles PATCH /api/v1/projects/{projectId}/todos/bulk.
func (h *ProjectHandler) BulkUpdateProjectTodos(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseID(r, "projectId")
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	var req dto.BulkUpdateTodosRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	updates := mapBulkUpdateRequest(req.Updates)

	result, err := h.svc.BulkUpdateTodos(r.Context(), projectID, updates)
	if err != nil {
		dto.WriteErrorResponse(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.ToBulkUpdateResponse(result))
}
