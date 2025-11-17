# Monitoring Architecture Specification

## Overview

The contextd monitoring system provides comprehensive observability through OpenTelemetry instrumentation, exporting metrics and traces to industry-standard backends. The architecture prioritizes zero-configuration local development while supporting production deployment patterns.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         contextd (Go)                            │
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐    │
│  │  Services   │  │   pkg/metrics │  │  pkg/telemetry    │    │
│  │             │  │               │  │                    │    │
│  │ - Checkpoint│──│ 31 Metrics   │──│ OTEL Provider     │    │
│  │ - Embedding │  │ - Counters   │  │ - Traces          │    │
│  │ - Remediation│ │ - Histograms │  │ - Metrics         │    │
│  │ - Skills    │  │ - Gauges     │  │                    │    │
│  └─────────────┘  └──────────────┘  └────────────────────┘    │
│                           │                      │               │
└───────────────────────────┼──────────────────────┼───────────────┘
                            │                      │
                            ▼                      ▼
                   OTLP/HTTP :4318         OTLP/HTTP :4318
                            │                      │
┌───────────────────────────┴──────────────────────┴───────────────┐
│              OpenTelemetry Collector :4317/:4318                  │
│                                                                    │
│  Receivers: OTLP (HTTP + gRPC)                                   │
│  Processors: Batch (10s, 1024 items)                             │
│  Exporters:                                                       │
│    - prometheusremotewrite → VictoriaMetrics                     │
│    - otlp/jaeger → Jaeger                                        │
│    - debug → stdout (troubleshooting)                            │
└──────────────────────┬─────────────────────┬─────────────────────┘
                       │                     │
           ┌───────────▼─────────┐  ┌────────▼────────┐
           │  VictoriaMetrics    │  │     Jaeger      │
           │  :8428              │  │     :4317       │
           │                     │  │                 │
           │  - 12 month         │  │  - Traces       │
           │    retention        │  │  - Spans        │
           │  - Prometheus       │  │  - Badger DB    │
           │    Remote Write     │  │                 │
           │  - PromQL           │  │                 │
           └─────────────────────┘  └─────────────────┘
                       │                     │
                       └──────────┬──────────┘
                                  │
                       ┌──────────▼──────────┐
                       │      Grafana        │
                       │      :3001          │
                       │                     │
                       │  Datasources:       │
                       │  - VictoriaMetrics  │
                       │  - Jaeger           │
                       │                     │
                       │  Dashboards:        │
                       │  - Overview         │
                       │  - Testing          │
                       │  - Agents & Skills  │
                       └─────────────────────┘
```

## Component Details

### 1. contextd Application (Go)

**Location**: `/home/dahendel/projects/contextd`

**Instrumentation**:
- **pkg/metrics**: Centralized metrics definitions (31 metrics)
- **pkg/telemetry**: OpenTelemetry initialization and configuration
- **Service Integration**: Checkpoint, Embedding, Remediation, Skills services

**Export Configuration**:
```go
// pkg/telemetry initialization
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=contextd
OTEL_ENVIRONMENT=development
```

**Metrics Instrumentation Points**:
- Checkpoint operations: Create, Search, List
- Embedding generation: OpenAI/TEI calls
- Remediation: Save, Search operations
- Skills: Create, Apply, Search (planned)
- Database: Insert, Search, Delete operations

### 2. OpenTelemetry Collector

**Image**: `otel/opentelemetry-collector-contrib:latest`
**Container**: `contextd-otel-collector`
**Config**: `/monitoring/otel-collector-config.yaml`

**Ports**:
- `:4318` - OTLP HTTP (metrics + traces)
- `:4317` - OTLP gRPC (metrics + traces)
- `:13133` - Health check endpoint

**Receivers**:
```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318
      grpc:
        endpoint: 0.0.0.0:4317
```

**Processors**:
```yaml
processors:
  batch:
    timeout: 10s              # Batch for 10 seconds
    send_batch_size: 1024     # Or 1024 items, whichever comes first
```

**Exporters**:
```yaml
exporters:
  # Metrics → VictoriaMetrics
  prometheusremotewrite:
    endpoint: http://victoriametrics:8428/api/v1/write
    tls:
      insecure: true

  # Traces → Jaeger
  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true

  # Debug (stdout)
  debug:
    verbosity: normal
    sampling_initial: 5
    sampling_thereafter: 200
