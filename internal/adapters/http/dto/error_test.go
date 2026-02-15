package dto_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

func TestNewErrorResponse_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantTitle  string
	}{
		{
			name:       "ErrNotFound maps to 404",
			err:        domain.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
		},
		{
			name:       "ErrValidation maps to 400",
			err:        &domain.ValidationError{Fields: map[string]string{"title": "is required"}},
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Bad Request",
		},
		{
			name:       "ErrConflict maps to 409",
			err:        domain.ErrConflict,
			wantStatus: http.StatusConflict,
			wantTitle:  "Conflict",
		},
		{
			name:       "ErrForbidden maps to 403",
			err:        domain.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantTitle:  "Forbidden",
		},
		{
			name:       "ErrUnavailable maps to 502",
			err:        domain.ErrUnavailable,
			wantStatus: http.StatusBadGateway,
			wantTitle:  "Bad Gateway",
		},
		{
			name:       "unknown error maps to 500",
			err:        errors.New("oops"),
			wantStatus: http.StatusInternalServerError,
			wantTitle:  "Internal Server Error",
		},
		{
			name:       "wrapped ErrNotFound preserves mapping",
			err:        fmt.Errorf("fetching todo: %w", domain.ErrNotFound),
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/todos/42", nil)
			got := dto.NewErrorResponse(r, tt.err)

			if got.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", got.Status, tt.wantStatus)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
		})
	}
}

func TestNewErrorResponse_Fields(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodPost, "/api/v1/todos", nil)
	err := domain.ErrNotFound

	got := dto.NewErrorResponse(r, err)

	if got.Type != "about:blank" {
		t.Errorf("Type = %q, want %q", got.Type, "about:blank")
	}
	if got.Instance != "/api/v1/todos" {
		t.Errorf("Instance = %q, want %q", got.Instance, "/api/v1/todos")
	}
	if got.Detail != err.Error() {
		t.Errorf("Detail = %q, want %q", got.Detail, err.Error())
	}
}

func TestNewErrorResponse_ValidationErrors(t *testing.T) {
	t.Parallel()

	verr := &domain.ValidationError{Fields: map[string]string{
		"title":       "is required",
		"description": "is required",
		"status":      "invalid: \"bad\"",
	}}

	r := httptest.NewRequest(http.MethodPost, "/api/v1/todos", nil)
	got := dto.NewErrorResponse(r, verr)

	if len(got.Errors) != 3 {
		t.Fatalf("len(Errors) = %d, want 3", len(got.Errors))
	}

	// Verify sorted by location.
	for i := 1; i < len(got.Errors); i++ {
		if got.Errors[i-1].Location >= got.Errors[i].Location {
			t.Errorf("Errors not sorted: %q >= %q", got.Errors[i-1].Location, got.Errors[i].Location)
		}
	}

	// Verify location format.
	for _, detail := range got.Errors {
		if len(detail.Location) < 6 || detail.Location[:5] != "body." {
			t.Errorf("Location %q does not start with %q", detail.Location, "body.")
		}
	}
}

func TestNewErrorResponse_NoValidationErrorsForNonValidation(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/api/v1/todos/1", nil)
	got := dto.NewErrorResponse(r, domain.ErrNotFound)

	if got.Errors != nil {
		t.Errorf("Errors = %v, want nil for non-validation error", got.Errors)
	}
}

func TestWriteErrorResponse_ContentType(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/todos/42", nil)

	dto.WriteErrorResponse(w, r, domain.ErrNotFound)

	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/problem+json")
	}
}

func TestWriteErrorResponse_StatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"validation", &domain.ValidationError{Fields: map[string]string{"x": "y"}}, http.StatusBadRequest},
		{"conflict", domain.ErrConflict, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			dto.WriteErrorResponse(w, r, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestWriteErrorResponse_ValidJSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/todos", nil)

	verr := &domain.ValidationError{Fields: map[string]string{
		"title": "is required",
	}}
	dto.WriteErrorResponse(w, r, verr)

	var resp dto.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if resp.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.Status, http.StatusBadRequest)
	}
	if resp.Type != "about:blank" {
		t.Errorf("Type = %q, want %q", resp.Type, "about:blank")
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(resp.Errors))
	}
	if resp.Errors[0].Location != "body.title" {
		t.Errorf("Errors[0].Location = %q, want %q", resp.Errors[0].Location, "body.title")
	}
	if resp.Errors[0].Message != "is required" {
		t.Errorf("Errors[0].Message = %q, want %q", resp.Errors[0].Message, "is required")
	}
}
