package domain

import (
	"fmt"
	"strings"
	"time"
)

// Todo represents a task item with progress tracking.
type Todo struct {
	ID              int64
	Title           string
	Description     string
	Status          TodoStatus
	Category        TodoCategory
	ProgressPercent int
	ProjectID       *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Validate checks business rules for the Todo entity.
// Returns a *ValidationError (wrapping ErrValidation) with per-field details,
// or nil if all rules pass.
func (t *Todo) Validate() error {
	fields := make(map[string]string)

	if strings.TrimSpace(t.Title) == "" {
		fields["title"] = msgRequired
	}
	if strings.TrimSpace(t.Description) == "" {
		fields["description"] = msgRequired
	}
	if !t.Status.IsValid() {
		fields["status"] = fmt.Sprintf("invalid: %q", t.Status)
	}
	if !t.Category.IsValid() {
		fields["category"] = fmt.Sprintf("invalid: %q", t.Category)
	}
	if t.ProgressPercent < 0 || t.ProgressPercent > 100 {
		fields["progress_percent"] = fmt.Sprintf("must be 0-100, got %d", t.ProgressPercent)
	}
	if t.ProjectID != nil && *t.ProjectID <= 0 {
		fields["project_id"] = fmt.Sprintf("must be positive, got %d", *t.ProjectID)
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}
