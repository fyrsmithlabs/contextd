# Package: telemetry

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides OpenTelemetry initialization and configuration for contextd. Sets up distributed tracing and metrics collection with OTLP exporters for observability.

## Specification

**Full Spec**: OpenTelemetry configuration is documented in [`pkg/CLAUDE.md`](../CLAUDE.md)

**Quick Summary**:
- **Problem**: Need observability for distributed system debugging and performance monitoring
- **Solution**: OpenTelemetry with traces and metrics via OTLP exporters
- **Key Features**:
  - OTLP/HTTP trace export (Jaeger, Grafana Tempo)
  - OTLP/HTTP metric export (Prometheus, VictoriaMetrics)
  - W3C Trace Context propagation
  - Resource attributes (service, version, environment)
  - Stdout exporters for testing (no external dependencies)

## Architecture

**Design Pattern**: Centralized telemetry initialization with graceful shutdown

**Dependencies**:
- `go.opentelemetry.io/otel` - OpenTelemetry SDK core
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` - OTLP trace export
- `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp` - OTLP metric export
- `go.opentelemetry.io/otel/exporters/stdout/*` - Stdout exporters for testing
- `go.opentelemetry.io/otel/sdk` - SDK components (trace, metric, resource)

**Used By**:
- `cmd/contextd` - Server initialization
- All packages - Tracing and metrics collection

## Key Components

### Main Types

```go
type Config struct {
    ServiceName    string  // Service identifier (e.g., "contextd")
    ServiceVersion string  // Semantic version (e.g., "1.0.0")
    Environment    string  // Deployment environment (dev/staging/prod)
    Endpoint       string  // OTLP collector endpoint (empty = stdout)
}
```

### Main Functions

```go
// Initialize OpenTelemetry with tracing and metrics
// Returns shutdown function that MUST be deferred
func Initialize(ctx context.Context, cfg Config) (func(context.Context) error, error)

// setupTraceProvider creates and configures the trace provider
func setupTraceProvider(ctx context.Context, res *resource.Resource, endpoint string) (*trace.TracerProvider, error)

// setupMeterProvider creates and configures the meter provider
func setupMeterProvider(ctx context.Context, res *resource.Resource, endpoint string) (*metric.MeterProvider, error)
```

## Usage Example

```go
import "github.com/axyzlabs/contextd/pkg/telemetry"

// Initialize telemetry at application startup
cfg := telemetry.Config{
    ServiceName:    "contextd",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    Endpoint:       "http://localhost:4318", // OTLP collector
}

shutdown, err := telemetry.Initialize(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer shutdown(context.Background()) // CRITICAL: Always defer

// Telemetry is now globally available
tracer := otel.Tracer("my-component")
ctx, span := tracer.Start(ctx, "operation")
defer span.End()

// Record metrics
meter := otel.Meter("my-component")
counter, _ := meter.Int64Counter("requests")
counter.Add(ctx, 1)
```

## Testing

**Test Coverage**: 85% (Target: ≥80%)

**Key Test Files**:
- `telemetry_test.go` - Initialization, config validation, error handling

**Test Scenarios**:
- Valid configs with stdout exporters
- Empty service name/version/environment
- Invalid endpoints (errors)
- Context timeout during initialization
- Resource creation with attributes
- Stdout exporters (no external dependencies)
- Double shutdown (idempotent)
- Shutdown with cancelled context

**Running Tests**:
```bash
go test ./pkg/telemetry/
go test -cover ./pkg/telemetry/
go test -race ./pkg/telemetry/
```

**Integration Testing**:
Tests use stdout exporters by default (no external dependencies). For full integration testing with OTLP collector:

```bash
# Start monitoring stack
docker-compose up -d

# Test with real OTLP endpoint
go test ./pkg/telemetry/ -endpoint http://localhost:4318
```

## Configuration

**Environment Variables**:
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP collector endpoint (default: `http://localhost:4318`)
- `OTEL_SERVICE_NAME` - Service name for traces (default: `contextd`)
- `OTEL_ENVIRONMENT` - Environment name (default: `development`)

**Trace Export Configuration**:
- Protocol: OTLP/HTTP with gzip compression
- Endpoint: `{ENDPOINT}/v1/traces`
- Batch timeout: 5 seconds
- Max batch size: 512 spans
- Max queue size: 2048 spans (default)

**Metric Export Configuration**:
- Protocol: OTLP/HTTP with gzip compression
- Endpoint: `{ENDPOINT}/v1/metrics`
- Export interval: 60 seconds
- Aggregation: Cumulative temporality

**Propagation**:
- W3C Trace Context (standard HTTP headers)
- W3C Baggage (context propagation)

**Stdout Exporters** (Testing):
When `Endpoint` is empty string, uses stdout exporters:
- Pretty-printed JSON output
- No network dependencies
- Useful for local development and CI

## Security Considerations

**CRITICAL Security Requirements**:

1. **Endpoint Security**:
   - Use HTTPS for production OTLP endpoints
   - Authenticate to telemetry collectors (API keys, mTLS)
   - Don't expose collector publicly (use internal network)
   - Validate endpoint URLs to prevent SSRF attacks

2. **Data Sanitization**:
   - Use `security.Redact()` for sensitive span attributes
   - Don't include credentials in traces or metrics
   - Redact PII from span names and attributes
   - Filter sensitive metadata before export

3. **Resource Limits**:
   - Batch size limits prevent memory exhaustion
   - Queue limits prevent unbounded growth
   - Timeouts prevent hanging during export
   - Monitor exporter backpressure

4. **Context Propagation**:
   - W3C Trace Context is safe to propagate
   - Be careful with baggage (don't include secrets)
   - Validate trace IDs from untrusted sources

## Performance Notes

- **Initialization time**: ~100ms (one-time cost at startup)
- **Trace overhead**: ~10μs per span (batched export)
- **Metric overhead**: ~1μs per measurement
- **Memory**: ~10MB for trace buffers
- **Shutdown time**: Up to 5s (flushes pending data)

**Optimization Tips**:
- Batch spans reduce export overhead (5s timeout)
- Use sampling for high-volume traces (not implemented yet)
- Aggregate metrics reduce cardinality
- Gzip compression reduces network bandwidth

**CRITICAL**: ALWAYS defer shutdown to flush pending traces/metrics before exit. Failure to shutdown gracefully will lose recent telemetry data.

**Shutdown Example**:
```go
shutdown, err := telemetry.Initialize(ctx, cfg)
if err != nil {
    return err
}

// ALWAYS defer with fresh context (not the request context)
defer func() {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := shutdown(shutdownCtx); err != nil {
        log.Printf("telemetry shutdown error: %v", err)
    }
}()
```

## Common Pitfalls

1. **Forgetting to defer shutdown**: Loses recent traces/metrics on exit
2. **Using request context for shutdown**: Context may be cancelled
3. **Not checking Initialize errors**: Silent telemetry failure
4. **Hardcoding endpoints**: Use environment variables for flexibility
5. **Including secrets in spans**: Always use security.Redact()
6. **Excessive span creation**: Only trace significant operations
7. **High cardinality labels**: Causes memory/storage issues

## Related Documentation

- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Project Root: [`CLAUDE.md`](../../CLAUDE.md)
- Monitoring Setup: [`docs/guides/MONITORING-SETUP.md`](../../docs/guides/MONITORING-SETUP.md)
- Metrics Package: [`pkg/metrics/CLAUDE.md`](../metrics/CLAUDE.md)
- Security Package: [`pkg/security/CLAUDE.md`](../security/CLAUDE.md)
