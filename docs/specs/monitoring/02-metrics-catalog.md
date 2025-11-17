# Metrics Catalog

## Overview

Complete reference of all metrics instrumented in contextd. All metrics follow OpenTelemetry semantic conventions and are exported via OTLP to VictoriaMetrics.

**Total Metrics**: 31 (19 implemented, 12 planned)

## Metric Naming Convention

```
contextd_<component>_<metric_name>
```

**Components**:
- `mcp` - Model Context Protocol tools
- `checkpoint` - Checkpoint service
- `remediation` - Remediation service
- `skill` - Skills service
- `embedding` - Embedding generation
- `database` - Database operations
- `test` - Testing metrics

**Metric Types**:
- **Counter**: Monotonically increasing value (e.g., total requests)
- **Histogram**: Distribution of values (e.g., latency, scores)
- **Gauge**: Point-in-time value (e.g., current total, percentage)

## Business Metrics

### MCP Service Metrics (2)

#### 1. contextd_mcp_tool_calls_total

**Type**: Counter
**Unit**: `{call}`
**Description**: Total number of MCP tool calls
**Status**: ✅ Implemented

**Labels**:
- `tool_name` (string): Name of MCP tool called
  - Values: `checkpoint_save`, `checkpoint_search`, `checkpoint_list`, `remediation_save`, `remediation_search`, `troubleshoot`, `list_patterns`, `index_repository`, `status`
- `status` (string): Call result status
  - Values: `success`, `error`

**Example Values**:
```promql
# Total checkpoint_save calls
contextd_mcp_tool_calls_total{tool_name="checkpoint_save",status="success"} 245

# Failed remediation searches
contextd_mcp_tool_calls_total{tool_name="remediation_search",status="error"} 3
```

**Usage**:
```go
meters.MCPToolCalls.Add(ctx, 1, metric.WithAttributes(
    attribute.String("tool_name", "checkpoint_save"),
    attribute.String("status", "success"),
))
```

**Dashboard Queries**:
```promql
# Rate of MCP calls per second
rate(contextd_mcp_tool_calls_total[5m])

# Success rate by tool
sum(rate(contextd_mcp_tool_calls_total{status="success"}[5m])) by (tool_name)
/ sum(rate(contextd_mcp_tool_calls_total[5m])) by (tool_name)
```

---

#### 2. contextd_mcp_tool_duration_seconds

**Type**: Histogram
**Unit**: `s` (seconds)
**Description**: MCP tool execution duration in seconds
**Status**: ✅ Implemented

**Labels**:
- `tool_name` (string): Name of MCP tool
  - Values: Same as `contextd_mcp_tool_calls_total`

**Buckets** (default OpenTelemetry):
```
0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
```

**Example Values**:
```promql
# P95 latency for checkpoint_search
histogram_quantile(0.95, contextd_mcp_tool_duration_seconds{tool_name="checkpoint_search"}) 0.234

# Average duration
avg(contextd_mcp_tool_duration_seconds) 0.156
```

**Usage**:
```go
start := time.Now()
// ... execute tool ...
duration := time.Since(start).Seconds()

meters.MCPToolDuration.Record(ctx, duration, metric.WithAttributes(
    attribute.String("tool_name", "checkpoint_search"),
))
```

**Dashboard Queries**:
```promql
# P50, P95, P99 latencies
histogram_quantile(0.50, rate(contextd_mcp_tool_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(contextd_mcp_tool_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(contextd_mcp_tool_duration_seconds_bucket[5m]))
```

---

### Checkpoint Service Metrics (4)

#### 3. contextd_checkpoints_total

**Type**: Gauge (Observable)
**Unit**: `{checkpoint}`
**Description**: Total number of checkpoints stored
**Status**: ⏳ Planned (needs callback implementation)

**Labels**: None

**Example Values**:
```promql
contextd_checkpoints_total 1247
```

