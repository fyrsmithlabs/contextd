# Tracing Specification

**Version**: 1.0.0
**Status**: Draft

---

## Overview

Distributed tracing for contextd using OpenTelemetry spans with standardized naming and attributes.

---

## Traced Operations

| Layer | Operations |
|-------|-----------|
| **gRPC services** | SafeExec, Memory, Checkpoint, Policy, Skill, Agent, Remediation, Ref, Session |
| **Tool execution** | safe_bash, safe_read, safe_write (process isolation boundary) |
| **Qdrant operations** | Vector search, store, collection management |
| **Secret scrubbing** | gitleaks pipeline (timing only, no content) |

---

## Span Naming Convention

| Service | Pattern | Example |
|---------|---------|---------|
| gRPC | `{service}/{method}` | `contextd.SafeExec/Bash` |
| Qdrant | `qdrant.{operation}` | `qdrant.search` |
| Scrubber | `scrubber.{stage}` | `scrubber.scan` |
| Executor | `executor.{tool}` | `executor.bash` |

---

## Standard Attributes

### Tenant Context (On Every Span)

```go
attribute.String("tenant.org", orgID)
attribute.String("tenant.team", teamID)
attribute.String("tenant.project", projectID)
```

### Session Context

```go
attribute.String("session.id", sessionID)
attribute.String("session.task", taskDescription)
```

---

## Operation-Specific Attributes

| Operation | Attributes |
|-----------|------------|
| **Bash** | `tool.cmd` (scrubbed), `tool.timeout`, `exit_code` |
| **Read** | `file.path`, `file.lines`, `file.bytes` |
| **Write** | `file.path`, `file.bytes`, `file.append` |
| **Memory search** | `query.scope`, `query.limit`, `results.count` |
| **Qdrant** | `qdrant.collection`, `qdrant.operation`, `qdrant.points` |
| **Scrubber** | `scrubber.rules_matched`, `scrubber.action`, `scrubber.confidence_min` |

---

## Example Trace

```
contextd.SafeExec/Bash (server span)
├── scrubber.scan (input validation)
├── executor.bash (process isolation)
│   └── [child process execution]
└── scrubber.redact (output scrubbing)
    └── gitleaks.detect
```

---

## Trace Propagation

### Context Builders

```go
// Add tenant context to span
func WithTenantAttributes(ctx context.Context, org, team, project string) context.Context {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("tenant.org", org),
        attribute.String("tenant.team", team),
        attribute.String("tenant.project", project),
    )
    return ctx
}

// Add session context to span
func WithSessionAttributes(ctx context.Context, sessionID, task string) context.Context {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("session.id", sessionID),
        attribute.String("session.task", task),
    )
    return ctx
}
```

---

## Debug Logging

When `logging.level: debug` or `logging.trace_debug: true`:

```json
{"level":"debug","msg":"span.start","span.name":"contextd.SafeExec/Bash","span.id":"abc123","trace.id":"xyz789","parent.id":""}
{"level":"debug","msg":"span.end","span.name":"contextd.SafeExec/Bash","span.id":"abc123","duration":"45.2ms","has_error":false}
```

---

## Error Recording

### Error Span Pattern

```go
func doOperation(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "operation.name")
    defer span.End()

    if err := riskyOperation(); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("operation failed: %w", err)
    }

    span.SetStatus(codes.Ok, "")
    return nil
}
```

### Error Attributes

| Attribute | Description |
|-----------|-------------|
| `error.type` | Error type/category |
| `error.message` | Scrubbed error message |
| `error.recoverable` | Whether operation can retry |

---

## Sampling

### Configuration

```yaml
sampling:
  rate: 1.0              # 100% in dev
  always_on_errors: true # Always capture error traces
```

### Sampling Strategy

| Environment | Rate | Notes |
|-------------|------|-------|
| Development | 1.0 | All traces |
| Staging | 0.5 | 50% sampling |
| Production | 0.1 | 10% + all errors |

---

## Scrubber Tracing

### Privacy-Safe Attributes

| Signal | Captured | Privacy |
|--------|----------|---------|
| Rule matched | Rule ID only | Never content |
| Match location | Generic type (env_var, config, output) | No paths |
| Confidence score | Numeric | Safe |
| Action taken | redact, tokenize, block | Safe |
| Source context | Tool name | No content |

### Scrubber Span Example

```
scrubber.redact
├── gitleaks.detect
│   attributes:
│     scrubber.rules_matched: 2
│     scrubber.action: "redact"
│     scrubber.confidence_min: 0.95
└── scrubber.tokenize
```

---

## References

- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [Span Status Codes](https://opentelemetry.io/docs/languages/go/instrumentation/#recording-errors)
