# Logging Architecture

**Parent**: @./SPEC.md

---

## Technology Choice: Zap

| Factor | Zap | slog | Decision |
|--------|-----|------|----------|
| Performance | 67 ns/op, 0 allocs | 193 ns/op, 0 allocs | Zap 3x faster |
| Sampling | Built-in, battle-tested | Manual | Zap native |
| OTEL Bridge | Experimental | Official stable | Acceptable |
| Ecosystem | Mature, widely used | Growing | Zap proven |

**Decision**: Zap for performance, sampling, and ecosystem maturity.

---

## Package Structure

```
internal/logging/
├── logger.go      # Logger wrapper, factory
├── levels.go      # Custom Trace level
├── config.go      # LogConfig struct
├── context.go     # Context extraction, injection
├── redact.go      # Sensitive field redaction
├── sampling.go    # Level-aware sampling
├── otel.go        # OTEL bridge integration
├── middleware.go  # gRPC interceptor
└── testing.go     # Test helpers, assertions
```

---

## Dual Output Architecture

```
Logger.Info(ctx, "message", fields...)
         │
         ▼
┌─────────────────────────────────────┐
│  Logger Wrapper                     │
│  Extract context → Redact → Output  │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│        zapcore.Tee                  │
├─────────────────┬───────────────────┤
│ Stdout Core     │ OTEL Core         │
│ JSON/Console    │ otelzap bridge    │
└─────────────────┴───────────────────┘
```

---

## Logger Interface

```go
type Logger struct {
    zap    *zap.Logger
    config *Config
}

func NewLogger(cfg *Config, otelProvider log.LoggerProvider) (*Logger, error)

// Context-aware methods (REQUIRED: always pass context)
func (l *Logger) Trace(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) DPanic(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field)

// Child logger creation
func (l *Logger) With(fields ...zap.Field) *Logger
func (l *Logger) Named(name string) *Logger
func (l *Logger) Enabled(level zapcore.Level) bool
func (l *Logger) Sync() error
```

---

## Configuration

```yaml
logging:
  level: "info"                    # trace, debug, info, warn, error
  format: "json"                   # json, console
  output:
    stdout: true                   # Write to stdout
    otel: true                     # Send to OTEL
  sampling:
    enabled: true
    tick: "1s"
    levels:
      trace: { initial: 1, thereafter: 0 }
      debug: { initial: 10, thereafter: 0 }
      info: { initial: 100, thereafter: 10 }
      warn: { initial: 100, thereafter: 100 }
  caller:
    enabled: true
    skip: 1
  stacktrace:
    level: "error"
  fields:
    service: "contextd"
    version: "${VERSION}"
  redaction:
    enabled: true
    fields: ["password", "secret", "token", "api_key", "authorization"]
    patterns: ["(?i)bearer\\s+[a-zA-Z0-9_-]+"]
```

### Environment Overrides

```bash
CONTEXTD_LOGGING_LEVEL=debug
CONTEXTD_LOGGING_FORMAT=console
CONTEXTD_LOGGING_SAMPLING_ENABLED=false
```

---

## Output Formats

### JSON (Production)

```json
{
  "ts": "2025-11-23T10:15:30.123Z",
  "level": "info",
  "caller": "safeexec/bash.go:142",
  "msg": "tool executed",
  "trace_id": "abc123def456789",
  "tenant.org": "acme",
  "session.id": "sess_001",
  "tool": "bash",
  "duration_ms": 45.2
}
```

### Console (Development)

```
2025-11-23T10:15:30.123Z  INFO   bash.go:142  tool executed  {"trace_id":"abc123","tool":"bash"}
```

---

## OTEL Integration

```go
func NewDualCore(cfg *Config, otelProvider log.LoggerProvider) (zapcore.Core, error) {
    cores := make([]zapcore.Core, 0, 2)

    if cfg.Output.Stdout {
        encoder := NewRedactingEncoder(newEncoder(cfg.Format), cfg.Redaction)
        cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), cfg.Level))
    }

    if cfg.Output.OTEL && otelProvider != nil {
        cores = append(cores, otelzap.NewCore("contextd", otelzap.WithLoggerProvider(otelProvider)))
    }

    return zapcore.NewTee(cores...), nil
}
```

---

## Graceful Shutdown

```go
func (l *Logger) Sync() error {
    err := l.zap.Sync()
    if err != nil && isStdoutSyncError(err) {
        return nil  // Ignore sync errors on stdout/stderr
    }
    return err
}
```

---

## Maintenance

**Update when**: Core architecture changes, new output targets added

**Keep**: Scannable, code examples minimal