```

**Pipelines**:
```yaml
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheusremotewrite, debug]

    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/jaeger, debug]
```

### 3. VictoriaMetrics (Metrics Storage)

**Image**: `victoriametrics/victoria-metrics:latest`
**Container**: `contextd-metrics`

**Ports**:
- `:8428` - HTTP API (Prometheus Remote Write + PromQL)

**Configuration**:
```yaml
command:
  - "-storageDataPath=/victoria-metrics-data"
  - "-retentionPeriod=12"  # 12 months
  - "-httpListenAddr=:8428"
```

**Endpoints**:
- `GET /api/v1/query` - PromQL instant queries
- `GET /api/v1/query_range` - PromQL range queries
- `POST /api/v1/write` - Prometheus Remote Write (OTEL Collector)
- `GET /api/v1/labels` - List metric labels
- `GET /api/v1/label/<name>/values` - List label values

**Storage**:
- Volume: `vm-data` (local driver)
- Retention: 12 months
- Path: `/victoria-metrics-data`

### 4. Jaeger (Distributed Tracing)

**Image**: `jaegertracing/all-in-one:latest`
**Container**: `contextd-jaeger`

**Ports**:
- `:16686` - Web UI
- `:14268` - Jaeger collector (HTTP)
- `:14269` - Health check
- `:4317` - OTLP gRPC receiver

**Configuration**:
```yaml
environment:
  COLLECTOR_OTLP_ENABLED: "true"
  SPAN_STORAGE_TYPE: badger
  BADGER_EPHEMERAL: "false"
  BADGER_DIRECTORY_VALUE: /badger/data
  BADGER_DIRECTORY_KEY: /badger/key
```

**Storage**:
- Volume: `jaeger-data` (local driver)
- Backend: Badger (embedded key-value DB)
- Persistent: Yes (survives container restarts)

### 5. Grafana (Visualization)

**Image**: `grafana/grafana:latest`
**Container**: `contextd-grafana`

**Ports**:
- `:3001` - Web UI (mapped from internal :3000)

**Configuration**:
```yaml
environment:
  GF_SECURITY_ADMIN_USER: admin
  GF_SECURITY_ADMIN_PASSWORD: admin
  GF_USERS_ALLOW_SIGN_UP: "false"
  GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH: /var/lib/grafana/dashboards/contextd-overview.json
```

**Volumes**:
- `/var/lib/grafana` - Persistent data (grafana-data volume)
- `/etc/grafana/provisioning/dashboards` - Dashboard provisioning config
- `/etc/grafana/provisioning/datasources` - Datasource provisioning config
- `/var/lib/grafana/dashboards` - Dashboard JSON files

**Provisioned Datasources**:
1. **VictoriaMetrics** (Prometheus-compatible)
   - URL: `http://victoriametrics:8428`
   - Type: Prometheus
   - Default: Yes

2. **Jaeger** (Traces)
   - URL: `http://jaeger:16686`
   - Type: Jaeger
   - Default: No

**Provisioned Dashboards** (3 total):
1. `contextd-overview.json` - Overall service metrics
2. `contextd-testing.json` - Test coverage and quality
3. `contextd-agents-skills.json` - Agent and skill performance

## Data Flow

### Metrics Pipeline

```
1. Service Operation (e.g., checkpoint.Create)
   │
   ├─→ Record metrics via pkg/metrics
   │   └─→ meters.CheckpointOperations.Add(ctx, 1)
   │       meters.CheckpointDuration.Record(ctx, duration)
   │
2. OpenTelemetry SDK (in-memory)
   │
   ├─→ Batch metrics (60s interval)
   │
3. Export via OTLP/HTTP to :4318
   │
4. OTEL Collector receives
   │
   ├─→ Batch processor (10s or 1024 items)
   │
5. Export to VictoriaMetrics
   │
   ├─→ Prometheus Remote Write
   │   POST http://victoriametrics:8428/api/v1/write
   │
6. VictoriaMetrics stores
   │
   ├─→ Time-series database
   │   12 month retention
   │
7. Grafana queries
   │
   └─→ PromQL queries via datasource
       Renders dashboards
```

### Traces Pipeline

