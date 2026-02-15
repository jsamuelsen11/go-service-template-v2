package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestBackoff_ExponentialIncrease(t *testing.T) {
	t.Parallel()

	cfg := retryConfig{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     10 * time.Second,
		multiplier:      2.0,
	}

	// Run multiple samples to account for jitter.
	const samples = 100
	for attempt := 1; attempt <= 3; attempt++ {
		baseDelay := float64(100*time.Millisecond) * pow(2.0, attempt-1)
		minExpected := time.Duration(baseDelay * (1 - jitterFraction))
		maxExpected := time.Duration(baseDelay * (1 + jitterFraction))

		for range samples {
			delay := backoff(attempt, cfg)
			if delay < minExpected || delay > maxExpected {
				t.Errorf("attempt %d: delay %v not in [%v, %v]", attempt, delay, minExpected, maxExpected)
			}
		}
	}
}

func TestBackoff_CappedAtMaxInterval(t *testing.T) {
	t.Parallel()

	cfg := retryConfig{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     500 * time.Millisecond,
		multiplier:      2.0,
	}

	// Attempt 10 would be 100ms * 2^9 = 51.2s without cap.
	maxWithJitter := time.Duration(float64(cfg.maxInterval) * (1 + jitterFraction))

	const samples = 100
	for range samples {
		delay := backoff(10, cfg)
		if delay > maxWithJitter {
			t.Errorf("delay %v exceeds max interval with jitter %v", delay, maxWithJitter)
		}
	}
}

func TestBackoff_JitterWithinBounds(t *testing.T) {
	t.Parallel()

	cfg := retryConfig{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     10 * time.Second,
		multiplier:      2.0,
	}

	baseDelay := 100 * time.Millisecond
	minExpected := time.Duration(float64(baseDelay) * (1 - jitterFraction))
	maxExpected := time.Duration(float64(baseDelay) * (1 + jitterFraction))

	const samples = 1000
	for range samples {
		delay := backoff(1, cfg)
		if delay < minExpected || delay > maxExpected {
			t.Errorf("delay %v not in [%v, %v]", delay, minExpected, maxExpected)
		}
	}
}

func TestIsRetryable_Nil(t *testing.T) {
	t.Parallel()

	if isRetryable(nil) {
		t.Error("nil error should not be retryable")
	}
}

func TestIsRetryable_ContextCanceled(t *testing.T) {
	t.Parallel()

	if isRetryable(context.Canceled) {
		t.Error("context.Canceled should not be retryable")
	}
}

func TestIsRetryable_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	if isRetryable(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should not be retryable")
	}
}

func TestIsRetryable_NetError(t *testing.T) {
	t.Parallel()

	err := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	if !isRetryable(err) {
		t.Error("net.Error should be retryable")
	}
}

func TestIsRetryable_GenericError(t *testing.T) {
	t.Parallel()

	if !isRetryable(errors.New("something failed")) {
		t.Error("generic error should be retryable")
	}
}

func TestIsRetryableStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{name: "200 OK", statusCode: http.StatusOK, want: false},
		{name: "201 Created", statusCode: http.StatusCreated, want: false},
		{name: "400 Bad Request", statusCode: http.StatusBadRequest, want: false},
		{name: "404 Not Found", statusCode: http.StatusNotFound, want: false},
		{name: "429 Too Many Requests", statusCode: http.StatusTooManyRequests, want: true},
		{name: "500 Internal Server Error", statusCode: http.StatusInternalServerError, want: true},
		{name: "502 Bad Gateway", statusCode: http.StatusBadGateway, want: true},
		{name: "503 Service Unavailable", statusCode: http.StatusServiceUnavailable, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isRetryableStatus(tt.statusCode); got != tt.want {
				t.Errorf("isRetryableStatus(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestSecureRandFloat64_InRange(t *testing.T) {
	t.Parallel()

	const samples = 1000
	for range samples {
		v := secureRandFloat64()
		if v < 0 || v >= 1 {
			t.Errorf("secureRandFloat64() = %v, want [0, 1)", v)
		}
	}
}

// pow is a test helper for integer-base exponentiation.
func pow(base float64, exp int) float64 {
	result := 1.0
	for range exp {
		result *= base
	}
	return result
}
