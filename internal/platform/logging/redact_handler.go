package logging

import (
	"log/slog"
	"regexp"

	"github.com/m-mizutani/masq"
)

// SensitiveHeaders is the canonical set of HTTP header names (lowercase) that
// carry credentials and must be redacted before logging. This set is shared
// between the masq defense-in-depth layer and the HTTP middleware's
// RedactHeaders utility so the two cannot silently drift apart.
var SensitiveHeaders = map[string]bool{
	"authorization": true,
	"x-api-key":     true,
	"cookie":        true,
}

// bearerPattern matches "Bearer <token>" strings that appear as raw values.
var bearerPattern = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-._~+/]+=*`)

// jwtPattern matches raw JWT strings (header.payload.signature). Requires at
// least 10 characters per segment to avoid false positives on short
// dot-separated strings like version numbers.
var jwtPattern = regexp.MustCompile(`[a-zA-Z0-9\-_]{10,}\.[a-zA-Z0-9\-_]{10,}\.[a-zA-Z0-9\-_]{10,}`)

// apiKeyInlinePattern matches inline "api_key=<value>" or "apikey:<value>"
// patterns that may appear in arbitrary string fields.
var apiKeyInlinePattern = regexp.MustCompile(`(?i)(api[_\-]?key|apikey)\s*[:=]\s*\S+`)

// fixedRedactOptions is the number of masq options beyond the dynamic
// SensitiveHeaders set (3 field names + 2 prefixes + 3 regexes).
const fixedRedactOptions = 8

// newRedactAttr returns a masq-powered ReplaceAttr function for use in
// slog.HandlerOptions. It redacts by field name for known sensitive fields
// and by regex for values that escape call-site redaction.
func newRedactAttr() func([]string, slog.Attr) slog.Attr {
	opts := make([]masq.Option, 0, fixedRedactOptions+len(SensitiveHeaders))

	// Sensitive header names shared with the HTTP middleware layer.
	for name := range SensitiveHeaders {
		opts = append(opts, masq.WithFieldName(name))
	}

	// Additional non-header fields for defense-in-depth.
	opts = append(opts,
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

	return masq.New(opts...)
}
