package httpclient_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/httpclient"
)

func testConfig(baseURL string) *config.ClientConfig {
	return &config.ClientConfig{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
		Retry: config.RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			MaxFailures:   3,
			Timeout:       1 * time.Second,
			HalfOpenLimit: 1,
		},
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestDo_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/test", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", string(body), "ok")
	}
}

func TestDo_RetryOnRetryableStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		failStatus   int
		failCount    int
		wantAttempts int32
	}{
		{
			name:         "5xx retries until success",
			failStatus:   http.StatusInternalServerError,
			failCount:    2,
			wantAttempts: 3,
		},
		{
			name:         "429 retries until success",
			failStatus:   http.StatusTooManyRequests,
			failCount:    1,
			wantAttempts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var count atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				n := count.Add(1)
				if int(n) <= tt.failCount {
					w.WriteHeader(tt.failStatus)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)

			client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/retry", http.NoBody)
			if err != nil {
				t.Fatalf("creating request: %v", err)
			}

			resp, err := client.Do(context.Background(), req)
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
			}

			if got := count.Load(); got != tt.wantAttempts {
				t.Errorf("request count = %d, want %d", got, tt.wantAttempts)
			}
		})
	}
}

func TestDo_NoRetryOn4xx(t *testing.T) {
	t.Parallel()

	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/bad", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	if got := count.Load(); got != 1 {
		t.Errorf("request count = %d, want 1 (no retries for 4xx)", got)
	}
}

func TestDo_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()

	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("unavailable"))
	}))
	t.Cleanup(srv.Close)

	client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/unavail", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err == nil {
		t.Fatal("Do() error = nil, want non-nil after max retries")
	}

	if got := count.Load(); got != 3 {
		t.Errorf("request count = %d, want 3", got)
	}

	// Last attempt's response should have body intact.
	if resp == nil {
		t.Fatal("resp is nil, want non-nil with body intact")
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "unavailable" {
		t.Errorf("body = %q, want %q", string(body), "unavailable")
	}
}

func TestDo_RequestBodyPreservedAcrossRetries(t *testing.T) {
	t.Parallel()

	var (
		count  atomic.Int32
		bodies []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		n := count.Add(1)
		if n <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := testConfig(srv.URL)
	client := httpclient.New(cfg, "test-svc", nil, testLogger())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/body", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if len(bodies) != 2 {
		t.Fatalf("request count = %d, want 2", len(bodies))
	}

	for i, b := range bodies {
		if b != "hello" {
			t.Errorf("attempt %d body = %q, want %q", i+1, b, "hello")
		}
	}
}

func TestDo_HeaderInjection(t *testing.T) {
	t.Parallel()

	var gotReqID, gotCorrID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReqID = r.Header.Get("X-Request-ID")
		gotCorrID = r.Header.Get("X-Correlation-ID")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

	ctx := httpclient.WithRequestID(context.Background(), "req-123")
	ctx = httpclient.WithCorrelationID(ctx, "corr-456")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/headers", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(ctx, req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if gotReqID != "req-123" {
		t.Errorf("X-Request-ID = %q, want %q", gotReqID, "req-123")
	}
	if gotCorrID != "corr-456" {
		t.Errorf("X-Correlation-ID = %q, want %q", gotCorrID, "corr-456")
	}
}

func TestDo_NoHeadersWithoutContext(t *testing.T) {
	t.Parallel()

	var gotReqID, gotCorrID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReqID = r.Header.Get("X-Request-ID")
		gotCorrID = r.Header.Get("X-Correlation-ID")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/noheaders", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if gotReqID != "" {
		t.Errorf("X-Request-ID = %q, want empty", gotReqID)
	}
	if gotCorrID != "" {
		t.Errorf("X-Correlation-ID = %q, want empty", gotCorrID)
	}
}

