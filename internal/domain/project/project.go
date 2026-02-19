package project

import (
	"strings"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain/todo"
)

// Project represents a collection of related todos.
// It maps to the downstream "Group" concept; the ACL translates between the two.
type Project struct {
	ID          int64
	Name        string
	Description string
	Todos       []todo.Todo
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate checks business rules for the Project entity.
// Returns a *domain.ValidationError (wrapping domain.ErrValidation) with per-field details,
// or nil if all rules pass.
func (p *Project) Validate() error {
	fields := make(map[string]string)

	if strings.TrimSpace(p.Name) == "" {
		fields["name"] = domain.MsgRequired
	}
	if strings.TrimSpace(p.Description) == "" {
		fields["description"] = domain.MsgRequired
	}

	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}
