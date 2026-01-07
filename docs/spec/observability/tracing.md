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
| **MCP tool handlers** | memory_search, memory_record, checkpoint_save, remediation_search, repository_search, etc. |
| **Internal services** | ReasoningBank, Remediation, Checkpoint, Repository, Troubleshoot, Folding |
| **Vectorstore operations** | Vector search, store, collection management (chromem or Qdrant) |
| **Secret scrubbing** | gitleaks pipeline (timing only, no content) |

---

## Span Naming Convention

| Service | Pattern | Example |
|---------|---------|---------|
| Internal Services | `{service}.{operation}` | `remediation.search`, `reasoningbank.record` |
| Vectorstore | `{provider}.{operation}` | `chromem.search`, `qdrant.add_documents` |
| Scrubber | `scrubber.{stage}` | `scrubber.scan` |
| Repository | `repository.{operation}` | `repository.index`, `repository.search` |
| Troubleshoot | `Service.{operation}` | `Service.Diagnose`, `Service.SavePattern` |

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
| **Memory search** | `query.text`, `query.limit`, `results.count`, `avg_confidence` |
| **Remediation** | `error_type`, `scope`, `limit`, `results.count` |
| **Repository** | `project_path`, `query`, `limit`, `results.count` |
| **Vectorstore** | `store.provider` (chromem/qdrant), `collection`, `operation`, `doc_count` |
| **Checkpoint** | `checkpoint.id`, `checkpoint.size`, `metadata_keys` |
| **Scrubber** | `scrubber.rules_matched`, `scrubber.action`, `scrubber.confidence_min` |

---

## Example Trace

```
remediation.search (service operation)
├── chromem.search (vectorstore query)
│   └── [embedding generation + similarity search]
└── remediation.score_results (confidence calculation)

reasoningbank.record (service operation)
├── scrubber.scan (input validation)
├── chromem.add_documents (vectorstore insert)
│   └── [embedding generation + document insert]
└── signal_store.update (confidence tracking)
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

When `logging.level: debug`:

```json
{"level":"debug","msg":"span.start","span.name":"remediation.search","span.id":"abc123","trace.id":"xyz789","parent.id":""}
{"level":"debug","msg":"span.end","span.name":"remediation.search","span.id":"abc123","duration":"45.2ms","has_error":false}
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
