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

## Logger in Context

```go
// Store logger in context for handler access
type ctxKey struct{}

func WithLogger(ctx context.Context, logger *Logger) context.Context {
    return context.WithValue(ctx, ctxKey{}, logger)
}

func FromContext(ctx context.Context) *Logger {
    if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
        return l
    }
    return defaultLogger
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

## gRPC Middleware Integration

```go
func UnaryServerInterceptor(logger *Logger) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        start := time.Now()

        // Create request-scoped logger with method context
        reqLogger := logger.WithContext(ctx).With(
            zap.String("grpc.method", info.FullMethod),
            zap.String("grpc.service", path.Dir(info.FullMethod)),
        )

        // Store in context for handlers
        ctx = WithLogger(ctx, reqLogger)

        // Log request start (debug level)
        reqLogger.Debug(ctx, "grpc.request.start")

        // Execute handler
        resp, err := handler(ctx, req)

        // Log completion
        duration := time.Since(start)
        fields := []zap.Field{
            zap.Duration("duration", duration),
            zap.Int64("duration_ms", duration.Milliseconds()),
        }

        if err != nil {
            code := status.Code(err)
            fields = append(fields,
                zap.Error(err),
                zap.String("grpc.code", code.String()),
            )
            reqLogger.Error(ctx, "grpc.request.error", fields...)
        } else {
            fields = append(fields, zap.String("grpc.code", "OK"))
            reqLogger.Info(ctx, "grpc.request.complete", fields...)
        }

        return resp, err
    }
}
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
  "tenant.org": "acme",
  "tenant.team": "platform",
  "tenant.project": "api",
  "session.id": "sess_001",
  "request.id": "req_456",
  "grpc.method": "/contextd.v1.SafeExecService/Bash",
  "tool": "bash",
  "duration_ms": 45
}
```

---

## Maintenance

**Update when**: New context fields added, correlation strategy changes

**Keep**: Field table accurate, middleware example current
