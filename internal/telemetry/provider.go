package telemetry

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc/credentials"
)

// newResource creates a resource describing the service.
func newResource(cfg *Config) (*resource.Resource, error) {
	// Create resource with service attributes
	// Note: We create a standalone resource to avoid schema URL conflicts
	// with resource.Default() which uses a different semconv version
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
	), nil
}

// newTracerProvider creates a TracerProvider with OTLP exporter.
func newTracerProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*trace.TracerProvider, error) {
	var exporter trace.SpanExporter
	var err error

	// Choose exporter based on protocol
	protocol := cfg.Protocol
	if protocol == "" {
		protocol = "grpc"
	}

	switch protocol {
	case "http/protobuf":
		// HTTP/protobuf exporter for HTTPS endpoints
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(stripScheme(cfg.Endpoint)),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		} else if cfg.TLSSkipVerify {
			// Skip TLS verification for internal CAs
			opts = append(opts, otlptracehttp.WithTLSClientConfig(&tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested
			}))
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	default: // "grpc"
		// gRPC exporter (default)
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		} else if cfg.TLSSkipVerify {
			// Skip TLS verification for internal CAs
			opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested
			})))
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	}

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

	var exporter metric.Exporter
	var err error

	// Choose exporter based on protocol
	protocol := cfg.Protocol
	if protocol == "" {
		protocol = "grpc"
	}

	// Cumulative temporality selector - required for Prometheus-compatible backends
	// like VictoriaMetrics. This overrides OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE
	// environment variable which may be set by parent processes (e.g., Claude Code).
	cumulativeSelector := func(metric.InstrumentKind) metricdata.Temporality {
		return metricdata.CumulativeTemporality
	}

	switch protocol {
	case "http/protobuf":
		// HTTP/protobuf exporter for HTTPS endpoints
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(stripScheme(cfg.Endpoint)),
			otlpmetrichttp.WithTemporalitySelector(cumulativeSelector),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		} else if cfg.TLSSkipVerify {
			// Skip TLS verification for internal CAs
			opts = append(opts, otlpmetrichttp.WithTLSClientConfig(&tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested
			}))
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
	default: // "grpc"
		// gRPC exporter (default)
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
			otlpmetricgrpc.WithTemporalitySelector(cumulativeSelector),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		} else if cfg.TLSSkipVerify {
			// Skip TLS verification for internal CAs
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested
			})))
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	}

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

// stripScheme removes http:// or https:// from an endpoint URL.
// The OTEL HTTP exporters expect just host:port, not full URLs.
func stripScheme(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return endpoint
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
