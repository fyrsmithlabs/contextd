# Temporal Workflows Metrics Documentation

**Status**: Complete
**Last Updated**: 2025-12-29

---

## Overview

This document describes the OpenTelemetry metrics emitted by contextd Temporal workflows for monitoring, alerting, and performance analysis.

## Metrics Architecture

All metrics follow the OpenTelemetry standard and are instrumented via the `go.opentelemetry.io/otel/metric` package.

**Instrumentation Scope**: `github.com/fyrsmithlabs/contextd/internal/workflows`

## Available Metrics

### Version Validation Workflow Metrics

#### `contextd.workflows.version_validation.executions`

- **Type**: Counter (Int64)
- **Unit**: `{execution}`
- **Description**: Total number of version validation workflow executions
- **Labels**: `status` (success/failure)
- **Use Cases**:
  - Track workflow execution rate
  - Calculate success rate
  - Alert on execution failures

**Example PromQL**:
```promql
# Workflow success rate
sum(rate(contextd_workflows_version_validation_executions_total{status="success"}[5m]))
/
sum(rate(contextd_workflows_version_validation_executions_total[5m]))
```

---

#### `contextd.workflows.version_validation.duration`

- **Type**: Histogram (Float64)
- **Unit**: `s` (seconds)
- **Description**: Duration of version validation workflow executions
- **Use Cases**:
  - Monitor workflow latency
  - Set SLOs for workflow execution time
  - Identify performance regressions

**Example PromQL**:
```promql
# P95 workflow latency
histogram_quantile(0.95,
  sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))

# Average duration
rate(contextd_workflows_version_validation_duration_sum[5m])
/
rate(contextd_workflows_version_validation_duration_count[5m])
```

---

#### `contextd.workflows.version_validation.mismatches`

- **Type**: Counter (Int64)
- **Unit**: `{mismatch}`
- **Description**: Number of version mismatches detected
- **Use Cases**:
  - Track version synchronization issues
  - Alert on high mismatch rates
  - Measure developer compliance with version management

**Example PromQL**:
```promql
# Version mismatch rate (per hour)
rate(contextd_workflows_version_validation_mismatches_total[1h])
```

---

#### `contextd.workflows.version_validation.matches`

- **Type**: Counter (Int64)
- **Unit**: `{match}`
- **Description**: Number of version matches detected
- **Use Cases**:
  - Track successful version synchronization
  - Calculate mismatch ratio

**Example PromQL**:
```promql
# Mismatch ratio
sum(rate(contextd_workflows_version_validation_mismatches_total[5m]))
/
sum(rate(contextd_workflows_version_validation_matches_total[5m]) + rate(contextd_workflows_version_validation_mismatches_total[5m]))
```

---

### Activity Metrics

#### `contextd.workflows.activity.duration`

- **Type**: Histogram (Float64)
- **Unit**: `s` (seconds)
- **Description**: Duration of workflow activity executions
- **Labels**:
  - `activity`: Activity name (FetchFileContent, PostVersionMismatchComment, etc.)
  - `file`: File path (for FetchFileContent)
- **Use Cases**:
  - Identify slow activities
  - Optimize activity performance
  - Track GitHub API latency

**Example PromQL**:
```promql
# Top 5 slowest activities
topk(5,
  rate(contextd_workflows_activity_duration_sum[5m])
  /
  rate(contextd_workflows_activity_duration_count[5m]))

# P95 latency by activity
histogram_quantile(0.95,
  sum(rate(contextd_workflows_activity_duration_bucket[5m])) by (le, activity))
```

---

#### `contextd.workflows.activity.errors`

- **Type**: Counter (Int64)
- **Unit**: `{error}`
- **Description**: Number of activity execution errors
- **Labels**:
  - `activity`: Activity name
  - `error_type`: Error category (invalid_path, rate_limit, api_error, etc.)
  - `file`: File path (for file-related activities)
- **Use Cases**:
  - Alert on specific error patterns
  - Troubleshoot activity failures
  - Monitor GitHub API issues

**Error Types**:
- `invalid_path`: Path validation failed
- `path_traversal`: Path traversal attempt detected
- `absolute_path`: Absolute path not allowed
- `client_creation`: Failed to create GitHub client
- `not_found`: GitHub API returned 404
- `rate_limit`: GitHub API rate limit exceeded
- `api_error`: Other GitHub API errors
- `decode_error`: Failed to decode file content
- `unknown`: Unclassified error

**Example PromQL**:
```promql
# Activity error rate by type
sum(rate(contextd_workflows_activity_errors_total[5m])) by (error_type)

# GitHub rate limit errors
sum(rate(contextd_workflows_activity_errors_total{error_type="rate_limit"}[5m]))
```

---

## Alerting Rules

### Recommended Prometheus Alerts

