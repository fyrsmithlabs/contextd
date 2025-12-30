# Qdrant Configuration Examples

This directory contains production-ready configuration examples for using contextd with Qdrant as the vector storage backend.

## Overview

contextd supports two vector storage providers:

| Provider | Type | Use Case |
|----------|------|----------|
| **chromem** | Embedded | Single-user, local development, small datasets |
| **Qdrant** | External | Multi-user, production, large datasets, horizontal scaling |

## Why Qdrant gRPC?

The Qdrant gRPC implementation (issue #15) solves the 256kB HTTP payload limit:

- **Problem**: Qdrant's HTTP REST API (port 6333) has a 256kB actix-web limit, causing 413 errors on large files
- **Solution**: Native gRPC client (port 6334) bypasses HTTP layer entirely
- **Benefits**: 50MB default limit (configurable to 200MB+), better performance, binary protobuf encoding

## Configuration Files

### [dev.yaml](./dev.yaml)

**For**: Local development with Docker Qdrant

```bash
# Start Qdrant
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant

# Run contextd with dev config
contextd --config examples/qdrant-config/dev.yaml
```

**Features**:
- Localhost connection
- No TLS (development only)
- 50MB max message size
- Debug logging

---

### [prod.yaml](./prod.yaml)

**For**: Production deployments

```bash
# Set environment variables
export QDRANT_HOST=qdrant.prod.example.com
export QDRANT_USE_TLS=true
export LOG_LEVEL=info

# Run contextd with prod config
contextd --config examples/qdrant-config/prod.yaml
```

**Features**:
- TLS encryption enabled
- 100MB max message size
- Environment variable configuration
- Structured JSON logging
- OpenTelemetry support

---

### [large-repos.yaml](./large-repos.yaml)

**For**: Indexing very large codebases (>1GB, files >10MB)

```bash
# Run contextd with large repo config
contextd --config examples/qdrant-config/large-repos.yaml
```

**Features**:
- 200MB max message size for huge files
- 10MB max file size for repository indexing
- Higher circuit breaker threshold (15 vs 5)
- Optimized retry backoff (5s vs 1s)
- Batch embedding configuration

**Verified capabilities**:
- ✅ 500KB documents (2x HTTP limit)
- ✅ 5MB documents (20x HTTP limit)
- ✅ 25MB documents (100x HTTP limit)
- ✅ Batch uploads (10MB total, 100 files)

---

## Quick Start

### 1. Choose Your Configuration

| Scenario | Config File |
|----------|-------------|
| Local development | `dev.yaml` |
| Production deployment | `prod.yaml` |
| Large monorepos | `large-repos.yaml` |

### 2. Start Qdrant

```bash
# Development (Docker)
docker run -d \
  -p 6333:6333 \
  -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant

# Production (see Qdrant docs for cluster setup)
```

### 3. Run contextd

```bash
# Copy and customize config
cp examples/qdrant-config/dev.yaml config.yaml
vim config.yaml  # Edit as needed

# Run contextd
contextd --config config.yaml
```

## Configuration Reference

### Required Settings

```yaml
vectorstore:
  provider: qdrant  # Use Qdrant instead of chromem

qdrant:
  host: localhost       # Qdrant server address
  port: 6334           # gRPC port (NOT 6333)
  collection_name: memories  # Default collection
  vector_size: 384     # Must match embedding model
```

### Performance Tuning

```yaml
qdrant:
  max_message_size: 104857600  # 100MB (adjust for your files)
  max_retries: 5               # Retry attempts
  retry_backoff: 2s            # Exponential backoff start
  circuit_breaker_threshold: 10  # Failures before opening
```

### Security

```yaml
qdrant:
  use_tls: true  # Enable for production

embeddings:
  # Use TEI for production instead of local FastEmbed
  provider: tei
  endpoint: http://embeddings.internal:8080
```

## Troubleshooting

### Error: "connection refused" on port 6334

**Cause**: Qdrant not running or using wrong port

**Solution**:
```bash
# Check Qdrant is running
docker ps | grep qdrant

# Verify ports are exposed
docker port <container-id>

# Should show:
# 6333/tcp -> 0.0.0.0:6333
# 6334/tcp -> 0.0.0.0:6334
```

### Error: "rpc error: code = Unauthenticated"

**Cause**: Missing or invalid API key for Qdrant Cloud

**Solution**:
```bash
# Set API key environment variable
export QDRANT_API_KEY=your_api_key_here

# Or add to prod.yaml
qdrant:
  api_key: ${QDRANT_API_KEY:}
```

**Note**: API keys are required for Qdrant Cloud but not for self-hosted instances.

### Error: "413 Payload Too Large"

**Cause**: Using HTTP REST API instead of gRPC

**Solution**:
```yaml
qdrant:
  port: 6334  # NOT 6333 (HTTP)
```

### Error: "rpc error: code = ResourceExhausted"

**Cause**: Payload exceeds `max_message_size`

**Solution**:
```yaml
qdrant:
  max_message_size: 209715200  # Increase to 200MB
```

### Slow indexing performance

**Cause**: Network latency or small batch sizes

**Solution**:
```yaml
embeddings:
  batch_size: 100  # Larger batches

qdrant:
  max_message_size: 104857600  # Allow larger batches
```

### Error: "rpc error: code = AlreadyExists"

**Cause**: Collection name conflict during migration or reindexing

**Solution**:
```bash
# Option 1: Delete existing collection (loses data)
ctxd collection delete <collection_name>

# Option 2: Use a different collection name
qdrant:
  collection_name: contextd_v2  # New name
```

**Note**: AlreadyExists is safe to ignore if you're resuming an interrupted indexing operation.

### Error: "tls: handshake failure" or "certificate verify failed"

**Cause**: TLS configuration mismatch or invalid certificate

**Solution**:
```yaml
# For Qdrant Cloud (TLS required)
qdrant:
  use_tls: true
  host: xyz-example.qdrant.io

# For self-hosted without TLS (development only)
qdrant:
  use_tls: false
  host: localhost
```

**Note**: Self-signed certificates may require additional CA configuration. For production, use valid TLS certificates.

### Error: "rpc error: code = DeadlineExceeded" or "context deadline exceeded"

**Cause**: Operation timeout (network latency, large payloads, or slow Qdrant instance)

**Solution**:
```yaml
# Increase timeout for large operations
qdrant:
  timeout: 60s  # Default is 30s

# Or reduce batch size to avoid timeouts
embeddings:
  batch_size: 50  # Smaller batches (default: 100)
```

**Note**: Persistent DeadlineExceeded errors may indicate insufficient Qdrant resources (CPU/memory).

## Migration from chromem

To migrate from chromem to Qdrant:

1. Export existing data (if needed):
   ```bash
   # chromem stores data in ~/.config/contextd/vectorstore/
   # Data is in gob format (not directly portable)
   ```

2. Update configuration:
   ```yaml
   vectorstore:
     provider: qdrant  # Change from chromem
   ```

3. Re-index repositories:
   ```bash
   # Use MCP tool or ctxd CLI
   repository_index(project_path="/path/to/repo")
   ```

4. Verify migration:
   ```bash
   # Check collection exists
   qdrant-cli collection list

   # Verify point count
   qdrant-cli collection info contextd_prod
   ```

## Performance Benchmarks

### Local Development (localhost)

- **Latency**: 5-10ms per operation
- **Throughput**: 1000-5000 docs/second (depending on size)
- **Batch upload**: 100 x 100KB files in ~2-5 seconds

### Production (network)

- **Latency**: 20-100ms per operation (+ network RTT)
- **Throughput**: 100-1000 queries/second
- **Bottleneck**: Network bandwidth and Qdrant capacity

## References

- **Qdrant Documentation**: https://qdrant.tech/documentation/
- **Qdrant Go Client**: https://github.com/qdrant/go-client
- **Issue #15**: feat: Implement QdrantGRPCStore to bypass 256kB HTTP payload limit
- **Implementation**: `internal/vectorstore/qdrant.go`
- **Tests**: `internal/vectorstore/qdrant_large_payload_test.go`
