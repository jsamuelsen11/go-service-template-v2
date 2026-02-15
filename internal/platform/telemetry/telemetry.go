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
//	metrics, err := telemetry.NewMetrics(mp)
//	metrics.ServerRequestTotal.Add(ctx, 1, ...)
package telemetry

import (
	"context"
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

// Attribute keys for metric labels, as specified in ARCHITECTURE.md.
var (
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
// The exporter parameter selects the span exporter: "otlp" uses OTLP/HTTP
// with the given endpoint; any other value (including "stdout") uses a
// pretty-printed stdout exporter for development.
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
// The exporter parameter selects the metric exporter: "otlp" uses OTLP/HTTP
// with the given endpoint; any other value (including "stdout") uses a
// stdout exporter for development.
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
// The meter is scoped to the service's module path.
func NewMetrics(mp *sdkmetric.MeterProvider) (*Metrics, error) {
	meter := mp.Meter("github.com/jsamuelsen11/go-service-template-v2")

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
	if exporter == "otlp" {
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(hostPort(endpoint))}
		if !isHTTPS(endpoint) {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	}
	return stdouttrace.New(stdouttrace.WithPrettyPrint())
}

func newMetricExporter(ctx context.Context, exporter, endpoint string) (sdkmetric.Exporter, error) {
	if exporter == "otlp" {
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(hostPort(endpoint))}
		if !isHTTPS(endpoint) {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		return otlpmetrichttp.New(ctx, opts...)
	}
	return stdoutmetric.New()
}

// hostPort extracts the host:port from a URL string
// (e.g., "http://otel-collector:4318" -> "otel-collector:4318").
func hostPort(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return endpoint
	}
	return u.Host
}

// isHTTPS returns true if the endpoint URL uses the https scheme.
func isHTTPS(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	return u.Scheme == "https"
}
