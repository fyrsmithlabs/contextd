# Configuration Reference

ContextD is configured entirely through environment variables. All settings have sensible defaults, making zero-configuration deployment possible.

---

## Quick Reference

```bash
# Minimal configuration (uses all defaults)
docker run -i --rm -v contextd-data:/data ghcr.io/fyrsmithlabs/contextd:latest

# Custom Qdrant connection
docker run -i --rm \
  -e QDRANT_HOST=qdrant.example.com \
  -e QDRANT_PORT=6334 \
  ghcr.io/fyrsmithlabs/contextd:latest
```

---

## Environment Variables

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `9090` | HTTP server port for health checks and metrics |
| `SERVER_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |

### Qdrant Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `QDRANT_HOST` | `localhost` | Qdrant server hostname |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port |
| `QDRANT_HTTP_PORT` | `6333` | Qdrant HTTP port (for health checks) |
| `QDRANT_COLLECTION` | `contextd_default` | Default collection name |
| `QDRANT_VECTOR_SIZE` | `384` | Vector dimensions (must match embedding model) |
| `CONTEXTD_DATA_PATH` | `/data` | Base path for persistent data |

### Embeddings Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `EMBEDDINGS_PROVIDER` | `fastembed` | Provider: `fastembed` or `tei` |
| `EMBEDDINGS_MODEL` | `BAAI/bge-small-en-v1.5` | Embedding model name |
| `EMBEDDING_BASE_URL` | `http://localhost:8080` | TEI server URL (if using TEI) |
| `EMBEDDINGS_ONNX_VERSION` | (default: 1.23.0) | ONNX runtime version override |
| `ONNX_PATH` | (auto-detected) | Path to libonnxruntime.so |

#### ONNX Runtime Auto-Download

contextd automatically downloads the ONNX runtime library on first use if not already installed. The library is downloaded to `~/.config/contextd/lib/`.

**Explicit Setup:**
```bash
# Download ONNX runtime before first use
ctxd init

# Force re-download
ctxd init --force
```

**Manual Override:**
Set `ONNX_PATH` environment variable to use your own ONNX installation:
```bash
export ONNX_PATH=/usr/local/lib/libonnxruntime.so
```

### Checkpoint Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CHECKPOINT_MAX_CONTENT_SIZE_KB` | `1024` | Maximum checkpoint content size (KB) |

### Telemetry Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_ENABLE` | `true` | Enable OpenTelemetry |
| `OTEL_SERVICE_NAME` | `contextd` | Service name for traces |

### Repository Indexing Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REPOSITORY_IGNORE_FILES` | `.gitignore,.dockerignore,.contextdignore` | Comma-separated list of ignore files to parse |
| `REPOSITORY_FALLBACK_EXCLUDES` | `.git/**,node_modules/**,vendor/**,__pycache__/**` | Fallback exclude patterns |

### Pre-fetch Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PREFETCH_ENABLED` | `true` | Enable pre-fetch engine |
| `PREFETCH_CACHE_TTL` | `5m` | Cache time-to-live |
| `PREFETCH_CACHE_MAX_ENTRIES` | `100` | Maximum cache entries |

#### Pre-fetch Rules

**Branch Diff Rule:**
| Variable | Default | Description |
|----------|---------|-------------|
| `PREFETCH_BRANCH_DIFF_ENABLED` | `true` | Enable branch diff pre-fetch |
| `PREFETCH_BRANCH_DIFF_MAX_FILES` | `10` | Maximum files to pre-fetch |
| `PREFETCH_BRANCH_DIFF_MAX_SIZE_KB` | `50` | Maximum file size (KB) |
| `PREFETCH_BRANCH_DIFF_TIMEOUT_MS` | `1000` | Timeout in milliseconds |

**Recent Commit Rule:**
| Variable | Default | Description |
|----------|---------|-------------|
| `PREFETCH_RECENT_COMMIT_ENABLED` | `true` | Enable recent commit pre-fetch |
| `PREFETCH_RECENT_COMMIT_MAX_SIZE_KB` | `20` | Maximum file size (KB) |
| `PREFETCH_RECENT_COMMIT_TIMEOUT_MS` | `500` | Timeout in milliseconds |

