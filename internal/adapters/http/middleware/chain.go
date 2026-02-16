package middleware

import "net/http"

// Chain composes multiple middleware into a single middleware. The first
// argument becomes the outermost middleware (executed first on request,
// last on response). This matches the intuitive reading order:
//
//	Chain(Recovery, RequestID, Logging)(handler)
//
// is equivalent to:
//
//	Recovery(RequestID(Logging(handler)))
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}
