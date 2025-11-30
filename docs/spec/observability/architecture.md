# Observability Architecture

**Version**: 1.0.0
**Status**: Draft

---

## Overview

Full-stack observability using OpenTelemetry with unified VictoriaMetrics backend.

```
┌─────────────────────────────────────────────────────────────┐
│                        contextd                             │
│                           │                                 │
│                     OTLP (4317)                             │
│                           ▼                                 │
│                   ┌───────────────┐                         │
│                   │ OTEL Collector│                         │
│                   └───────┬───────┘                         │
│            ┌──────────────┼──────────────┐                  │
│            ▼              ▼              ▼                  │
│   ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│   │ Victoria   │  │ Victoria   │  │ Victoria   │           │
│   │ Metrics    │  │ Logs       │  │ Traces     │           │
│   └─────┬──────┘  └─────┬──────┘  └─────┬──────┘           │
│         └───────────────┼───────────────┘                   │
│                         ▼                                   │
│                   ┌───────────┐                             │
│                   │  Grafana  │                             │
│                   └───────────┘                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Instrumentation** | OpenTelemetry Go SDK | Industry standard, vendor-agnostic |
| **Collector** | OTEL Collector | Decouples app from backends |
| **Metrics** | VictoriaMetrics | Prometheus-compatible, efficient |
| **Logs** | VictoriaLogs | High cardinality support, fast queries |
| **Traces** | VictoriaTraces | Unified ecosystem, low resource usage |
| **Visualization** | Grafana | Universal dashboards, correlation |
| **Logging** | Zap | Structured, fast, trace ID injection |
| **Config** | Koanf | Lightweight, supports file + env + flags |

---

## Package Structure

```
internal/
├── config/
│   ├── config.go           # Root config struct, Koanf setup
│   ├── telemetry.go        # TelemetryConfig sub-struct
│   └── validation.go       # Startup validation
└── telemetry/
    ├── provider.go         # TracerProvider, MeterProvider setup
    ├── attributes.go       # Standard attribute builders
    ├── metrics.go          # Metric definitions
    ├── middleware.go       # gRPC stats handlers
    └── testing.go          # In-memory exporter, assertions
```

---

## Configuration

```yaml
telemetry:
  enabled: true
  endpoint: "localhost:4317"
  service_name: "contextd"

  sampling:
    rate: 1.0                    # 100% in dev, lower in prod
    always_on_errors: true       # Always keep error traces

  logging:
    level: "info"                # debug, info, warn, error
    format: "json"               # json, console
    trace_debug: false           # Log span lifecycle at debug level

  metrics:
    enabled: true
    export_interval: "15s"

  experience_metrics:
    enabled: false               # Opt-in user experience tracking
    retention_days: 30
```

### Environment Overrides

```bash
CONTEXTD_TELEMETRY_ENDPOINT=collector.prod:4317
CONTEXTD_TELEMETRY_SAMPLING_RATE=0.1
CONTEXTD_TELEMETRY_LOGGING_LEVEL=warn
```

---

## gRPC Integration

### Stats Handlers (Recommended)

```go
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

// Server
grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))

// Client
grpc.NewClient(addr, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
```

Note: Interceptors are deprecated in favor of stats handlers.

---

## Graceful Shutdown

### Shutdown Order

1. Stop accepting new requests (gRPC server)
2. Wait for in-flight requests (with timeout)
3. Flush telemetry providers:
   - TracerProvider.Shutdown()
   - MeterProvider.Shutdown()
   - Logger.Sync()
4. Close Qdrant connections
5. Exit

### Implementation Pattern

```go
func (t *Telemetry) Shutdown(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, t.shutdownTimeout)
    defer cancel()

    var errs []error

    if err := t.tracerProvider.Shutdown(ctx); err != nil {
        errs = append(errs, fmt.Errorf("trace provider: %w", err))
    }

    if err := t.meterProvider.Shutdown(ctx); err != nil {
        errs = append(errs, fmt.Errorf("meter provider: %w", err))
    }

    if err := t.logger.Sync(); err != nil && !isStdoutSyncError(err) {
        errs = append(errs, fmt.Errorf("logger sync: %w", err))
    }

    return errors.Join(errs...)
}
```

---

## Error Handling

### Principle

Telemetry failures MUST NOT crash the application.

### Patterns

```go
// Span creation - return no-op if disabled
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
    if t == nil || t.tracer == nil {
        return ctx, trace.SpanFromContext(ctx)
    }
    return t.tracer.Start(ctx, name)
}

// Metric recording - swallow errors
func (m *Metrics) RecordDuration(ctx context.Context, name string, d time.Duration) {
    if m == nil {
        return
    }
    // Record, log errors at debug level only
}
```

### Degraded Mode

```go
type Telemetry struct {
    healthy  atomic.Bool
    degraded atomic.Bool
}

func (t *Telemetry) Health() HealthStatus {
    return HealthStatus{
        Healthy:  t.healthy.Load(),
        Degraded: t.degraded.Load(),
    }
}
```

---

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| VictoriaMetrics unified stack | Single ecosystem, lower operational complexity |
| OTEL Collector middleware | Backend flexibility, sampling control |
| Stats handlers over interceptors | Modern approach, better integration |
| Zap over slog | Performance, trace ID injection support |
| Koanf over viper | Lighter weight, cleaner API |

---

## References

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [otelgrpc Package](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc)
- [VictoriaMetrics](https://docs.victoriametrics.com/)
