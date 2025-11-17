# Text Embeddings Inference (TEI) Deployment Guide

This guide shows how to deploy and use TEI (Text Embeddings Inference by HuggingFace) as a local, quota-free alternative to OpenAI embeddings.

## Why TEI?

- ✅ **No API quotas or rate limits** - Run unlimited embeddings locally
- ✅ **Cost-free** - No per-token charges
- ✅ **Privacy** - Data never leaves your infrastructure
- ✅ **OpenAI-compatible API** - Drop-in replacement
- ✅ **Production-ready** - Optimized for throughput and latency
- ✅ **Multiple deployment options** - Docker, Kubernetes, bare metal

## Quick Start (Docker)

TEI is already configured in `docker-compose.yml`:

```bash
# Start TEI service
docker-compose up -d tei

# Verify it's running
curl http://localhost:8080/health

# Test embeddings
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{"input": "Hello world", "model": "BAAI/bge-small-en-v1.5"}'
```

## Configuration

### Docker Compose (Included)

```yaml
tei:
  container_name: local-tei
  image: ghcr.io/huggingface/text-embeddings-inference:cpu-1.2
  command:
    - "--model-id"
    - "BAAI/bge-small-en-v1.5"  # 384-dim, fast, high quality
    - "--port"
    - "8080"
  ports:
    - "8080:8080"
  volumes:
    - ${HOME}/.cache/huggingface:/data
  restart: unless-stopped
```

### Available Models

TEI supports many embedding models. Popular choices:

| Model | Dimensions | Size | Speed | Quality |
|-------|-----------|------|-------|---------|
| `BAAI/bge-small-en-v1.5` | 384 | 133MB | Fast | Good |
| `BAAI/bge-base-en-v1.5` | 768 | 438MB | Medium | Better |
| `BAAI/bge-large-en-v1.5` | 1024 | 1.34GB | Slow | Best |
| `sentence-transformers/all-MiniLM-L6-v2` | 384 | 80MB | Fastest | Good |
| `thenlper/gte-large` | 1024 | 670MB | Medium | Excellent |

To change models:
```yaml
# In docker-compose.yml
command:
  - "--model-id"
  - "BAAI/bge-base-en-v1.5"  # Change this line
```

## Contextd Integration

### Option 1: Using claude mcp add (Recommended)

```bash
claude mcp add --scope user --transport stdio contextd \
  --env EMBEDDING_BASE_URL=http://localhost:8080/v1 \
  --env EMBEDDING_MODEL=BAAI/bge-small-en-v1.5 \
  --env OTEL_ENVIRONMENT=local \
  --env OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
  --env OTEL_SERVICE_NAME=contextd \
  -- /usr/local/bin/contextd --mcp
```

### Option 2: Manual Configuration

Edit `~/.claude.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "/usr/local/bin/contextd",
      "args": ["--mcp"],
      "env": {
        "EMBEDDING_BASE_URL": "http://localhost:8080/v1",
        "EMBEDDING_MODEL": "BAAI/bge-small-en-v1.5",
        "OTEL_ENVIRONMENT": "local",
        "OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318",
        "OTEL_SERVICE_NAME": "contextd"
      }
    }
  }
}
```

### Verify Connection

```bash
# Check MCP server status
claude mcp list

# Should show:
# contextd: /usr/local/bin/contextd --mcp - ✓ Connected
```

## Environment Variables

