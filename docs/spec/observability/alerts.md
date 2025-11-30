# Alerts Specification

**Version**: 1.0.0
**Status**: Draft

---

## Overview

Monitoring signals and alert definitions for contextd using RED+USE methodology.

---

## Critical Signals (RED + USE)

### RED Signals

| Signal | What | Alert Threshold |
|--------|------|-----------------|
| **Rate** | Requests/sec | N/A (baseline) |
| **Error Rate** | % failing | >1% for 5min |
| **Duration** | p95 latency | >500ms for 5min |

### USE Signals

| Signal | What | Alert Threshold |
|--------|------|-----------------|
| **Utilization** | Sessions / max | >80% capacity |
| **Saturation** | Queue depth | >100 pending |
| **Errors** | System errors | >0 |

---

## Alert Definitions

### P0 - Page Immediately

Critical issues requiring immediate response.

| Alert | Condition | Action |
|-------|-----------|--------|
| IsolationViolation | `contextd_isolation_violations > 0` | Security incident |
| ServiceDown | No requests for 5min | Service restart |
| QdrantUnreachable | Qdrant errors >50% for 2min | Check Qdrant |
| ScrubberBypass | Unscrubbed output detected | Security incident |

### P1 - Page Business Hours

Issues requiring same-day resolution.

| Alert | Condition | Action |
|-------|-----------|--------|
| HighErrorRate | Error rate >5% for 5min | Investigate |
| HighLatency | p95 >1s for 10min | Performance check |
| CheckpointFailures | Any save failures | Storage check |
| ScrubberFailure | Scrubber errors >1% | Config review |
| MemoryStoreFailure | Memory store errors >1% | Qdrant check |

### P2 - Ticket, 24h Fix

Issues requiring near-term resolution.

| Alert | Condition | Action |
|-------|-----------|--------|
| ElevatedErrorRate | Error rate >1% for 15min | Ticket |
| ToolTimeouts | Timeout rate >5% | Timeout tuning |
| SessionCapacity | >80% max sessions | Capacity planning |
| QdrantSlowQueries | p95 >200ms for 30min | Index review |
| DiskUsageHigh | >80% disk | Cleanup/expand |

### P3 - Review Weekly

Trends requiring attention.

| Alert | Condition | Action |
|-------|-----------|--------|
| SecretDetectionSpike | 10x normal rate | Pattern review |
| UnusualToolDistribution | Usage pattern change | Behavioral |
| LongSessions | Sessions >1h | UX review |
| MemoryLowHitRate | <30% useful results | Quality review |
| CheckpointLargeSize | >1MB average | Optimization |

---

## Alert Configuration

### Prometheus Alert Rules

```yaml
groups:
  - name: contextd-critical
    rules:
      - alert: IsolationViolation
        expr: contextd_isolation_violations > 0
        for: 0m
        labels:
          severity: critical
          team: security
        annotations:
          summary: "Tenant isolation violation detected"
          runbook: "https://docs.internal/runbooks/isolation-violation"

      - alert: ServiceDown
        expr: absent(up{job="contextd"}) or up{job="contextd"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "contextd service is down"

  - name: contextd-high
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(contextd_request_total{status="error"}[5m])) /
          sum(rate(contextd_request_total[5m])) > 0.05
        for: 5m
        labels:
          severity: high
        annotations:
          summary: "Error rate exceeds 5%"
```

---

## Key Dashboards

### Overview Dashboard

**Purpose**: Is contextd healthy right now?

| Panel | Metrics |
|-------|---------|
| Request Rate | `rate(contextd_request_total[5m])` |
| Error Rate | Error requests / total requests |
| Latency p50/p95/p99 | `histogram_quantile(contextd_request_duration)` |
| Active Sessions | `contextd_session_active` |

### Investigation Dashboard

**Purpose**: Why is it unhealthy?

| Panel | Metrics |
|-------|---------|
| Error breakdown | By method, error type |
| Latency by operation | Per-service histograms |
| Qdrant health | Connection, query latency |
| Tool execution | Duration by tool type |

### Security Dashboard

**Purpose**: Is isolation working? Scrubber effective?

| Panel | Metrics |
|-------|---------|
| Isolation violations | `contextd_isolation_violations` |
| Secrets detected | By type, confidence |
| Scrubber latency | p95 scrubbing time |
| User feedback | Helpful/missed/false positive |

### Tools Dashboard

**Purpose**: Tool execution patterns

| Panel | Metrics |
|-------|---------|
| Tool usage | By type over time |
| Execution duration | Histograms by tool |
| Timeout rate | Timeouts / total |
| Error patterns | By tool, error type |

### Sessions Dashboard

**Purpose**: Session lifecycle, checkpoints

| Panel | Metrics |
|-------|---------|
| Session rate | Start/end over time |
| Duration distribution | Histogram |
| Checkpoint saves | Rate, size |
| Memory search hits | Results per query |

---

## Grafana Provisioning

### Dashboard JSON Location

```
deploy/grafana/dashboards/
├── overview.json
├── investigation.json
├── security.json
├── tools.json
└── sessions.json
```

### Datasource Configuration

```yaml
datasources:
  - name: VictoriaMetrics
    type: prometheus
    url: http://victoriametrics:8428
    isDefault: true

  - name: VictoriaLogs
    type: loki
    url: http://victorialogs:9428

  - name: VictoriaTraces
    type: jaeger
    url: http://victoriatraces:9420
```

---

## References

- [RED Method](https://www.weave.works/blog/the-red-method-key-metrics-for-microservices-architecture/)
- [USE Method](https://www.brendangregg.com/usemethod.html)
- [Prometheus Alerting](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
