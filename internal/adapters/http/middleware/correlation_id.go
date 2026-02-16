package middleware

import (
	"context"
	"net/http"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/httpclient"
)

const headerCorrelationID = "X-Correlation-ID"

// correlationIDKey is the context key for storing correlation IDs.
type correlationIDKey struct{}

// WithCorrelationID returns a new context with the given correlation ID stored
// in it. It also stores the ID via httpclient.WithCorrelationID so that
// outbound HTTP calls automatically include the X-Correlation-ID header.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	ctx = context.WithValue(ctx, correlationIDKey{}, id)
	ctx = httpclient.WithCorrelationID(ctx, id)
	return ctx
}

// CorrelationIDFromContext extracts the correlation ID from the context.
// Returns an empty string if no correlation ID is stored.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return id
	}
	return ""
}

// CorrelationID returns middleware that extracts or derives an
// X-Correlation-ID for each request. If the incoming request has an
// X-Correlation-ID header, it is reused; otherwise the request ID from
// context is used as a fallback. The ID is stored in the request context
// and set as a response header.
//
// This middleware must run after RequestID so that the fallback value is
// available.
func CorrelationID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(headerCorrelationID)
			if id == "" {
				id = RequestIDFromContext(r.Context())
			}
			ctx := WithCorrelationID(r.Context(), id)
			w.Header().Set(headerCorrelationID, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
