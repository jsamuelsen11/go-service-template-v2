// Package acl implements the Anti-Corruption Layer that translates between
// downstream TODO API representations and domain types. Domain-specific
// translators live in subpackages (acl/todo, acl/project); shared error
// mapping lives here.
package acl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// maxErrorBodySize limits how much of an error response body we read.
const maxErrorBodySize = 1 << 20 // 1 MB

// problemDetail represents an RFC 7807 Problem Details response from the
// downstream API.
type problemDetail struct {
	Detail string        `json:"detail"`
	Errors []errorDetail `json:"errors"`
}

// errorDetail represents a single field-level error within an RFC 7807 response.
type errorDetail struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

// TranslateHTTPError maps an HTTP error response to a domain error.
// It parses the response body as RFC 7807 when the content type is
// application/problem+json, using the detail field for context.
// For 400/422 responses with field-level errors, it returns a
// *domain.ValidationError.
func TranslateHTTPError(resp *http.Response) error {
	pd := parseProblemDetail(resp)

	detail := pd.Detail
	if detail == "" {
		detail = http.StatusText(resp.StatusCode)
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%s: %w", detail, domain.ErrNotFound)

	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity:
		if len(pd.Errors) > 0 {
			return toValidationError(pd.Errors)
		}
		return fmt.Errorf("%s: %w", detail, domain.ErrValidation)

	case resp.StatusCode == http.StatusConflict:
		return fmt.Errorf("%s: %w", detail, domain.ErrConflict)

	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%s: %w", detail, domain.ErrForbidden)

	case resp.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("%s: %w", detail, domain.ErrUnavailable)

	default:
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, detail)
	}
}

// parseProblemDetail attempts to read and parse an RFC 7807 body from the
// response. Returns an empty problemDetail if parsing fails.
func parseProblemDetail(resp *http.Response) problemDetail {
	if resp.Body == nil {
		return problemDetail{}
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/problem+json") {
		return problemDetail{}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	if err != nil {
		return problemDetail{}
	}

	var pd problemDetail
	if err := json.Unmarshal(body, &pd); err != nil {
		return problemDetail{}
	}
	return pd
}

// toValidationError converts RFC 7807 error details to a domain ValidationError.
// It strips the "body." prefix from locations to produce clean field names.
func toValidationError(details []errorDetail) *domain.ValidationError {
	fields := make(map[string]string, len(details))
	for _, d := range details {
		field := strings.TrimPrefix(d.Location, "body.")
		fields[field] = d.Message
	}
	return &domain.ValidationError{Fields: fields}
}
