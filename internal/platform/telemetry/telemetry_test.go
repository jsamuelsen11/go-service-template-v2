package telemetry_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/jsamuelsen11/go-service-template-v2/internal/platform/telemetry"
)

func TestInitTracer_Stdout(t *testing.T) {
	ctx := context.Background()

	tp, err := telemetry.InitTracer(ctx, "test-service", telemetry.ExporterStdout, "")
	if err != nil {
		t.Fatalf("InitTracer(stdout) error = %v", err)
	}
	t.Cleanup(func() {
		if err := tp.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown error = %v", err)
		}
	})

	if tp == nil {
		t.Fatal("InitTracer(stdout) returned nil TracerProvider")
	}
}

func TestInitTracer_OTLP(t *testing.T) {
	ctx := context.Background()

	tp, err := telemetry.InitTracer(ctx, "test-service", telemetry.ExporterOTLP, "http://localhost:4318")
	if err != nil {
		t.Fatalf("InitTracer(otlp) error = %v", err)
	}
	t.Cleanup(func() {
		// Shutdown may fail when no collector is running; this is expected in unit tests.
		_ = tp.Shutdown(ctx)
	})

	if tp == nil {
		t.Fatal("InitTracer(otlp) returned nil TracerProvider")
	}
}

func TestInitTracer_SetsGlobalPropagator(t *testing.T) {
	ctx := context.Background()

	tp, err := telemetry.InitTracer(ctx, "test-service", telemetry.ExporterStdout, "")
	if err != nil {
		t.Fatalf("InitTracer error = %v", err)
	}
	t.Cleanup(func() {
		if err := tp.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown error = %v", err)
		}
	})

	prop := otel.GetTextMapPropagator()
	if _, ok := prop.(propagation.TraceContext); ok {
		// Single TraceContext is fine but we expect a composite.
		return
	}
	// Composite propagator should have non-empty Fields().
	if len(prop.Fields()) == 0 {
		t.Error("global propagator has no fields, want TraceContext + Baggage fields")
	}
}

func TestInitTracer_UnsupportedExporter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := telemetry.InitTracer(ctx, "test-service", "invalid", "")
	if err == nil {
		t.Fatal("InitTracer with unsupported exporter should return error")
	}
}

func TestInitTracer_OTLPEmptyEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := telemetry.InitTracer(ctx, "test-service", telemetry.ExporterOTLP, "")
	if err == nil {
		t.Fatal("InitTracer with otlp and empty endpoint should return error")
	}
}

func TestInitMeter_Stdout(t *testing.T) {
	ctx := context.Background()

	mp, err := telemetry.InitMeter(ctx, "test-service", telemetry.ExporterStdout, "")
	if err != nil {
		t.Fatalf("InitMeter(stdout) error = %v", err)
	}
	t.Cleanup(func() {
		if err := mp.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown error = %v", err)
		}
	})

	if mp == nil {
		t.Fatal("InitMeter(stdout) returned nil MeterProvider")
	}
}

func TestInitMeter_OTLP(t *testing.T) {
	ctx := context.Background()

	mp, err := telemetry.InitMeter(ctx, "test-service", telemetry.ExporterOTLP, "http://localhost:4318")
	if err != nil {
		t.Fatalf("InitMeter(otlp) error = %v", err)
	}
	t.Cleanup(func() {
		// Shutdown may fail when no collector is running; this is expected in unit tests.
		_ = mp.Shutdown(ctx)
	})

	if mp == nil {
		t.Fatal("InitMeter(otlp) returned nil MeterProvider")
	}
}

func TestInitMeter_UnsupportedExporter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := telemetry.InitMeter(ctx, "test-service", "invalid", "")
	if err == nil {
		t.Fatal("InitMeter with unsupported exporter should return error")
	}
}

func TestInitMeter_OTLPEmptyEndpoint(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := telemetry.InitMeter(ctx, "test-service", telemetry.ExporterOTLP, "")
	if err == nil {
		t.Fatal("InitMeter with otlp and empty endpoint should return error")
	}
}

func TestNewMetrics(t *testing.T) {
	ctx := context.Background()

	mp, err := telemetry.InitMeter(ctx, "test-service", telemetry.ExporterStdout, "")
	if err != nil {
		t.Fatalf("InitMeter error = %v", err)
	}
	t.Cleanup(func() {
		if err := mp.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown error = %v", err)
		}
	})

	metrics, err := telemetry.NewMetrics(mp, "test-service")
	if err != nil {
		t.Fatalf("NewMetrics error = %v", err)
	}

	if metrics.ServerRequestDuration == nil {
		t.Error("ServerRequestDuration is nil")
	}
	if metrics.ServerRequestTotal == nil {
		t.Error("ServerRequestTotal is nil")
	}
	if metrics.ClientRequestDuration == nil {
		t.Error("ClientRequestDuration is nil")
	}
	if metrics.ClientRequestTotal == nil {
		t.Error("ClientRequestTotal is nil")
	}
}
