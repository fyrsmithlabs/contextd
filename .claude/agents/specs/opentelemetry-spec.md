# OpenTelemetry Specification

## Official References

- **OTel Docs**: https://opentelemetry.io/docs/
- **Go SDK**: https://pkg.go.dev/go.opentelemetry.io/otel
- **Semantic Conventions**: https://opentelemetry.io/docs/specs/semconv/
- **Best Practices**: https://opentelemetry.io/docs/specs/otel/trace/sdk/

## Core Concepts

### Signals
- **Traces**: Request flow through system
- **Metrics**: Numerical measurements over time
- **Logs**: Timestamped text records (future in Go SDK)

### Trace Hierarchy
```
Trace (request lifecycle)
└── Span (operation)
    ├── Attributes (metadata)
    ├── Events (point-in-time logs)
    ├── Links (cross-trace relationships)
    └── Status (error state)
```

## Go SDK Patterns

### Initialization

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initTracer() (*trace.TracerProvider, error) {
    // Create resource
    res, err := resource.New(
        context.Background(),
        resource.WithAttributes(
            semconv.ServiceName("contextd"),
            semconv.ServiceVersion("1.0.0"),
            semconv.DeploymentEnvironment("production"),
        ),
    )

    // Create exporter
    exporter, err := otlptracehttp.New(
        context.Background(),
        otlptracehttp.WithEndpoint("otel.example.com"),
        otlptracehttp.WithInsecure(),
    )

    // Create tracer provider
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter,
            trace.WithBatchTimeout(5*time.Second),
            trace.WithMaxExportBatchSize(512),
        ),
        trace.WithResource(res),
        trace.WithSampler(trace.ParentBased(
            trace.TraceIDRatioBased(0.1),  // Sample 10%
        )),
    )

    // Set global provider
    otel.SetTracerProvider(tp)

    // Set propagator for context
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return tp, nil
}
```

### Creating Spans

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("contextd")

func ProcessCheckpoint(ctx context.Context, cp Checkpoint) error {
    // Start span
    ctx, span := tracer.Start(ctx, "checkpoint.process")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("checkpoint.id", cp.ID),
        attribute.String("project", cp.Project),
        attribute.Int("size_bytes", len(cp.Content)),
    )

    // Process checkpoint
    err := processInternal(ctx, cp)
    if err != nil {
        // Record error
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    // Success
    span.SetStatus(codes.Ok, "")
    return nil
}
```

### Nested Spans

```go
func SearchCheckpoints(ctx context.Context, query string) ([]Checkpoint, error) {
    ctx, span := tracer.Start(ctx, "checkpoint.search")
    defer span.End()

    // Child span for embedding
    embedding, err := generateEmbedding(ctx, query)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Int("results.count", len(results)),
    )

    return results, nil
}

func generateEmbedding(ctx context.Context, text string) ([]float32, error) {
    // Child span automatically linked
    ctx, span := tracer.Start(ctx, "embedding.generate")
    defer span.End()

    span.SetAttributes(
        attribute.String("provider", "openai"),
        attribute.Int("input.length", len(text)),
    )

    // Generate embedding
    emb, err := openaiClient.CreateEmbedding(ctx, text)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Int("output.dimension", len(emb)),
    )

    return emb, nil
}
```

## Semantic Conventions

### Service Attributes (Required)

```go
semconv.ServiceName("contextd")           // Service identifier
semconv.ServiceVersion("1.0.0")           // Version
semconv.DeploymentEnvironment("prod")     // Environment
semconv.ServiceInstanceID("instance-1")   // Instance ID
```

### HTTP Attributes

```go
// Server spans
semconv.HTTPMethod("POST")
semconv.HTTPRoute("/api/v1/search")
semconv.HTTPStatusCode(200)
semconv.HTTPRequestContentLength(1024)
semconv.HTTPResponseContentLength(2048)

// Client spans
semconv.HTTPUrl("https://api.openai.com/v1/embeddings")
semconv.NetPeerName("api.openai.com")
semconv.NetPeerPort(443)
```

### Database Attributes

```go
semconv.DBName("default")
semconv.DBOperation("search")
semconv.DBStatement("embedding search with filter")  // No PII!
```

### Custom Attributes

```go
// Use namespaced keys
attribute.String("contextd.operation", "checkpoint.create")
attribute.String("contextd.project", "myproject")
attribute.Int("contextd.batch_size", 100)
```

## Sampling Strategies

### Head-Based Sampling (Before Processing)

```go
// Always sample errors
sampler := trace.ParentBased(
    trace.AlwaysSample(),  // Root spans
)

// Probability-based
sampler := trace.ParentBased(
    trace.TraceIDRatioBased(0.1),  // 10% of traces
)
```

### Tail-Based Sampling (After Processing)

Configure in collector:
```yaml
processors:
  tail_sampling:
    decision_wait: 10s
    num_traces: 100
    policies:
      # Always keep errors
      - name: errors
        type: status_code
        status_code: {status_codes: [ERROR]}

      # Keep slow requests
      - name: slow
        type: latency
        latency: {threshold_ms: 500}

      # Sample fast requests
      - name: fast
        type: probabilistic
        probabilistic: {sampling_percentage: 1}
```