**Implementation Required**:
```go
// Register callback to query Qdrant for total count
_, err := meter.Int64ObservableGauge(
    "contextd_checkpoints_total",
    metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
        count, err := checkpointService.Count(ctx)
        if err != nil {
            return err
        }
        observer.Observe(int64(count))
        return nil
    }),
)
```

---

#### 4. contextd_checkpoint_operations_total

**Type**: Counter
**Unit**: `{operation}`
**Description**: Checkpoint operations performed
**Status**: ✅ Implemented

**Labels**:
- `operation` (string): Type of operation
  - Values: `create`, `search`, `list`, `update`, `delete`

**Example Values**:
```promql
contextd_checkpoint_operations_total{operation="create"} 458
contextd_checkpoint_operations_total{operation="search"} 892
```

**Usage**:
```go
meters.CheckpointOperations.Add(ctx, 1, metric.WithAttributes(
    attribute.String("operation", "create"),
))
```

**Dashboard Queries**:
```promql
# Operations per second by type
rate(contextd_checkpoint_operations_total[5m])

# Top operation types
topk(5, sum(rate(contextd_checkpoint_operations_total[5m])) by (operation))
```

---

#### 5. contextd_checkpoint_duration_seconds

**Type**: Histogram
**Unit**: `s` (seconds)
**Description**: Checkpoint operation duration
**Status**: ✅ Implemented

**Labels**:
- `operation` (string): Type of operation (same as above)

**Example Values**:
```promql
# P95 create latency
histogram_quantile(0.95, contextd_checkpoint_duration_seconds{operation="create"}) 0.342
```

**Usage**:
```go
start := time.Now()
// ... perform operation ...
meters.CheckpointDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
    attribute.String("operation", "create"),
))
```

---

#### 6. contextd_checkpoint_search_score

**Type**: Histogram
**Unit**: `1` (dimensionless, 0-1 range)
**Description**: Checkpoint search relevance scores
**Status**: ✅ Implemented

**Labels**: None

**Buckets** (custom for scores):
```
0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0
```

**Example Values**:
```promql
# Average search relevance
avg(contextd_checkpoint_search_score) 0.78

# Distribution of scores
rate(contextd_checkpoint_search_score_bucket[5m])
```

**Usage**:
```go
for _, result := range searchResults {
    meters.CheckpointSearchScore.Record(ctx, result.Score)
}
```

**Dashboard Queries**:
```promql
# P50 search relevance (median)
histogram_quantile(0.50, rate(contextd_checkpoint_search_score_bucket[5m]))

# Percentage of high-quality results (>0.8)
sum(rate(contextd_checkpoint_search_score_bucket{le="1.0"}[5m]))
- sum(rate(contextd_checkpoint_search_score_bucket{le="0.8"}[5m]))
```

---

### Remediation Service Metrics (4)

#### 7. contextd_remediations_total

**Type**: Gauge (Observable)
**Unit**: `{remediation}`
**Description**: Total number of remediations stored
**Status**: ⏳ Planned (needs callback implementation)

**Labels**: None

**Example Values**:
```promql
contextd_remediations_total 89
```

---

#### 8. contextd_remediation_operations_total

**Type**: Counter
**Unit**: `{operation}`
**Description**: Remediation operations performed
**Status**: ⏳ Planned (service integration needed)

**Labels**:
- `operation` (string): Type of operation
  - Values: `save`, `search`

**Example Values**:
```promql
contextd_remediation_operations_total{operation="save"} 89
contextd_remediation_operations_total{operation="search"} 456
```

---

#### 9. contextd_remediation_match_score

**Type**: Histogram
**Unit**: `1` (dimensionless, 0-1 range)
**Description**: Remediation hybrid match scores (70% semantic + 30% string)
**Status**: ⏳ Planned (service integration needed)

**Labels**: None

**Example Values**:
```promql
# Average match quality
avg(contextd_remediation_match_score) 0.68

# High-confidence matches (>0.75)
histogram_quantile(0.95, contextd_remediation_match_score_bucket)
```

**Implementation Note**: Score combines semantic similarity (embedding cosine) and string matching (Levenshtein distance).

