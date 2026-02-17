package acl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/httpclient"
)

// Requester centralizes the HTTP request lifecycle for ACL clients:
// request creation, JSON marshaling, execution via httpclient.Client,
// response body cleanup on error, status code validation, error
// translation, and JSON decoding.
type Requester struct {
	client *httpclient.Client
	logger *slog.Logger
}

// NewRequester creates a Requester backed by the given HTTP client and logger.
func NewRequester(client *httpclient.Client, logger *slog.Logger) *Requester {
	return &Requester{client: client, logger: logger}
}

// Do executes an HTTP request against the configured base URL.
//
// It marshals reqBody to JSON (if non-nil), sends the request, validates the
// status code matches wantStatus, and decodes the response body into respBody
// (if non-nil). For DELETE-style calls where no response body is expected,
// pass nil for respBody.
//
// On non-matching status codes, the response is passed to TranslateHTTPError.
func (r *Requester) Do(ctx context.Context, method, path string, wantStatus int, reqBody, respBody any) error {
	switch method {
	case http.MethodGet:
		return r.get(ctx, path, wantStatus, respBody)
	case http.MethodPost, http.MethodPut:
		return r.withBody(ctx, method, path, wantStatus, reqBody, respBody)
	case http.MethodDelete:
		return r.delete(ctx, path, wantStatus)
	default:
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}
}

// BaseURL returns the base URL from the underlying HTTP client.
func (r *Requester) BaseURL() string {
	return r.client.BaseURL()
}

// CircuitBreakerState returns the circuit breaker state from the underlying
// HTTP client.
func (r *Requester) CircuitBreakerState() string {
	return r.client.CircuitBreakerState()
}

func (r *Requester) get(ctx context.Context, path string, wantStatus int, respBody any) error {
	url := r.client.BaseURL() + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating GET request for %s: %w", path, err)
	}

	return r.execute(req, wantStatus, respBody)
}

func (r *Requester) withBody(ctx context.Context, method, path string, wantStatus int, reqBody, respBody any) error {
	url := r.client.BaseURL() + path

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling %s body for %s: %w", method, path, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating %s request for %s: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")

	return r.execute(req, wantStatus, respBody)
}

func (r *Requester) delete(ctx context.Context, path string, wantStatus int) error {
	url := r.client.BaseURL() + path

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating DELETE request for %s: %w", path, err)
	}

	return r.execute(req, wantStatus, nil)
}

// closeBody is a helper that closes an HTTP response body and logs on failure.
func (r *Requester) closeBody(ctx context.Context, resp *http.Response) {
	if err := resp.Body.Close(); err != nil {
		r.logger.WarnContext(ctx, "failed to close response body",
			slog.String("error", err.Error()),
		)
	}
}

// execute sends the request, checks the status code, and optionally decodes
// the response body. It ensures resp.Body is always closed.
func (r *Requester) execute(req *http.Request, wantStatus int, respBody any) error {
	resp, err := r.client.Do(req.Context(), req)
	if err != nil {
		// httpclient.Do can return both resp and err when retries are exhausted
		// on a retryable status (e.g. 5xx). In that case, translate the HTTP
		// response into a domain error rather than returning the raw retry error.
		if resp != nil {
			defer r.closeBody(req.Context(), resp)
			if resp.StatusCode != wantStatus {
				return TranslateHTTPError(resp)
			}
		}
		r.logger.ErrorContext(req.Context(), "request failed",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("%s %s: %w", req.Method, req.URL.Path, err)
	}
	defer r.closeBody(req.Context(), resp)

	if resp.StatusCode != wantStatus {
		translateErr := TranslateHTTPError(resp)
		r.logger.ErrorContext(req.Context(), "unexpected status",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.Int("status", resp.StatusCode),
			slog.Int("want_status", wantStatus),
		)
		return translateErr
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("decoding response from %s %s: %w", req.Method, req.URL.Path, err)
		}
	}

	return nil
}
