package todo

import (
	"fmt"
	"strings"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// Todo represents a task item with progress tracking.
type Todo struct {
	ID              int64
	Title           string
	Description     string
	Status          Status
	Category        Category
	ProgressPercent int
	ProjectID       *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Validate checks business rules for the Todo entity.
// Returns a *domain.ValidationError (wrapping domain.ErrValidation) with per-field details,
// or nil if all rules pass.
func (t *Todo) Validate() error {
	fields := make(map[string]string)

	if strings.TrimSpace(t.Title) == "" {
		fields["title"] = domain.MsgRequired
	}
	if strings.TrimSpace(t.Description) == "" {
		fields["description"] = domain.MsgRequired
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
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}
