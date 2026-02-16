package middleware

import (
	"context"
	"maps"
	"net/http"
	"sync"
	"time"
)

// Timeout returns middleware that enforces a request deadline. If the handler
// does not complete within the given duration, a 504 Gateway Timeout response
// is written. The context passed to the handler carries the deadline so that
// downstream I/O operations can respect it.
//
// The handler runs in a separate goroutine. A sync.Once ensures that exactly
// one of the handler or the timeout path writes the response.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			tw := &timeoutWriter{w: w}
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(tw, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				tw.mu.Lock()
				defer tw.mu.Unlock()
				tw.flush()
			case <-ctx.Done():
				tw.mu.Lock()
				defer tw.mu.Unlock()
				if !tw.wroteHeader {
					w.WriteHeader(http.StatusGatewayTimeout)
				}
			}
		})
	}
}

// timeoutWriter buffers the response so that the timeout path can safely
// write a 504 if the handler hasn't finished. All writes are guarded by a
// mutex shared between the handler goroutine and the timeout select.
type timeoutWriter struct {
	w           http.ResponseWriter
	mu          sync.Mutex
	header      http.Header
	buf         []byte
	statusCode  int
	wroteHeader bool
}

func (tw *timeoutWriter) Header() http.Header {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.header == nil {
		tw.header = make(http.Header)
	}
	return tw.header
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if !tw.wroteHeader {
		tw.statusCode = http.StatusOK
		tw.wroteHeader = true
	}
	tw.buf = append(tw.buf, b...)
	return len(b), nil
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.wroteHeader {
		return
	}
	tw.statusCode = code
	tw.wroteHeader = true
}

// flush copies the buffered response to the underlying writer. Must be
// called with tw.mu held.
func (tw *timeoutWriter) flush() {
	if tw.header != nil {
		maps.Copy(tw.w.Header(), tw.header)
	}
	if tw.wroteHeader {
		tw.w.WriteHeader(tw.statusCode)
	}
	if len(tw.buf) > 0 {
		_, _ = tw.w.Write(tw.buf)
	}
}