func TestDo_CircuitBreakerOpens(t *testing.T) {
	t.Parallel()

	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfg := testConfig(srv.URL)
	cfg.CircuitBreaker.MaxFailures = 1
	cfg.Retry.MaxAttempts = 1 // Disable retries to count CB trips easily.

	client := httpclient.New(cfg, "test-svc", nil, testLogger())

	// First request: triggers failure, CB counts it.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/cb", http.NoBody)
	resp, _ := client.Do(context.Background(), req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	// Second request: CB should be open, no server hit.
	countBefore := count.Load()
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/cb", http.NoBody)
	resp, err := client.Do(context.Background(), req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	if err == nil {
		t.Fatal("Do() error = nil, want circuit breaker error")
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("error = %v, want gobreaker.ErrOpenState", err)
	}
	if count.Load() != countBefore {
		t.Error("server was hit while circuit breaker should be open")
	}
}

func TestDo_CircuitBreakerRecovery(t *testing.T) {
	t.Parallel()

	var shouldFail atomic.Bool
	shouldFail.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if shouldFail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := testConfig(srv.URL)
	cfg.CircuitBreaker.MaxFailures = 1
	cfg.CircuitBreaker.Timeout = 100 * time.Millisecond // Short timeout for test.
	cfg.Retry.MaxAttempts = 1

	client := httpclient.New(cfg, "test-svc", nil, testLogger())

	// Trip the circuit breaker.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/recover", http.NoBody)
	resp, _ := client.Do(context.Background(), req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	// Verify CB is open.
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/recover", http.NoBody)
	resp, err := client.Do(context.Background(), req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("expected circuit breaker open, got: %v", err)
	}

	// Wait for CB timeout to transition to half-open.
	time.Sleep(150 * time.Millisecond)

	// Fix the downstream service.
	shouldFail.Store(false)

	// Half-open probe should succeed, closing the circuit.
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/recover", http.NoBody)
	resp, err = client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil (circuit should recover)", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d after recovery", resp.StatusCode, http.StatusOK)
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/cancel", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(ctx, req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err == nil {
		t.Fatal("Do() error = nil, want context error")
	}
}

func TestClient_Name(t *testing.T) {
	t.Parallel()

	client := httpclient.New(testConfig("http://localhost"), "todo-api", nil, testLogger())

	if got := client.Name(); got != "todo-api" {
		t.Errorf("Name() = %q, want %q", got, "todo-api")
	}
}

func TestClient_HealthCheck_Closed(t *testing.T) {
	t.Parallel()

	// A fresh client has a closed circuit breaker — healthy.
	client := httpclient.New(testConfig("http://localhost"), "todo-api", nil, testLogger())

	if err := client.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck() = %v, want nil (closed breaker)", err)
	}
}

func TestClient_HealthCheck_Open(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfg := testConfig(srv.URL)
	cfg.CircuitBreaker.MaxFailures = 1
	cfg.Retry.MaxAttempts = 1

	client := httpclient.New(cfg, "todo-api", nil, testLogger())

	// Trip the circuit breaker with a failing request.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/health", http.NoBody)
	resp, _ := client.Do(context.Background(), req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck() = nil, want error (open breaker)")
	}
	if !strings.Contains(err.Error(), "failing") {
		t.Errorf("HealthCheck() = %q, want error containing %q", err, "failing")
	}
}

func TestClient_HealthCheck_HalfOpen(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfg := testConfig(srv.URL)
	cfg.CircuitBreaker.MaxFailures = 1
	cfg.CircuitBreaker.Timeout = 100 * time.Millisecond
	cfg.Retry.MaxAttempts = 1

	client := httpclient.New(cfg, "todo-api", nil, testLogger())

	// Trip the circuit breaker.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/health", http.NoBody)
	resp, _ := client.Do(context.Background(), req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	// Wait for the CB timeout so it transitions to half-open.
	time.Sleep(150 * time.Millisecond)

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck() = nil, want error (half-open breaker)")
	}
	if !strings.Contains(err.Error(), "degraded") {
		t.Errorf("HealthCheck() = %q, want error containing %q", err, "degraded")
	}
}

func TestDo_NilMetrics(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	// Explicitly pass nil metrics to verify no panic.
	client := httpclient.New(testConfig(srv.URL), "test-svc", nil, testLogger())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/nil-metrics", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
