# Future Work - Monitoring Enhancements

## Overview

Planned enhancements for the contextd monitoring system, organized by implementation phase and priority. This roadmap aligns with the overall contextd product roadmap while maintaining observability as code evolves.

## Phase 3: Runtime Metrics (Months 2-3)

### Go Runtime Instrumentation

**Priority**: HIGH
**Estimated Effort**: 2-4 hours
**Dependencies**: None

**Metrics to Add**:

1. **Memory Metrics**:
   - `process_runtime_go_mem_heap_alloc_bytes` - Current heap allocation
   - `process_runtime_go_mem_heap_sys_bytes` - Heap obtained from system
   - `process_runtime_go_mem_heap_inuse_bytes` - Bytes in in-use spans
   - `process_runtime_go_mem_heap_released_bytes` - Bytes released to OS
   - `process_runtime_go_mem_stack_inuse_bytes` - Stack memory in use

2. **GC Metrics**:
   - `process_runtime_go_gc_duration_seconds` - GC pause duration
   - `process_runtime_go_gc_count_total` - Total GC runs
   - `process_runtime_go_gc_pause_ns` - GC pause in nanoseconds

3. **Goroutine Metrics**:
   - `process_runtime_go_goroutines` - Number of goroutines
   - `process_runtime_go_goroutines_blocked` - Blocked goroutines

4. **System Metrics**:
   - `process_runtime_go_cgo_calls` - Number of cgo calls
   - `process_runtime_go_num_cpu` - Number of CPUs

**Implementation**:

```go
// pkg/telemetry/telemetry.go

import (
    "go.opentelemetry.io/contrib/instrumentation/runtime"
)

func Init(ctx context.Context, serviceName, environment, version string) (func(context.Context) error, error) {
    // ... existing OTEL setup ...

    // Enable runtime metrics collection
    err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
    if err != nil {
        return nil, fmt.Errorf("failed to start runtime instrumentation: %w", err)
    }

    // ... rest of initialization ...
}
```

**Grafana Dashboard Panel**:
```json
{
  "title": "Go Runtime Metrics",
  "targets": [
    {
      "expr": "process_runtime_go_mem_heap_alloc_bytes",
      "legendFormat": "Heap Allocated"
    },
    {
      "expr": "process_runtime_go_goroutines",
      "legendFormat": "Goroutines"
    },
    {
      "expr": "rate(process_runtime_go_gc_duration_seconds_sum[5m])",
      "legendFormat": "GC Duration"
    }
  ]
}
```

**Value**:
- Detect memory leaks early
- Optimize GC performance
- Monitor goroutine leaks
- Capacity planning

---

### HTTP Server Metrics

**Priority**: HIGH
**Estimated Effort**: 2-3 hours
**Dependencies**: None

**Metrics to Add**:

1. **Request Metrics**:
   - `http_server_requests_total` - Total requests by route/status
   - `http_server_request_duration_seconds` - Request latency distribution
   - `http_server_request_size_bytes` - Request body size
   - `http_server_response_size_bytes` - Response body size

2. **Connection Metrics**:
   - `http_server_active_requests` - Current active requests
   - `http_server_connections_total` - Total connections

**Labels**:
- `http.method` - GET, POST, DELETE, etc.
- `http.route` - /health, /checkpoints, /search
- `http.status_code` - 200, 404, 500, etc.

**Implementation**:

```go
// cmd/contextd/main.go or internal/api/server.go

import (
    "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func setupAPI(e *echo.Echo, serviceName string) {
    // Add OpenTelemetry middleware
    e.Use(otelecho.Middleware(serviceName,
        otelecho.WithTracerProvider(otel.GetTracerProvider()),
        otelecho.WithMeterProvider(otel.GetMeterProvider()),
    ))

    // ... register routes ...
}
```

