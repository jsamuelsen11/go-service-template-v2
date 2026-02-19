package middleware_test

import (
	"net/http"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
)

const redactedValue = "[REDACTED]"

func TestRedactHeaders_RedactsAuthorization(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": {"Bearer secret-token"},
	}
	attrs := middleware.RedactHeaders(headers)

	if len(attrs) != 1 {
		t.Fatalf("len(attrs) = %d, want 1", len(attrs))
	}
	if attrs[0].Value.String() != redactedValue {
		t.Errorf("Authorization value = %q, want %q", attrs[0].Value.String(), redactedValue)
	}
}

func TestRedactHeaders_RedactsXAPIKey(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"X-Api-Key": {"my-api-key-value"},
	}
	attrs := middleware.RedactHeaders(headers)

	if len(attrs) != 1 {
		t.Fatalf("len(attrs) = %d, want 1", len(attrs))
	}
	if attrs[0].Value.String() != redactedValue {
		t.Errorf("X-Api-Key value = %q, want %q", attrs[0].Value.String(), redactedValue)
	}
}

func TestRedactHeaders_RedactsCookie(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Cookie": {"session=abc123"},
	}
	attrs := middleware.RedactHeaders(headers)

	if len(attrs) != 1 {
		t.Fatalf("len(attrs) = %d, want 1", len(attrs))
	}
	if attrs[0].Value.String() != redactedValue {
		t.Errorf("Cookie value = %q, want %q", attrs[0].Value.String(), redactedValue)
	}
}

func TestRedactHeaders_PassesThroughNonSensitive(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}
	attrs := middleware.RedactHeaders(headers)

	if len(attrs) != 2 {
		t.Fatalf("len(attrs) = %d, want 2", len(attrs))
	}

	found := false
	for _, a := range attrs {
		if a.Key == "Content-Type" && a.Value.String() == "application/json" {
			found = true
		}
	}
	if !found {
		t.Error("Content-Type not found or value incorrect in redacted attrs")
	}
}

func TestRedactHeaders_JoinsMultiValueHeaders(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Accept": {"text/html", "application/json"},
	}
	attrs := middleware.RedactHeaders(headers)

	if len(attrs) != 1 {
		t.Fatalf("len(attrs) = %d, want 1", len(attrs))
	}
	if attrs[0].Value.String() != "text/html,application/json" {
		t.Errorf("Accept value = %q, want %q", attrs[0].Value.String(), "text/html,application/json")
	}
}

func TestRedactHeaders_EmptyHeaders(t *testing.T) {
	t.Parallel()

	attrs := middleware.RedactHeaders(http.Header{})

	if len(attrs) != 0 {
		t.Errorf("len(attrs) = %d, want 0 for empty headers", len(attrs))
	}
}

func TestRedactHeaders_MixedSensitiveAndNonSensitive(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": {"Bearer secret"},
		"Content-Type":  {"application/json"},
	}
	attrs := middleware.RedactHeaders(headers)

	if len(attrs) != 2 {
		t.Fatalf("len(attrs) = %d, want 2", len(attrs))
	}

	values := map[string]string{}
	for _, a := range attrs {
		values[a.Key] = a.Value.String()
	}

	if values["Authorization"] != redactedValue {
		t.Errorf("Authorization = %q, want %q", values["Authorization"], redactedValue)
	}
	if values["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want %q", values["Content-Type"], "application/json")
	}
}
