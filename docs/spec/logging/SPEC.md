# Logging Specification

**Feature**: Structured Logging
**Status**: Draft
**Created**: 2025-11-23

---

## Overview

Structured logging for contextd using Zap with dual output (stdout + OpenTelemetry), context-aware methods, secret redaction, and intelligent sampling. Logs correlate with traces via trace_id/span_id injection.

---

## Quick Reference

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| **Library** | Zap | 3x faster than slog, native sampling |
| **Output** | Dual (stdout + OTEL) | Local dev + observability |
| **Levels** | Trace-Fatal (custom Trace) | Ultra-verbose debugging |
| **Redaction** | Field names + patterns | No secrets in logs |
| **Sampling** | Level-aware | Volume control, errors never sampled |

---

## Architecture

@./architecture.md

```
┌─────────────────────────────────────────────────────────────────┐
│                        contextd                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              internal/logging/                           │    │
│  │  ├── logger.go      (Logger wrapper, factory)           │    │
│  │  ├── levels.go      (Custom Trace level)                │    │
│  │  ├── config.go      (LogConfig struct)                  │    │
│  │  ├── context.go     (Context extraction, injection)     │    │
│  │  ├── redact.go      (Sensitive field redaction)         │    │
│  │  ├── sampling.go    (Level-aware sampling)              │    │
│  │  ├── otel.go        (OTEL bridge integration)           │    │
│  │  ├── middleware.go  (gRPC interceptor)                  │    │
│  │  └── testing.go     (Test helpers, assertions)          │    │
│  └─────────────────────────────────────────────────────────┘    │
│              │                         │                         │
│      ┌───────┴───────┐         ┌───────┴───────┐                │
│      │ JSON Encoder  │         │ OTEL Bridge   │                │
│      │ (stdout)      │         │ (otelzap)     │                │
│      └───────────────┘         └───────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Detailed Documentation

| File | Content |
|------|---------|
| @./architecture.md | Logger design, Zap choice, dual output |
| @./levels.md | Log levels (Trace-Fatal), usage guidelines |
| @./context.md | Context injection, correlation IDs |
| @./redaction.md | Secret redaction, patterns, encoder |
| @./sampling.md | Level-aware sampling strategy |
| @./testing.md | Test helpers, assertions |

---

## Goals

| Goal | Description |
|------|-------------|
| **Correlation** | Every log linked to trace/span for debugging |
| **Security** | No secrets, tokens, or PII in logs |
| **Performance** | Minimal overhead, zero-allocation hot paths |
| **Volume Control** | Sampling prevents log flood without losing errors |
| **Dual Output** | Local dev (stdout) + observability (OTEL) simultaneously |

## Non-Goals

- Log aggregation from external systems
- Custom log storage (use VictoriaLogs via OTEL)
- Metrics derived from logs (use proper metrics)

---

## Requirements Summary

### Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-001 | All logging methods accept `context.Context` as first parameter |
| FR-002 | Logs include `trace_id` and `span_id` when available |
| FR-003 | Logs include tenant attributes (org, team, project) |
| FR-004 | Logs include `session.id` when available |
| FR-005 | System redacts sensitive fields before output |
| FR-006 | Support simultaneous stdout and OTEL output |
| FR-007 | Sample logs by level, never sampling Error+ |
| FR-008 | Support Trace level below Debug |
| FR-009 | Production logs JSON formatted |
| FR-010 | Logger flushes all buffered logs on Sync() |

### Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-001 | Zero secret leakage in any log output |
| NFR-002 | >99% request logs include trace_id when tracing enabled |
| NFR-003 | Logging overhead <1ms per entry in hot paths |
| NFR-004 | Error logs never sampled/dropped |
| NFR-005 | Logging package >80% test coverage |

---

## Implementation Phases

| Phase | Description | Status |
|-------|-------------|--------|
| 1 | Core Logger (wrapper, Trace level, JSON, stdout) | Pending |
| 2 | Redaction (Secret type, encoder, patterns) | Pending |
| 3 | Context Integration (trace/tenant/session extraction) | Pending |
| 4 | OTEL Bridge (otelzap, dual output) | Pending |
| 5 | Sampling (level-aware, metrics) | Pending |
| 6 | Middleware (gRPC interceptors) | Pending |

---

## Configuration Quick Reference

```yaml
logging:
  level: "info"           # trace, debug, info, warn, error
  format: "json"          # json, console
  output:
    stdout: true
    otel: true
  sampling:
    enabled: true
    tick: "1s"
  redaction:
    enabled: true
```

Full configuration: @./architecture.md#configuration

---

## References

- [Zap Documentation](https://pkg.go.dev/go.uber.org/zap)
- [Zap GitHub](https://github.com/uber-go/zap)
- [OpenTelemetry Zap Bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelzap)
- [Zap Best Practices](https://betterstack.com/community/guides/logging/go/zap/)
- [Sensitive Data in Logs](https://betterstack.com/community/guides/logging/sensitive-data/)

---

## Maintenance

**Update when**: Architecture changes, new levels added, security model changes

**Keep**: Scannable, noun-heavy, @imports for implementation details
