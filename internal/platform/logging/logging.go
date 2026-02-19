// Package logging provides structured logger construction and context propagation
// using the standard library slog package.
//
// Logger construction:
//
//	logger := logging.New("info", "json", os.Stderr)
//
// Context propagation (used by middleware to enrich with request metadata):
//
//	ctx = logging.WithLogger(ctx, logger)
//	logger = logging.FromContext(ctx)
//
// Error logging convention for application services:
//
//	logger.ErrorContext(ctx, "failed to fetch todo",
//	    slog.String("operation", "GetTodo"),
//	    slog.Int64("todo_id", id),
//	    slog.Any("error", err),
//	)
//
// Every error log should include the operation name, entity identifiers, and
// the full error chain via slog.Any("error", err). When logging middleware is
// active, the context carries request_id and correlation_id automatically.
package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// contextKey is the unexported key type for storing loggers in context.
type contextKey struct{}

// New creates a configured *slog.Logger.
//
// The level parameter sets the minimum log level. Valid values are "debug",
// "info", "warn", and "error". Unrecognized values default to info.
//
// The format parameter selects the output handler. "text" uses
// slog.NewTextHandler; all other values (including "json") use
// slog.NewJSONHandler.
//
// When level is "debug", source code location is included in log output.
func New(level, format string, w io.Writer) *slog.Logger {
	lvl := parseLevel(level)

	opts := &slog.HandlerOptions{
		Level:       lvl,
		AddSource:   lvl == slog.LevelDebug,
		ReplaceAttr: newRedactAttr(),
	}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(w, opts)
	} else {
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}

// WithLogger returns a new context with the given logger stored in it.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext extracts a *slog.Logger from the context.
// If no logger is stored, it returns slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// parseLevel converts a level string to slog.Level.
// Unrecognized values default to slog.LevelInfo.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
