// Package httpclient provides an instrumented HTTP client with circuit breaker,
// retry with exponential backoff, OpenTelemetry tracing, and header injection
// for outbound requests.
//
// The client applies middleware-like processing in this order:
//
//	Circuit Breaker → Rate Limiter → Header Injection → OTEL Span → Retry → HTTP
//
// Construction:
//
//	client := httpclient.New(&cfg.Client, "todo-api", metrics, logger)
//
// Executing requests:
//
//	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
//	resp, err := client.Do(ctx, req)
//
// Context propagation for header injection (set by inbound middleware):
//
//	ctx = httpclient.WithRequestID(ctx, "req-123")
//	ctx = httpclient.WithCorrelationID(ctx, "corr-456")
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/telemetry"
)

// Context key types for request metadata propagation.
type (
	requestIDKey     struct{}
	correlationIDKey struct{}
)

// WithRequestID returns a new context with the given request ID stored in it.
// Inbound middleware should call this to propagate request IDs to outbound calls.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// WithCorrelationID returns a new context with the given correlation ID stored
// in it. Inbound middleware should call this to propagate correlation IDs to
// outbound calls.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// retryConfig holds the retry policy values extracted from config.RetryConfig
// using unexported types to avoid leaking the config package through the API.
type retryConfig struct {
	maxAttempts     int
	initialInterval time.Duration
	maxInterval     time.Duration
	multiplier      float64
}

// Client is an instrumented HTTP client with circuit breaker, rate limiting,
// retry, header injection, and OpenTelemetry tracing for outbound requests.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	serviceName string
	breaker     *gobreaker.CircuitBreaker[struct{}]
	limiter     *rate.Limiter // nil when rate limiting is disabled
	retryCfg    retryConfig
	metrics     *telemetry.Metrics
	logger      *slog.Logger
}

// New creates an instrumented HTTP client configured with circuit breaker,
// retry with exponential backoff, OpenTelemetry tracing, and header injection.
//
// The serviceName identifies the downstream service in traces and metrics
// (e.g., "todo-api"). If metrics is nil, metric recording is skipped.
func New(cfg *config.ClientConfig, serviceName string, metrics *telemetry.Metrics, logger *slog.Logger) *Client {
	cb := gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
		Name:        serviceName,
		MaxRequests: toUint32(cfg.CircuitBreaker.HalfOpenLimit),
		Timeout:     cfg.CircuitBreaker.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return int(counts.ConsecutiveFailures) >= cfg.CircuitBreaker.MaxFailures
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.Warn("circuit breaker state change",
				slog.String("breaker", name),
				slog.String("from", from.String()),
				slog.String("to", to.String()),
			)
		},
	})

	var limiter *rate.Limiter
	if cfg.RateLimit.RequestsPerSecond > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.RateLimit.RequestsPerSecond), cfg.RateLimit.BurstSize)
	}

	return &Client{
		httpClient:  &http.Client{Timeout: cfg.Timeout},
		baseURL:     cfg.BaseURL,
		serviceName: serviceName,
		breaker:     cb,
		limiter:     limiter,
		retryCfg: retryConfig{
			maxAttempts:     cfg.Retry.MaxAttempts,
			initialInterval: cfg.Retry.InitialInterval,
			maxInterval:     cfg.Retry.MaxInterval,
			multiplier:      cfg.Retry.Multiplier,
		},
		metrics: metrics,
		logger:  logger,
	}
}

// Do executes an HTTP request through the full middleware pipeline:
// Circuit Breaker → Rate Limiter → Header Injection → OTEL Span → Retry → HTTP.
//
// The request's context is used for cancellation, tracing, and to extract
// Request-ID and Correlation-ID for header propagation.
//
// When the request succeeds (non-retryable status), resp is non-nil with an
// open body that the caller must close. When all retries are exhausted for a
// retryable status, both resp (with open body) and err are non-nil; the caller
// should close resp.Body. When the circuit breaker rejects or a network error
// occurs, resp is nil.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	start := time.Now()
	method := req.Method

	var resp *http.Response
	_, err := c.breaker.Execute(func() (struct{}, error) {
		if err := c.waitForRateLimit(ctx); err != nil {
			return struct{}{}, err
		}

		c.injectHeaders(ctx, req)

		spanCtx, span := c.startSpan(ctx, req)
		defer span.End()

		// Bind span context to the request so http.Client.Do uses it for
		// cancellation, deadlines, and trace propagation.
		req = req.WithContext(spanCtx)

		retryErr := c.doWithRetry(spanCtx, req, &resp)
		c.finishSpan(span, resp, retryErr)

		return struct{}{}, retryErr
	})

	c.recordMetrics(ctx, method, start, resp, err)

	return resp, err
}