**Grafana Dashboard Panel**:
```json
{
  "title": "HTTP Request Rate by Route",
  "targets": [
    {
      "expr": "sum(rate(http_server_requests_total[5m])) by (http_route)",
      "legendFormat": "{{http_route}}"
    }
  ]
}
```

**Value**:
- Monitor API performance
- Identify slow endpoints
- Track error rates by route
- Capacity planning

---

### Connection Pool Metrics

**Priority**: MEDIUM
**Estimated Effort**: 4-6 hours
**Dependencies**: Vectorstore implementation

**Metrics to Add**:

1. **Pool Metrics**:
   - `contextd_connection_pool_size` - Total connections in pool
   - `contextd_connection_pool_active` - Active connections
   - `contextd_connection_pool_idle` - Idle connections
   - `contextd_connection_pool_wait_duration_seconds` - Wait time for connection

2. **Connection Metrics**:
   - `contextd_connections_opened_total` - Total connections opened
   - `contextd_connections_closed_total` - Total connections closed
   - `contextd_connection_errors_total` - Connection errors

**Implementation**:

```go
// pkg/vectorstore/qdrant/pool.go (example)

type Pool struct {
    meters *metrics.Meters
    // ... pool fields ...
}

func (p *Pool) Get(ctx context.Context) (*Connection, error) {
    start := time.Now()

    // Record wait time
    defer func() {
        if p.meters != nil {
            p.meters.ConnectionPoolWaitDuration.Record(ctx,
                time.Since(start).Seconds())
        }
    }()

    // ... get connection ...

    // Record active connections
    if p.meters != nil {
        p.meters.ConnectionPoolActive.Add(ctx, 1)
    }

    return conn, nil
}
```

**Value**:
- Detect connection leaks
- Optimize pool size
- Identify connection bottlenecks

---

## Phase 4: Advanced Observability (Months 3-6)

### Continuous Profiling

**Priority**: MEDIUM
**Estimated Effort**: 1-2 days
**Dependencies**: None

**Tool Options**:

1. **pprof (Built-in)**:
   - Pros: Built into Go, no dependencies
   - Cons: Manual profiling, not continuous
   - Implementation: `import _ "net/http/pprof"`

2. **Pyroscope**:
   - Pros: Continuous profiling, flame graphs, low overhead
   - Cons: Additional service to run
   - Implementation: Pyroscope agent + server

3. **Grafana Phlare**:
   - Pros: Integrates with Grafana, open source
   - Cons: Newer project, less mature
   - Implementation: Phlare agent + server

**Recommended**: Start with pprof, migrate to Pyroscope if needed.

**pprof Implementation**:

```go
// cmd/contextd/main.go

import (
    "net/http"
    _ "net/http/pprof"
)

func main() {
    // ... existing setup ...

    // Enable pprof endpoints (dev mode only)
    if config.Environment == "development" {
        go func() {
            log.Println("pprof server listening on :6060")
            log.Println(http.ListenAndServe("localhost:6060", nil))
        }()
    }

    // ... start contextd ...
}
```

**pprof Endpoints**:
- `http://localhost:6060/debug/pprof/` - Index
- `http://localhost:6060/debug/pprof/heap` - Heap profile
- `http://localhost:6060/debug/pprof/goroutine` - Goroutine stacks
- `http://localhost:6060/debug/pprof/profile?seconds=30` - CPU profile (30s)
- `http://localhost:6060/debug/pprof/trace?seconds=5` - Execution trace (5s)

**Usage**:
```bash
# CPU profile (30 seconds)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Heap profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Interactive visualization
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

**Pyroscope Implementation** (future):

```yaml
# docker-compose.yml
services:
  pyroscope:
    image: pyroscope/pyroscope:latest
    ports:
      - "4040:4040"
    command:
      - "server"
```

```go
// pkg/telemetry/telemetry.go
import "github.com/pyroscope-io/client/pyroscope"

