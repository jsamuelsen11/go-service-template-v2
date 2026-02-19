// Package main is the entry point for the service. It wires all dependencies
// using samber/do v2, starts the HTTP server, and handles graceful shutdown
// on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	adapthttp "github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http"
	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/handlers"
	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/middleware"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/clients/acl"
	"github.com/jsamuelsen11/go-service-template-v2/internal/app"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/config"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/health"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/httpclient"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/logging"
	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/telemetry"
	"github.com/jsamuelsen11/go-service-template-v2/internal/ports"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	serverShutdownTimeout = 15 * time.Second
	otelShutdownTimeout   = 5 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	profile := os.Getenv("APP_PROFILE")
	if profile == "" {
		return errors.New("APP_PROFILE environment variable is required (e.g. local, dev, qa, prod)")
	}

	// Bootstrap: config, logger, telemetry.
	cfg, err := config.Load(profile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := logging.New(cfg.Log.Level, cfg.Log.Format, os.Stderr)

	ctx := context.Background()
	otel, err := initTelemetry(ctx, cfg)
	if err != nil {
		return fmt.Errorf("initializing telemetry: %w", err)
	}

	// DI container.
	injector := do.New()

	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)
	do.ProvideValue(injector, otel.metrics)

	registerDependencies(injector, cfg, logger)

	// Resolve the server (eagerly wires the full graph).
	server, err := do.Invoke[*adapthttp.Server](injector)
	if err != nil {
		return fmt.Errorf("resolving server: %w", err)
	}

	// Register health checkers after the graph is wired.
	registry := do.MustInvoke[ports.HealthRegistry](injector)
	httpClient := do.MustInvoke[*httpclient.Client](injector)
	registry.Register(httpClient)

	// Start server in background.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for shutdown signal or server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	case err := <-serverErr:
		return fmt.Errorf("server failed: %w", err)
	}

	// Graceful shutdown: drain HTTP requests.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.Any("error", err))
	}

	// Wait for Start() goroutine to return.
	<-serverErr

	// Flush telemetry.
	otelCtx, otelCancel := context.WithTimeout(context.Background(), otelShutdownTimeout)
	defer otelCancel()

	if err := otel.Shutdown(otelCtx); err != nil {
		logger.Error("telemetry shutdown error", slog.Any("error", err))
	}

	logger.Info("shutdown complete")
	return nil
}

// otelProviders bundles OpenTelemetry provider lifecycle. All fields are nil
// when telemetry is disabled.
type otelProviders struct {
	tracer  *sdktrace.TracerProvider
	meter   *sdkmetric.MeterProvider
	metrics *telemetry.Metrics
}

// Shutdown flushes both providers. Nil-safe.
func (o *otelProviders) Shutdown(ctx context.Context) error {
	var errs []error
	if o.tracer != nil {
		if err := o.tracer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}
	if o.meter != nil {
		if err := o.meter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
	}
	return errors.Join(errs...)
}

func initTelemetry(ctx context.Context, cfg *config.Config) (*otelProviders, error) {
	if !cfg.Telemetry.Enabled {
		return &otelProviders{}, nil
	}

	tp, err := telemetry.InitTracer(ctx,
		cfg.Telemetry.ServiceName,
		cfg.Telemetry.Exporter,
		cfg.Telemetry.Endpoint,
	)
	if err != nil {
		return nil, fmt.Errorf("init tracer: %w", err)
	}

	mp, err := telemetry.InitMeter(ctx,
		cfg.Telemetry.ServiceName,
		cfg.Telemetry.Exporter,
		cfg.Telemetry.Endpoint,
	)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("init meter: %w", err)
	}

	metrics, err := telemetry.NewMetrics(mp, cfg.Telemetry.ServiceName)
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, fmt.Errorf("creating metrics: %w", err)
	}

	return &otelProviders{
		tracer:  tp,
		meter:   mp,
		metrics: metrics,
	}, nil
}

func registerDependencies(injector *do.RootScope, cfg *config.Config, logger *slog.Logger) {
	do.Provide(injector, func(i do.Injector) (*httpclient.Client, error) {
		metrics := do.MustInvoke[*telemetry.Metrics](i)
		return httpclient.New(&cfg.Client, "todo-api", metrics, logger), nil
	})

	do.Provide(injector, func(i do.Injector) (ports.TodoClient, error) {
		client := do.MustInvoke[*httpclient.Client](i)
		return acl.NewTodoClient(client, logger), nil
	})

	do.Provide(injector, func(i do.Injector) (ports.ProjectService, error) {
		todoClient := do.MustInvoke[ports.TodoClient](i)
		return app.NewProjectService(todoClient, logger), nil
	})

	do.Provide(injector, func(_ do.Injector) (ports.HealthRegistry, error) {
		return health.New(), nil
	})

	do.Provide(injector, func(i do.Injector) (*handlers.ProjectHandler, error) {
		svc := do.MustInvoke[ports.ProjectService](i)
		return handlers.NewProjectHandler(svc), nil
	})

	do.Provide(injector, func(i do.Injector) (*handlers.HealthHandler, error) {
		registry := do.MustInvoke[ports.HealthRegistry](i)
		return handlers.NewHealthHandler(registry), nil
	})

	do.Provide(injector, func(i do.Injector) (nethttp.Handler, error) {
		projH := do.MustInvoke[*handlers.ProjectHandler](i)
		healthH := do.MustInvoke[*handlers.HealthHandler](i)
		metrics := do.MustInvoke[*telemetry.Metrics](i)

		return adapthttp.NewRouter(projH, healthH,
			middleware.Recovery(logger),
			middleware.RequestID(),
			middleware.CorrelationID(),
			middleware.OpenTelemetry(metrics),
			middleware.Logging(logger),
			middleware.Timeout(cfg.Server.WriteTimeout),
		), nil
	})

	do.Provide(injector, func(i do.Injector) (*adapthttp.Server, error) {
		handler := do.MustInvoke[nethttp.Handler](i)
		return adapthttp.NewServer(cfg.Server, handler, logger), nil
	})
}
