package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"
)

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(new(bytes.Buffer), nil))
}

func TestRecovery_NoPanic(t *testing.T) {
	t.Parallel()

	handler := middleware.Recovery(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestRecovery_HandlesPanic(t *testing.T) {
	t.Parallel()

	handler := middleware.Recovery(discardLogger())(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("something went wrong")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/problem+json")
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
	if title, _ := body["title"].(string); title != "Internal Server Error" {
		t.Errorf("title = %q, want %q", title, "Internal Server Error")
	}
}

func TestRecovery_LogsPanicWithStack(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	handler := middleware.Recovery(testLogger(&buf))(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic value")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/log-test", http.NoBody)
	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Error("log output missing 'panic recovered'")
	}
	if !strings.Contains(logOutput, "test panic value") {
		t.Error("log output missing panic value")
	}
	if !strings.Contains(logOutput, "goroutine") {
		t.Error("log output missing stack trace")
	}
}

func TestRecovery_HandlesNonStringPanic(t *testing.T) {
	t.Parallel()

	handler := middleware.Recovery(discardLogger())(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(42)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRecovery_SkipsResponseIfHeadersAlreadyWritten(t *testing.T) {
	t.Parallel()

	handler := middleware.Recovery(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("partial"))
		panic("late panic")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	handler.ServeHTTP(rec, req)

	// Original status should be preserved since headers were already written.
	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d (original, not 500)", rec.Code, http.StatusAccepted)
	}
}