func Init(...) {
    // Start Pyroscope profiler
    pyroscope.Start(pyroscope.Config{
        ApplicationName: "contextd",
        ServerAddress:   "http://localhost:4040",
        ProfileTypes: []pyroscope.ProfileType{
            pyroscope.ProfileCPU,
            pyroscope.ProfileAllocObjects,
            pyroscope.ProfileAllocSpace,
            pyroscope.ProfileInuseObjects,
            pyroscope.ProfileInuseSpace,
        },
    })
}
```

**Value**:
- Identify CPU hotspots
- Find memory leaks
- Optimize performance
- Reduce resource usage

---

### Alerting System

**Priority**: HIGH (for production)
**Estimated Effort**: 2-3 days
**Dependencies**: None

**Alert Rules to Implement**:

#### 1. High Error Rate
```yaml
- alert: HighMCPErrorRate
  expr: |
    rate(contextd_mcp_tool_calls_total{status="error"}[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High MCP tool error rate detected"
    description: "MCP error rate is {{ $value }} errors/sec"
```

#### 2. Slow Embeddings
```yaml
- alert: SlowEmbeddings
  expr: |
    histogram_quantile(0.95,
      rate(contextd_embedding_duration_seconds_bucket[5m])
    ) > 2.0
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Embedding generation is slow"
    description: "P95 latency is {{ $value }}s (threshold: 2s)"
```

#### 3. Database Errors
```yaml
- alert: DatabaseErrors
  expr: |
    rate(contextd_database_errors_total[5m]) > 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Database errors detected"
    description: "{{ $value }} database errors/sec"
```

#### 4. Memory Usage High
```yaml
- alert: HighMemoryUsage
  expr: |
    process_runtime_go_mem_heap_alloc_bytes / 1024 / 1024 > 1024
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "High memory usage detected"
    description: "Heap allocation is {{ $value }}MB"
```

#### 5. Service Down
```yaml
- alert: ContextdDown
  expr: up{job="contextd"} == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "contextd service is down"
    description: "contextd has been down for 1 minute"
```

**Notification Channels**:

1. **Slack** (recommended):
   ```bash
   # In Grafana UI
   Alerting → Contact points → Add contact point
   Type: Slack
   Webhook URL: https://hooks.slack.com/services/...
   ```

2. **Email**:
   ```yaml
   # grafana.ini
   [smtp]
   enabled = true
   host = smtp.gmail.com:587
   user = alerts@example.com
   password = ***
   ```

3. **PagerDuty** (production):
   ```bash
   # In Grafana UI
   Alerting → Contact points → Add contact point
   Type: PagerDuty
   Integration Key: <key>
   ```

**Alert Routing**:
```yaml
# Grafana notification policies
- Severity: critical → PagerDuty + Slack
- Severity: warning → Slack
- Severity: info → Log only
```

**Value**:
- Early problem detection
- Reduced downtime
- Faster incident response
- Better SLA compliance

---

### SLO/SLI Dashboards

**Priority**: MEDIUM (for production)
**Estimated Effort**: 1-2 days
**Dependencies**: 3-6 months of historical data

**SLIs to Track**:

1. **Availability SLI**:
   ```promql
   # Uptime percentage
   avg_over_time(up{job="contextd"}[30d]) * 100
   ```
   Target: 99.9% (43 minutes downtime/month)

2. **Latency SLI**:
   ```promql
   # P95 MCP tool latency
   histogram_quantile(0.95,
     rate(contextd_mcp_tool_duration_seconds_bucket[30d])
   )
   ```
   Target: <500ms

3. **Error Rate SLI**:
   ```promql
   # Error percentage
   sum(rate(contextd_mcp_tool_calls_total{status="error"}[30d]))
   / sum(rate(contextd_mcp_tool_calls_total[30d]))
   * 100
   ```
   Target: <1%

4. **Throughput SLI**:
   ```promql
   # Requests per second
   sum(rate(contextd_mcp_tool_calls_total[30d]))
   ```
   Target: Support 10 req/sec minimum

**SLO Dashboard**:
```json
{
  "title": "Service Level Objectives",
  "panels": [
    {
      "title": "Availability SLO",
      "targets": [
        {
          "expr": "avg_over_time(up[30d]) * 100",
          "legendFormat": "Uptime %"
        }
      ],
      "thresholds": [
        {
          "value": 99.9,
          "color": "green"
        },
        {
          "value": 99.0,
          "color": "yellow"
        },
        {
          "value": 0,
          "color": "red"
        }
      ]
    }
  ]
}
```

**Error Budget Tracking**:
```promql
# Error budget remaining (monthly)
# Assuming 99.9% SLO = 0.1% error budget
0.1 - (
  sum(rate(contextd_mcp_tool_calls_total{status="error"}[30d]))
  / sum(rate(contextd_mcp_tool_calls_total[30d]))
  * 100
)
```

**Value**:
- Clear performance targets
- Track SLA compliance
- Prioritize improvements
- Customer communication

---

### Log Aggregation Integration

**Priority**: LOW (use OTEL traces instead)
**Estimated Effort**: 2-3 days
**Dependencies**: Loki or Elasticsearch

**Options**:

1. **Grafana Loki**:
   - Pros: Integrates with Grafana, lightweight
   - Cons: Less powerful query language
   - Use case: Structured logs, grep-like queries

2. **Elasticsearch + Kibana**:
   - Pros: Powerful search, mature ecosystem
   - Cons: Heavy resource usage
   - Use case: Complex log analysis

**Recommendation**: Stick with OpenTelemetry traces for now. Logs add complexity with minimal benefit given robust tracing.

**If Needed** (Loki example):

```yaml
# docker-compose.yml
services:
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    command: -config.file=/etc/loki/local-config.yaml

  promtail:
    image: grafana/promtail:latest
    volumes:
      - /var/log:/var/log
    command: -config.file=/etc/promtail/config.yml
```

```go
// pkg/telemetry/telemetry.go
import "github.com/grafana/loki-client-go/loki"

// Configure Loki client
lokiClient, _ := loki.NewClient(loki.Config{
    URL: "http://localhost:3100",
})
```

---

## Additional Metrics

### Batch Processing Metrics

**Priority**: MEDIUM
**Use Case**: Repository indexing, bulk operations

**Metrics**:
```go
// Batch size distribution
contextd_batch_size_bytes (histogram)

// Batch processing duration
contextd_batch_duration_seconds (histogram)

// Items processed per batch
contextd_batch_items_total (histogram)

// Batch errors
contextd_batch_errors_total (counter)
```

**Implementation**:
```go
func (s *IndexService) ProcessBatch(ctx context.Context, items []Item) error {
    start := time.Now()
    defer func() {
        s.meters.BatchDuration.Record(ctx, time.Since(start).Seconds())
        s.meters.BatchSize.Record(ctx, float64(len(items)))
    }()

    // ... process items ...
}
```

---

### Rate Limiting Metrics

**Priority**: LOW (when rate limiting added)
**Use Case**: API rate limiting

**Metrics**:
```go
// Rate limit hits
contextd_rate_limit_hits_total (counter)

// Rate limit rejects
contextd_rate_limit_rejects_total (counter)

// Current request rate
contextd_request_rate_per_second (gauge)
```

---

### Cache Metrics

**Priority**: MEDIUM
**Use Case**: Embedding cache, query cache

**Metrics**:
```go
// Cache hits
contextd_cache_hits_total{cache="embeddings"} (counter)

// Cache misses
contextd_cache_misses_total{cache="embeddings"} (counter)

// Cache hit rate
(sum(rate(contextd_cache_hits_total[5m]))
/ (sum(rate(contextd_cache_hits_total[5m]))
   + sum(rate(contextd_cache_misses_total[5m]))))

// Cache evictions
contextd_cache_evictions_total (counter)

// Cache size
contextd_cache_size_bytes (gauge)
```

**Implementation**:
```go
// pkg/embedding/embedding.go
func (s *Service) Embed(ctx context.Context, text string) (*EmbeddingResult, error) {
    cacheKey := s.getCacheKey(text)

    // Check cache
    if cached, found := s.cache[cacheKey]; found {
        if s.meters != nil {
            s.meters.CacheHits.Add(ctx, 1, metric.WithAttributes(
                attribute.String("cache", "embeddings")))
        }
        return &EmbeddingResult{Embedding: cached, FromCache: true}, nil
    }

    // Cache miss
    if s.meters != nil {
        s.meters.CacheMisses.Add(ctx, 1, metric.WithAttributes(
            attribute.String("cache", "embeddings")))
    }

    // ... generate embedding ...
}
```

---

### Security Metrics

**Priority**: MEDIUM
**Use Case**: Security monitoring, threat detection

**Metrics**:
```go
// Authentication failures
contextd_auth_failures_total (counter)

// Invalid tokens
contextd_invalid_tokens_total (counter)

// Secrets detected (and redacted)
contextd_secrets_detected_total (counter)

// Permission denied errors
contextd_permission_denied_total (counter)
```

**Implementation**:
```go
// pkg/auth/middleware.go
func BearerAuthMiddleware(validToken string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            token := extractToken(c)
            if token != validToken {
                // Record auth failure
                if meters != nil {
                    meters.AuthFailures.Add(c.Request().Context(), 1)
                }
                return echo.ErrUnauthorized
            }
            return next(c)
        }
    }
}
```

---

## Dashboard Enhancements

### Real User Monitoring (RUM) Dashboard

**Priority**: LOW
**Use Case**: Understanding user behavior

**Panels**:
- Most used MCP tools (pie chart)
- Peak usage hours (heatmap)
- Average session duration
- Tool usage by project
- User journey flow (Sankey diagram)

**Implementation**: Requires user tracking (privacy considerations).

---

### Cost Analysis Dashboard

**Priority**: MEDIUM
**Use Case**: Cost optimization

**Panels**:
1. **Current Costs**:
   - Daily embedding cost
   - Monthly projection
   - Cost by model
   - Cost per operation

2. **Cost Trends**:
   - 30-day cost trend
   - Week-over-week comparison
   - Cost optimization opportunities

3. **Resource Usage**:
   - Tokens per day
   - Cache hit rate
   - Operations per dollar

**PromQL Queries**:
```promql
# Daily cost
increase(contextd_embedding_cost_total[1d])

