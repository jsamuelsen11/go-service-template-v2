package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
)

// Logging returns middleware that logs request start and completion events.
// It creates a child logger enriched with the request ID and correlation ID
// from context, stores it via logging.WithLogger for downstream use, and
// logs completion with method, path, status code, and duration.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := r.Context()

			reqID := RequestIDFromContext(ctx)
			corrID := CorrelationIDFromContext(ctx)

			child := logger.With(
				slog.String("request_id", reqID),
				slog.String("correlation_id", corrID),
			)
			ctx = logging.WithLogger(ctx, child)

			child.InfoContext(ctx, "request started",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)

			if child.Enabled(ctx, slog.LevelDebug) {
				headerAttrs := RedactHeaders(r.Header)
				args := make([]any, 0, len(headerAttrs))
				for _, a := range headerAttrs {
					args = append(args, a)
				}
				child.DebugContext(ctx, "request headers", args...)
			}

			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r.WithContext(ctx))

			child.InfoContext(ctx, "request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
