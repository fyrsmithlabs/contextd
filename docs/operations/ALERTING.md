# contextd Alerting Configuration

This guide covers Prometheus alerting rules and Alertmanager configuration for contextd vectorstore health monitoring.

## Overview

contextd exposes Prometheus metrics for monitoring vectorstore health. This document describes:
- Alert rules for detecting issues
- Alertmanager routing configuration
- Integration with notification systems

## Prerequisites

- Prometheus server with alerting enabled
- Alertmanager for notification routing
- contextd running with HTTP server enabled (`--no-http` not set)

## Alert Rules

Alert rules are defined in `deploy/prometheus/alerts.yml`.

### Critical Alerts

| Alert | Condition | Description |
|-------|-----------|-------------|
| `VectorstoreDegraded` | `contextd_vectorstore_health_status == 0` | Vectorstore operating in degraded mode |
| `CorruptCollectionsDetected` | `collections_total{status="corrupt"} > 0` | Corrupt collections found |
| `NoHealthyCollections` | All collections corrupt or empty | Complete service degradation |

### Warning Alerts

| Alert | Condition | Description |
|-------|-----------|-------------|
| `HealthCheckFailing` | Health checks returning errors | Filesystem or config issues |
| `HealthCheckSlow` | p99 latency > 1s | Performance degradation |

### Info Alerts

| Alert | Condition | Description |
|-------|-----------|-------------|
| `QuarantineOperationOccurred` | Quarantine in last hour | Collection was auto-quarantined |
| `ManyEmptyCollections` | > 10 empty collections | Potential cleanup needed |

## Deployment

### 1. Deploy Alert Rules to Prometheus

```bash
# Copy rules file
cp deploy/prometheus/alerts.yml /etc/prometheus/rules/contextd.yml

# Validate rules
promtool check rules /etc/prometheus/rules/contextd.yml

# Reload Prometheus
curl -X POST http://prometheus:9090/-/reload
```

### 2. Configure Alertmanager

```bash
# Copy and customize example config
cp deploy/prometheus/alertmanager.yml.example /etc/alertmanager/alertmanager.yml

# Edit with your settings:
# - Slack webhook URL
# - PagerDuty service key
# - Email settings
vim /etc/alertmanager/alertmanager.yml

# Validate config
amtool check-config /etc/alertmanager/alertmanager.yml

# Reload Alertmanager
curl -X POST http://alertmanager:9093/-/reload
```

### 3. Configure Prometheus Scraping

Add contextd to your Prometheus scrape config:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'contextd'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
    metrics_path: /metrics
```

## Alert Routing

The example Alertmanager configuration routes alerts by severity:

| Severity | Destination | Timing |
|----------|-------------|--------|
| `critical` | PagerDuty + Slack | Immediate |
| `warning` | Slack #platform-alerts | 1min group wait |
| `info` | Slack #platform-info | 5min group wait |

### Inhibition Rules

- If `ContextdDown`, suppress all other contextd alerts
- If `NoHealthyCollections`, suppress `VectorstoreDegraded`

## Metrics Reference

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `contextd_vectorstore_health_status` | Gauge | - | 1=healthy, 0=degraded |
| `contextd_vectorstore_collections_total` | Gauge | status | Collections by status |
| `contextd_vectorstore_health_check_duration_seconds` | Histogram | - | Check latency |
| `contextd_vectorstore_health_checks_total` | Counter | result | Check count |
| `contextd_vectorstore_corrupt_collections_detected_total` | Counter | - | Corruption count |
| `contextd_vectorstore_quarantine_operations_total` | Counter | result | Quarantine ops |

## Grafana Dashboard

Import the pre-built dashboards from `grafana/dashboards/`:
- `overview.json` - Overall system health
- `memory-checkpoint.json` - Memory and checkpoint metrics

Example panel query for health status:
```promql
contextd_vectorstore_health_status{instance="$instance"}
```

## Runbook Links

All critical alerts include `runbook_url` annotations pointing to:
- [Metadata Recovery Guide](./METADATA_RECOVERY.md)
- [Health Monitoring Guide](./METADATA_HEALTH_MONITORING.md)

## Testing Alerts

### Test alert firing (dry run)
```bash
# Check if alert would fire with current metrics
promtool query instant http://prometheus:9090 'contextd_vectorstore_health_status == 0'
```

### Send test alert to Alertmanager
```bash
# Create test alert
cat > /tmp/test-alert.json << 'EOF'
[
  {
    "labels": {
      "alertname": "VectorstoreDegraded",
      "severity": "critical",
      "service": "contextd",
      "instance": "localhost:9090"
    },
    "annotations": {
      "summary": "TEST: Vectorstore degraded",
      "description": "This is a test alert"
    }
  }
]
EOF

# Send to Alertmanager
curl -X POST -d @/tmp/test-alert.json http://alertmanager:9093/api/v1/alerts
```

## Troubleshooting

### Alert not firing

1. Check Prometheus is scraping contextd:
   ```bash
   curl http://prometheus:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job=="contextd")'
   ```

2. Verify metrics exist:
   ```bash
   curl http://localhost:9090/metrics | grep contextd_vectorstore
   ```

3. Check rule evaluation:
   ```bash
   curl http://prometheus:9090/api/v1/rules | jq '.data.groups[] | select(.name=="contextd_vectorstore")'
   ```

### Alert stuck in pending

- Check the `for` duration in the alert rule
- Verify the condition is still true after the `for` period

### Notifications not received

1. Check Alertmanager logs
2. Verify receiver configuration (webhook URLs, API keys)
3. Test receiver directly with `amtool`

## See Also

- [Background Scanner Implementation](../../internal/vectorstore/background_scanner.go)
- [Prometheus Alert Rules](../../deploy/prometheus/alerts.yml)
- [Alertmanager Example Config](../../deploy/prometheus/alertmanager.yml.example)
