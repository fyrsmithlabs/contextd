# Testing Specification

**Version**: 1.0.0
**Status**: Draft

---

## Overview

Three-layer testing approach for observability using in-memory exporters and structured assertions.

---

## Three-Layer Approach

| Layer | Tool | What It Tests |
|-------|------|---------------|
| **Unit** | In-memory exporter | Individual span/metric creation |
| **Integration** | ManualReader snapshots | Correct attributes after operations |
| **E2E** | Test collector container | Full pipeline: app -> collector -> backend |

---

## TestTelemetry Helper

### Structure

```go
// internal/telemetry/testing.go

type TestTelemetry struct {
    TraceExporter  *tracetest.InMemoryExporter
    MetricReader   *metric.ManualReader
    Logger         *observer.ObservedLogs
    TracerProvider trace.TracerProvider
    MeterProvider  metric.MeterProvider
}

func NewTestTelemetry(t *testing.T) *TestTelemetry {
    t.Helper()

    traceExporter := tracetest.NewInMemoryExporter()
    metricReader := metric.NewManualReader()
    core, logs := observer.New(zap.DebugLevel)

    tracerProvider := sdktrace.NewTracerProvider(
        sdktrace.WithSyncer(traceExporter),
    )
    meterProvider := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(metricReader),
    )

    t.Cleanup(func() {
        tracerProvider.Shutdown(context.Background())
        meterProvider.Shutdown(context.Background())
    })

    return &TestTelemetry{
        TraceExporter:  traceExporter,
        MetricReader:   metricReader,
        Logger:         logs,
        TracerProvider: tracerProvider,
        MeterProvider:  meterProvider,
    }
}
```

---

## Span Assertions

### AssertSpanExists

```go
func (tt *TestTelemetry) AssertSpanExists(t *testing.T, name string) tracetest.SpanStub {
    t.Helper()

    spans := tt.TraceExporter.GetSpans()
    for _, span := range spans {
        if span.Name == name {
            return span
        }
    }

    t.Fatalf("span %q not found in %d spans", name, len(spans))
    return tracetest.SpanStub{}
}
```

### AssertSpanHasAttribute

```go
func (tt *TestTelemetry) AssertSpanHasAttribute(t *testing.T, span tracetest.SpanStub, key string, expected any) {
    t.Helper()

    for _, attr := range span.Attributes {
        if string(attr.Key) == key {
            actual := attr.Value.AsInterface()
            if actual != expected {
                t.Errorf("attribute %q: got %v, want %v", key, actual, expected)
            }
            return
        }
    }

    t.Errorf("attribute %q not found on span %q", key, span.Name)
}
```

### AssertSpanHasParent

```go
func (tt *TestTelemetry) AssertSpanHasParent(t *testing.T, child, parent tracetest.SpanStub) {
    t.Helper()

    if child.Parent.SpanID() != parent.SpanContext.SpanID() {
        t.Errorf("span %q parent mismatch: got %s, want %s",
            child.Name,
            child.Parent.SpanID(),
            parent.SpanContext.SpanID(),
        )
    }
}
```

### AssertSpanHasError

```go
func (tt *TestTelemetry) AssertSpanHasError(t *testing.T, span tracetest.SpanStub) {
    t.Helper()

    if span.Status.Code != codes.Error {
        t.Errorf("span %q status: got %v, want Error", span.Name, span.Status.Code)
    }
}
```

---

## Metric Assertions

### AssertMetricValue

```go
func (tt *TestTelemetry) AssertMetricValue(t *testing.T, name string, expected float64, attrs ...attribute.KeyValue) {
    t.Helper()

    rm := metricdata.ResourceMetrics{}
    if err := tt.MetricReader.Collect(context.Background(), &rm); err != nil {
        t.Fatalf("collect metrics: %v", err)
    }

    for _, sm := range rm.ScopeMetrics {
        for _, m := range sm.Metrics {
            if m.Name == name {
                // Check data points based on type
                // ... implementation details
            }
        }
    }

    t.Errorf("metric %q not found", name)
}
```

### AssertCounterIncremented

