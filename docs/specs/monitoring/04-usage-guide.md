# Monitoring Usage Guide

## Overview

Practical guide for using the contextd monitoring system. Covers starting services, accessing dashboards, querying metrics, viewing traces, and troubleshooting common issues.

## Quick Start

### 1. Start Monitoring Stack

```bash
# Navigate to project root
cd /home/dahendel/projects/contextd

# Start all services
docker-compose up -d

# Verify all services are healthy
docker-compose ps

# Expected output:
# contextd-otel-collector  running  0.0.0.0:4317-4318->4317-4318/tcp
# contextd-metrics         running  0.0.0.0:8428->8428/tcp
# contextd-grafana         running  0.0.0.0:3001->3000/tcp
# contextd-jaeger          running  0.0.0.0:16686->16686/tcp
# contextd-qdrant          running  0.0.0.0:6333-6334->6333-6334/tcp
# contextd-tei             running  0.0.0.0:8080->8080/tcp
```

### 2. Start contextd

```bash
# Build contextd (if not already built)
go build -o contextd ./cmd/contextd/

# Run contextd (API mode)
./contextd

# Or run in MCP mode
./contextd --mcp

# Verify contextd is exporting metrics
# Check logs for OTEL initialization
# 2025-11-04T12:00:00.000Z INFO Telemetry initialized successfully
```

### 3. Access UIs

**Grafana** (Primary Dashboard):
```bash
# Web browser
open http://localhost:3001

# Login credentials
Username: admin
Password: admin

# Default dashboard: contextd Overview
```

**Jaeger** (Trace Viewer):
```bash
# Web browser
open http://localhost:16686

# Select service: contextd
# Click "Find Traces"
```

**VictoriaMetrics** (Direct Queries):
```bash
# Web browser
open http://localhost:8428/vmui

# Or use curl
curl 'http://localhost:8428/api/v1/query?query=up'
```

---

## Grafana Dashboards

### Overview Dashboard

**Location**: Home → contextd Overview

**Panels**:

1. **MCP Tool Calls** (Top Row)
   - Total calls by tool name
   - Success/error breakdown
   - Rate over time

2. **Checkpoint Operations** (Second Row)
   - Operations by type (create, search, list, delete)
   - Operation rate
   - Average search score

3. **Embedding Metrics** (Third Row)
   - Total tokens consumed
   - Total cost (USD)
   - Operations by provider (OpenAI vs TEI)
   - P95 latency

4. **Database Operations** (Fourth Row, when implemented)
   - Operations by type
   - P95 query latency
   - Error rate

**Time Range**: Last 6 hours (adjustable)
**Refresh**: 30s (adjustable)

**Example Workflow**:
```
1. Open dashboard
2. Select time range (e.g., Last 1 hour)
3. Review MCP tool success rate
   - Green = >95% success rate
   - Yellow = 90-95%
   - Red = <90%
4. Check embedding costs
   - Monthly projection shown
5. Review checkpoint search scores
   - P95 should be >0.7 for good relevance
```

---

### Testing Dashboard

**Location**: Dashboards → contextd Testing & Quality

**Panels**:

1. **Test Coverage** (Top Row)
   - Overall coverage percentage
   - Coverage by package
   - Trend over time

2. **Bug Tracking** (Second Row)
   - Total bugs by severity
   - Bugs by status (open, fixed, verified)
   - Bug creation rate

3. **Regression Tests** (Third Row)
   - Total regression tests
   - Tests by category (bugs, security, performance)
   - Test execution time

4. **Quality Metrics** (Fourth Row)
   - Code quality score
   - Technical debt
   - Code complexity

**Note**: Test metrics are planned (Phase 4). Dashboard shows placeholder panels.

---

### Agents & Skills Dashboard

**Location**: Dashboards → contextd Agents & Skills

**Panels**:

1. **Agent Performance** (Top Row)
   - Agent executions by type
   - Average execution time
   - Success rate by agent

2. **Skill Operations** (Second Row)
   - Skills created
   - Skills applied
   - Skill success rate

3. **Time Saved** (Third Row)
   - Estimated time saved by automation
   - Most valuable skills
   - ROI metrics

**Note**: Skills metrics are planned (Phase 2). Dashboard shows placeholder panels.

---

## Querying Metrics

### VictoriaMetrics Query UI

**Access**: http://localhost:8428/vmui

**Example Queries**:

#### 1. Total MCP Tool Calls
```promql
sum(contextd_mcp_tool_calls_total) by (tool_name)
```

