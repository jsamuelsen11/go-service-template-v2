package domain

import (
	"strings"
	"time"
)

// Project represents a collection of related todos.
// It maps to the downstream "Group" concept; the ACL translates between the two.
type Project struct {
	ID          int64
	Name        string
	Description string
	Todos       []Todo
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate checks business rules for the Project entity.
// Returns a *ValidationError (wrapping ErrValidation) with per-field details,
// or nil if all rules pass.
func (p *Project) Validate() error {
	fields := make(map[string]string)

	if strings.TrimSpace(p.Name) == "" {
		fields["name"] = msgRequired
	}
	if strings.TrimSpace(p.Description) == "" {
		fields["description"] = msgRequired
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}
