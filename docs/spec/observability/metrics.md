# Metrics Specification

**Version**: 1.0.0
**Status**: Draft

---

## Overview

Metric instrumentation for contextd using OpenTelemetry meter primitives with standardized naming and histogram buckets.

---

## Metric Instruments

### Request Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `contextd.request.duration` | Histogram | ms | gRPC request latency |
| `contextd.request.total` | Counter | 1 | Total requests by method, status |

### Tool Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `contextd.tool.duration` | Histogram | ms | Tool execution time |
| `contextd.tool.total` | Counter | 1 | Tool invocations by type |
| `contextd.tool.errors` | Counter | 1 | Tool errors by type |

### Qdrant Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `contextd.qdrant.duration` | Histogram | ms | Qdrant operation latency |
| `contextd.qdrant.points` | Counter | 1 | Points read/written |
| `contextd.qdrant.errors` | Counter | 1 | Qdrant errors |

### Session Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `contextd.session.active` | UpDownCounter | 1 | Current active sessions |
| `contextd.session.duration` | Histogram | s | Session duration |

### Memory Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `contextd.memory.search.duration` | Histogram | ms | Memory search latency |
| `contextd.memory.search.results` | Histogram | 1 | Results per search |

### Checkpoint Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `contextd.checkpoint.save.total` | Counter | 1 | Checkpoints saved |
| `contextd.checkpoint.size.bytes` | Histogram | bytes | Checkpoint sizes |

---

## Scrubber Metrics

### Core Scrubber Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `contextd.scrubber.duration` | Histogram | ms | Scrubbing latency |
| `contextd.scrubber.secrets` | Counter | 1 | Secrets detected by type |
| `contextd.scrubber.bytes` | Counter | 1 | Bytes scanned |

### Scrubber Quality Metrics

| Metric | Type | Attributes |
|--------|------|------------|
| `contextd.scrubber.secrets` | Counter | `rule_id`, `secret_type`, `confidence_band` |
| `contextd.scrubber.false_positives` | Counter | `rule_id`, `context` |
| `contextd.scrubber.false_negatives` | Counter | `rule_id`, `context` |
| `contextd.scrubber.user_reports` | Counter | `type` (missed, false_positive, helpful) |

### Confidence Bands

| Band | Range | Description |
|------|-------|-------------|
| `high` | 0.9 - 1.0 | High confidence detection |
| `medium` | 0.7 - 0.9 | Medium confidence |
| `low` | 0.5 - 0.7 | Low confidence |

---

## Histogram Buckets

### Fast Operations (1ms - 100ms)

Used for: scrubber, simple reads, cache lookups

```go
fastBuckets = []float64{0.5, 1, 2, 5, 10, 25, 50, 100}
```

### Standard Operations (1ms - 5s)

Used for: gRPC requests, Qdrant queries

```go
standardBuckets = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000}
```

### Slow Operations (10ms - 60s)

Used for: tool execution (bash, read, write)

```go
slowBuckets = []float64{10, 50, 100, 500, 1000, 5000, 10000, 30000, 60000}
```

### Size Buckets (bytes)

Used for: checkpoint sizes, response sizes

```go
sizeBuckets = []float64{256, 1024, 4096, 16384, 65536, 262144, 1048576}
```

---

## Attribute Constants

### Status Values

```go
StatusOK       = "ok"
StatusError    = "error"
StatusTimeout  = "timeout"
StatusCanceled = "canceled"
```

### Tool Names

```go
ToolBash  = "bash"
ToolRead  = "read"
ToolWrite = "write"
```

### Secret Types (Generic, No Actual Secrets)

```go
SecretTypeAPIKey     = "api_key"
SecretTypePassword   = "password"
SecretTypePrivateKey = "private_key"
SecretTypeToken      = "token"
SecretTypeGeneric    = "generic"
```

---

## Recording Patterns

### Duration Recording

```go
func RecordRequestDuration(ctx context.Context, method string, status string, duration time.Duration) {
    requestDuration.Record(ctx, float64(duration.Milliseconds()),
        metric.WithAttributes(
            attribute.String("method", method),
            attribute.String("status", status),
        ),
    )
}
```

### Counter Increment

```go
func IncrementToolTotal(ctx context.Context, tool string, status string) {
    toolTotal.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("tool", tool),
            attribute.String("status", status),
        ),
    )
}
```

### UpDownCounter Pattern

```go
func (s *SessionManager) Start(ctx context.Context) {
    activeSessions.Add(ctx, 1)
}

func (s *SessionManager) End(ctx context.Context, duration time.Duration) {
    activeSessions.Add(ctx, -1)
    sessionDuration.Record(ctx, duration.Seconds())
}
```

---

## Experience Metrics (Opt-In)

### Configuration

```yaml
telemetry:
  experience_metrics:
    enabled: false           # Must explicitly enable
    include_task_summaries: false
    retention_days: 30
```

### Captured Signals (When Enabled)

| Metric | What | Why |
|--------|------|-----|
| Session outcome | success, failure, abandoned | Product effectiveness |
| Memory hit rate | Did ReasoningBank help? | Core value validation |
| Checkpoint usage | How often resumed? | Feature validation |
| Error recovery | Did user continue after error? | UX improvement |
| Scrubber feedback | Helpful ratio | Security UX |

### Privacy Guarantees

- Opt-in only, default off
- All free text scrubbed before storage
- No actual commands/code captured
- No session-reconstructing information
- Aggregated, not individual tracking

---

## User Feedback API

```protobuf
service FeedbackService {
    rpc ReportScrubberIssue(ScrubberFeedbackRequest) returns (FeedbackResponse);
}

message ScrubberFeedbackRequest {
    string session_id = 1;
    string type = 2;        // "missed_secret", "false_positive", "helpful"
    string context = 3;     // "bash_output", "file_read"
    string rule_id = 4;     // If known
    string description = 5; // Free text (scrubbed before storage)
}
```

---

## Metric Registration

### Provider Setup

```go
func NewMetrics(provider metric.MeterProvider) (*Metrics, error) {
    meter := provider.Meter("contextd")

    requestDuration, err := meter.Float64Histogram(
        "contextd.request.duration",
        metric.WithUnit("ms"),
        metric.WithDescription("gRPC request latency"),
        metric.WithExplicitBucketBoundaries(standardBuckets...),
    )
    if err != nil {
        return nil, fmt.Errorf("request duration: %w", err)
    }

    // ... register other metrics

    return &Metrics{
        requestDuration: requestDuration,
        // ...
    }, nil
}
```

---

## References

- [OpenTelemetry Metrics SDK](https://opentelemetry.io/docs/languages/go/instrumentation/#metrics)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
