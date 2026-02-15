// Package telemetry provides OpenTelemetry tracer and meter initialization
// with support for stdout (development) and OTLP/HTTP (production) exporters.
//
// Tracer initialization:
//
//	tp, err := telemetry.InitTracer(ctx, "my-service", "stdout", "")
//	defer tp.Shutdown(ctx)
//
// Meter initialization:
//
//	mp, err := telemetry.InitMeter(ctx, "my-service", "stdout", "")
//	defer mp.Shutdown(ctx)
//
// Pre-registered metrics:
//
//	metrics, err := telemetry.NewMetrics(mp, "my-service")
//	metrics.ServerRequestTotal.Add(ctx, 1, ...)
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

// Exporter type constants.
const (
	ExporterStdout = "stdout"
	ExporterOTLP   = "otlp"
)

// Attribute keys for metric labels, as specified in ARCHITECTURE.md.
const (
	AttrHTTPMethod  = attribute.Key("http.method")
	AttrHTTPStatus  = attribute.Key("http.status_code")
	AttrPeerService = attribute.Key("peer.service")
	AttrResult      = attribute.Key("result")
)

// Metrics holds pre-registered OpenTelemetry metric instruments.
type Metrics struct {
	ServerRequestDuration metric.Float64Histogram
	ServerRequestTotal    metric.Int64Counter
	ClientRequestDuration metric.Float64Histogram
	ClientRequestTotal    metric.Int64Counter
}

// InitTracer creates and registers a global TracerProvider.
//
// The exporter parameter selects the span exporter: ExporterOTLP ("otlp")
// uses OTLP/HTTP with the given endpoint; ExporterStdout ("stdout") uses a
// pretty-printed stdout exporter for development. Unrecognized values return
// an error.
//
// The returned TracerProvider must be shut down when the application exits.
func InitTracer(ctx context.Context, serviceName, exporter, endpoint string) (*sdktrace.TracerProvider, error) {
	res, err := newResource(serviceName)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	spanExporter, err := newSpanExporter(ctx, exporter, endpoint)
	if err != nil {
		return nil, fmt.Errorf("creating span exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// InitMeter creates and registers a global MeterProvider.
//
// The exporter parameter selects the metric exporter: ExporterOTLP ("otlp")
// uses OTLP/HTTP with the given endpoint; ExporterStdout ("stdout") uses a
// stdout exporter for development. Unrecognized values return an error.
//
// The returned MeterProvider must be shut down when the application exits.
func InitMeter(ctx context.Context, serviceName, exporter, endpoint string) (*sdkmetric.MeterProvider, error) {
	res, err := newResource(serviceName)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	metricExporter, err := newMetricExporter(ctx, exporter, endpoint)
	if err != nil {
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}

// NewMetrics creates and registers all metric instruments using the given MeterProvider.
// The instrumentationName scopes the meter (typically the service name or module path).
func NewMetrics(mp *sdkmetric.MeterProvider, instrumentationName string) (*Metrics, error) {
	meter := mp.Meter(instrumentationName)

	serverDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of incoming HTTP requests"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating http.server.request.duration: %w", err)
	}

	serverTotal, err := meter.Int64Counter(
		"http.server.request.total",
		metric.WithDescription("Total number of incoming HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating http.server.request.total: %w", err)
	}

	clientDuration, err := meter.Float64Histogram(
		"http.client.request.duration",
		metric.WithDescription("Duration of outgoing HTTP requests"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating http.client.request.duration: %w", err)
	}

	clientTotal, err := meter.Int64Counter(
		"http.client.request.total",
		metric.WithDescription("Total number of outgoing HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating http.client.request.total: %w", err)
	}

	return &Metrics{
		ServerRequestDuration: serverDuration,
		ServerRequestTotal:    serverTotal,
		ClientRequestDuration: clientDuration,
		ClientRequestTotal:    clientTotal,
	}, nil
}

func newResource(serviceName string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
}

func newSpanExporter(ctx context.Context, exporter, endpoint string) (sdktrace.SpanExporter, error) {
	switch exporter {
	case ExporterOTLP:
		opts, err := otlpHTTPOptions(endpoint)
		if err != nil {
			return nil, err
		}
		traceOpts := make([]otlptracehttp.Option, 0, len(opts))
		for _, o := range opts {
			traceOpts = append(traceOpts, o.trace)
		}
		return otlptracehttp.New(ctx, traceOpts...)
	case ExporterStdout:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		return nil, fmt.Errorf("unsupported exporter %q, must be %q or %q", exporter, ExporterStdout, ExporterOTLP)
	}
}

func newMetricExporter(ctx context.Context, exporter, endpoint string) (sdkmetric.Exporter, error) {
	switch exporter {
	case ExporterOTLP:
		opts, err := otlpHTTPOptions(endpoint)
		if err != nil {
			return nil, err
		}
		metricOpts := make([]otlpmetrichttp.Option, 0, len(opts))
		for _, o := range opts {
			metricOpts = append(metricOpts, o.metric)
		}
		return otlpmetrichttp.New(ctx, metricOpts...)
	case ExporterStdout:
		return stdoutmetric.New()
	default:
		return nil, fmt.Errorf("unsupported exporter %q, must be %q or %q", exporter, ExporterStdout, ExporterOTLP)
	}
}

// otlpOption pairs trace and metric options so they stay in sync.
type otlpOption struct {
	trace  otlptracehttp.Option
	metric otlpmetrichttp.Option
}

// otlpHTTPOptions parses the endpoint URL and returns matched trace/metric
// option pairs. The endpoint must be a valid URL with a host component
// (e.g. "http://otel-collector:4318"). Any path component is preserved
// via WithURLPath.
func otlpHTTPOptions(endpoint string) ([]otlpOption, error) {
	if endpoint == "" {
		return nil, errors.New("otlp endpoint must not be empty")
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing otlp endpoint %q: %w", endpoint, err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("otlp endpoint %q must include a host", endpoint)
	}

	opts := []otlpOption{
		{
			trace:  otlptracehttp.WithEndpoint(u.Host),
			metric: otlpmetrichttp.WithEndpoint(u.Host),
		},
	}

	if u.Path != "" && u.Path != "/" {
		opts = append(opts, otlpOption{
			trace:  otlptracehttp.WithURLPath(u.Path),
			metric: otlpmetrichttp.WithURLPath(u.Path),
		})
	}

	if u.Scheme != "https" {
		opts = append(opts, otlpOption{
			trace:  otlptracehttp.WithInsecure(),
			metric: otlpmetrichttp.WithInsecure(),
		})
	}

	return opts, nil
}
