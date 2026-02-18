// Package http provides the inbound HTTP adapter including routing and server lifecycle.
package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/handlers"
)

// NewRouter creates an HTTP handler with all application routes registered.
// Middleware is applied globally in the order given.
func NewRouter(
	projectHandler *handlers.ProjectHandler,
	todoHandler *handlers.TodoHandler,
	healthHandler *handlers.HealthHandler,
	middlewares ...func(http.Handler) http.Handler,
) http.Handler {
	r := chi.NewRouter()

	for _, mw := range middlewares {
		r.Use(mw)
	}

	// Health endpoints (outside /api/v1 prefix).
	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	// API v1 routes.
	r.Route("/api/v1", func(r chi.Router) {
		// Project CRUD.
		r.Get("/projects", projectHandler.ListProjects)
		r.Post("/projects", projectHandler.CreateProject)
		r.Get("/projects/{id}", projectHandler.GetProject)
		r.Patch("/projects/{id}", projectHandler.UpdateProject)
		r.Delete("/projects/{id}", projectHandler.DeleteProject)

		// Nested project-todo operations.
		r.Post("/projects/{projectId}/todos", projectHandler.AddProjectTodo)
		r.Patch("/projects/{projectId}/todos/{todoId}", projectHandler.UpdateProjectTodo)
		r.Delete("/projects/{projectId}/todos/{todoId}", projectHandler.RemoveProjectTodo)

		// Flat todo CRUD.
		r.Get("/todos", todoHandler.ListTodos)
		r.Post("/todos", todoHandler.CreateTodo)
		r.Get("/todos/{id}", todoHandler.GetTodo)
		r.Patch("/todos/{id}", todoHandler.UpdateTodo)
		r.Delete("/todos/{id}", todoHandler.DeleteTodo)
	})

	return r
}
