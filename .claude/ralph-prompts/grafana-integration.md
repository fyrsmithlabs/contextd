# Ralph Wiggum Prompt: Grafana Integration for contextd

## Objective
Use the Grafana MCP to create a complete observability stack for contextd, ensuring every dashboard and every panel has data and meaningful metrics from what contextd already provides.

## Context
contextd emits OpenTelemetry metrics across multiple services:
- Workflows: `contextd.workflows.*` metrics
- Context Folding: `folding.*` metrics
- Compression: `compression.*` metrics
- Checkpoint: `contextd.checkpoint.*` metrics
- Remediation: `contextd.remediation.*` metrics
- ReasoningBank: (needs investigation - may not have metrics yet)

Reference documentation:
- `/home/dahendel/projects/contextd/internal/workflows/METRICS.md`
- `/home/dahendel/projects/contextd/internal/folding/telemetry.go`
- `/home/dahendel/projects/contextd/internal/compression/service.go`
- `/home/dahendel/projects/contextd/internal/checkpoint/service.go`
- `/home/dahendel/projects/contextd/internal/remediation/service.go`

## Tasks

### 1. Verify Grafana MCP Connection
- Use `mcp__grafana__list_datasources` to confirm Prometheus/VictoriaMetrics datasource is available
- Record the datasource UID for use in dashboard panels

### 2. Check Existing contextd Dashboards
- Use `mcp__grafana__search_dashboards` with query "contextd"
- Use `mcp__grafana__search_folders` with query "contextd"
- Document any existing dashboards found

### 3. Create contextd Folder
- Use `mcp__grafana__create_folder` with:
  - title: "contextd"
  - uid: "contextd"
- Record the folder UID for organizing dashboards

### 4. Create Service Dashboards

For each contextd service, create a dashboard using `mcp__grafana__update_dashboard`:

#### Dashboard 1: Workflows Overview
**Panels to create:**
- Workflow Execution Rate: `rate(contextd_workflows_version_validation_executions_total[5m])`
- Workflow Success Rate: Success executions / total executions
- Workflow Duration (P50, P95, P99): `histogram_quantile()` on `contextd_workflows_version_validation_duration_bucket`
- Version Matches vs Mismatches: Stacked area chart
- Activity Duration by Activity: P95 latency grouped by `activity` label
- Activity Errors by Type: Errors grouped by `error_type` label

#### Dashboard 2: Context Folding
**Panels to create:**
- Branch Creation Rate: `rate(folding_branch_created_total[5m])`
- Branch Active Count: `folding_branch_active_count` (gauge)
- Branch Outcomes: Success/timeout/failed rates stacked
- Branch Duration Distribution: Histogram percentiles
- Budget Consumed Distribution: `folding_budget_consumed_tokens` histogram
- Budget Utilization: `folding_budget_utilization_ratio` with thresholds

#### Dashboard 3: Compression & Checkpoints
**Panels to create:**
- Compression Operations Rate: `rate(compression_operations_total[5m])`
- Compression Duration: `compression_duration_seconds` percentiles
- Compression Ratio: `compression_ratio` histogram
- Compression Quality Score: `compression_quality_score` with thresholds
- Checkpoint Saves/Resumes: `contextd_checkpoint_saves_total`, `contextd_checkpoint_resumes_total`
- Checkpoint Count: `contextd_checkpoint_count` gauge

#### Dashboard 4: Remediation & Memory
**Panels to create:**
- Remediation Searches: `rate(contextd_remediation_searches_total[5m])`
- Remediation Records: `rate(contextd_remediation_records_total[5m])`
- Remediation Feedback: `rate(contextd_remediation_feedbacks_total[5m])`
- ReasoningBank metrics: **INVESTIGATE** - check if metrics exist

### 5. Test Each Panel for Data

For each panel created:
- Use `mcp__grafana__query_prometheus` to verify the query returns data
- Test with time range: last 6 hours
- Document panels that return empty results
- For empty results:
  - Check if the metric exists: Use `mcp__grafana__list_prometheus_metric_names` with regex filter
  - Check if the query syntax is correct
  - Identify if this is a gap (metric not instrumented)

### 6. Document Gaps and Create Issues

For each gap identified (missing metrics or no data):
- Create a GitHub issue using the GitHub MCP (if available) or document for manual creation
- Issue template:
  ```
  Title: [Metrics Gap] Missing <metric_name> for <service>

  Description:
  - Service: <service_name>
  - Expected Metric: <metric_name>
  - Dashboard Panel: <panel_name>
  - Impact: Cannot monitor <what can't be monitored>

  Implementation:
  - Add metric in <file_path>
  - Follow existing pattern from <similar_service>
  - Add test coverage

  Labels: observability, metrics, enhancement
  ```

### 7. Validate Dashboard Collection

- Use `mcp__grafana__search_dashboards` to list all dashboards in contextd folder
- Use `mcp__grafana__get_dashboard_summary` for each dashboard
- Generate a summary report of:
  - Total dashboards created
  - Total panels created
  - Panels with data vs panels without data
  - List of gaps requiring instrumentation

## Success Criteria

- [ ] contextd folder exists in Grafana
- [ ] All service dashboards created (Workflows, Folding, Compression/Checkpoints, Remediation/Memory)
- [ ] Every panel query has been tested
- [ ] Data presence documented for each panel
- [ ] Gaps identified and issues created for missing metrics
- [ ] Summary report generated

## Output Format

Provide a structured report:

```markdown
# Grafana Integration Report

## Dashboards Created
1. Workflows Overview (uid: contextd-workflows) - X panels
2. Context Folding (uid: contextd-folding) - X panels
3. Compression & Checkpoints (uid: contextd-compression-checkpoints) - X panels
4. Remediation & Memory (uid: contextd-remediation-memory) - X panels

## Data Coverage
- Panels with data: X/Y (Z%)
- Panels without data: X/Y (Z%)

## Metrics Gaps Identified
1. [Issue #XX] Missing reasoningbank.searches_total
2. [Issue #XX] Missing reasoningbank.records_total
3. ...

## Recommendations
- High priority metrics to add
- Dashboard improvements
- Alert rules to consider
```

## Notes for Ralph

- Use Grafana best practices: https://grafana.com/docs/grafana/latest/visualizations/dashboards/
- Follow the dashboard JSON structure from https://grafana.com/docs/grafana/latest/developers/http_api/dashboard/
- Test queries before adding to dashboards to avoid empty panels
- For panels without data, investigate whether it's a query issue or missing metric instrumentation
- Create issues individually for each gap so they can be tracked and prioritized separately