## Performance Optimization

### Minimize Allocations

```go
// ❌ BAD: Creates new slice
span.SetAttributes(
    attribute.String("key1", val1),
    attribute.String("key2", val2),
)

// ✅ GOOD: Reuse slice
attrs := []attribute.KeyValue{
    attribute.String("key1", val1),
    attribute.String("key2", val2),
}
span.SetAttributes(attrs...)
```

### Batch Export

```go
trace.WithBatchTimeout(5*time.Second)     // Export every 5s
trace.WithMaxExportBatchSize(512)         // Or when batch reaches 512
trace.WithMaxQueueSize(2048)              // Queue up to 2048 spans
```

### Conditional Instrumentation

```go
func hotPath(ctx context.Context) {
    // Skip instrumentation in production hot paths
    if shouldTrace(ctx) {
        ctx, span := tracer.Start(ctx, "hot.path")
        defer span.End()
    }

    // ... operation
}

func shouldTrace(ctx context.Context) bool {
    // Only trace if already in active trace
    return trace.SpanFromContext(ctx).SpanContext().IsValid()
}
```

## Error Handling

### Recording Errors

```go
if err != nil {
    // Record error with stack trace
    span.RecordError(err,
        trace.WithAttributes(
            attribute.String("error.type", fmt.Sprintf("%T", err)),
        ),
        trace.WithStackTrace(true),
    )

    // Set span status
    span.SetStatus(codes.Error, err.Error())

    return err
}
```

### Error Types

```go
// Classify errors
func classifyError(err error) codes.Code {
    if errors.Is(err, context.Canceled) {
        return codes.Canceled
    }
    if errors.Is(err, context.DeadlineExceeded) {
        return codes.DeadlineExceeded
    }
    // ... more classifications
    return codes.Error
}

span.SetStatus(classifyError(err), err.Error())
```

## Context Propagation

### HTTP Server (Echo)

```go
import (
    "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

// Middleware automatically propagates context
e.Use(otelecho.Middleware("contextd"))
```

### HTTP Client

```go
import (
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Wrap client transport
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}

// Context automatically propagated in requests
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := client.Do(req)
```

## Common Pitfalls

### ❌ Don't: High Cardinality Attributes

```go
// ❌ BAD: User IDs (millions of values)
span.SetAttributes(
    attribute.String("user.id", userID),
)

// ❌ BAD: Timestamps (infinite values)
span.SetAttributes(
    attribute.String("timestamp", time.Now().String()),
)

// ✅ GOOD: Bucketed values
span.SetAttributes(
    attribute.String("user.tier", getUserTier(userID)),  // "free", "pro", "enterprise"
    attribute.Int("hour_of_day", time.Now().Hour()),     // 0-23
)
```

### ❌ Don't: Include PII

```go
// ❌ BAD: Personal information
span.SetAttributes(
    attribute.String("user.email", email),
    attribute.String("credit.card", cardNumber),
)

// ✅ GOOD: Anonymized data
span.SetAttributes(
    attribute.String("user.id_hash", hashUserID(email)),
    attribute.Bool("payment.succeeded", true),
)
```

### ❌ Don't: Forget to End Spans

```go
// ❌ BAD: Span never ends
ctx, span := tracer.Start(ctx, "operation")
if err != nil {
    return err  // Span leaked!
}
span.End()

// ✅ GOOD: Always defer
ctx, span := tracer.Start(ctx, "operation")
defer span.End()
```

## Troubleshooting

### "spans not appearing in Jaeger"
```
Check:
1. Exporter endpoint configured correctly
2. Tracer provider set globally (otel.SetTracerProvider)
3. Spans actually created (tracer.Start called)
4. Sampling allows spans through
5. Batch export timeout not too long
```

### "context not propagating"
```
Check:
1. Context passed to child functions
2. Propagator configured (otel.SetTextMapPropagator)
3. HTTP middleware installed (otelecho, otelhttp)
4. Parent context not nil
```

### "high memory usage from tracing"
```
Fix:
1. Reduce sampling rate
2. Decrease batch size
3. Lower queue size
4. Remove high-frequency spans
5. Reduce attribute count
```

## Best Practices

### ✅ DO

1. **Always defer span.End()**
2. **Pass context as first parameter**
3. **Use semantic conventions** for standard attributes
4. **Record errors** with RecordError()
5. **Set span status** explicitly
6. **Keep attribute cardinality low** (<100 unique values)
7. **Sample intelligently** (errors always, fast requests rarely)
8. **Use batch export** (not sync)
9. **Propagate context** through call chain
10. **Monitor overhead** (<5% CPU/memory)

### ❌ DON'T

1. **Include PII** in attributes
2. **Create unbounded cardinality**
3. **Forget to end spans**
4. **Block on export** (use async batch)
5. **Instrument every function** (overhead)
6. **Hardcode service names** (use config)
7. **Ignore sampling** in production
8. **Log full payloads** to spans

---

**Reference**: OpenTelemetry Go SDK - https://opentelemetry.io/docs/instrumentation/go/
