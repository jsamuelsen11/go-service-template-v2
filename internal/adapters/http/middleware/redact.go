package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
)

// RedactHeaders converts an http.Header map into a slice of slog.Attr values
// suitable for structured logging. Headers whose lowercase name appears in
// [logging.SensitiveHeaders] are replaced with "[REDACTED]"; all others are
// included as-is. Multi-value headers are joined with a comma.
func RedactHeaders(headers http.Header) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(headers))
	for key, vals := range headers {
		if logging.SensitiveHeaders[strings.ToLower(key)] {
			attrs = append(attrs, slog.String(key, "[REDACTED]"))
		} else {
			attrs = append(attrs, slog.String(key, strings.Join(vals, ",")))
		}
	}
	return attrs
}

// attrsToArgs converts a slice of slog.Attr into []any for use with
// slog.Logger methods that accept variadic any arguments.
func attrsToArgs(attrs []slog.Attr) []any {
	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	return args
}
