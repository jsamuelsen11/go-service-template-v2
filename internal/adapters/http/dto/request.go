package dto

import (
	"fmt"
	"strings"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

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
		fields["title"] = "is required"
	}
	if strings.TrimSpace(r.Description) == "" {
		fields["description"] = "is required"
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
		fields["title"] = "must not be empty"
	}
	if r.Description != nil && strings.TrimSpace(*r.Description) == "" {
		fields["description"] = "must not be empty"
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
