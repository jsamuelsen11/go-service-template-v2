package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
)

const defaultShutdownTimeout = 10 * time.Second

// Server wraps http.Server with graceful shutdown support.
type Server struct {
	srv    *http.Server
	logger *slog.Logger
}

// NewServer creates a new HTTP server from the given config and handler.
func NewServer(cfg config.ServerConfig, handler http.Handler, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Server{
		srv: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		logger: logger,
	}
}

// Start begins listening and serving HTTP requests.
// It blocks until the server stops. Returns nil on graceful shutdown.
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", slog.String("addr", s.srv.Addr))

	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server, waiting for in-flight requests
// to complete within the given context deadline. If ctx has no deadline,
// a default 10-second timeout is applied.
func (s *Server) Shutdown(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultShutdownTimeout)
		defer cancel()
	}

	s.logger.Info("shutting down HTTP server")
	return s.srv.Shutdown(ctx)
}

// Addr returns the server's configured listen address string.
func (s *Server) Addr() string {
	return s.srv.Addr
}
