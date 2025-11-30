package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// newResource creates a resource describing the service.
func newResource(cfg *Config) (*resource.Resource, error) {
	// Create resource with service attributes
	// Note: We create a standalone resource to avoid schema URL conflicts
	// with resource.Default() which uses a different semconv version
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion("0.1.0"), // TODO: Make configurable
	), nil
}

// newTracerProvider creates a TracerProvider with OTLP exporter.
func newTracerProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*trace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(), // TODO: Make TLS configurable
	)
	if err != nil {
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}

	// Configure sampler based on config
	var sampler trace.Sampler
	if cfg.Sampling.Rate >= 1.0 {
		sampler = trace.AlwaysSample()
	} else if cfg.Sampling.Rate <= 0 {
		sampler = trace.NeverSample()
	} else {
		sampler = trace.TraceIDRatioBased(cfg.Sampling.Rate)
	}

	// Wrap with parent-based sampler for proper context propagation
	sampler = trace.ParentBased(sampler)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	return tp, nil
}

// newMeterProvider creates a MeterProvider with OTLP exporter.
func newMeterProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*metric.MeterProvider, error) {
	if !cfg.Metrics.Enabled {
		return nil, nil
	}

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithInsecure(), // TODO: Make TLS configurable
	)
	if err != nil {
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			metric.NewPeriodicReader(
				exporter,
				metric.WithInterval(cfg.Metrics.ExportInterval.Duration()),
			),
		),
	)

	return mp, nil
}

// TracerProviderOption configures TracerProvider creation.
type TracerProviderOption func(*tracerProviderOptions)

type tracerProviderOptions struct {
	exporter trace.SpanExporter
}

// WithTraceExporter overrides the default OTLP exporter (for testing).
func WithTraceExporter(exp trace.SpanExporter) TracerProviderOption {
	return func(opts *tracerProviderOptions) {
		opts.exporter = exp
	}
}

// MeterProviderOption configures MeterProvider creation.
type MeterProviderOption func(*meterProviderOptions)

type meterProviderOptions struct {
	exporter metric.Exporter
}

// WithMetricExporter overrides the default OTLP exporter (for testing).
func WithMetricExporter(exp metric.Exporter) MeterProviderOption {
	return func(opts *meterProviderOptions) {
		opts.exporter = exp
	}
}
