package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

// parseID extracts an int64 path parameter from the chi URL params.
func parseID(r *http.Request, param string) (int64, error) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, &domain.ValidationError{
			Fields: map[string]string{param: "must be a valid integer"},
		}
	}
	return id, nil
}

// mapCreateTodoRequest converts a CreateTodoRequest DTO to a domain Todo entity.
func mapCreateTodoRequest(req *dto.CreateTodoRequest) *todo.Todo {
	t := &todo.Todo{
		Title:           req.Title,
		Description:     req.Description,
		Status:          todo.StatusPending,
		Category:        todo.CategoryPersonal,
		ProgressPercent: req.ProgressPercent,
	}
	if req.Status != "" {
		t.Status = todo.Status(req.Status)
	}
	if req.Category != "" {
		t.Category = todo.Category(req.Category)
	}
	return t
}

// mapUpdateTodoRequest converts an UpdateTodoRequest DTO to a domain Todo entity.
func mapUpdateTodoRequest(req *dto.UpdateTodoRequest) *todo.Todo {
	t := &todo.Todo{}
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Status != nil {
		t.Status = todo.Status(*req.Status)
	}
	if req.Category != nil {
		t.Category = todo.Category(*req.Category)
	}
	if req.ProgressPercent != nil {
		t.ProgressPercent = *req.ProgressPercent
	}
	return t
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", slog.Any("error", err))
	}
}

// maxJSONBodyBytes is the maximum allowed size for a JSON request body (1 MB).
const maxJSONBodyBytes = 1 << 20

// decodeJSONBody decodes the request body as JSON into dst. The body is
// limited to maxJSONBodyBytes to prevent resource exhaustion. On failure,
// it writes a 400 error response and returns false.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		dto.WriteErrorResponse(w, r, &domain.ValidationError{
			Fields: map[string]string{"body": "invalid JSON"},
		})
		return false
	}
	return true
}

// validatable is implemented by request DTOs that support validation.
type validatable interface {
	Validate() error
}

// decodeAndValidate decodes the JSON request body into dst and validates it.
// On decode or validation failure it writes an error response and returns false.
func decodeAndValidate[T validatable](w http.ResponseWriter, r *http.Request, dst T) bool {
	if !decodeJSONBody(w, r, dst) {
		return false
	}
	if err := dst.Validate(); err != nil {
		dto.WriteErrorResponse(w, r, err)
		return false
	}
	return true
}

// decodeTodoCreate decodes and validates a CreateTodoRequest, returning the
// mapped domain Todo. Returns nil and writes an error response on failure.
func decodeTodoCreate(w http.ResponseWriter, r *http.Request) *todo.Todo {
	var req dto.CreateTodoRequest
	if !decodeAndValidate(w, r, &req) {
		return nil
	}
	return mapCreateTodoRequest(&req)
}

// decodeTodoUpdate decodes and validates an UpdateTodoRequest, returning the
// mapped domain Todo. Returns nil and writes an error response on failure.
func decodeTodoUpdate(w http.ResponseWriter, r *http.Request) *todo.Todo {
	var req dto.UpdateTodoRequest
	if !decodeAndValidate(w, r, &req) {
		return nil
	}
	return mapUpdateTodoRequest(&req)
}