```
1. Service Operation (e.g., checkpoint.Search)
   │
   ├─→ Start span: tracer.Start(ctx, "checkpoint.search")
   │   └─→ Add attributes: checkpoint_id, project_path, top_k
   │       Record events/errors
   │       End span
   │
2. OpenTelemetry SDK (in-memory)
   │
   ├─→ Batch spans (5s timeout, 512 spans)
   │
3. Export via OTLP/HTTP to :4318
   │
4. OTEL Collector receives
   │
   ├─→ Batch processor (10s or 1024 items)
   │
5. Export to Jaeger
   │
   ├─→ OTLP/gRPC to jaeger:4317
   │
6. Jaeger stores
   │
   ├─→ Badger database
   │   Persistent storage
   │
7. View in Jaeger UI
   │
   └─→ http://localhost:16686
       Search traces
       View spans and timing
```

## Network Architecture

### Docker Network

**Network Name**: `contextd-network`
**Type**: Bridge (default)

**Service Communication**:
```
contextd (host) → otel-collector:4318  (OTLP HTTP)
otel-collector  → victoriametrics:8428 (Prometheus Remote Write)
otel-collector  → jaeger:4317          (OTLP gRPC)
grafana         → victoriametrics:8428 (PromQL queries)
grafana         → jaeger:16686         (Trace queries)
```

**Host Access**:
```
localhost:3001  → Grafana UI
localhost:16686 → Jaeger UI
localhost:8428  → VictoriaMetrics API
localhost:4318  → OTEL Collector HTTP
localhost:4317  → OTEL Collector gRPC
```

## Configuration Files

### 1. OpenTelemetry Collector

**File**: `/monitoring/otel-collector-config.yaml`

Key configuration points:
- Receivers: OTLP HTTP/gRPC on all interfaces (0.0.0.0)
- Batch processing: 10s timeout, 1024 item batches
- Exporters: VictoriaMetrics (Prometheus Remote Write), Jaeger (OTLP)
- Debug exporter: First 5 batches, then 1 in 200 (for troubleshooting)

### 2. Grafana Provisioning

**Datasources**: `/monitoring/grafana/datasources/datasources.yml`
```yaml
apiVersion: 1
datasources:
  - name: VictoriaMetrics
    type: prometheus
    access: proxy
    url: http://victoriametrics:8428
    isDefault: true

  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
```

**Dashboards**: `/monitoring/grafana/dashboards/dashboards.yml`
```yaml
apiVersion: 1
providers:
  - name: 'contextd'
    folder: ''
    type: file
    options:
      path: /var/lib/grafana/dashboards
```

**Dashboard Files**: `/monitoring/grafana/dashboard-configs/`
- `contextd-overview.json` - Main dashboard (default home)
- `contextd-testing.json` - Test metrics dashboard
- `contextd-agents-skills.json` - Agent performance dashboard

### 3. Docker Compose

**File**: `/docker-compose.yml`

Services:
1. `tei` - Text Embeddings Inference (embedding generation)
2. `qdrant` - Vector database
3. `victoriametrics` - Metrics storage
4. `grafana` - Visualization
5. `otel-collector` - Metrics/traces aggregation
6. `jaeger` - Trace storage
7. `redis` - Caching (future use)

Depends on:
- `grafana` depends on `victoriametrics`
- `otel-collector` depends on `victoriametrics` and `jaeger`

## Security Considerations

### 1. Network Isolation

- All services on isolated Docker bridge network (`contextd-network`)
- No external network exposure by default
- Host access only through explicitly mapped ports

### 2. Authentication

**Grafana**:
- Default credentials: `admin/admin`
- Change in production: `GF_SECURITY_ADMIN_PASSWORD`
- Sign-up disabled: `GF_USERS_ALLOW_SIGN_UP: "false"`

**Other Services**:
- VictoriaMetrics: No auth (network isolated)
- Jaeger: No auth (network isolated)
- OTEL Collector: No auth (network isolated)

**Production Recommendations**:
- Enable Grafana OAuth/LDAP
- Add VictoriaMetrics authentication
- Use TLS for inter-service communication
- Implement network policies

### 3. Data Security

**Secrets in Metrics**:
- `pkg/security` redacts sensitive data before embedding
- Patterns: API keys, tokens, passwords, file paths with usernames
- Applied in `pkg/embedding` before OpenAI API calls

**Trace Data**:
- Avoid logging sensitive request bodies
- Redact credentials in span attributes
- Use sampling in production to reduce data volume