#### 2. Success Rate by Tool
```promql
sum(rate(contextd_mcp_tool_calls_total{status="success"}[5m])) by (tool_name)
/
sum(rate(contextd_mcp_tool_calls_total[5m])) by (tool_name)
* 100
```

#### 3. P95 MCP Tool Latency
```promql
histogram_quantile(0.95,
  rate(contextd_mcp_tool_duration_seconds_bucket[5m])
) by (tool_name)
```

#### 4. Checkpoint Operations Per Second
```promql
rate(contextd_checkpoint_operations_total[5m])
```

#### 5. Average Checkpoint Search Score
```promql
histogram_quantile(0.50,
  rate(contextd_checkpoint_search_score_bucket[5m])
)
```

#### 6. Embedding Cost Per Day
```promql
increase(contextd_embedding_cost_total[1d])
```

#### 7. Monthly Cost Projection
```promql
increase(contextd_embedding_cost_total[1d]) * 30
```

#### 8. Tokens Per Operation
```promql
rate(contextd_embedding_tokens_total[5m])
/
rate(contextd_embedding_operations_total[5m])
```

---

### Using curl for Queries

**Instant Query** (current value):
```bash
curl -G 'http://localhost:8428/api/v1/query' \
  --data-urlencode 'query=contextd_mcp_tool_calls_total'
```

**Range Query** (time series):
```bash
curl -G 'http://localhost:8428/api/v1/query_range' \
  --data-urlencode 'query=rate(contextd_mcp_tool_calls_total[5m])' \
  --data-urlencode 'start=2025-11-04T10:00:00Z' \
  --data-urlencode 'end=2025-11-04T12:00:00Z' \
  --data-urlencode 'step=60s'
```

**List All Metrics**:
```bash
curl 'http://localhost:8428/api/v1/label/__name__/values'
```

**List Label Values**:
```bash
# List all tool names
curl -G 'http://localhost:8428/api/v1/label/tool_name/values' \
  --data-urlencode 'match[]=contextd_mcp_tool_calls_total'
```

---

## Viewing Traces

### Jaeger UI

**Access**: http://localhost:16686

**Finding Traces**:

1. **By Service**:
   ```
   Service: contextd
   Operation: All
   Lookback: Last 1 hour
   Click "Find Traces"
   ```

2. **By Operation**:
   ```
   Service: contextd
   Operation: checkpoint.create
   Click "Find Traces"
   ```

3. **By Duration**:
   ```
   Service: contextd
   Min Duration: 500ms
   Click "Find Traces"
   ```

4. **By Tags**:
   ```
   Tags: error=true
   Click "Find Traces"
   ```

**Trace Details**:

**Example Trace Structure**:
```
checkpoint.create (342ms)
├─ embedding.batch (234ms)
│  └─ openai.create_embeddings (223ms)
├─ vectorstore.insert (56ms)
│  └─ qdrant.insert (45ms)
└─ database.create_checkpoint (12ms)
```

**Span Attributes**:
- `checkpoint_id`: UUID of checkpoint
- `project_path`: Project directory
- `tokens_used`: Tokens consumed
- `embedding_cost`: Cost in USD
- `from_cache`: Whether embedding was cached

**Error Traces**:
- Red highlights on failed spans
- Error message in span logs
- Stack trace if available

---

### Using Jaeger API

**Get Services**:
```bash
curl http://localhost:16686/api/services
```

**Get Operations**:
```bash
curl http://localhost:16686/api/services/contextd/operations
```

**Search Traces**:
```bash
curl -G http://localhost:16686/api/traces \
  --data-urlencode 'service=contextd' \
  --data-urlencode 'limit=10' \
  --data-urlencode 'lookback=1h'
```

---

## Generating Test Data

### Create Checkpoints

```bash
# Single checkpoint
./ctxd checkpoint save "Test checkpoint" \
  --description "Testing monitoring" \
  --tag "test" \
  --tag "monitoring"

# Multiple checkpoints (bash loop)
for i in {1..10}; do
  ./ctxd checkpoint save "Checkpoint $i" \
    --description "Testing monitoring - iteration $i"
  sleep 2
done
```

**Expected Metrics**:
```promql
contextd_checkpoint_operations_total{operation="create"} += 10
contextd_embedding_operations_total += 10
contextd_embedding_tokens_total += ~1000-2000 (depends on text length)
```

---

### Search Checkpoints

