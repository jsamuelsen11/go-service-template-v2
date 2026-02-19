package logging

import (
	"log/slog"
	"regexp"

	"github.com/m-mizutani/masq"
)

// bearerPattern matches "Bearer <token>" strings that appear as raw values.
var bearerPattern = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-._~+/]+=*`)

// jwtPattern matches raw JWT strings (header.payload.signature). Requires at
// least 10 characters per segment to avoid false positives on short
// dot-separated strings like version numbers.
var jwtPattern = regexp.MustCompile(`[a-zA-Z0-9\-_]{10,}\.[a-zA-Z0-9\-_]{10,}\.[a-zA-Z0-9\-_]{10,}`)

// apiKeyInlinePattern matches inline "api_key=<value>" or "apikey:<value>"
// patterns that may appear in arbitrary string fields.
var apiKeyInlinePattern = regexp.MustCompile(`(?i)(api[_\-]?key|apikey)\s*[:=]\s*\S+`)

// newRedactAttr returns a masq-powered ReplaceAttr function for use in
// slog.HandlerOptions. It redacts by field name for known sensitive fields
// and by regex for values that escape call-site redaction.
func newRedactAttr() func([]string, slog.Attr) slog.Attr {
	return masq.New(
		// Field-name redaction: catches fields logged with these names.
		masq.WithFieldName("authorization"),
		masq.WithFieldName("x-api-key"),
		masq.WithFieldName("cookie"),
		masq.WithFieldName("password"),
		masq.WithFieldName("secret"),
		masq.WithFieldName("token"),

		// Prefix-based redaction for variations like "secret_key", "api_key_v2".
		masq.WithFieldPrefix("secret_"),
		masq.WithFieldPrefix("api_key"),

		// Regex-based defense-in-depth for raw sensitive values.
		masq.WithRegex(bearerPattern),
		masq.WithRegex(jwtPattern),
		masq.WithRegex(apiKeyInlinePattern),
	)
}
