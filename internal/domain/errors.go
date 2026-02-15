package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for errors.Is() checking.
var (
	ErrNotFound    = errors.New("not found")
	ErrValidation  = errors.New("validation error")
	ErrConflict    = errors.New("conflict")
	ErrForbidden   = errors.New("forbidden")
	ErrUnavailable = errors.New("unavailable")
)

// ValidationError provides programmatic access to field-level validation failures.
// Use errors.Is(err, ErrValidation) for simple checks, or errors.As(err, &verr) to
// access verr.Fields for per-field error details.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for field, msg := range e.Fields {
		parts = append(parts, field+": "+msg)
	}
	return fmt.Sprintf("%s: %s", ErrValidation.Error(), strings.Join(parts, "; "))
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}