**Common Files Rule:**
| Variable | Default | Description |
|----------|---------|-------------|
| `PREFETCH_COMMON_FILES_ENABLED` | `true` | Enable common files pre-fetch |
| `PREFETCH_COMMON_FILES_MAX_FILES` | `3` | Maximum files to pre-fetch |
| `PREFETCH_COMMON_FILES_TIMEOUT_MS` | `500` | Timeout in milliseconds |

---

## Embedding Models

### Supported FastEmbed Models

| Model | Dimensions | Language | Notes |
|-------|------------|----------|-------|
| `BAAI/bge-small-en-v1.5` | 384 | English | Default, fast, recommended |
| `BAAI/bge-base-en-v1.5` | 768 | English | Higher quality, slower |
| `sentence-transformers/all-MiniLM-L6-v2` | 384 | English | General purpose |
| `BAAI/bge-small-zh-v1.5` | 512 | Chinese | Chinese language support |

**Important:** The `QDRANT_VECTOR_SIZE` must match the model dimensions.

### Using TEI Instead of FastEmbed

If you prefer to use HuggingFace Text Embeddings Inference (TEI):

```bash
# Start TEI server
docker run -p 8080:80 ghcr.io/huggingface/text-embeddings-inference:latest \
  --model-id BAAI/bge-small-en-v1.5

# Configure ContextD to use TEI
docker run -i --rm \
  -e EMBEDDINGS_PROVIDER=tei \
  -e EMBEDDING_BASE_URL=http://host.docker.internal:8080 \
  ghcr.io/fyrsmithlabs/contextd:latest
```

---

## Data Persistence

### Volume Structure

```
/data/
├── qdrant/           # Qdrant storage
│   └── storage/      # Vector collections
├── config/           # Runtime config (future)
└── logs/             # Log files (if enabled)
```

### Docker Volume Management

```bash
# Create named volume
docker volume create contextd-data

# Run with volume
docker run -i --rm -v contextd-data:/data ghcr.io/fyrsmithlabs/contextd:latest

# Inspect volume
docker volume inspect contextd-data

# Backup
docker run --rm -v contextd-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/contextd-backup-$(date +%Y%m%d).tar.gz /data

# Restore
docker run --rm -v contextd-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/contextd-backup-20241202.tar.gz -C /

# Delete (WARNING: destroys all data)
docker volume rm contextd-data
```

---

## Configuration File (Optional)

ContextD can also load configuration from `~/.config/contextd/config.yaml`:

```yaml
# ~/.config/contextd/config.yaml
server:
  port: 9090
  shutdown_timeout: 10s

qdrant:
  host: localhost
  port: 6334
  collection_name: contextd_default
  vector_size: 384

embeddings:
  provider: fastembed
  model: BAAI/bge-small-en-v1.5
  onnx_version: "1.23.0"  # Optional: override ONNX runtime version

checkpoint:
  max_content_size_kb: 1024

telemetry:
  enable: true
  service_name: contextd
```

**Priority:** Environment variables override config file values.

---

## Claude Code Integration

### MCP Configuration

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "ghcr.io/fyrsmithlabs/contextd:latest"
      ]
    }
  }
}
```

### With Custom Configuration

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "-e", "EMBEDDINGS_MODEL=BAAI/bge-base-en-v1.5",
        "-e", "QDRANT_VECTOR_SIZE=768",
        "ghcr.io/fyrsmithlabs/contextd:latest"
      ]
    }
  }
}
```

### Using External Qdrant

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "QDRANT_HOST=qdrant.example.com",
        "-e", "QDRANT_PORT=6334",
        "ghcr.io/fyrsmithlabs/contextd:latest"
      ]
    }
  }
}
```

---

## Validation

Configuration is validated at startup. Invalid configuration causes ContextD to exit with an error.

### Validation Rules

- `SERVER_PORT`: Must be 1-65535
- `SERVER_SHUTDOWN_TIMEOUT`: Must be positive
- `OTEL_SERVICE_NAME`: Required if telemetry is enabled
- `QDRANT_VECTOR_SIZE`: Must match embedding model dimensions

### Health Check

```bash
# Check if ContextD is healthy
curl http://localhost:9090/health

# Check Qdrant connection
curl http://localhost:6333/health
```
