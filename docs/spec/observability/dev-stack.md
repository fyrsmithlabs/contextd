# Development Stack

**Version**: 1.0.0
**Status**: Draft

---

## Overview

Local observability stack using Docker Compose with VictoriaMetrics ecosystem and Grafana.

---

## Components

| Component | Image | Ports |
|-----------|-------|-------|
| OTEL Collector | `otel/opentelemetry-collector-contrib` | 4317, 4318 |
| VictoriaMetrics | `victoriametrics/victoria-metrics` | 8428 |
| VictoriaLogs | `victoriametrics/victoria-logs` | 9428 |
| VictoriaTraces | `victoriametrics/victoria-traces` | 9420 |
| Grafana | `grafana/grafana` | 3000 |
| Qdrant | `qdrant/qdrant` | 6334 |

---

## Quick Start

```bash
# Start observability stack
make dev-stack

# Run contextd with telemetry
make dev-run

# Stop stack
make dev-stack-down

# Reset all data
make dev-stack-reset
```

---

## URLs

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / contextd |
| VictoriaMetrics UI | http://localhost:8428/vmui | - |
| VictoriaLogs UI | http://localhost:9428/select/vmui | - |
| VictoriaTraces | http://localhost:9420 | - |
| Qdrant Dashboard | http://localhost:6333/dashboard | - |

---

## Docker Compose Configuration

```yaml
# deploy/docker-compose.dev.yaml

version: "3.8"

services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP

  victoriametrics:
    image: victoriametrics/victoria-metrics:latest
    command:
      - "--storageDataPath=/victoria-metrics-data"
      - "--httpListenAddr=:8428"
    ports:
      - "8428:8428"
    volumes:
      - vm-data:/victoria-metrics-data

  victorialogs:
    image: victoriametrics/victoria-logs:latest
    command:
      - "--storageDataPath=/victoria-logs-data"
      - "--httpListenAddr=:9428"
    ports:
      - "9428:9428"
    volumes:
      - vl-data:/victoria-logs-data

  victoriatraces:
    image: victoriametrics/victoria-traces:latest
    ports:
      - "9420:9420"

  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=contextd
      - GF_AUTH_ANONYMOUS_ENABLED=true
    ports:
      - "3000:3000"
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning
      - ./grafana/dashboards:/var/lib/grafana/dashboards

  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"   # HTTP
      - "6334:6334"   # gRPC
    volumes:
      - qdrant-data:/qdrant/storage

volumes:
  vm-data:
  vl-data:
  qdrant-data:
```

---

## OTEL Collector Configuration

```yaml
# deploy/otel-collector-config.yaml

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  prometheusremotewrite:
    endpoint: http://victoriametrics:8428/api/v1/write

  otlphttp/logs:
    endpoint: http://victorialogs:9428/insert/opentelemetry/v1/logs

  otlphttp/traces:
    endpoint: http://victoriatraces:9420/insert/opentelemetry/v1/traces

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheusremotewrite]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/logs]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/traces]
```

---

## Makefile Targets

```makefile
# Makefile

.PHONY: dev-stack dev-stack-down dev-stack-reset dev-run

dev-stack:
	docker compose -f deploy/docker-compose.dev.yaml up -d

dev-stack-down:
	docker compose -f deploy/docker-compose.dev.yaml down

dev-stack-reset:
	docker compose -f deploy/docker-compose.dev.yaml down -v
	docker compose -f deploy/docker-compose.dev.yaml up -d

dev-run:
	CONTEXTD_TELEMETRY_ENDPOINT=localhost:4317 go run ./cmd/contextd
```

---

## Verification

### Check Collector Health

```bash
curl http://localhost:4318/health
```

### Check VictoriaMetrics

```bash
curl http://localhost:8428/api/v1/query?query=up
```

### Send Test Span

```bash
# Using otel-cli
otel-cli span \
  --endpoint localhost:4317 \
  --service "test" \
  --name "test-span" \
  --attrs "test.attr=value"
```

---

## References

- [VictoriaMetrics Docker](https://docs.victoriametrics.com/quickstart/#docker)
- [OTEL Collector Configuration](https://opentelemetry.io/docs/collector/configuration/)
- [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)