```bash
# Single search
./ctxd checkpoint search "monitoring"

# Multiple searches
for query in "test" "monitoring" "checkpoint" "embedding"; do
  ./ctxd checkpoint search "$query"
  sleep 1
done
```

**Expected Metrics**:
```promql
contextd_checkpoint_operations_total{operation="search"} += 4
contextd_checkpoint_search_score (histogram with scores)
```

---

### Generate Embeddings

```bash
# Direct embedding API call (if exposed)
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "input": "Test embedding generation",
    "model": "BAAI/bge-small-en-v1.5"
  }'
```

**Expected Metrics**:
```promql
contextd_embedding_operations_total{provider="tei"} += 1
contextd_embedding_duration_seconds (histogram with latency)
```

---

## Monitoring Health

### Service Health Checks

**Docker Compose**:
```bash
# All services
docker-compose ps

# Specific service
docker inspect contextd-otel-collector --format='{{.State.Health.Status}}'
# Expected: healthy
```

**Individual Services**:

**OTEL Collector**:
```bash
curl http://localhost:13133/
# Expected: 200 OK
```

**VictoriaMetrics**:
```bash
curl http://localhost:8428/health
# Expected: 200 OK
```

**Jaeger**:
```bash
curl http://localhost:14269/
# Expected: 200 OK
```

**Grafana**:
```bash
curl http://localhost:3001/api/health
# Expected: {"database":"ok","version":"..."}
```

**Qdrant**:
```bash
curl http://localhost:6333/health
# Expected: {"status":"ok"}
```

**TEI**:
```bash
curl http://localhost:8080/health
# Expected: 200 OK
```

---

### Checking Metric Export

**1. Check OTEL Collector Logs**:
```bash
docker logs -f contextd-otel-collector

# Expected output (every ~10s):
# 2025-11-04T12:00:00.000Z info MetricsExporter {"#metrics": 42}
# 2025-11-04T12:00:05.000Z info TracesExporter {"#spans": 15}
```

**2. Query VictoriaMetrics**:
```bash
# Check if metrics are being written
curl -G 'http://localhost:8428/api/v1/query' \
  --data-urlencode 'query=up'

# Check specific contextd metrics
curl -G 'http://localhost:8428/api/v1/query' \
  --data-urlencode 'query=contextd_checkpoint_operations_total'
```

**3. Check Grafana Datasource**:
```bash
# Open Grafana
open http://localhost:3001

# Navigate to Configuration → Data sources
# Click on VictoriaMetrics
# Click "Test" button
# Expected: "Data source is working"
```

---

## Troubleshooting

### Metrics Not Appearing in Grafana

**Symptom**: Dashboard panels show "No data"

**Diagnosis**:

1. **Check contextd is running**:
   ```bash
   ps aux | grep contextd
   # Should show running process
   ```

2. **Check OTEL environment variables**:
   ```bash
   # In contextd startup
   export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
   export OTEL_SERVICE_NAME=contextd
   export OTEL_ENVIRONMENT=development
   ```

3. **Verify OTEL Collector is receiving data**:
   ```bash
   docker logs contextd-otel-collector | grep -i "metric"
   # Should show metrics being received
   ```

4. **Check VictoriaMetrics has data**:
   ```bash
   curl 'http://localhost:8428/api/v1/query?query=contextd_checkpoint_operations_total'
   # Should return data with values
   ```

5. **Verify Grafana datasource**:
   - Open Grafana → Configuration → Data sources
   - Click VictoriaMetrics
   - Click "Test" button
   - Should show "Data source is working"

**Resolution**:
```bash
# Restart services
docker-compose restart otel-collector victoriametrics grafana

# Restart contextd
./contextd
```

---

### Traces Not Appearing in Jaeger

**Symptom**: No traces found when searching in Jaeger UI

**Diagnosis**:

1. **Check Jaeger is running**:
   ```bash
   curl http://localhost:16686/api/services
   # Should return list of services including "contextd"
   ```

2. **Check OTEL Collector is forwarding traces**:
   ```bash
   docker logs contextd-otel-collector | grep -i "trace"
   # Should show traces being exported
   ```

3. **Check contextd is creating spans**:
   ```bash
   # In contextd logs
   grep -i "span" contextd.log
   ```

**Resolution**:
```bash
# Restart Jaeger and OTEL Collector
docker-compose restart jaeger otel-collector

# Generate test traces
./ctxd checkpoint save "Test trace"

# Wait 10-15 seconds for batching
sleep 15

# Search in Jaeger UI
open http://localhost:16686
```

