# Log Levels

**Parent**: @./SPEC.md

---

## Level Definitions

| Level | Value | When to Use |
|-------|-------|-------------|
| **Trace** | -2 | Ultra-verbose: function entry/exit, wire data, byte counts |
| **Debug** | -1 | Development: internal state, cache behavior, config details |
| **Info** | 0 | Normal operations: requests, completions, state changes |
| **Warn** | 1 | Recoverable issues: retries, fallbacks, deprecations |
| **Error** | 2 | Failures: tool errors, connection failures, timeouts |
| **DPanic** | 3 | Invariants violated: panics in dev, errors in prod |
| **Fatal** | 4 | Unrecoverable: startup failures only |

---

## Custom Trace Level

```go
const TraceLevel = zapcore.Level(-2)

func init() {
    zap.RegisterLevel("trace", TraceLevel)
}
```

---

## Level Usage Examples

### Trace / Debug

```go
// Trace: ultra-verbose, almost always filtered
logger.Trace(ctx, "entering function", zap.String("func", "ProcessBashRequest"))

// Debug: development, internal details
logger.Debug(ctx, "cache lookup", zap.String("key", cacheKey), zap.Bool("hit", hit))
```

### Info / Warn

```go
// Info: normal operations, audit trail
logger.Info(ctx, "tool executed", zap.String("tool", "bash"), zap.Duration("duration", elapsed))

// Warn: something unexpected but handled
logger.Warn(ctx, "qdrant connection retry", zap.Int("attempt", attempt), zap.Error(err))
```

### Error / DPanic / Fatal

```go
// Error: failure requiring attention (stacktrace auto-included)
logger.Error(ctx, "tool execution failed", zap.String("tool", "bash"), zap.Error(err))

// DPanic: should-never-happen (panics in dev)
logger.DPanic(ctx, "nil session in authenticated context", zap.String("endpoint", endpoint))

// Fatal: only during startup (calls os.Exit)
logger.Fatal(ctx, "failed to initialize qdrant", zap.String("host", cfg.Qdrant.Host), zap.Error(err))
```

---

## Level Selection Decision Tree

```
Is this normal operation output?
├── Yes → INFO
└── No ↓

Is something wrong?
├── No → TRACE (very verbose) or DEBUG (normal detail)
└── Yes ↓

Was it handled/recovered?
├── Yes → WARN
└── No ↓

Is it unrecoverable (startup only)?
├── Yes → FATAL
└── No → DPANIC (invariant) or ERROR (normal failure)
```

---

## Level Filtering by Environment

| Environment | Recommended Level |
|-------------|------------------|
| Development | debug or trace |
| Staging | debug |
| Production | info |
| Debugging Production | debug (temporary) |

---

## Trace Level Output Example

```json
{
  "ts": "2025-11-23T10:15:30.100Z",
  "level": "trace",
  "caller": "safeexec/bash.go:98",
  "msg": "entering ProcessBashRequest",
  "trace_id": "abc123def456789",
  "func": "ProcessBashRequest",
  "input_bytes": 256
}
```

---

## Maintenance

**Update when**: New level added, level semantics change

**Keep**: Clear distinction between levels, concrete examples
