# Observability Architecture

**Version**: 1.0.0
**Status**: Draft

---

## Overview

Full-stack observability using OpenTelemetry with flexible OTLP export to any compatible backend.

**Current Architecture (Post-v2 Simplification)**:
- MCP server over stdio (no gRPC)
- Direct service calls within the process
- OTLP export for traces and metrics
- Structured logging with Zap

```
┌─────────────────────────────────────────────────────────────┐
│                    contextd MCP Server                      │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Internal Services                                   │   │
│  │  • reasoningbank (memory)                            │   │
│  │  • remediation (error patterns)                      │   │
│  │  • checkpoint (context snapshots)                    │   │
│  │  • repository (semantic search)                      │   │
│  │  • troubleshoot (diagnostics)                        │   │
│  │  • folding (context management)                      │   │
│  │                                                       │   │
│  │  Each service:                                       │   │
│  │  • Creates spans (otel.Tracer)                       │   │
│  │  • Records metrics (otel.Meter)                      │   │
│  │  • Logs with Zap                                     │   │
│  └──────────────────────────────────────────────────────┘   │
│                           │                                 │
│                     OTLP Export                             │
│              (gRPC :4317 or HTTP :4318)                     │
└───────────────────────────┼─────────────────────────────────┘
                            ▼
                    ┌───────────────┐
                    │ OTEL Collector│ (optional)
                    └───────┬───────┘
             ┌──────────────┼──────────────┐
             ▼              ▼              ▼
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │  Metrics   │  │   Logs     │  │   Traces   │
    │  Backend   │  │  Backend   │  │  Backend   │
    └─────┬──────┘  └─────┬──────┘  └─────┬──────┘
          └───────────────┼───────────────┘
                          ▼
                    ┌───────────┐
                    │Visualization│
                    └───────────┘
```

---

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Instrumentation** | OpenTelemetry Go SDK | Industry standard, vendor-agnostic |
| **Transport** | MCP over stdio | Simplified architecture, no gRPC overhead |
| **Export** | OTLP (gRPC/HTTP) | Flexible backend support, standard protocol |
| **Collector** | OTEL Collector (optional) | Decouples app from backends, sampling control |
| **Backend Examples** | VictoriaMetrics, Prometheus, Jaeger | User choice, OTLP-compatible |
| **Visualization** | Grafana (typical) | Universal dashboards, correlation |
| **Logging** | Zap | Structured, fast, OTEL bridge support |
| **Config** | Koanf | Lightweight, supports file + env + flags |

---

## Package Structure

```
internal/
├── config/
│   ├── config.go           # Root config struct, Koanf setup
│   └── duration.go         # Duration type for config parsing
├── telemetry/
│   ├── telemetry.go        # Main Telemetry type with providers
│   ├── config.go           # Telemetry configuration
│   ├── provider.go         # TracerProvider, MeterProvider setup
│   └── doc.go              # Package documentation
├── logging/
│   ├── logger.go           # Zap logger wrapper
│   ├── config.go           # Logging configuration
│   ├── context.go          # Context-aware logging helpers
│   ├── otel.go             # OTEL bridge integration
│   └── redact.go           # Secret redaction for logs
└── [services]/             # Each service instruments itself
    └── service.go          # Uses otel.Tracer() and otel.Meter()
```

---

## Configuration

```yaml
observability:
  service_name: "contextd"
  enable_telemetry: false        # Disabled by default
  otlp_endpoint: "localhost:4317"
  otlp_protocol: "grpc"          # "grpc" or "http/protobuf"
  otlp_insecure: true            # For localhost, use false for remote
  otlp_tls_skip_verify: false    # For internal CAs

# Telemetry package config (internal/telemetry)
telemetry:
  enabled: false                 # Synced with observability.enable_telemetry
  endpoint: "localhost:4317"
  protocol: "grpc"
  insecure: true
  service_name: "contextd"
  service_version: "0.1.0"

  sampling:
    rate: 1.0                    # 100% in dev, lower in prod
    always_on_errors: true       # Always keep error traces

  metrics:
    enabled: true
    export_interval: "15s"

  shutdown:
    timeout: "5s"

# Logging config (internal/logging)
logging:
  level: "info"                  # trace, debug, info, warn, error
  format: "json"                 # json, console
  caller:
    enabled: true
    skip: 0
  stacktrace:
    level: "error"               # Level at which to add stacktraces
```

### Environment Overrides

```bash
# Main observability config
CONTEXTD_OBSERVABILITY_ENABLE_TELEMETRY=true
CONTEXTD_OBSERVABILITY_OTLP_ENDPOINT=collector.prod:4317
CONTEXTD_OBSERVABILITY_OTLP_PROTOCOL=grpc

# Telemetry-specific overrides
CONTEXTD_TELEMETRY_ENABLED=true
CONTEXTD_TELEMETRY_ENDPOINT=collector.prod:4317
CONTEXTD_TELEMETRY_SAMPLING_RATE=0.1

# Logging
CONTEXTD_LOGGING_LEVEL=warn
CONTEXTD_LOGGING_FORMAT=json
```

---

## Service Instrumentation

### Pattern for Internal Services

Each service initializes its own tracer and meter from the global OTEL registry:

```go
package myservice

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/fyrsmithlabs/contextd/internal/myservice"

type Service struct {
    tracer trace.Tracer
    meter  metric.Meter
    // ...
}

func NewService(...) (*Service, error) {
    return &Service{
        tracer: otel.Tracer(instrumentationName),
        meter:  otel.Meter(instrumentationName),
    }, nil
}

func (s *Service) SomeOperation(ctx context.Context, ...) error {
    ctx, span := s.tracer.Start(ctx, "myservice.operation")
    defer span.End()

    // Record metrics
    s.counter.Add(ctx, 1)

    // ... business logic ...

    return nil
}
```

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
| MCP over stdio (no gRPC) | Simplified architecture, eliminates RPC overhead |
| OTLP export with protocol flexibility | Backend-agnostic, supports gRPC and HTTP/protobuf |
| OTEL Collector optional | Users can export directly to backends or use collector for flexibility |
| Telemetry disabled by default | Non-intrusive for users without OTLP infrastructure |
| Per-service instrumentation | Each service owns its tracer/meter, clean separation |
| Zap with OTEL bridge | Performance, structured logging, optional OTEL integration |
| Koanf over viper | Lighter weight, cleaner API |

---

## References

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [otelgrpc Package](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc)
- [VictoriaMetrics](https://docs.victoriametrics.com/)