---

### High Embedding Costs

**Symptom**: `contextd_embedding_cost_total` increasing rapidly

**Diagnosis**:

1. **Check token usage**:
   ```promql
   rate(contextd_embedding_tokens_total[5m])
   ```

2. **Check operations count**:
   ```promql
   rate(contextd_embedding_operations_total[5m])
   ```

3. **Check tokens per operation**:
   ```promql
   rate(contextd_embedding_tokens_total[5m])
   / rate(contextd_embedding_operations_total[5m])
   ```

4. **Identify source**:
   ```bash
   # Check checkpoint descriptions (might be too long)
   # Check cache hit rate (should be >30%)
   ```

**Resolution**:

1. **Enable TEI (free local embeddings)**:
   ```bash
   # docker-compose.yml already has TEI
   docker-compose up -d tei

   # Configure contextd to use TEI
   export EMBEDDING_BASE_URL=http://localhost:8080/v1
   export EMBEDDING_MODEL=BAAI/bge-small-en-v1.5
   # No OPENAI_API_KEY needed
   ```

2. **Optimize checkpoint text length**:
   ```go
   // Limit description length
   if len(description) > 1000 {
       description = description[:1000]
   }
   ```

3. **Increase cache hit rate**:
   ```bash
   # Check cache settings in embedding service
   EnableCaching: true
   ```

---

### Slow Query Performance

**Symptom**: Dashboard loading slowly or timeouts

**Diagnosis**:

1. **Check VictoriaMetrics query duration**:
   ```bash
   # In VictoriaMetrics UI (vmui)
   # Look at query execution time at bottom of page
   ```

2. **Check cardinality**:
   ```bash
   curl 'http://localhost:8428/api/v1/label/__name__/values' | jq length
   # Should be <1000 metrics

   curl 'http://localhost:8428/api/v1/label/tool_name/values'
   # Should be ~10 values (not thousands)
   ```

3. **Check data retention**:
   ```bash
   # Verify retention is reasonable (12 months default)
   docker inspect contextd-metrics | grep retentionPeriod
   ```

**Resolution**:

1. **Reduce query time range**:
   ```
   # In Grafana
   Change time range from "Last 30 days" to "Last 6 hours"
   ```

2. **Optimize PromQL queries**:
   ```promql
   # BAD: Queries entire history
   sum(contextd_checkpoint_operations_total)

   # GOOD: Queries rate over recent window
   sum(rate(contextd_checkpoint_operations_total[5m]))
   ```

3. **Reduce cardinality** (if needed):
   ```go
   // Avoid high-cardinality labels
   // BAD: attribute.String("checkpoint_id", id)  // Millions of values
   // GOOD: attribute.String("operation", "create")  // ~5 values
   ```

---

### OTEL Collector Not Starting

**Symptom**: `docker-compose ps` shows collector as unhealthy or restarting

**Diagnosis**:

1. **Check logs**:
   ```bash
   docker logs contextd-otel-collector
   # Look for error messages
   ```

2. **Common errors**:
   - `failed to load config`: YAML syntax error
   - `failed to start receiver`: Port already in use
   - `failed to export`: Backend (VM/Jaeger) not available

**Resolution**:

1. **Validate config**:
   ```bash
   # Install otelcol-contrib locally
   # brew install opentelemetry-collector-contrib

   otelcol-contrib validate --config monitoring/otel-collector-config.yaml
   ```

2. **Check ports**:
   ```bash
   lsof -i :4318  # OTLP HTTP
   lsof -i :4317  # OTLP gRPC
   # Should not be in use by other processes
   ```

3. **Verify backends**:
   ```bash
   curl http://localhost:8428/health  # VictoriaMetrics
   curl http://localhost:14269/       # Jaeger
   ```

4. **Restart with dependencies**:
   ```bash
   docker-compose up -d victoriametrics jaeger
   sleep 5
   docker-compose up -d otel-collector
   ```

---

### Grafana Dashboard Not Loading

**Symptom**: Dashboard shows errors or blank panels

**Diagnosis**:

1. **Check Grafana logs**:
   ```bash
   docker logs contextd-grafana | grep -i error
   ```

2. **Verify datasource**:
   - Open Grafana → Configuration → Data sources
   - Click VictoriaMetrics
   - Click "Test"
   - Should succeed

