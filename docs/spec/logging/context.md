# Context Injection

**Parent**: @./SPEC.md

---

## Overview

Every log entry automatically includes correlation fields extracted from `context.Context`. This enables end-to-end request tracing across services.

---

## Automatic Field Extraction

```go
// internal/logging/context.go

// ContextFields extracts correlation data from context
func ContextFields(ctx context.Context) []zap.Field {
    fields := make([]zap.Field, 0, 8)

    // Trace correlation (from OpenTelemetry)
    if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
        sc := span.SpanContext()
        fields = append(fields,
            zap.String("trace_id", sc.TraceID().String()),
            zap.String("span_id", sc.SpanID().String()),
        )
        if sc.IsSampled() {
            fields = append(fields, zap.Bool("trace_sampled", true))
        }
    }

    // Tenant context
    if tenant := TenantFromContext(ctx); tenant != nil {
        fields = append(fields,
            zap.String("tenant.org", tenant.OrgID),
            zap.String("tenant.team", tenant.TeamID),
            zap.String("tenant.project", tenant.ProjectID),
        )
    }

    // Session context
    if sessionID := SessionIDFromContext(ctx); sessionID != "" {
        fields = append(fields, zap.String("session.id", sessionID))
    }

    // Request ID
    if requestID := RequestIDFromContext(ctx); requestID != "" {
        fields = append(fields, zap.String("request.id", requestID))
    }

    return fields
}
```

---

## Fields Injected

| Field | Source | Purpose |
|-------|--------|---------|
| `trace_id` | OpenTelemetry span | Cross-service correlation |
| `span_id` | OpenTelemetry span | Within-service correlation |
| `trace_sampled` | OpenTelemetry span | Sampling decision |
| `tenant.org` | Tenant context | Organization isolation |
| `tenant.team` | Tenant context | Team filtering |
| `tenant.project` | Tenant context | Project filtering |
| `session.id` | Session context | Agent session tracking |
| `request.id` | Request context | Request-level correlation |

---

## Tenant Context

```go
// Tenant represents multi-tenant context
type Tenant struct {
    OrgID     string
    TeamID    string
    ProjectID string
}

// WithTenant adds tenant to context
func WithTenant(ctx context.Context, tenant *Tenant) context.Context

// TenantFromContext extracts tenant from context
func TenantFromContext(ctx context.Context) *Tenant

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context

// SessionIDFromContext extracts session ID from context
func SessionIDFromContext(ctx context.Context) string

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context

// RequestIDFromContext extracts request ID from context
func RequestIDFromContext(ctx context.Context) string
```

---

## Logger in Context

```go
// loggerCtxKey is the context key for Logger
type loggerCtxKey struct{}

// WithLogger stores logger in context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
    return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// FromContext retrieves logger from context.
// Returns a default nop logger if not found.
func FromContext(ctx context.Context) *Logger {
    if l, ok := ctx.Value(loggerCtxKey{}).(*Logger); ok {
        return l
    }
    return &Logger{zap: zap.NewNop(), config: NewDefaultConfig()}
}
```

---

## Context Requirement

All logging methods require `context.Context` as first parameter:

```go
// Correct - context enables correlation
logger.Info(ctx, "request processed", zap.Duration("duration", d))

// Wrong - loses trace correlation, tenant context
logger.zap.Info("request processed", zap.Duration("duration", d))
```

---

## Result: Correlated Log Entry

```json
{
  "ts": "2025-11-23T10:15:30.123Z",
  "level": "info",
  "msg": "tool executed",
  "trace_id": "abc123def456789",
  "span_id": "xyz789",
  "trace_sampled": true,
  "tenant.org": "acme",
  "tenant.team": "platform",
  "tenant.project": "api",
  "session.id": "sess_001",
  "request.id": "req_456",
  "tool": "bash",
  "duration_ms": 45
}
```

---

## Maintenance

**Update when**: New context fields added, correlation strategy changes

**Keep**: Field table accurate, middleware example current
