package middleware

import (
	"log/slog"
	"net/http"
	"strings"
)

// sensitiveHeaders is the set of header names (lowercase) that must be
// redacted before logging. These headers commonly carry credentials.
var sensitiveHeaders = map[string]bool{
	"authorization": true,
	"x-api-key":     true,
	"cookie":        true,
}

// RedactHeaders converts an http.Header map into a slice of slog.Attr values
// suitable for structured logging. Headers whose lowercase name appears in
// sensitiveHeaders are replaced with "[REDACTED]"; all others are included
// as-is. Multi-value headers are joined with a comma.
func RedactHeaders(headers http.Header) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(headers))
	for key, vals := range headers {
		if sensitiveHeaders[strings.ToLower(key)] {
			attrs = append(attrs, slog.String(key, "[REDACTED]"))
		} else {
			attrs = append(attrs, slog.String(key, strings.Join(vals, ",")))
		}
	}
	return attrs
}