3. **Check dashboard JSON**:
   ```bash
   # Validate JSON syntax
   jq . monitoring/grafana/dashboard-configs/contextd-overview.json
   ```

**Resolution**:

1. **Reimport dashboard**:
   ```bash
   # In Grafana UI
   # Dashboards → Browse → Import
   # Upload contextd-overview.json
   ```

2. **Fix datasource UID**:
   ```bash
   # Check datasource UID in Grafana
   # Configuration → Data sources → VictoriaMetrics → Settings
   # Copy UID

   # Update dashboard JSON
   # "datasource": {"uid": "CORRECT_UID"}
   ```

3. **Clear Grafana cache**:
   ```bash
   docker-compose restart grafana
   ```

---

## Performance Tips

### 1. Use Time Range Filters

**Grafana**:
- Default to "Last 6 hours" instead of "Last 30 days"
- Use auto-refresh: 30s or 1m (not 5s)

**PromQL**:
```promql
# Query last 5 minutes of data (fast)
rate(contextd_mcp_tool_calls_total[5m])

# Don't query all history (slow)
sum(contextd_mcp_tool_calls_total)
```

---

### 2. Optimize Metric Cardinality

**Good** (low cardinality):
```go
attribute.String("operation", "create")  // ~5-10 values
attribute.String("status", "success")    // 2-3 values
```

**Bad** (high cardinality):
```go
attribute.String("user_id", "...")       // Thousands of values
attribute.String("checkpoint_id", "...")  // Millions of values
```

---

### 3. Batch Metric Recording

**Good**:
```go
// Record multiple metrics in one operation
for _, result := range results {
    meters.CheckpointSearchScore.Record(ctx, result.Score)
}
```

**Bad**:
```go
// Don't make individual OTEL API calls for each point
for _, result := range results {
    // This creates excessive overhead
}
```

---

### 4. Use Aggregation in Queries

**Good** (aggregated):
```promql
sum(rate(contextd_checkpoint_operations_total[5m])) by (operation)
```

**Bad** (raw time series):
```promql
contextd_checkpoint_operations_total
```

---

## Advanced Usage

### Custom Dashboards

**Creating New Dashboard**:

1. **In Grafana UI**:
   - Click "+" → Create Dashboard
   - Add Panel
   - Select VictoriaMetrics datasource
   - Enter PromQL query
   - Configure visualization
   - Save dashboard

2. **Export to JSON**:
   - Dashboard Settings → JSON Model
   - Copy JSON
   - Save to `monitoring/grafana/dashboard-configs/custom-dashboard.json`

3. **Auto-provision** (optional):
   ```bash
   # Add to monitoring/grafana/dashboards/dashboards.yml
   # Restart Grafana
   docker-compose restart grafana
   ```

---

### Alerting (Future)

**Setting Up Alerts** (when implemented):

1. **In Grafana**:
   - Open dashboard panel
   - Panel → Edit → Alert tab
   - Create alert rule
   - Configure threshold
   - Set notification channel

2. **Example Alert**:
   ```yaml
   Name: High MCP Error Rate
   Condition: rate(contextd_mcp_tool_calls_total{status="error"}[5m]) > 0.1
   For: 5m
   Actions: Send to Slack
   ```

---

### Exporting Data

**Export Metrics to CSV**:
```bash
# Query VictoriaMetrics
curl -G 'http://localhost:8428/api/v1/query_range' \
  --data-urlencode 'query=rate(contextd_checkpoint_operations_total[5m])' \
  --data-urlencode 'start=2025-11-04T00:00:00Z' \
  --data-urlencode 'end=2025-11-04T23:59:59Z' \
  --data-urlencode 'step=60s' \
  | jq -r '.data.result[].values[] | @csv' > metrics.csv
```

**Export Traces**:
```bash
# Use Jaeger API
curl 'http://localhost:16686/api/traces?service=contextd&limit=100' \
  > traces.json
```

---

## References

- [01-architecture.md](./01-architecture.md) - Architecture overview
- [02-metrics-catalog.md](./02-metrics-catalog.md) - All metrics reference
- [03-implementation-status.md](./03-implementation-status.md) - Current status
- [05-future-work.md](./05-future-work.md) - Future enhancements
- VictoriaMetrics PromQL: https://docs.victoriametrics.com/MetricsQL.html
- Jaeger UI Guide: https://www.jaegertracing.io/docs/latest/frontend-ui/
- Grafana Docs: https://grafana.com/docs/grafana/latest/
