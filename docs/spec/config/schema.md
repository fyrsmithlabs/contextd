# Configuration Schema

**Parent**: @./SPEC.md

Full YAML configuration reference. All values shown are defaults unless noted.

---

## Server Configuration

```yaml
server:
  grpc:
    port: 50051                    # gRPC server port
    max_recv_msg_size: 16777216    # 16MB
    max_send_msg_size: 16777216    # 16MB
    keepalive:
      time: 30s                    # Ping interval if no activity
      timeout: 10s                 # Wait for ping ack
      min_time: 10s                # Min time between client pings

  http:
    port: 8080                     # HTTP server port (health, metrics)
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s

  shutdown_timeout: 30s            # Graceful shutdown wait
```

---

## Vector Store Configuration

```yaml
vectorstore:
  provider: chroma                 # "chroma" (default) or "qdrant"

  chroma:
    path: ~/.config/contextd/chroma.db  # SQLite database path
    model: sentence-transformers/all-mpnet-base-v2  # Embedding model
    dimension: 768                 # Must match model (768 for mpnet)
    distance: cosine               # cosine, l2, ip

  qdrant:
    host: localhost                # Qdrant host
    port: 6334                     # Qdrant gRPC port
    api_key: ""                    # API key (use env: CONTEXTD_QDRANT_API_KEY)
    vector_size: 384               # Embedding dimensions
    tls:
      enabled: false
      cert_file: ""
      key_file: ""
      ca_file: ""
    timeout: 30s                   # Operation timeout
    pool:
      max_connections: 10
      min_connections: 2
```

### Chroma Model/Dimension Mapping

| Model | Dimension | Notes |
|-------|-----------|-------|
| `sentence-transformers/all-MiniLM-L6-v2` | 384 | Fast, lightweight |
| `sentence-transformers/all-mpnet-base-v2` | 768 | **Default**, balanced |
| `sentence-transformers/all-roberta-large-v1` | 1024 | Highest accuracy |

**Validation**: Dimension must match model output. Mismatches return error.

---

## Qdrant Configuration (Legacy)

> **Note**: Use `vectorstore.qdrant` instead. This section preserved for backward compatibility.

```yaml
qdrant:
  host: localhost                  # Qdrant host
  port: 6334                       # Qdrant gRPC port
  api_key: ""                      # API key (use env: CONTEXTD_QDRANT_API_KEY)
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
    ca_file: ""
  timeout: 30s                     # Operation timeout
  pool:
    max_connections: 10
    min_connections: 2
```

---

## Telemetry Configuration

```yaml
telemetry:
  enabled: true                    # Master switch
  service_name: contextd           # Service identifier
  service_version: ""              # Set via build flags

  endpoint: localhost:4317         # OTLP collector endpoint
  protocol: grpc                   # grpc or http
  insecure: true                   # TLS disabled (dev)

  sampling:
    rate: 1.0                      # 0.0-1.0 (1.0 = 100%)
    always_on_errors: true         # Never sample out errors

  metrics:
    enabled: true
    export_interval: 15s

  traces:
    enabled: true

  experience_metrics:
    enabled: false                 # Opt-in user experience tracking
    retention_days: 30
```

---

## Logging Configuration

```yaml
logging:
  level: info                      # trace, debug, info, warn, error
  format: json                     # json, console

  output:
    stdout: true                   # Write to stdout
    otel: true                     # Send to OTEL (if telemetry enabled)

  sampling:
    enabled: true
    tick: 1s
    levels:
      trace:
        initial: 1
        thereafter: 0
      debug:
        initial: 10
        thereafter: 0
      info:
        initial: 100
        thereafter: 10
      warn:
        initial: 100
        thereafter: 100

  caller:
    enabled: true
    skip: 1

  stacktrace:
    level: error

  fields:
    service: contextd
    version: ""
    environment: ""

  redaction:
    enabled: true
    fields:
      - password
      - secret
      - token
      - api_key
      - authorization
```

---

## Scrubber Configuration

```yaml
scrubber:
  enabled: true

  gitleaks:
    config_path: ""                # Custom gitleaks config
    baseline_path: ""              # Baseline for known secrets

  action: redact                   # redact, tokenize, block

  token_store:
    enabled: false                 # Store tokens for reference resolution
    ttl: 24h

  patterns:                        # Additional regex patterns
    - name: custom_api_key
      regex: "CUSTOM_[A-Z0-9]{32}"
```

---

## Session Configuration

```yaml
session:
  max_duration: 4h                 # Max session length
  idle_timeout: 30m                # Timeout without activity

  checkpoint:
    auto_save: true
    interval: 10m                  # Auto-checkpoint interval
    context_threshold: 0.7         # Save when context usage > 70%
    max_size: 10485760             # 10MB max checkpoint size

  distillation:
    enabled: true
    async: true                    # Don't block session end
    queue_size: 100
```

---

## Memory Configuration

```yaml
memory:
  search:
    default_limit: 10
    max_limit: 100
    min_confidence: 0.5

  embedding:
    model: text-embedding-3-small  # OpenAI model
    dimensions: 1536
    batch_size: 100
```

---

## Tools Configuration

```yaml
tools:
  bash:
    default_timeout: 30s
    max_timeout: 300s              # 5 minutes max
    allowed_commands: []           # Empty = all allowed
    blocked_commands:
      - rm -rf /
      - mkfs
      - dd

  read:
    max_file_size: 10485760        # 10MB
    blocked_paths:
      - /etc/shadow
      - /etc/passwd

  write:
    max_file_size: 10485760
    blocked_paths:
      - /etc
      - /usr
      - /bin
```

---

## Tenancy Configuration

```yaml
tenancy:
  enabled: false                   # Single-tenant by default

  jwt:
    secret: ""                     # Use env: CONTEXTD_TENANCY_JWT_SECRET
    issuer: ""
    audience: ""

  isolation:
    collection_per_tenant: true    # Physical isolation in Qdrant
```
