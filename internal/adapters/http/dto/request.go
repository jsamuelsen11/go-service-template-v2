package dto

import (
	"fmt"
	"strings"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

const (
	msgRequired     = "is required"
	msgMustNotEmpty = "must not be empty"

	// maxBulkUpdateItems is the maximum number of items in a bulk update request.
	maxBulkUpdateItems = 20
)

// CreateProjectRequest represents the JSON body for creating a new project.
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate checks that required fields are present.
// Returns a *domain.ValidationError if any checks fail.
func (r *CreateProjectRequest) Validate() error {
	fields := make(map[string]string)

	if strings.TrimSpace(r.Name) == "" {
		fields["name"] = msgRequired
	}
	if strings.TrimSpace(r.Description) == "" {
		fields["description"] = msgRequired
	}

	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}

// UpdateProjectRequest represents the JSON body for updating an existing project.
// All fields are optional; nil means "do not change this field.".
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Validate checks that any provided fields have valid values.
// Returns a *domain.ValidationError if any checks fail.
func (r *UpdateProjectRequest) Validate() error {
	fields := make(map[string]string)

	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		fields["name"] = msgMustNotEmpty
	}
	if r.Description != nil && strings.TrimSpace(*r.Description) == "" {
		fields["description"] = msgMustNotEmpty
	}

	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}

// CreateTodoRequest represents the JSON body for creating a new TODO item.
type CreateTodoRequest struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Status          string `json:"status,omitempty"`
	Category        string `json:"category,omitempty"`
	ProgressPercent int    `json:"progress_percent,omitempty"`
}

// Validate checks that required fields are present and optional fields have
// valid values. Returns a *domain.ValidationError if any checks fail.
func (r *CreateTodoRequest) Validate() error {
	fields := make(map[string]string)

	if strings.TrimSpace(r.Title) == "" {
		fields["title"] = msgRequired
	}
	if strings.TrimSpace(r.Description) == "" {
		fields["description"] = msgRequired
	}
	if r.Status != "" && !todo.Status(r.Status).IsValid() {
		fields["status"] = fmt.Sprintf("invalid: %q", r.Status)
	}
	if r.Category != "" && !todo.Category(r.Category).IsValid() {
		fields["category"] = fmt.Sprintf("invalid: %q", r.Category)
	}
	if r.ProgressPercent < 0 || r.ProgressPercent > 100 {
		fields["progress_percent"] = fmt.Sprintf("must be 0-100, got %d", r.ProgressPercent)
	}

	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}

// UpdateTodoRequest represents the JSON body for updating an existing TODO item.
// All fields are optional; nil means "do not change this field.".
type UpdateTodoRequest struct {
	Title           *string `json:"title,omitempty"`
	Description     *string `json:"description,omitempty"`
	Status          *string `json:"status,omitempty"`
	Category        *string `json:"category,omitempty"`
	ProgressPercent *int    `json:"progress_percent,omitempty"`
}

// Validate checks that any provided fields have valid values.
// Returns a *domain.ValidationError if any checks fail.
func (r *UpdateTodoRequest) Validate() error {
	fields := make(map[string]string)

	if r.Title != nil && strings.TrimSpace(*r.Title) == "" {
		fields["title"] = msgMustNotEmpty
	}
	if r.Description != nil && strings.TrimSpace(*r.Description) == "" {
		fields["description"] = msgMustNotEmpty
	}
	if r.Status != nil && !todo.Status(*r.Status).IsValid() {
		fields["status"] = fmt.Sprintf("invalid: %q", *r.Status)
	}
	if r.Category != nil && !todo.Category(*r.Category).IsValid() {
		fields["category"] = fmt.Sprintf("invalid: %q", *r.Category)
	}
	if r.ProgressPercent != nil && (*r.ProgressPercent < 0 || *r.ProgressPercent > 100) {
		fields["progress_percent"] = fmt.Sprintf("must be 0-100, got %d", *r.ProgressPercent)
	}

	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}

// BulkUpdateTodoItem represents a single item within a bulk update request.
// It pairs a todo ID with optional fields to update (same fields as UpdateTodoRequest).
type BulkUpdateTodoItem struct {
	TodoID          int64   `json:"todo_id"`
	Title           *string `json:"title,omitempty"`
	Description     *string `json:"description,omitempty"`
	Status          *string `json:"status,omitempty"`
	Category        *string `json:"category,omitempty"`
	ProgressPercent *int    `json:"progress_percent,omitempty"`
}

// BulkUpdateTodosRequest represents the JSON body for bulk updating todos
// within a project.
type BulkUpdateTodosRequest struct {
	Updates []BulkUpdateTodoItem `json:"updates"`
}

// validateBulkUpdateItem validates a single item within a bulk update request,
// adding any errors to fields with the given prefix.
func validateBulkUpdateItem(item BulkUpdateTodoItem, prefix string, fields map[string]string) {
	if item.TodoID <= 0 {
		fields[prefix+".todo_id"] = "must be a positive integer"
	}
	if item.Title != nil && strings.TrimSpace(*item.Title) == "" {
		fields[prefix+".title"] = msgMustNotEmpty
	}
	if item.Description != nil && strings.TrimSpace(*item.Description) == "" {
		fields[prefix+".description"] = msgMustNotEmpty
	}
	if item.Status != nil && !todo.Status(*item.Status).IsValid() {
		fields[prefix+".status"] = fmt.Sprintf("invalid: %q", *item.Status)
	}
	if item.Category != nil && !todo.Category(*item.Category).IsValid() {
		fields[prefix+".category"] = fmt.Sprintf("invalid: %q", *item.Category)
	}
	if item.ProgressPercent != nil && (*item.ProgressPercent < 0 || *item.ProgressPercent > 100) {
		fields[prefix+".progress_percent"] = fmt.Sprintf("must be 0-100, got %d", *item.ProgressPercent)
	}
}

// Validate checks that the request has at least one update, does not exceed
// the maximum batch size, contains no duplicate todo IDs, and each item
// has valid field values.
func (r *BulkUpdateTodosRequest) Validate() error {
	fields := make(map[string]string)

	if len(r.Updates) == 0 {
		fields["updates"] = "must not be empty"
	}
	if len(r.Updates) > maxBulkUpdateItems {
		fields["updates"] = fmt.Sprintf("exceeds maximum of %d items", maxBulkUpdateItems)
	}

	seen := make(map[int64]bool, len(r.Updates))
	for i, item := range r.Updates {
		prefix := fmt.Sprintf("updates[%d]", i)

		if seen[item.TodoID] {
			fields[prefix+".todo_id"] = fmt.Sprintf("duplicate todo ID %d", item.TodoID)
		}
		seen[item.TodoID] = true

		validateBulkUpdateItem(item, prefix, fields)
	}

	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}
