package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
)

// errInternalServer is the generic error returned to clients when a panic is
// recovered. The actual panic value and stack trace are logged but never
// exposed in the HTTP response.
var errInternalServer = errors.New("internal server error")

// Recovery returns middleware that recovers from panics in downstream handlers.
// When a panic occurs the middleware logs the error with the full stack trace
// and returns an RFC 9457 500 response. If the response headers have already
// been written, only the log entry is emitted.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := newResponseWriter(w)

			defer func() {
				if v := recover(); v != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						slog.String("panic", fmt.Sprint(v)),
						slog.String("stack", string(debug.Stack())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)

					if !rw.headerWritten {
						dto.WriteErrorResponse(rw, r, errInternalServer)
					}
				}
			}()

			next.ServeHTTP(rw, r)
		})
	}
}