---

#### 10. contextd_remediation_duration_seconds

**Type**: Histogram
**Unit**: `s` (seconds)
**Description**: Remediation operation duration
**Status**: ⏳ Planned (service integration needed)

**Labels**:
- `operation` (string): Type of operation (save, search)

**Example Values**:
```promql
histogram_quantile(0.95, contextd_remediation_duration_seconds{operation="search"}) 0.156
```

---

### Skills Service Metrics (4)

#### 11. contextd_skills_total

**Type**: Gauge (Observable)
**Unit**: `{skill}`
**Description**: Total number of skills stored
**Status**: ⏳ Planned (needs callback implementation)

**Labels**: None

**Example Values**:
```promql
contextd_skills_total 42
```

---

#### 12. contextd_skill_operations_total

**Type**: Counter
**Unit**: `{operation}`
**Description**: Skill operations performed
**Status**: ⏳ Planned (service integration needed)

**Labels**:
- `operation` (string): Type of operation
  - Values: `create`, `apply`, `search`, `update`, `delete`

**Example Values**:
```promql
contextd_skill_operations_total{operation="apply"} 234
contextd_skill_operations_total{operation="create"} 42
```

---

#### 13. contextd_skill_success_rate

**Type**: Gauge (Observable)
**Unit**: `1` (dimensionless, 0-1 range)
**Description**: Skill application success rate
**Status**: ⏳ Planned (needs callback + tracking)

**Labels**:
- `skill_id` (string): Unique skill identifier

**Example Values**:
```promql
contextd_skill_success_rate{skill_id="git-workflow"} 0.92
contextd_skill_success_rate{skill_id="test-pattern"} 0.85
```

**Implementation Note**: Requires tracking skill application outcomes (success/failure) and calculating rolling success rate.

---

#### 14. contextd_skill_duration_seconds

**Type**: Histogram
**Unit**: `s` (seconds)
**Description**: Skill operation duration
**Status**: ⏳ Planned (service integration needed)

**Labels**:
- `operation` (string): Type of operation

**Example Values**:
```promql
histogram_quantile(0.95, contextd_skill_duration_seconds{operation="apply"}) 2.45
```

---

### Embedding Service Metrics (4)

#### 15. contextd_embedding_duration_seconds

**Type**: Histogram
**Unit**: `s` (seconds)
**Description**: Embedding generation duration (includes API latency)
**Status**: ✅ Implemented

**Labels**:
- `provider` (string): Embedding provider
  - Values: `openai`, `tei`
- `model` (string): Model used
  - Values: `text-embedding-3-small`, `BAAI/bge-small-en-v1.5`

**Example Values**:
```promql
# P95 OpenAI latency
histogram_quantile(0.95, contextd_embedding_duration_seconds{provider="openai"}) 0.456

# TEI average latency
avg(contextd_embedding_duration_seconds{provider="tei"}) 0.123
```

**Usage**:
```go
// In pkg/embedding/embedding.go
start := time.Now()
// ... generate embedding ...
s.meters.EmbeddingDuration.Record(ctx, time.Since(start).Seconds())
```

---

#### 16. contextd_embedding_operations_total

**Type**: Counter
**Unit**: `{operation}`
**Description**: Total embedding generation operations
**Status**: ✅ Implemented

**Labels**:
- `provider` (string): Embedding provider (openai, tei)
- `model` (string): Model used

**Example Values**:
```promql
contextd_embedding_operations_total{provider="openai",model="text-embedding-3-small"} 1247
contextd_embedding_operations_total{provider="tei",model="BAAI/bge-small-en-v1.5"} 89
```

**Usage**:
```go
meters.EmbeddingOperations.Add(ctx, 1, metric.WithAttributes(
    attribute.String("provider", "openai"),
    attribute.String("model", s.config.Model),
))
```

---

#### 17. contextd_embedding_tokens_total

**Type**: Counter
**Unit**: `{token}`
**Description**: Total tokens consumed for embeddings
**Status**: ✅ Implemented