### 4. Resource Limits

**Docker Compose** (production recommendations):
```yaml
services:
  victoriametrics:
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '1.0'

  otel-collector:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
```

## Performance Considerations

### 1. Batch Configuration

**OpenTelemetry SDK** (in contextd):
```go
// pkg/telemetry
metricReader := sdkmetric.NewPeriodicReader(
    sdkmetric.NewOTLPHTTPExporter(...),
    sdkmetric.WithInterval(60*time.Second),  // 60s metric export
)

traceExporter := otlptracehttp.New(ctx,
    otlptracehttp.WithTimeout(5*time.Second),  // 5s batch timeout
)
```

**OTEL Collector**:
```yaml
processors:
  batch:
    timeout: 10s
    send_batch_size: 1024
```

**Impact**:
- Metrics: Exported every 60s from app, batched 10s in collector
- Traces: Exported every 5s from app, batched 10s in collector
- Network calls reduced 6x (60s/10s) for metrics
- Memory usage: ~512MB for collector, ~2GB for VictoriaMetrics

### 2. Cardinality Management

**Metric Labels** (avoid high cardinality):
```go
// GOOD: Low cardinality labels
attribute.String("operation", "create")     // ~10 values
attribute.String("status", "success")       // 2-3 values
attribute.String("provider", "openai")      // 2-3 values

// BAD: High cardinality (avoid)
attribute.String("user_id", "...")          // Thousands/millions
attribute.String("checkpoint_id", "...")    // Unique per checkpoint
```

**Current Labels**:
- Total unique combinations: ~200-500
- Acceptable for VictoriaMetrics at this scale

### 3. Retention and Storage

**VictoriaMetrics**:
- Retention: 12 months
- Compression: ~10:1 typical for time-series
- Estimated storage: ~1-5GB for 1 year (depends on metric volume)

**Jaeger**:
- Storage: Badger (embedded key-value)
- No automatic retention (configure separately)
- Estimated storage: ~5-10GB for 6 months of traces

## Deployment Patterns

### Development (Current)

```bash
# Start all services
docker-compose up -d

# Start contextd
./contextd

# Access UIs
# Grafana: http://localhost:3001 (admin/admin)
# Jaeger: http://localhost:16686
# VictoriaMetrics: http://localhost:8428
```

### Production (Recommended)

1. **Separate Infrastructure Tier**:
   - Run monitoring stack on dedicated hosts
   - Use managed services (Grafana Cloud, AWS CloudWatch)

2. **High Availability**:
   - Multiple OTEL Collector instances (load balanced)
   - VictoriaMetrics cluster mode
   - Grafana high availability setup

3. **Security**:
   - TLS for all inter-service communication
   - Authentication on all services
   - Network policies/firewalls

4. **Scaling**:
   - Horizontal: Multiple contextd instances → multiple collectors
   - Vertical: Increase VictoriaMetrics memory for higher cardinality

## Monitoring the Monitoring

### Health Checks

**Docker Compose Built-in**:
```bash
# Check all services
docker-compose ps

# Check specific service
docker inspect contextd-otel-collector --format='{{.State.Health.Status}}'
```

**Endpoints**:
- OTEL Collector: `http://localhost:13133/` (200 OK)
- VictoriaMetrics: `http://localhost:8428/health` (200 OK)
- Jaeger: `http://localhost:14269/` (200 OK)
- Grafana: `http://localhost:3001/api/health` (200 OK)

### Debug Exporter

**OTEL Collector** debug exporter logs to stdout:
```bash
# View collector logs
docker logs -f contextd-otel-collector

# Sample output
2025-11-04T12:00:00.000Z info MetricsExporter {"#metrics": 42}
2025-11-04T12:00:00.000Z info TracesExporter {"#spans": 15}
```

## Troubleshooting

See [04-usage-guide.md](./04-usage-guide.md) for comprehensive troubleshooting guide.

## References

- **OpenTelemetry**: https://opentelemetry.io/docs/
- **VictoriaMetrics**: https://docs.victoriametrics.com/
- **Jaeger**: https://www.jaegertracing.io/docs/
- **Grafana**: https://grafana.com/docs/grafana/latest/
- **Prometheus Remote Write**: https://prometheus.io/docs/specs/remote_write_spec/
