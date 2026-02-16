// Package middleware provides HTTP middleware for the inbound request pipeline.
//
// The middleware chain processes requests in this order:
//
//	Recovery → RequestID → CorrelationID → OpenTelemetry → Logging → Timeout → Handler
//
// Each middleware is a func(http.Handler) http.Handler and can be composed
// using the Chain helper.
package middleware

import "net/http"

// responseWriter wraps http.ResponseWriter to capture the status code and
// bytes written. It is used by recovery, otel, and logging middleware.
type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
	written       int64
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code and delegates to the underlying writer.
// Only the first call takes effect; subsequent calls are ignored.
func (rw *responseWriter) WriteHeader(code int) {
	if rw.headerWritten {
		return
	}
	rw.statusCode = code
	rw.headerWritten = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write delegates to the underlying writer, triggering an implicit 200 OK if
// WriteHeader has not been called.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.headerWritten = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// Unwrap returns the underlying http.ResponseWriter so that
// http.ResponseController and type assertions (http.Flusher, http.Hijacker)
// work through the wrapper.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