**Labels**:
- `model` (string): Model used

**Example Values**:
```promql
# Total tokens consumed
contextd_embedding_tokens_total{model="text-embedding-3-small"} 456789

# Tokens per day
increase(contextd_embedding_tokens_total[1d])
```

**Usage**:
```go
meters.EmbeddingTokens.Add(ctx, int64(totalTokens), metric.WithAttributes(
    attribute.String("model", s.config.Model),
))
```

**Dashboard Queries**:
```promql
# Token consumption rate (tokens/sec)
rate(contextd_embedding_tokens_total[5m])

# Projected monthly usage
increase(contextd_embedding_tokens_total[1d]) * 30
```

---

#### 18. contextd_embedding_cost_total

**Type**: Counter
**Unit**: `USD`
**Description**: Total embedding cost in USD (OpenAI: $0.02/1M tokens)
**Status**: ✅ Implemented

**Labels**:
- `model` (string): Model used

**Example Values**:
```promql
contextd_embedding_cost_total{model="text-embedding-3-small"} 9.14

# Cost per day
increase(contextd_embedding_cost_total[1d]) 0.42
```

**Usage**:
```go
cost := (float64(tokens) / 1_000_000) * 0.02
meters.EmbeddingCost.Add(ctx, cost, metric.WithAttributes(
    attribute.String("model", s.config.Model),
))
```

**Dashboard Queries**:
```promql
# Projected monthly cost
increase(contextd_embedding_cost_total[1d]) * 30

# Cost per operation
rate(contextd_embedding_cost_total[5m]) / rate(contextd_embedding_operations_total[5m])
```

---

## Infrastructure Metrics

### Database Metrics (3)

#### 19. contextd_database_operations_total

**Type**: Counter
**Unit**: `{operation}`
**Description**: Database operations performed
**Status**: ⏳ Planned (vectorstore integration needed)

**Labels**:
- `operation` (string): Type of operation
  - Values: `insert`, `search`, `delete`, `get`, `update`
- `database` (string): Database name
  - Values: `shared`, `project_<hash>`
- `collection` (string): Collection name
  - Values: `checkpoints`, `remediations`, `skills`, `documents`

**Example Values**:
```promql
contextd_database_operations_total{operation="insert",database="shared",collection="remediations"} 89
contextd_database_operations_total{operation="search",database="project_abc123",collection="checkpoints"} 1247
```

---

#### 20. contextd_database_duration_seconds

**Type**: Histogram
**Unit**: `s` (seconds)
**Description**: Database operation duration
**Status**: ⏳ Planned (vectorstore integration needed)

**Labels**:
- `operation` (string): Type of operation
- `database` (string): Database name

**Example Values**:
```promql
# P95 search latency
histogram_quantile(0.95, contextd_database_duration_seconds{operation="search"}) 0.045
```

---

#### 21. contextd_database_errors_total

**Type**: Counter
**Unit**: `{error}`
**Description**: Database errors encountered
**Status**: ⏳ Planned (vectorstore integration needed)

**Labels**:
- `error_type` (string): Type of error
  - Values: `connection_timeout`, `query_failed`, `not_found`, `permission_denied`
- `database` (string): Database name

**Example Values**:
```promql
contextd_database_errors_total{error_type="connection_timeout",database="shared"} 3
```

---

## Runtime Metrics (Planned - Phase 3)

### Go Runtime Metrics (4)

These metrics will be automatically collected via OpenTelemetry Go runtime instrumentation.

#### 22. process_runtime_go_mem_heap_alloc_bytes

**Type**: Gauge
**Description**: Bytes allocated and still in use
**Status**: ⏳ Planned

---

#### 23. process_runtime_go_mem_heap_sys_bytes

**Type**: Gauge
**Description**: Bytes obtained from system
**Status**: ⏳ Planned

---

#### 24. process_runtime_go_gc_duration_seconds

**Type**: Histogram
**Description**: GC pause duration
**Status**: ⏳ Planned

---

#### 25. process_runtime_go_goroutines

