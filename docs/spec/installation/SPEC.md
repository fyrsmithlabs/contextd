# contextd Installation & Deployment Specification

**Status**: Partially Outdated
**Created**: 2024-12-02
**Updated**: 2026-01-06
**Author**: Claude + dahendel

**⚠️ NOTE**: This spec is partially outdated. contextd v2 uses **chromem** (embedded) as the default vector store, with Qdrant as an optional external provider. The Docker and deployment patterns described here need updating.

---

## Overview

contextd is distributed as an all-in-one Docker container that includes:
- contextd MCP server
- **chromem vector store (embedded, default)** or Qdrant (optional external)
- FastEmbed for local embeddings (no external API needed)

Users need only Docker to run contextd. All data persists in a named volume.

---

## Quick Start

### 1. Add to Claude Code MCP Config

**Location**: `~/.claude/claude_desktop_config.json` (macOS/Linux) or `%APPDATA%/Claude/claude_desktop_config.json` (Windows)

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "fyrsmithlabs/contextd:latest"
      ]
    }
  }
}
```

### 2. Restart Claude Code

That's it. contextd will automatically:
- Pull the image on first run
- Create the `contextd-data` volume
- Start Qdrant and contextd
- Persist all memories, checkpoints, and remediations

---

## Environment Variables

All configuration is optional. Sensible defaults are provided.

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `CONTEXTD_DATA_PATH` | `/data` | Base path for all persistent data |
| `CONTEXTD_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### Vector Store Settings

**Default**: chromem (embedded, no configuration needed)

**Qdrant (Optional External)**:

| Variable | Default | Description |
|----------|---------|-------------|
| `VECTORSTORE_PROVIDER` | `chromem` | Provider: `chromem` (default) or `qdrant` |
| `QDRANT_HOST` | `localhost` | Qdrant host (if using Qdrant) |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port |
| `QDRANT_HTTP_PORT` | `6333` | Qdrant HTTP port |
| `VECTORSTORE_DEFAULT_COLLECTION` | `memories` | Default collection name |
| `VECTORSTORE_VECTOR_SIZE` | `384` | Vector dimensions (FastEmbed default) |

### Embeddings Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `EMBEDDINGS_PROVIDER` | `fastembed` | Provider: `fastembed` or `tei` |
| `EMBEDDINGS_MODEL` | `BAAI/bge-small-en-v1.5` | Embedding model |
| `EMBEDDING_BASE_URL` | `http://localhost:8080` | TEI URL (if using TEI) |
| `ONNX_PATH` | (auto-detected) | Path to libonnxruntime.so (FastEmbed only) |

**Supported FastEmbed Models:**

| Model | Dimensions | Notes |
|-------|------------|-------|
| `BAAI/bge-small-en-v1.5` | 384 | Default, fast, English |
| `BAAI/bge-base-en-v1.5` | 768 | Higher quality, English |
| `sentence-transformers/all-MiniLM-L6-v2` | 384 | General purpose |
| `BAAI/bge-small-zh-v1.5` | 512 | Chinese |

### ONNX Runtime Setup (FastEmbed only)

FastEmbed requires ONNX Runtime. The Docker image includes it automatically.

**For local development:**

```bash
# Ubuntu/Debian
wget https://github.com/microsoft/onnxruntime/releases/download/v1.16.3/onnxruntime-linux-x64-1.16.3.tgz
tar xzf onnxruntime-linux-x64-1.16.3.tgz
sudo cp onnxruntime-linux-x64-1.16.3/lib/libonnxruntime.so* /usr/lib/
sudo ldconfig

# macOS
brew install onnxruntime

# Or set ONNX_PATH manually
export ONNX_PATH=/path/to/libonnxruntime.so
```

**Fallback to TEI:** If ONNX is unavailable, set `EMBEDDINGS_PROVIDER=tei` and run a TEI server.

### Server Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `9090` | HTTP API port |
| `SERVER_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |

### Checkpoint Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `CHECKPOINT_MAX_CONTENT_SIZE_KB` | `1024` | Max checkpoint size (KB) |
| `CHECKPOINT_THRESHOLD_PERCENT` | `70` | Auto-checkpoint trigger (%) |

### Telemetry Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_ENABLE` | `true` | Enable OpenTelemetry |
| `OTEL_SERVICE_NAME` | `contextd` | Service name for traces |

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

### Volume Management

```bash
# List volumes
docker volume ls | grep contextd

# Inspect volume
docker volume inspect contextd-data

# Backup volume
docker run --rm -v contextd-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/contextd-backup.tar.gz /data

# Restore volume
docker run --rm -v contextd-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/contextd-backup.tar.gz -C /

# Delete volume (WARNING: destroys all data)
docker volume rm contextd-data
```

---

## Docker Image

### Building

```bash
# Clone repo
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd

# Build image
docker build -t fyrsmithlabs/contextd:latest .

# Build with specific version
docker build -t fyrsmithlabs/contextd:v0.1.0 .
```

### Dockerfile

