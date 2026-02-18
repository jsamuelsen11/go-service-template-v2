package http_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	adapthttp "github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestNewServer_NilLogger(t *testing.T) {
	t.Parallel()

	cfg := config.ServerConfig{Host: "127.0.0.1", Port: 0}
	s := adapthttp.NewServer(cfg, http.NotFoundHandler(), nil)

	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestServer_Addr(t *testing.T) {
	t.Parallel()

	cfg := config.ServerConfig{Host: "127.0.0.1", Port: 9090}
	s := adapthttp.NewServer(cfg, http.NotFoundHandler(), discardLogger())

	if got := s.Addr(); got != "127.0.0.1:9090" {
		t.Errorf("Addr() = %q, want %q", got, "127.0.0.1:9090")
	}
}

func TestServer_StartAndShutdown(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})

	cfg := config.ServerConfig{
		Host:         "127.0.0.1",
		Port:         0,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	s := adapthttp.NewServer(cfg, handler, discardLogger())

	// Start returns nil on graceful shutdown, so we collect the error in a channel.
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	// Give the server a moment to start listening.
	time.Sleep(50 * time.Millisecond)

	// Gracefully shut down.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error: %v", err)
	}

	// Start should have returned nil.
	if err := <-errCh; err != nil {
		t.Fatalf("Start() error after shutdown: %v", err)
	}
}

func TestServer_ShutdownDefaultTimeout(t *testing.T) {
	t.Parallel()

	cfg := config.ServerConfig{Host: "127.0.0.1", Port: 0}
	s := adapthttp.NewServer(cfg, http.NotFoundHandler(), discardLogger())

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	// Pass a context without a deadline -- should use the default 10s timeout.
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Start() error after shutdown: %v", err)
	}
}
