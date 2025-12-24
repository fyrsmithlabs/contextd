# Observability Specification

**Feature**: Observability (Traces, Metrics, Logs)
**Status**: Draft
**Created**: 2025-11-23

---

## Overview

Full-stack observability for contextd using OpenTelemetry instrumentation with unified VictoriaMetrics backend. Provides distributed tracing, metrics collection, and structured logging for debugging, performance analysis, and security validation.

---

## Quick Reference

| Aspect | Choice |
|--------|--------|
| **Instrumentation** | OpenTelemetry Go SDK |
| **Collector** | OTEL Collector (decouples app from backends) |
| **Backend** | VictoriaMetrics (metrics, logs, traces) |
| **Visualization** | Grafana |
| **Logging** | Zap (structured, trace ID injection) |
| **Config** | Koanf (file + env + flags) |

---

## Architecture

```
contextd → OTLP (4317) → OTEL Collector
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
        VictoriaMetrics  VictoriaLogs   VictoriaTraces
              └───────────────┼───────────────┘
                              ▼
                           Grafana
```

@./architecture.md

---

## Detailed Documentation

| File | Content |
|------|---------|
| @./architecture.md | System design, stack decisions, config, gRPC integration |
| @./tracing.md | Span naming, attributes, propagation, error recording |
| @./metrics.md | Metric definitions, histograms, recording patterns |
| @./alerts.md | RED+USE signals, alert tiers, dashboards |
| @./testing.md | Three-layer test strategy, TestTelemetry helpers |
| @./dev-stack.md | Docker Compose setup, quick start commands |

---

## Goals

| Goal | Description |
|------|-------------|
| **Proof-of-work** | Traces proving tools executed correctly |
| **Debugging** | Trace request flows to find where things break |
| **Performance** | Identify latency bottlenecks across services |
| **Security visibility** | Validate secret scrubbing and tenant isolation |

## Non-Goals

- APM/RUM (browser-side monitoring)
- Business analytics beyond operational metrics
- Log aggregation from external systems

---

## Requirements Summary

### Functional

| ID | Requirement |
|----|-------------|
| FR-001 | Use OpenTelemetry Go SDK for all instrumentation |
| FR-002 | Create spans for gRPC, tool execution, Qdrant, scrubbing |
| FR-003 | Include tenant context (org, team, project) on all telemetry |
| FR-004 | Include session_id for correlation |
| FR-005 | No secret content in telemetry (rule IDs, types, counts only) |
| FR-006 | Configurable sampling; always capture errors |
| FR-007 | Debug-level span lifecycle logging |
| FR-008 | Graceful shutdown with telemetry flush |
| FR-009 | Telemetry failures must not crash application |
| FR-010 | Experience metrics opt-in only |

### Non-Functional

| ID | Requirement |
|----|-------------|
| NFR-001 | <5% telemetry overhead on request latency |
| NFR-002 | >80% test coverage for telemetry package |
| NFR-003 | >99% telemetry exported on graceful shutdown |

### Security

| ID | Requirement |
|----|-------------|
| SEC-001 | Zero secret content in any telemetry data |
| SEC-002 | Scrubber metrics capture rule IDs, not content |
| SEC-003 | User feedback scrubbed before storage |

---

## Success Criteria

| ID | Criteria |
|----|----------|
| SC-001 | 100% of gRPC requests produce complete traces |
| SC-002 | Metrics match actual counts/durations within 1% |
| SC-003 | 0 instances of secret content in telemetry |
| SC-004 | >99% pending telemetry exported on shutdown |
| SC-005 | <5% request latency overhead |
| SC-006 | >80% test coverage |

---

## Implementation Phases

| Phase | Scope |
|-------|-------|
| **1: Core Setup** | TracerProvider, MeterProvider, gRPC stats handlers, dev stack |
| **2: Full Instrumentation** | All service spans, all metrics, debug logging, graceful shutdown |
| **3: Scrubber Observability** | Scrubber metrics, rule tracking, user feedback API, security dashboard |
| **4: Experience Metrics** | Opt-in config, session outcomes, memory effectiveness, privacy guarantees |
| **5: Alerting** | Prometheus rules, dashboard provisioning, runbooks |

---

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| VictoriaMetrics unified stack | Single ecosystem, lower operational complexity |
| OTEL Collector middleware | Backend flexibility, sampling control |
| Stats handlers over interceptors | Modern approach, better integration |
| Zap over slog | Performance, trace ID injection support |
| Koanf over viper | Lighter weight, cleaner API |

---

## References

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [VictoriaMetrics Documentation](https://docs.victoriametrics.com/)
- [VictoriaLogs Documentation](https://docs.victoriametrics.com/victorialogs/)
- [VictoriaTraces Documentation](https://docs.victoriametrics.com/victoriatraces/)
- [otelgrpc Package](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc)
