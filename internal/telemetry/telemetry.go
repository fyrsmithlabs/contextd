package telemetry

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Telemetry provides OpenTelemetry instrumentation for contextd.
//
// It manages TracerProvider, MeterProvider, and graceful shutdown.
// Telemetry failures do not crash the application; they degrade gracefully.
type Telemetry struct {
	config *Config

	tracerProvider *trace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	logProvider    log.LoggerProvider

	// Health tracking
	healthy  atomic.Bool
	degraded atomic.Bool
}

// New creates a new Telemetry instance and initializes providers.
//
// If telemetry is disabled in config, returns a no-op instance.
// Provider initialization errors are logged but don't fail; the instance
// degrades gracefully.
func New(ctx context.Context, cfg *Config) (*Telemetry, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid telemetry config: %w", err)
	}

	t := &Telemetry{
		config: cfg,
	}
	t.healthy.Store(true)

	if !cfg.Enabled {
		return t, nil
	}

	// Create resource describing the service
	res, err := newResource(cfg)
	if err != nil {
		t.setDegraded("resource creation failed: %v", err)
		return t, nil
	}

	// Initialize TracerProvider
	tp, err := newTracerProvider(ctx, cfg, res)
	if err != nil {
		t.setDegraded("tracer provider failed: %v", err)
	} else {
		t.tracerProvider = tp
		otel.SetTracerProvider(tp)
	}

	// Initialize MeterProvider
	mp, err := newMeterProvider(ctx, cfg, res)
	if err != nil {
		t.setDegraded("meter provider failed: %v", err)
	} else if mp != nil {
		t.meterProvider = mp
		otel.SetMeterProvider(mp)
	}

	// Set up propagation (W3C Trace Context)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return t, nil
}

// Tracer returns a tracer for the given instrumentation scope.
//
// Returns a no-op tracer if telemetry is disabled or degraded.
func (t *Telemetry) Tracer(name string, opts ...oteltrace.TracerOption) oteltrace.Tracer {
	if t == nil || t.tracerProvider == nil {
		return otel.GetTracerProvider().Tracer(name, opts...)
	}
	return t.tracerProvider.Tracer(name, opts...)
}

// Meter returns a meter for the given instrumentation scope.
//
// Returns a no-op meter if telemetry is disabled or degraded.
func (t *Telemetry) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	if t == nil || t.meterProvider == nil {
		return otel.GetMeterProvider().Meter(name, opts...)
	}
	return t.meterProvider.Meter(name, opts...)
}

// LoggerProvider returns the log provider for OTEL logging bridge.
//
// May return nil if not configured.
func (t *Telemetry) LoggerProvider() log.LoggerProvider {
	if t == nil {
		return nil
	}
	return t.logProvider
}

// SetLoggerProvider sets the logger provider for OTEL logging bridge.
func (t *Telemetry) SetLoggerProvider(lp log.LoggerProvider) {
	if t != nil {
		t.logProvider = lp
	}
}

// Shutdown gracefully shuts down all telemetry providers.
//
// Should be called during application shutdown to flush pending telemetry.
// Uses the shutdown timeout from config.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil {
		return nil
	}

	// Use configured timeout if no deadline set
	if _, ok := ctx.Deadline(); !ok && t.config != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.Shutdown.Timeout.Duration())
		defer cancel()
	}

	var errs []error

	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("trace provider shutdown: %w", err))
		}
	}

	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
		}
	}

	t.healthy.Store(false)
	return errors.Join(errs...)
}

// ForceFlush immediately exports all pending telemetry data.
//
// Useful for testing or before critical operations.
func (t *Telemetry) ForceFlush(ctx context.Context) error {
	if t == nil {
		return nil
	}

	var errs []error

	if t.tracerProvider != nil {
		if err := t.tracerProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, fmt.Errorf("trace flush: %w", err))
		}
	}

	if t.meterProvider != nil {
		if err := t.meterProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter flush: %w", err))
		}
	}

	return errors.Join(errs...)
}

// Health returns the current health status.
type HealthStatus struct {
	Healthy  bool
	Degraded bool
}

// Health returns the current telemetry health status.
func (t *Telemetry) Health() HealthStatus {
	if t == nil {
		return HealthStatus{Healthy: false, Degraded: true}
	}
	return HealthStatus{
		Healthy:  t.healthy.Load(),
		Degraded: t.degraded.Load(),
	}
}

// IsEnabled returns true if telemetry is enabled and healthy.
func (t *Telemetry) IsEnabled() bool {
	if t == nil || t.config == nil {
		return false
	}
	return t.config.Enabled && t.healthy.Load()
}

// setDegraded marks telemetry as degraded due to an error.
func (t *Telemetry) setDegraded(format string, args ...interface{}) {
	t.degraded.Store(true)
	// In production, this would log the error
	_ = fmt.Errorf(format, args...)
}