| Variable | Description | Default | TEI Value |
|----------|-------------|---------|-----------|
| `EMBEDDING_BASE_URL` | API endpoint URL | OpenAI | `http://localhost:8080/v1` |
| `EMBEDDING_MODEL` | Model identifier | `text-embedding-3-small` | `BAAI/bge-small-en-v1.5` |
| `OPENAI_API_KEY` | API key | Required | `dummy-key` (not used by TEI) |
| `EMBEDDING_DIM` | Vector dimensions | 768 | 384 (for bge-small) |
| `EMBEDDING_MAX_BATCH_SIZE` | Max batch size | 2048 | 512 |
| `EMBEDDING_ENABLE_CACHE` | Enable caching | true | true |

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tei-embedding
spec:
  replicas: 2
  selector:
    matchLabels:
      app: tei
  template:
    metadata:
      labels:
        app: tei
    spec:
      containers:
      - name: tei
        image: ghcr.io/huggingface/text-embeddings-inference:cpu-1.2
        args:
          - "--model-id"
          - "BAAI/bge-small-en-v1.5"
          - "--port"
          - "8080"
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 60
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: tei-service
spec:
  selector:
    app: tei
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

Then configure contextd to use `http://tei-service:8080/v1`.

## GPU Acceleration (Optional)

For better performance with GPU:

```yaml
tei:
  image: ghcr.io/huggingface/text-embeddings-inference:1.2
  deploy:
    resources:
      reservations:
        devices:
          - driver: nvidia
            count: 1
            capabilities: [gpu]
  command:
    - "--model-id"
    - "BAAI/bge-large-en-v1.5"  # Use larger model with GPU
    - "--port"
    - "8080"
```

## Performance Tuning

### Batch Size

TEI automatically batches requests. Configure optimal batch size:

```yaml
command:
  - "--model-id"
  - "BAAI/bge-small-en-v1.5"
  - "--max-batch-tokens"
  - "16384"  # Adjust based on available memory
  - "--max-client-batch-size"
  - "512"    # Max texts per request
```

### Concurrent Requests

```yaml
command:
  - "--max-concurrent-requests"
  - "512"  # Increase for high throughput
```

### Memory Management

```yaml
deploy:
  resources:
    limits:
      memory: 4G  # Increase for larger models
```

## Monitoring

TEI exposes Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

Key metrics:
- `te_request_duration_seconds` - Request latency
- `te_batch_size` - Batch sizes
- `te_queue_size` - Request queue depth

## Troubleshooting

### TEI not responding

```bash
# Check container status
docker-compose ps tei

# View logs
docker-compose logs -f tei

# Restart service
docker-compose restart tei
```

### Model download issues

TEI downloads models on first start. This can take time:

```bash
# Monitor download progress
docker-compose logs -f tei | grep -i download
```

### Out of memory errors

Reduce batch size or use smaller model:

```yaml
command:
  - "--model-id"
  - "sentence-transformers/all-MiniLM-L6-v2"  # Smaller model
  - "--max-batch-tokens"
  - "8192"  # Reduce batch size
```

### Connection refused errors

Ensure TEI is fully started before starting contextd:

```bash
# Wait for TEI to be ready
curl --retry 10 --retry-delay 5 http://localhost:8080/health

# Then start contextd
systemctl --user start contextd
```

## Switching Back to OpenAI

To switch back to OpenAI embeddings:

```bash
# Remove TEI configuration
claude mcp remove contextd

# Add with OpenAI (no EMBEDDING_BASE_URL)
claude mcp add --scope user --transport stdio contextd \
  -- /usr/local/bin/contextd --mcp
```

And ensure `OPENAI_API_KEY` is set:
```bash
echo "sk-your-key" > ~/.config/contextd/openai_api_key
chmod 0600 ~/.config/contextd/openai_api_key
```

## Cost Comparison

### OpenAI (text-embedding-3-small)
- **Cost**: $0.02 per 1M tokens
- **1,000 embeddings** (~500 tokens each): $0.01
- **Monthly (100K embeddings)**: $1.00

### TEI (Local)
- **Cost**: $0 (hardware already owned)
- **1,000 embeddings**: $0
- **Monthly (unlimited)**: $0

**Break-even**: Instant! Any usage saves money.

## Further Reading

- [TEI Documentation](https://huggingface.co/docs/text-embeddings-inference)
- [Model Selection Guide](https://www.sbert.net/docs/pretrained_models.html)
- [Embedding Best Practices](https://huggingface.co/blog/getting-started-with-embeddings)
