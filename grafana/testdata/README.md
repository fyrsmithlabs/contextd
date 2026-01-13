# Dashboard Test Data Generator

This utility generates sample metrics data to test Grafana dashboards without using real production data.

## Quick Start

```bash
# Build the generator
go build -o metrics-generator ./generate_metrics.go

# Run it (exposes metrics on :9090 by default)
./metrics-generator

# Or specify a different port
PORT=9091 ./metrics-generator
```

## Usage with Prometheus

Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'contextd-test'
    static_configs:
      - targets: ['localhost:9090']
```

## Usage with Docker Compose

```bash
cd grafana/testdata
docker-compose up -d
```

This starts:
- Metrics generator on port 9090
- Prometheus on port 9091
- Grafana on port 3001

Access Grafana at http://localhost:3001 (admin/admin) and import the dashboards from `grafana/dashboards/`.

## Metrics Generated

| Category | Metrics |
|----------|---------|
| Checkpoint | `contextd_checkpoint_saves_total`, `contextd_checkpoint_resumes_total`, `contextd_checkpoint_errors_total`, `contextd_checkpoint_count` |
| Memory | `contextd_memory_searches_total`, `contextd_memory_records_total`, `contextd_memory_feedbacks_total`, `contextd_memory_outcomes_total`, `contextd_memory_errors_total`, `contextd_memory_count` |
| Remediation | `contextd_remediation_searches_total`, `contextd_remediation_records_total`, `contextd_remediation_feedbacks_total`, `contextd_remediation_errors_total` |
| Compression | `compression_operations_total`, `compression_duration_seconds`, `compression_errors_total`, `compression_ratio`, `compression_input_tokens_total`, `compression_output_tokens_total` |
| Context Folding | `folding_branch_created_total`, `folding_branch_returned_total`, `folding_branch_duration_seconds`, `folding_branch_tokens_used`, `folding_branch_depth`, `folding_active_branches` |
| Workflows | `contextd_workflows_version_validation_executions`, `contextd_workflows_duration_seconds`, `contextd_workflows_activity_errors` |
| HTTP | `contextd_http_requests_total`, `contextd_http_request_duration_seconds` |

## Behavior

- On startup, generates initial sample data with realistic distributions
- Every 5 seconds, adds incremental activity to simulate ongoing usage
- Counters increment, gauges fluctuate
- Some operations randomly "fail" to generate error metrics