**Type**: Gauge
**Description**: Number of goroutines
**Status**: ⏳ Planned

---

## Testing Metrics (4)

### Quality Metrics

#### 26. contextd_test_coverage_percent

**Type**: Gauge (Observable)
**Unit**: `%`
**Description**: Overall test coverage percentage
**Status**: ⏳ Planned (CI integration needed)

**Labels**: None

**Example Values**:
```promql
contextd_test_coverage_percent 68.4
```

**Implementation**: Parse coverage report from CI and push to pushgateway or expose via /metrics endpoint.

---

#### 27. contextd_tests_total

**Type**: Gauge (Observable)
**Unit**: `{test}`
**Description**: Total number of tests
**Status**: ⏳ Planned (CI integration needed)

**Labels**:
- `type` (string): Test type
  - Values: `unit`, `integration`, `e2e`

**Example Values**:
```promql
contextd_tests_total{type="unit"} 245
contextd_tests_total{type="integration"} 34
```

---

#### 28. contextd_bugs_total

**Type**: Gauge (Observable)
**Unit**: `{bug}`
**Description**: Total bugs tracked in tests/regression/bugs/
**Status**: ⏳ Planned (filesystem scan needed)

**Labels**:
- `severity` (string): Bug severity
  - Values: `critical`, `high`, `medium`, `low`
- `status` (string): Bug status
  - Values: `open`, `fixed`, `verified`

**Example Values**:
```promql
contextd_bugs_total{severity="high",status="open"} 2
contextd_bugs_total{severity="medium",status="fixed"} 12
```

---

#### 29. contextd_regression_tests_total

**Type**: Gauge (Observable)
**Unit**: `{test}`
**Description**: Total regression tests
**Status**: ⏳ Planned (filesystem scan needed)

**Labels**:
- `category` (string): Test category
  - Values: `bugs`, `security`, `performance`

**Example Values**:
```promql
contextd_regression_tests_total{category="bugs"} 15
contextd_regression_tests_total{category="security"} 8
```

---

## HTTP Server Metrics (Planned)

### Echo Framework Metrics

#### 30. http_server_request_duration_seconds

**Type**: Histogram
**Description**: HTTP request duration
**Status**: ⏳ Planned (otelecho middleware)

**Labels**:
- `http.method` (string): HTTP method (GET, POST, etc.)
- `http.route` (string): Route pattern (/health, /checkpoints, etc.)
- `http.status_code` (int): Response status code

---

#### 31. http_server_requests_total

**Type**: Counter
**Description**: Total HTTP requests
**Status**: ⏳ Planned (otelecho middleware)

**Labels**:
- `http.method` (string): HTTP method
- `http.route` (string): Route pattern
- `http.status_code` (int): Response status code

---

## Metric Implementation Status

### Phase 1: Infrastructure (COMPLETED)
- ✅ OpenTelemetry initialization (pkg/telemetry)
- ✅ Metrics package (pkg/metrics)
- ✅ OTEL Collector deployment
- ✅ VictoriaMetrics + Grafana

### Phase 2: Business Metrics (67% Complete)
- ✅ MCP metrics (2/2)
- ✅ Checkpoint metrics (3/4) - Missing: checkpoints_total callback
- ✅ Embedding metrics (4/4)
- ⏳ Remediation metrics (0/4) - Service integration needed
- ⏳ Skills metrics (0/4) - Service integration needed
- ⏳ Database metrics (0/3) - Vectorstore integration needed

### Phase 3: Runtime Metrics (PLANNED)
- ⏳ Go runtime metrics (0/4)
- ⏳ HTTP server metrics (0/2)

### Phase 4: Testing Metrics (PLANNED)
- ⏳ Quality metrics (0/4)

## References

- OpenTelemetry Semantic Conventions: https://opentelemetry.io/docs/specs/semconv/
- Prometheus Metric Types: https://prometheus.io/docs/concepts/metric_types/
- VictoriaMetrics PromQL: https://docs.victoriametrics.com/MetricsQL.html
