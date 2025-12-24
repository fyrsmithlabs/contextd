# Log Sampling

**Parent**: @./SPEC.md

---

## Overview

Level-aware sampling prevents log volume explosion while preserving all error-level and above logs. Each level has independent sampling configuration.

---

## Sampling Configuration

| Level | Initial | Thereafter | Effect |
|-------|---------|------------|--------|
| Trace | 1 | 0 | First 1 per tick, drop rest |
| Debug | 10 | 0 | First 10 per tick, drop rest |
| Info | 100 | 10 | First 100, then 1 every 10 |
| Warn | 100 | 100 | First 100, then 1 every 100 |
| Error+ | - | - | Never sampled |

**Tick**: Default 1 second window

---

## Configuration

```yaml
sampling:
  enabled: true
  tick: "1s"
  levels:
    trace:
      initial: 1
      thereafter: 0
    debug:
      initial: 10
      thereafter: 0
    info:
      initial: 100
      thereafter: 10
    warn:
      initial: 100
      thereafter: 100
    # error+ never sampled
```

---

## Level-Aware Sampling Code

```go
// internal/logging/sampling.go

// SamplingConfig per level
type LevelSamplingConfig struct {
    Initial    int  // First N logged per tick
    Thereafter int  // Then 1 every M (0 = drop all after Initial)
}

// Default sampling (production)
var DefaultSampling = map[zapcore.Level]LevelSamplingConfig{
    TraceLevel:         {Initial: 1, Thereafter: 0},
    zapcore.DebugLevel: {Initial: 10, Thereafter: 0},
    zapcore.InfoLevel:  {Initial: 100, Thereafter: 10},
    zapcore.WarnLevel:  {Initial: 100, Thereafter: 100},
    // Error+ never sampled
}
```

---

## Sampled Core Implementation

```go
func NewSampledCore(core zapcore.Core, cfg SamplingConfig) zapcore.Core {
    if !cfg.Enabled {
        return core
    }

    // Error and above bypass sampling
    errorCore := &levelFilterCore{
        Core:     core,
        minLevel: zapcore.ErrorLevel,
    }

    // Below error gets sampled
    sampledCore := zapcore.NewSamplerWithOptions(
        &levelFilterCore{Core: core, maxLevel: zapcore.WarnLevel},
        cfg.Tick,
        cfg.Levels[zapcore.InfoLevel].Initial,
        cfg.Levels[zapcore.InfoLevel].Thereafter,
        zapcore.SamplerHook(func(entry zapcore.Entry, dec zapcore.SamplingDecision) {
            if dec == zapcore.LogDropped {
                sampledDroppedTotal.WithLabelValues(entry.Level.String()).Inc()
            }
        }),
    )

    return zapcore.NewTee(errorCore, sampledCore)
}
```

---

## Sampling Metrics

Track dropped logs for observability:

```go
var sampledDroppedTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "contextd_log_sampled_dropped_total",
        Help: "Total log entries dropped by sampling",
    },
    []string{"level"},
)
```

---

## When to Disable Sampling

| Scenario | Recommendation |
|----------|----------------|
| Development | Disable |
| Debugging production issue | Temporary disable |
| Security audit | Disable |
| Normal production | Enable |

```bash
# Disable via environment
CONTEXTD_LOGGING_SAMPLING_ENABLED=false
```

---

## Sampling Guarantees

1. **Error+ never dropped**: All error, dpanic, and fatal logs pass through
2. **Burst handling**: Initial quota handles burst starts
3. **Steady state**: Thereafter rate controls ongoing volume
4. **Metrics visibility**: Dropped count exposed for monitoring

---

## Volume Estimation

Given 1000 logs/second at Info level with default config:
- First second: 100 logged, 900 dropped
- Subsequent: ~100 logged/sec (100 initial + 90 thereafter)
- **Reduction**: ~90%

---

## Maintenance

**Update when**: Default sampling rates tuned, new levels added

**Keep**: Rate table accurate, guarantees documented