// BaseURL returns the base URL configured for this client.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Name returns the downstream service identifier (e.g., "todo-api").
// Together with HealthCheck, this method lets Client satisfy the
// ports.HealthChecker interface via structural typing — no import needed.
func (c *Client) Name() string {
	return c.serviceName
}

// HealthCheck reports the downstream service's availability based on the
// circuit breaker state — no network call is made.
//
// State mapping:
//   - "closed"    — downstream is operating normally; returns nil.
//   - "half-open" — circuit breaker is probing recovery; returns a
//     descriptive error indicating degraded state.
//   - "open"      — downstream is unavailable and the breaker is rejecting
//     requests; returns a descriptive error indicating failure.
//
// This reports downstream status, not service readiness. The service itself
// is always ready to handle requests even when a downstream is failing.
func (c *Client) HealthCheck(_ context.Context) error {
	state := c.breaker.State()
	switch state {
	case gobreaker.StateClosed:
		return nil
	case gobreaker.StateHalfOpen:
		return fmt.Errorf("%s: degraded (circuit breaker half-open)", c.serviceName)
	case gobreaker.StateOpen:
		return fmt.Errorf("%s: failing (circuit breaker open)", c.serviceName)
	default:
		return fmt.Errorf("%s: unknown circuit breaker state %v", c.serviceName, state)
	}
}

// waitForRateLimit blocks until the rate limiter allows the request or the
// context is canceled. Returns nil immediately when rate limiting is disabled.
func (c *Client) waitForRateLimit(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	return c.limiter.Wait(ctx)
}

// injectHeaders adds Request-ID and Correlation-ID headers to the outbound
// request if present in the context.
func (c *Client) injectHeaders(ctx context.Context, req *http.Request) {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok && id != "" {
		req.Header.Set("X-Request-ID", id)
	}
	if id, ok := ctx.Value(correlationIDKey{}).(string); ok && id != "" {
		req.Header.Set("X-Correlation-ID", id)
	}
}

// startSpan creates an OTEL client span for the outbound request and injects
// trace context (W3C Trace Context) into the request headers.
func (c *Client) startSpan(ctx context.Context, req *http.Request) (context.Context, trace.Span) {
	tracer := otel.GetTracerProvider().Tracer("httpclient")

	spanName := fmt.Sprintf("HTTP %s %s", req.Method, c.serviceName)
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL.String()),
			attribute.String("peer.service", c.serviceName),
		),
	)

	// Propagate trace context into outbound request headers.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	return ctx, span
}

// finishSpan records the response outcome on the span.
func (c *Client) finishSpan(span trace.Span, resp *http.Response, err error) {
	if resp != nil {
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// recordMetrics records client request duration and count metrics.
// Metrics are recorded outside the circuit breaker so that circuit-open
// rejections are captured. Safe to call with nil metrics.
func (c *Client) recordMetrics(ctx context.Context, method string, start time.Time, resp *http.Response, err error) {
	if c.metrics == nil {
		return
	}

	duration := time.Since(start).Seconds()

	statusCode := 0
	result := "error"
	if resp != nil {
		statusCode = resp.StatusCode
		if statusCode < http.StatusBadRequest {
			result = "success"
		}
	}
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		result = "circuit_open"
	}

	attrs := metric.WithAttributes(
		telemetry.AttrHTTPMethod.String(method),
		telemetry.AttrHTTPStatus.Int(statusCode),
		telemetry.AttrPeerService.String(c.serviceName),
		telemetry.AttrResult.String(result),
	)

	c.metrics.ClientRequestDuration.Record(ctx, duration, attrs)
	c.metrics.ClientRequestTotal.Add(ctx, 1, attrs)
}

// toUint32 safely converts a non-negative int to uint32, clamping at the
// uint32 maximum. Negative values are treated as zero.
func toUint32(v int) uint32 {
	if v <= 0 {
		return 0
	}
	if v > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v)
}
