package acl

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

func TestTranslateHTTPError_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{
			name:       "404 maps to ErrNotFound",
			statusCode: http.StatusNotFound,
			wantErr:    domain.ErrNotFound,
		},
		{
			name:       "400 maps to ErrValidation",
			statusCode: http.StatusBadRequest,
			wantErr:    domain.ErrValidation,
		},
		{
			name:       "422 maps to ErrValidation",
			statusCode: http.StatusUnprocessableEntity,
			wantErr:    domain.ErrValidation,
		},
		{
			name:       "409 maps to ErrConflict",
			statusCode: http.StatusConflict,
			wantErr:    domain.ErrConflict,
		},
		{
			name:       "401 maps to ErrForbidden",
			statusCode: http.StatusUnauthorized,
			wantErr:    domain.ErrForbidden,
		},
		{
			name:       "403 maps to ErrForbidden",
			statusCode: http.StatusForbidden,
			wantErr:    domain.ErrForbidden,
		},
		{
			name:       "500 maps to ErrUnavailable",
			statusCode: http.StatusInternalServerError,
			wantErr:    domain.ErrUnavailable,
		},
		{
			name:       "502 maps to ErrUnavailable",
			statusCode: http.StatusBadGateway,
			wantErr:    domain.ErrUnavailable,
		},
		{
			name:       "503 maps to ErrUnavailable",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    domain.ErrUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     http.Header{},
				Body:       http.NoBody,
			}

			got := TranslateHTTPError(resp)

			if !errors.Is(got, tt.wantErr) {
				t.Errorf("TranslateHTTPError() = %v, want errors.Is %v", got, tt.wantErr)
			}
		})
	}
}

func TestTranslateHTTPError_RFC7807Parsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantSubstr string
	}{
		{
			name:       "extracts detail from RFC 7807 body",
			statusCode: http.StatusNotFound,
			body:       `{"type":"about:blank","title":"Not Found","status":404,"detail":"todo 42 not found"}`,
			wantSubstr: "todo 42 not found",
		},
		{
			name:       "falls back to status text for non-JSON body",
			statusCode: http.StatusNotFound,
			body:       "Not Found",
			wantSubstr: "Not Found",
		},
		{
			name:       "falls back to status text for empty body",
			statusCode: http.StatusConflict,
			body:       "",
			wantSubstr: "Conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			header := http.Header{}
			if strings.HasPrefix(tt.body, "{") {
				header.Set("Content-Type", "application/problem+json")
			}

			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     header,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			got := TranslateHTTPError(resp)

			if !strings.Contains(got.Error(), tt.wantSubstr) {
				t.Errorf("error = %q, want substring %q", got.Error(), tt.wantSubstr)
			}
		})
	}
}

func TestTranslateHTTPError_ValidationErrorWithDetails(t *testing.T) {
	t.Parallel()

	body := `{
		"type": "about:blank",
		"title": "Bad Request",
		"status": 400,
		"detail": "validation failed",
		"errors": [
			{"location": "body.title", "message": "is required"},
			{"location": "body.description", "message": "is required"}
		]
	}`

	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/problem+json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	got := TranslateHTTPError(resp)

	if !errors.Is(got, domain.ErrValidation) {
		t.Fatalf("error is not ErrValidation: %v", got)
	}

	var verr *domain.ValidationError
	if !errors.As(got, &verr) {
		t.Fatalf("error is not *ValidationError: %v", got)
	}

	if len(verr.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(verr.Fields))
	}
	if verr.Fields["title"] != "is required" {
		t.Errorf("Fields[title] = %q, want %q", verr.Fields["title"], "is required")
	}
	if verr.Fields["description"] != "is required" {
		t.Errorf("Fields[description] = %q, want %q", verr.Fields["description"], "is required")
	}
}

func TestTranslateHTTPError_ValidationErrorStripsBodyPrefix(t *testing.T) {
	t.Parallel()

	body := `{
		"detail": "validation failed",
		"errors": [{"location": "body.status", "message": "invalid"}]
	}`

	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/problem+json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	got := TranslateHTTPError(resp)

	var verr *domain.ValidationError
	if !errors.As(got, &verr) {
		t.Fatalf("error is not *ValidationError: %v", got)
	}

	if _, ok := verr.Fields["status"]; !ok {
		t.Errorf("Fields = %v, want key %q (body. prefix stripped)", verr.Fields, "status")
	}
}

func TestTranslateHTTPError_UnexpectedStatus(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusTeapot,
		Header:     http.Header{},
		Body:       http.NoBody,
	}

	got := TranslateHTTPError(resp)

	if errors.Is(got, domain.ErrNotFound) ||
		errors.Is(got, domain.ErrValidation) ||
		errors.Is(got, domain.ErrConflict) ||
		errors.Is(got, domain.ErrForbidden) ||
		errors.Is(got, domain.ErrUnavailable) {
		t.Errorf("unexpected status should not match any domain error, got: %v", got)
	}

	if !strings.Contains(got.Error(), "418") {
		t.Errorf("error = %q, want status code 418 in message", got.Error())
	}
}

func TestTranslateHTTPError_NilBody(t *testing.T) {
	t.Parallel()

	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/problem+json"}},
		Body:       nil,
	}

	got := TranslateHTTPError(resp)

	if !errors.Is(got, domain.ErrNotFound) {
		t.Errorf("error is not ErrNotFound: %v", got)
	}
}
