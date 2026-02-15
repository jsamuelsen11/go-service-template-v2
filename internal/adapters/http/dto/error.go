package dto

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// ErrorResponse represents an RFC 9457 Problem Details response.
type ErrorResponse struct {
	Type     string        `json:"type"`
	Title    string        `json:"title"`
	Status   int           `json:"status"`
	Detail   string        `json:"detail,omitempty"`
	Instance string        `json:"instance,omitempty"`
	Errors   []ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail represents a single field-level validation error within
// an ErrorResponse.
type ErrorDetail struct {
	Location string `json:"location"`
	Message  string `json:"message"`
	Value    any    `json:"value,omitempty"`
}

// NewErrorResponse creates an RFC 9457 ErrorResponse from a domain error.
// The request is used to populate the instance field with the request URI.
func NewErrorResponse(r *http.Request, err error) ErrorResponse {
	status := domainErrorToStatus(err)

	resp := ErrorResponse{
		Type:     "about:blank",
		Title:    http.StatusText(status),
		Status:   status,
		Detail:   err.Error(),
		Instance: r.RequestURI,
	}

	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		resp.Errors = validationFieldsToDetails(verr.Fields)
	}

	return resp
}

// WriteErrorResponse writes an RFC 9457 error response for the given domain
// error. It sets the Content-Type to application/problem+json, writes the
// appropriate HTTP status code, and marshals the error body as JSON.
func WriteErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	resp := NewErrorResponse(r, err)

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(resp.Status)

	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		slog.ErrorContext(r.Context(), "failed to encode error response",
			slog.Any("error", encErr),
		)
	}
}

// domainErrorToStatus maps domain sentinel errors to HTTP status codes.
func domainErrorToStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrUnavailable):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// validationFieldsToDetails converts domain validation fields to sorted
// ErrorDetail entries.
func validationFieldsToDetails(fields map[string]string) []ErrorDetail {
	details := make([]ErrorDetail, 0, len(fields))
	for field, msg := range fields {
		details = append(details, ErrorDetail{
			Location: "body." + field,
			Message:  msg,
		})
	}
	sort.Slice(details, func(i, j int) bool {
		return details[i].Location < details[j].Location
	})
	return details
}
