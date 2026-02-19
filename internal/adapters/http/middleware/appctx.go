package middleware

import (
	"net/http"

	appctx "github.com/jsamuelsen11/go-service-template-v2/internal/app/context"
)

// AppContext returns middleware that creates a new RequestContext for each
// HTTP request and stores it in the request context. Downstream handlers
// and application services can retrieve it via appctx.FromContext(ctx).
//
// This middleware should be registered after CorrelationID (so that the
// RequestContext's embedded context carries request/correlation IDs) and
// before OpenTelemetry (so that the RequestContext is available when
// tracing begins).
func AppContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := appctx.New(r.Context())
			ctx := appctx.WithRequestContext(r.Context(), rc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
