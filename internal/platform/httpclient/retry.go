package httpclient

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
)

// jitterFraction is the maximum jitter as a fraction of the delay (±25%).
const jitterFraction = 0.25

// doWithRetry executes the HTTP request with retry logic using exponential
// backoff and ±25% jitter. Request bodies are buffered so they can be
// replayed on each attempt. The result is written to resp rather than returned
// to avoid false positives from the bodyclose linter; the caller is
// responsible for closing the response body.
func (c *Client) doWithRetry(ctx context.Context, req *http.Request, resp **http.Response) error {
	if c.retryCfg.maxAttempts <= 0 {
		return fmt.Errorf("httpclient: maxAttempts must be >= 1, got %d", c.retryCfg.maxAttempts)
	}

	bodyBytes, err := bufferRequestBody(req)
	if err != nil {
		return err
	}

	var lastErr error

	for attempt := range c.retryCfg.maxAttempts {
		if attempt > 0 {
			if err := c.waitForRetry(ctx, req, attempt, lastErr); err != nil {
				return err
			}
		}

		resetRequestBody(req, bodyBytes)

		r, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if !isRetryable(err) {
				return err
			}
			continue
		}

		if !isRetryableStatus(r.StatusCode) {
			*resp = r
			return nil
		}

		lastErr = fmt.Errorf("HTTP %d from %s", r.StatusCode, c.serviceName)

		// On last attempt, return response with body intact for the caller.
		if attempt == c.retryCfg.maxAttempts-1 {
			*resp = r
			return lastErr
		}

		drainResponseBody(r)
	}

	return lastErr
}

// bufferRequestBody reads and closes the request body, returning the bytes
// for replay on subsequent retry attempts. Returns nil if the body is nil.
func bufferRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}
	_ = req.Body.Close()

	return bodyBytes, nil
}

// resetRequestBody replaces the request body with a fresh reader over the
// buffered bytes. No-op if bodyBytes is nil.
func resetRequestBody(req *http.Request, bodyBytes []byte) {
	if bodyBytes == nil {
		return
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.ContentLength = int64(len(bodyBytes))
}

// drainResponseBody reads and discards the response body to enable HTTP
// connection reuse before a retry attempt.
func drainResponseBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// waitForRetry calculates the backoff delay, logs the retry attempt at WARN
// level, and waits for the delay or context cancellation.
func (c *Client) waitForRetry(ctx context.Context, req *http.Request, attempt int, lastErr error) error {
	delay := backoff(attempt, c.retryCfg)

	logger := logging.FromContext(ctx)
	logger.WarnContext(ctx, "retrying HTTP request",
		slog.String("operation", "httpclient.Do"),
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.String("peer_service", c.serviceName),
		slog.Int("attempt", attempt+1),
		slog.Int("max_attempts", c.retryCfg.maxAttempts),
		slog.Duration("backoff", delay),
		slog.Any("error", lastErr),
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// backoff calculates the delay for a given retry attempt using exponential
// backoff with ±25% jitter. The attempt parameter is 1-indexed (attempt 1 is
// the first retry).
func backoff(attempt int, cfg retryConfig) time.Duration {
	delay := float64(cfg.initialInterval) * math.Pow(cfg.multiplier, float64(attempt-1))

	// Cap at max interval before applying jitter.
	if delay > float64(cfg.maxInterval) {
		delay = float64(cfg.maxInterval)
	}

	// Apply ±25% jitter to prevent thundering herd.
	jitter := delay * jitterFraction
	delay += jitter * (2*secureRandFloat64() - 1)

	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// IEEE 754 double-precision constants for random float generation.
const (
	significandBits = 53
	uint64Bits      = 64
)

// secureRandFloat64 returns a random float64 in [0, 1) using crypto/rand.
func secureRandFloat64() float64 {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		return 0
	}
	return float64(binary.BigEndian.Uint64(b[:])>>(uint64Bits-significandBits)) / float64(uint64(1)<<significandBits)
}

// isRetryable determines whether a request error is retryable.
// Context cancellation and deadline exceeded are not retryable.
// Network errors (including timeouts) and unknown errors are retryable.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network errors are retryable.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Default to retryable for unknown errors.
	return true
}

// isRetryableStatus determines whether an HTTP status code is retryable.
// Server errors (5xx) and 429 Too Many Requests are retryable.
func isRetryableStatus(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	return statusCode >= http.StatusInternalServerError
}