```yaml
groups:
  - name: contextd_workflows
    interval: 30s
    rules:
      # Alert on high version mismatch rate
      - alert: HighVersionMismatchRate
        expr: |
          rate(contextd_workflows_version_validation_mismatches_total[1h]) > 0.5
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "High version mismatch rate detected"
          description: "More than 0.5 version mismatches per hour"

      # Alert on workflow failures
      - alert: WorkflowFailureRate
        expr: |
          sum(rate(contextd_workflows_version_validation_executions_total{status="failure"}[5m]))
          /
          sum(rate(contextd_workflows_version_validation_executions_total[5m]))
          > 0.1
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "High workflow failure rate"
          description: "More than 10% of workflows are failing"

      # Alert on high activity error rate
      - alert: HighActivityErrorRate
        expr: |
          sum(rate(contextd_workflows_activity_errors_total[5m])) by (error_type) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High activity error rate for {{ $labels.error_type }}"
          description: "More than 1 error per second"

      # Alert on slow workflows
      - alert: SlowWorkflowExecution
        expr: |
          histogram_quantile(0.95,
            sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))
          > 120
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Slow workflow execution detected"
          description: "P95 workflow duration is over 2 minutes"

      # Alert on GitHub rate limiting
      - alert: GitHubRateLimitHit
        expr: |
          sum(rate(contextd_workflows_activity_errors_total{error_type="rate_limit"}[5m])) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "GitHub API rate limit exceeded"
          description: "Workflows are being rate limited by GitHub API"
```

---

## Grafana Dashboards

### Example Panels

**Workflow Execution Rate**:
```promql
rate(contextd_workflows_version_validation_executions_total[5m])
```

**Workflow Duration Percentiles (P50, P95, P99)**:
```promql
# P50
histogram_quantile(0.50, sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))
# P95
histogram_quantile(0.95, sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))
# P99
histogram_quantile(0.99, sum(rate(contextd_workflows_version_validation_duration_bucket[5m])) by (le))
```

**Version Matches vs Mismatches**:
```promql
rate(contextd_workflows_version_validation_matches_total[5m])
rate(contextd_workflows_version_validation_mismatches_total[5m])
```

**Activity Errors by Type**:
```promql
sum(rate(contextd_workflows_activity_errors_total[5m])) by (error_type)
```

**Activity Duration Heatmap**:
```promql
sum(rate(contextd_workflows_activity_duration_bucket[5m])) by (le, activity)
```

---

## Implementation Details

### Metrics Initialization

Metrics are initialized once during package init via `initMetrics()` in `internal/workflows/metrics.go`.

```go
func initMetrics() {
    meter := otel.Meter(instrumentationName)
    // ... metric creation ...
}

func init() {
    initMetrics()
}
```

### Metric Recording

Metrics are recorded in workflow and activity code:

```go
// In workflows (via deferred function)
defer func(start time.Time) {
    duration := workflow.Now(ctx).Sub(start).Seconds()
    logger.Info("Version validation completed", "duration_seconds", duration)
}(startTime)

// In activities (direct recording)
activityDuration.Record(ctx, duration,
    metric.WithAttributes(
        attribute.String("activity", activityName),
        attribute.String("file", input.Path)))
```

### Testing

Metrics are tested in `internal/workflows/version_validation_metrics_test.go`:

- `TestVersionValidationWorkflowMetrics`: Verifies workflow metrics
- `TestActivityMetrics`: Verifies activity metrics
- `TestMetricsInitialization`: Verifies metrics are initialized
- `BenchmarkWorkflowExecution`: Performance benchmarks

Run tests:
```bash
go test ./internal/workflows/... -run TestMetrics -v
```

---

## Best Practices

### 1. Use Labels Consistently

Always use the same label names across metrics:
- `status`: success/failure
- `activity`: Activity name
- `error_type`: Error category
- `file`: File path

### 2. Monitor Key Metrics

Essential metrics to monitor:
- Workflow execution rate and success rate
- P95/P99 latency
- Error rates by type
- GitHub API rate limits

### 3. Set Appropriate Alerts

- **Critical**: Workflow failure rate > 10%, GitHub rate limits
- **Warning**: High error rates, slow execution (P95 > 2min)
- **Info**: Version mismatch trends

### 4. Dashboard Organization

Group metrics by:
- Workflow overview (execution rate, duration, success rate)
- Activity performance (latency heatmaps, slow activities)
- Errors and failures (error types, GitHub API issues)

---

## Future Enhancements

Planned metrics additions:

1. **Plugin Validation Workflow Metrics**:
   - `contextd.workflows.plugin_validation.executions`
   - `contextd.workflows.plugin_validation.duration`
   - `contextd.workflows.plugin_validation.plugin_updates_required`
   - `contextd.workflows.plugin_validation.schema_validation_failures`

2. **Documentation Validation Metrics**:
   - `contextd.workflows.documentation_validation.executions`
   - `contextd.workflows.documentation_validation.issues_found`
   - `contextd.workflows.documentation_validation.agent_latency`

3. **GitHub API Metrics**:
   - `contextd.workflows.github.api_calls`
   - `contextd.workflows.github.rate_limit_remaining`
   - `contextd.workflows.github.retry_attempts`

---

## See Also

- [TEMPORAL_WORKFLOWS.md](../../../docs/TEMPORAL_WORKFLOWS.md) - Complete workflow documentation
- [version_validation_metrics_test.go](./version_validation_metrics_test.go) - Metrics test examples
- [OpenTelemetry Go Metrics](https://opentelemetry.io/docs/languages/go/instrumentation/#metrics) - Official documentation

---

*Last Updated: 2025-12-29*
*Author: Claude Code (Sonnet 4.5)*