```dockerfile
# Stage 1: Build contextd
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o contextd ./cmd/contextd

# Stage 2: Runtime with Qdrant
FROM qdrant/qdrant:latest AS qdrant

FROM alpine:latest
RUN apk add --no-cache ca-certificates supervisor

# Copy Qdrant binary
COPY --from=qdrant /qdrant/qdrant /usr/local/bin/qdrant

# Copy contextd binary
COPY --from=builder /build/contextd /usr/local/bin/contextd

# Copy supervisor config
COPY deploy/supervisord.conf /etc/supervisord.conf

# Copy entrypoint
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Data volume
VOLUME /data

# Expose ports (optional, for debugging)
EXPOSE 6333 6334 9090

ENTRYPOINT ["/entrypoint.sh"]
```

### Entrypoint Script

```bash
#!/bin/sh
set -e

# Create data directories
mkdir -p /data/qdrant/storage
mkdir -p /data/logs

# Set Qdrant storage path
export QDRANT__STORAGE__STORAGE_PATH=/data/qdrant/storage

# Start supervisor (manages Qdrant + contextd)
exec /usr/bin/supervisord -c /etc/supervisord.conf
```

### Supervisor Config

```ini
[supervisord]
nodaemon=true
logfile=/data/logs/supervisord.log

[program:qdrant]
command=/usr/local/bin/qdrant
autostart=true
autorestart=true
stdout_logfile=/data/logs/qdrant.log
stderr_logfile=/data/logs/qdrant.err

[program:contextd]
command=/usr/local/bin/contextd
autostart=true
autorestart=true
startsecs=5
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
```

---

## Alternative: Docker Compose

For users who want more control or to run Qdrant separately:

```yaml
version: '3.8'

services:
  qdrant:
    image: qdrant/qdrant:latest
    volumes:
      - qdrant-data:/qdrant/storage
    ports:
      - "6333:6333"
      - "6334:6334"
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:6333/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  contextd:
    image: fyrsmithlabs/contextd:latest
    depends_on:
      qdrant:
        condition: service_healthy
    environment:
      - QDRANT_HOST=qdrant
      - QDRANT_PORT=6334
    volumes:
      - contextd-config:/data/config
    stdin_open: true
    tty: true

volumes:
  qdrant-data:
  contextd-config:
```

---

## CLI Setup Command (Future)

```bash
# Auto-configure Claude Code
contextd setup

# What it does:
# 1. Detects Claude Code config location
# 2. Adds contextd MCP server entry
# 3. Creates docker volume if needed
# 4. Verifies connection

# Manual config location override
contextd setup --config ~/.claude/claude_desktop_config.json

# Show current config
contextd setup --show
```

---

## Verification

### Check MCP Connection

In Claude Code:
```
/mcp
```

Should show `contextd` as connected.

### Test Tools

```
# In Claude Code conversation:
Use memory_search to find any existing memories for this project.
```

### Health Check

```bash
# If HTTP port exposed
curl http://localhost:9090/health
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs $(docker ps -aq --filter ancestor=fyrsmithlabs/contextd:latest)

# Run interactively
docker run -it --rm -v contextd-data:/data fyrsmithlabs/contextd:latest /bin/sh
```

### Qdrant Connection Issues

```bash
# Check Qdrant is running inside container
docker exec -it <container_id> wget -q --spider http://localhost:6333/health && echo "OK"
```

### Data Not Persisting

```bash
# Verify volume is mounted
docker inspect <container_id> | grep -A5 "Mounts"

# Check volume exists
docker volume inspect contextd-data
```

### Reset Everything

```bash
# Stop all contextd containers
docker stop $(docker ps -q --filter ancestor=fyrsmithlabs/contextd:latest)

# Remove volume (WARNING: destroys data)
docker volume rm contextd-data

# Pull fresh image
docker pull fyrsmithlabs/contextd:latest
```

---

## Security Considerations

### Data Privacy

- All data stays local in Docker volume
- No external API calls (FastEmbed runs locally)
- No telemetry sent externally by default

### Container Security

- Runs as non-root user (future)
- Minimal Alpine base image
- No unnecessary ports exposed

### Secrets

- contextd scrubs secrets from all tool responses using gitleaks
- Never stores API keys, tokens, or credentials

---

## Implementation Phases

### Phase 1: Environment Variable Config (Current)
- [ ] Add Qdrant env var loading to config.go
- [ ] Add Embeddings env var loading
- [ ] Add data path configuration

### Phase 2: FastEmbed Integration ✅
- [x] Replace TEI with Qdrant FastEmbed
- [x] Update embeddings service (provider pattern)
- [x] Test vector dimensions

### Phase 3: Docker Image
- [ ] Create Dockerfile
- [ ] Create entrypoint.sh
- [ ] Create supervisord.conf
- [ ] Test all-in-one container

### Phase 4: Distribution
- [ ] Push to Docker Hub (fyrsmithlabs/contextd)
- [ ] Push to GHCR (ghcr.io/fyrsmithlabs/contextd)
- [ ] Create GitHub release workflow

### Phase 5: CLI Setup Command
- [ ] Implement `contextd setup` command
- [ ] Auto-detect Claude Code config
- [ ] Add verification step
