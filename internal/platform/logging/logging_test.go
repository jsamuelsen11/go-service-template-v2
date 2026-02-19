package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
)

// --- New tests ---

func TestNew_JSONFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	logger.Info("hello")

	out := buf.String()
	if !strings.Contains(out, `"level":"INFO"`) {
		t.Errorf("output = %q, want it to contain '\"level\":\"INFO\"'", out)
	}
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("output = %q, want it to contain '\"msg\":\"hello\"'", out)
	}
}

func TestNew_TextFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "text", &buf)

	logger.Info("hello")

	out := buf.String()
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("output = %q, want it to contain 'level=INFO'", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("output = %q, want it to contain 'hello'", out)
	}
}

func TestNew_DebugLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("debug", "json", &buf)

	logger.Debug("debug message")

	if buf.Len() == 0 {
		t.Error("debug message was filtered out, want it to appear at debug level")
	}
}

func TestNew_DebugLevelIncludesSource(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("debug", "json", &buf)

	logger.Debug("with source")

	out := buf.String()
	if !strings.Contains(out, `"source"`) {
		t.Errorf("output = %q, want it to contain '\"source\"' at debug level", out)
	}
}

func TestNew_InfoLevelExcludesSource(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	logger.Info("no source")

	out := buf.String()
	if strings.Contains(out, `"source"`) {
		t.Errorf("output = %q, want no '\"source\"' at info level", out)
	}
}

func TestNew_InfoLevelFiltersDebug(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	logger.Debug("should not appear")

	if buf.Len() != 0 {
		t.Errorf("debug message appeared at info level, output = %q", buf.String())
	}
}

func TestNew_ErrorLevelFiltersWarn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("error", "json", &buf)

	logger.Warn("should not appear")

	if buf.Len() != 0 {
		t.Errorf("warn message appeared at error level, output = %q", buf.String())
	}
}

func TestNew_UnknownLevelDefaultsToInfo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("verbose", "json", &buf)

	// Debug should be filtered (defaulted to info).
	logger.Debug("should not appear")
	if buf.Len() != 0 {
		t.Errorf("debug message appeared with unknown level, output = %q", buf.String())
	}

	// Info should pass through.
	logger.Info("should appear")
	if buf.Len() == 0 {
		t.Error("info message was filtered with unknown level, want it to appear")
	}
}

func TestNew_UnknownFormatDefaultsToJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "xml", &buf)

	logger.Info("hello")

	out := buf.String()
	if !strings.Contains(out, `"level":"INFO"`) {
		t.Errorf("output = %q, want JSON format for unknown format string", out)
	}
}

func TestNew_LevelCaseInsensitive(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("DEBUG", "json", &buf)

	logger.Debug("should appear")

	if buf.Len() == 0 {
		t.Error("debug message was filtered with uppercase 'DEBUG', want case-insensitive parsing")
	}
}

// --- Context tests ---

func TestFromContext_WithLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	ctx := logging.WithLogger(context.Background(), logger)
	got := logging.FromContext(ctx)

	if got != logger {
		t.Error("FromContext returned different logger than the one stored with WithLogger")
	}
}

func TestFromContext_NoLogger(t *testing.T) {
	t.Parallel()

	got := logging.FromContext(context.Background())

	if got != slog.Default() {
		t.Error("FromContext on bare context returned something other than slog.Default()")
	}
}

func TestWithLogger_OverwritesPrevious(t *testing.T) {
	t.Parallel()

	var buf1, buf2 bytes.Buffer
	logger1 := logging.New("info", "json", &buf1)
	logger2 := logging.New("debug", "json", &buf2)

	ctx := logging.WithLogger(context.Background(), logger1)
	ctx = logging.WithLogger(ctx, logger2)

	got := logging.FromContext(ctx)
	if got != logger2 {
		t.Error("FromContext returned first logger, want second (overwritten) logger")
	}
}

// --- Redaction tests ---

func TestNew_RedactsAuthorizationFieldName(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	logger.Info("request", slog.String("authorization", "Bearer supersecret-token"))

	out := buf.String()
	if strings.Contains(out, "supersecret-token") {
		t.Error("log output contains raw token, want it redacted")
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Error("log output missing [REDACTED] marker")
	}
}

func TestNew_RedactsPasswordFieldName(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	logger.Info("login", slog.String("password", "hunter2"))

	out := buf.String()
	if strings.Contains(out, "hunter2") {
		t.Error("log output contains raw password, want it redacted")
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Error("log output missing [REDACTED] marker")
	}
}

func TestNew_DefenseInDepthBearerRegex(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	logger.Info("debug trace", slog.String("raw_header", "Bearer eyJhbGciOiJSUzI1NiJ9"))

	out := buf.String()
	if strings.Contains(out, "eyJhbGciOiJSUzI1NiJ9") {
		t.Error("log output contains raw Bearer token, want it redacted by regex")
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Error("log output missing [REDACTED] marker")
	}
}

func TestNew_DoesNotRedactNonSensitiveFields(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := logging.New("info", "json", &buf)

	logger.Info("event",
		slog.String("user_id", "usr-123"),
		slog.String("path", "/api/projects"),
	)

	out := buf.String()
	if !strings.Contains(out, "usr-123") {
		t.Error("log output missing user_id, non-sensitive field should not be redacted")
	}
	if !strings.Contains(out, "/api/projects") {
		t.Error("log output missing path, non-sensitive field should not be redacted")
	}
}
