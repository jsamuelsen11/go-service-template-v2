package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/httpclient"
)

const headerRequestID = "X-Request-ID"

// requestIDKey is the context key for storing request IDs within the middleware
// package. A separate key from httpclient's is used to avoid a dependency
// inversion (middleware reads its own key; httpclient reads its own key).
type requestIDKey struct{}

// WithRequestID returns a new context with the given request ID stored in it.
// It also stores the ID via httpclient.WithRequestID so that outbound HTTP
// calls automatically include the X-Request-ID header.
func WithRequestID(ctx context.Context, id string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey{}, id)
	ctx = httpclient.WithRequestID(ctx, id)
	return ctx
}

// RequestIDFromContext extracts the request ID from the context.
// Returns an empty string if no request ID is stored.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// RequestID returns middleware that generates or extracts an X-Request-ID for
// each request. If the incoming request has an X-Request-ID header, it is
// reused; otherwise a new UUID v4 is generated. The ID is stored in the
// request context and set as a response header.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(headerRequestID)
			if id == "" {
				id = generateID()
			}
			ctx := WithRequestID(r.Context(), id)
			w.Header().Set(headerRequestID, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UUID v4 bit manipulation constants.
const (
	uuidVersion4    = 0x40 // Version 4 (random) in bits 4-7 of byte 6.
	uuidVersionMask = 0x0f // Mask to clear version bits before setting.
	uuidVariant10   = 0x80 // RFC 4122 variant (10xx) in bits 6-7 of byte 8.
	uuidVariantMask = 0x3f // Mask to clear variant bits before setting.
)

// generateID produces a UUID v4 string using crypto/rand.
// Format: "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx" where y is 8, 9, a, or b.
func generateID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])

	uuid[6] = (uuid[6] & uuidVersionMask) | uuidVersion4
	uuid[8] = (uuid[8] & uuidVariantMask) | uuidVariant10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