# Monthly projection
increase(contextd_embedding_cost_total[1d]) * 30

# Cost per operation
rate(contextd_embedding_cost_total[5m])
/ rate(contextd_embedding_operations_total[5m])

# Potential savings from cache
(sum(rate(contextd_cache_misses_total[1d]))
 * avg_embedding_cost_per_operation)
```

---

### Capacity Planning Dashboard

**Priority**: MEDIUM
**Use Case**: Infrastructure scaling

**Panels**:
1. **Resource Utilization**:
   - CPU usage trend
   - Memory usage trend
   - Disk usage trend
   - Network I/O

2. **Growth Metrics**:
   - Daily active users (proxied from operations)
   - Requests per day trend
   - Storage growth rate

3. **Projections**:
   - Time to resource exhaustion
   - Recommended scaling actions

---

## Testing Improvements

### Load Testing Metrics

**Priority**: MEDIUM
**Estimated Effort**: 1-2 days

**Tool**: k6 or Locust

**k6 Example**:
```javascript
// load-test.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 10 },  // Ramp up
    { duration: '5m', target: 10 },  // Stay at 10 users
    { duration: '2m', target: 0 },   // Ramp down
  ],
};

export default function() {
  // Create checkpoint
  let res = http.post('http://localhost:8080/checkpoints', JSON.stringify({
    summary: 'Load test checkpoint',
    description: 'Testing performance',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'duration < 500ms': (r) => r.timings.duration < 500,
  });
}
```

**Metrics to Track**:
- Request rate (RPS)
- Response time (P50, P95, P99)
- Error rate
- Resource usage under load

---

### Synthetic Monitoring

**Priority**: LOW
**Use Case**: Proactive health checking

**Tool**: Blackbox Exporter + Prometheus

**Checks**:
1. **Health Endpoint**: GET /health every 30s
2. **Create Checkpoint**: POST /checkpoints every 5m
3. **Search Checkpoint**: GET /checkpoints/search every 5m
4. **End-to-End**: Full user workflow every 15m

**Metrics**:
```
probe_success{endpoint="/health"} - Probe success (1 or 0)
probe_duration_seconds{endpoint="/health"} - Probe duration
```

---

## Documentation Improvements

### Runbook Creation

**Priority**: HIGH (for production)
**Format**: Markdown documents

**Runbooks to Create**:

1. **Service Down**:
   - Symptoms: up == 0
   - Investigation steps
   - Resolution procedures
   - Escalation path

2. **High Error Rate**:
   - Symptoms: error rate > 1%
   - Common causes
   - Debugging steps
   - Mitigation strategies

3. **Performance Degradation**:
   - Symptoms: P95 latency > 2s
   - Profiling steps
   - Optimization options
   - Scaling procedures

4. **Database Issues**:
   - Connection errors
   - Query timeouts
   - Storage exhaustion
   - Recovery procedures

---

### Monitoring Best Practices Guide

**Priority**: MEDIUM
**Topics**:
- When to add new metrics
- Avoiding high cardinality
- Alert fatigue prevention
- Dashboard design principles
- Cost optimization strategies

---

## Implementation Priority

### High Priority (Next 1-2 months)
1. ✅ Go runtime metrics - Essential for production
2. ✅ HTTP server metrics - Essential for API monitoring
3. ✅ Alerting system - Critical for production readiness
4. ⏳ Remediation service instrumentation - Complete Phase 2
5. ⏳ Database metrics - Visibility into data layer

### Medium Priority (Months 3-6)
6. ⏳ Connection pool metrics - Performance optimization
7. ⏳ SLO/SLI dashboards - Production readiness
8. ⏳ Cost analysis dashboard - Cost optimization
9. ⏳ Capacity planning dashboard - Scaling decisions
10. ⏳ Continuous profiling - Performance optimization

### Low Priority (Months 6-12)
11. ⏳ Log aggregation - Only if traces insufficient
12. ⏳ Synthetic monitoring - Proactive health checks
13. ⏳ Real user monitoring - User behavior analysis
14. ⏳ Advanced analytics - ML-based anomaly detection

---

## Success Metrics

### Phase 3 Success Criteria
- [ ] Runtime metrics exported and visible in Grafana
- [ ] HTTP metrics showing request patterns
- [ ] Connection pool metrics tracking resource usage
- [ ] All metrics < 1% overhead on application performance

### Phase 4 Success Criteria
- [ ] Profiling endpoints accessible
- [ ] Alerts configured and tested
- [ ] SLO dashboard showing compliance
- [ ] Runbooks created for all critical alerts
- [ ] Mean Time To Detection (MTTD) < 5 minutes
- [ ] Mean Time To Resolution (MTTR) < 30 minutes

---

## References

- [01-architecture.md](./01-architecture.md) - Architecture overview
- [02-metrics-catalog.md](./02-metrics-catalog.md) - Current metrics
- [03-implementation-status.md](./03-implementation-status.md) - Current status
- [04-usage-guide.md](./04-usage-guide.md) - Usage guide
- OpenTelemetry Runtime Instrumentation: https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/runtime
- Grafana Alerting: https://grafana.com/docs/grafana/latest/alerting/
- SLO/SLI Guide: https://sre.google/sre-book/service-level-objectives/