```go
func (tt *TestTelemetry) AssertCounterIncremented(t *testing.T, name string, delta int64, attrs ...attribute.KeyValue) {
    t.Helper()
    // ... verify counter increased by delta
}
```

### AssertHistogramRecorded

```go
func (tt *TestTelemetry) AssertHistogramRecorded(t *testing.T, name string, attrs ...attribute.KeyValue) {
    t.Helper()
    // ... verify histogram has at least one data point
}
```

---

## Log Assertions

### AssertDebugSpanLog

```go
func (tt *TestTelemetry) AssertDebugSpanLog(t *testing.T, spanName string) {
    t.Helper()

    logs := tt.Logger.FilterMessage("span.start").All()
    for _, log := range logs {
        if log.ContextMap()["span.name"] == spanName {
            return
        }
    }

    t.Errorf("debug log for span %q not found", spanName)
}
```

---

## Unit Test Examples

### Test Span Creation

```go
func TestBash_CreatesSpanWithAttributes(t *testing.T) {
    tt := telemetry.NewTestTelemetry(t)
    svc := NewService(tt.TracerProvider, tt.MeterProvider, logger)

    ctx := WithSession(context.Background(), "sess_123")
    ctx = WithTenant(ctx, "acme", "platform", "contextd")

    _, err := svc.Bash(ctx, &BashRequest{Cmd: "echo hello", Timeout: 30})
    require.NoError(t, err)

    span := tt.AssertSpanExists(t, "contextd.SafeExec/Bash")
    tt.AssertSpanHasAttribute(t, span, "session.id", "sess_123")
    tt.AssertSpanHasAttribute(t, span, "tenant.org", "acme")
    tt.AssertSpanHasAttribute(t, span, "tool.name", "bash")
}
```

### Test Error Recording

```go
func TestBash_RecordsErrorSpan(t *testing.T) {
    tt := telemetry.NewTestTelemetry(t)
    svc := NewService(tt.TracerProvider, tt.MeterProvider, logger)

    ctx := context.Background()
    _, err := svc.Bash(ctx, &BashRequest{Cmd: "exit 1", Timeout: 30})
    require.Error(t, err)

    span := tt.AssertSpanExists(t, "contextd.SafeExec/Bash")
    tt.AssertSpanHasError(t, span)
    tt.AssertSpanHasAttribute(t, span, "exit_code", int64(1))
}
```

### Test Metrics Recording

```go
func TestBash_RecordsDuration(t *testing.T) {
    tt := telemetry.NewTestTelemetry(t)
    svc := NewService(tt.TracerProvider, tt.MeterProvider, logger)

    ctx := context.Background()
    _, err := svc.Bash(ctx, &BashRequest{Cmd: "sleep 0.1", Timeout: 30})
    require.NoError(t, err)

    tt.AssertHistogramRecorded(t, "contextd.tool.duration",
        attribute.String("tool", "bash"),
        attribute.String("status", "ok"),
    )
}
```

---

## Integration Test Pattern

```go
func TestService_TracesFullOperation(t *testing.T) {
    tt := telemetry.NewTestTelemetry(t)
    // Setup full service with real components

    // Execute operation

    // Verify trace hierarchy
    parent := tt.AssertSpanExists(t, "contextd.SafeExec/Bash")
    scrubInput := tt.AssertSpanExists(t, "scrubber.scan")
    executor := tt.AssertSpanExists(t, "executor.bash")
    scrubOutput := tt.AssertSpanExists(t, "scrubber.redact")

    tt.AssertSpanHasParent(t, scrubInput, parent)
    tt.AssertSpanHasParent(t, executor, parent)
    tt.AssertSpanHasParent(t, scrubOutput, parent)
}
```

---

## E2E Test Setup

### Test Collector Container

```yaml
# docker-compose.test.yaml
services:
  test-collector:
    image: otel/opentelemetry-collector:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./testdata/otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4317:4317"
```

### E2E Test Pattern

```go
func TestE2E_TelemetryPipeline(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }

    // Start test collector
    // Execute operations
    // Query test collector for exported data
    // Verify data arrived correctly
}
```

---

## References

- [OpenTelemetry Go Testing](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace/tracetest)
- [Zap Observer](https://pkg.go.dev/go.uber.org/zap/zaptest/observer)
