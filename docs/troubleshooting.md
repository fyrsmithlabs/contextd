# Troubleshooting Guide

This guide covers common issues when running ContextD and how to resolve them.

---

## Quick Diagnostics

### Health Check

```bash
# Check ContextD HTTP server
curl http://localhost:9090/health

# Check Qdrant
curl http://localhost:6333/health

# Check container is running
docker ps | grep contextd
```

### Logs

```bash
# View container logs
docker logs <container_id>

# Follow logs in real-time
docker logs -f <container_id>

# Last 100 lines
docker logs --tail 100 <container_id>
```

---

## Common Issues

### Container Won't Start

**Symptom:** Container exits immediately after starting.

**Diagnostic Steps:**

```bash
# Check container exit code and logs
docker logs <container_id>

# Run interactively to see output
docker run -it --rm ghcr.io/fyrsmithlabs/contextd:latest
```

**Common Causes:**

1. **Port conflict** - Another service using port 6333 or 6334
   ```bash
   # Check what's using the ports
   lsof -i :6333
   lsof -i :6334
   ```

2. **Volume permissions** - Data directory not writable
   ```bash
   # Check volume permissions
   docker run --rm -v contextd-data:/data alpine ls -la /data
   ```

3. **Memory constraints** - Not enough memory for Qdrant + embeddings
   ```bash
   # Check available memory
   docker stats --no-stream
   ```
   **Solution:** Allocate at least 512MB RAM to the container.

---

### Qdrant Client Version Warning

**Symptom:** Log message like:
```
WARN Client version is not compatible with server version. Major versions should match...
clientVersion=v1.16.2 serverVersion=1.12.1
```

**Cause:** The Go Qdrant client (v1.16.2) is newer than the bundled Qdrant server (v1.12.1).

**Impact:** This is a **warning only** - the system works correctly. The client maintains backward compatibility.

**Solution:** No action needed. This warning can be safely ignored.

---

### Qdrant Connection Failed

**Symptom:** Error like `connection refused: dial tcp 127.0.0.1:6334`

**Causes:**

1. **Qdrant not started** - The embedded Qdrant hasn't finished starting
   ```bash
   # Check if Qdrant is responding
   curl http://localhost:6333/health
   ```

2. **External Qdrant misconfigured** - Wrong host/port
   ```bash
   # Verify environment variables
   echo $QDRANT_HOST
   echo $QDRANT_PORT
   ```

**Solution:**

```bash
# For external Qdrant, ensure correct configuration
docker run -i --rm \
  -e QDRANT_HOST=your-qdrant-host \
  -e QDRANT_PORT=6334 \
  ghcr.io/fyrsmithlabs/contextd:latest
```

---

### ONNX Runtime Errors

**Symptom:** Errors mentioning ONNX, ORT, or embedding failures.

**Common Errors:**

1. **API version mismatch**
   ```
   Error: API version [22] is not available, only API versions [1, 20] are supported
   ```
   **Cause:** ONNX runtime version mismatch.
   **Solution:** The Docker image includes the correct ONNX runtime (v1.23.2). If building from source, ensure you use matching versions.

2. **Library not found**
   ```
   error while loading shared libraries: libonnxruntime.so
   ```
   **Solution:** Set `LD_LIBRARY_PATH` to include the ONNX runtime library path:
   ```bash
   export LD_LIBRARY_PATH=/usr/lib:$LD_LIBRARY_PATH
   ```

---

### Vector Dimension Mismatch

**Symptom:** Error about vector dimensions not matching.

**Cause:** The `QDRANT_VECTOR_SIZE` doesn't match the embedding model dimensions.

| Model | Dimensions |
|-------|------------|
| `BAAI/bge-small-en-v1.5` | 384 |
| `BAAI/bge-base-en-v1.5` | 768 |
| `sentence-transformers/all-MiniLM-L6-v2` | 384 |

**Solution:**

```bash
# Match vector size to model
docker run -i --rm \
  -e EMBEDDINGS_MODEL=BAAI/bge-base-en-v1.5 \
  -e QDRANT_VECTOR_SIZE=768 \
  ghcr.io/fyrsmithlabs/contextd:latest
```

**Warning:** Changing models after data is stored requires recreating collections.

---

### Memory Search Returns No Results

**Symptom:** `memory_search` returns empty even when memories exist.

**Diagnostic Steps:**

1. **Check if memories were recorded**
   - Memories need the correct `project_id`
   - Verify the project ID matches when searching

2. **Check confidence threshold**
   - Memories with very low confidence may be filtered
   - Provide positive feedback to boost confidence

3. **Check tenant isolation**
   - Different tenants cannot see each other's data
   - Verify `tenant_id` is consistent

**Solution:**

```json
{
  "tool": "memory_search",
  "arguments": {
    "project_id": "contextd",
    "query": "your search query",
    "limit": 10
  }
}
```

---

### Checkpoint Resume Fails

**Symptom:** `checkpoint_resume` returns "not found" error.

**Causes:**

1. **Wrong checkpoint ID** - Typo or deleted checkpoint
2. **Wrong tenant** - Checkpoints are tenant-isolated
3. **Checkpoint expired** - Old checkpoints may be cleaned up

**Diagnostic:**

```json
{
  "tool": "checkpoint_list",
  "arguments": {
    "tenant_id": "your-tenant-id",
    "limit": 50
  }
}
```

---

### Slow First Startup

**Symptom:** Container takes 30-40 seconds to respond on first run.

**Cause:** Multiple initialization steps on first startup:
1. Qdrant database initialization (~10s)
2. Embedding model download (~5s on first run, cached after)
3. Service initialization (~5s)

**Solution:** This is expected behavior. Subsequent startups are faster because:
- The embedding model is cached in the Docker volume
- Qdrant data is persisted

### Slow Embedding Performance

**Symptom:** First embedding takes a long time (10-30 seconds).

**Cause:** The ONNX model is loaded on first use.

**Solution:** This is expected behavior. Subsequent embeddings are fast (10-20ms).

For faster cold starts, consider:
1. Using a smaller model (`bge-small` instead of `bge-base`)
2. Pre-warming the embedding model by making a dummy query at startup

---

### Docker Volume Issues

**Symptom:** Data not persisting between container restarts.

**Check:**

```bash
# Verify volume exists
docker volume ls | grep contextd

# Inspect volume
docker volume inspect contextd-data

# Check volume contents
docker run --rm -v contextd-data:/data alpine ls -la /data
```

**Solution:**

```bash
# Create volume if missing
docker volume create contextd-data

# Run with volume
docker run -i --rm -v contextd-data:/data ghcr.io/fyrsmithlabs/contextd:latest
```

---

### Secret Detection False Positives

**Symptom:** Non-secret content is being redacted.

**Cause:** Gitleaks pattern matching is aggressive by design.

**Examples of false positives:**
- UUIDs that look like API keys
- Base64-encoded content
- Long alphanumeric strings

**Note:** False positives are preferred over false negatives for security.

---

## MCP Integration Issues

### Claude Code Not Seeing ContextD

**Symptom:** ContextD tools not appearing in Claude Code.

**Check:**

1. **Configuration file location**
   ```bash
   cat ~/.claude/claude_desktop_config.json
   ```

2. **JSON syntax**
   ```bash
   # Validate JSON
   python3 -m json.tool ~/.claude/claude_desktop_config.json
   ```

3. **Docker available**
   ```bash
   docker --version
   ```

**Solution:**

1. Ensure valid JSON in config
2. Restart Claude Code
3. Check Docker daemon is running

### Tool Calls Timing Out

**Symptom:** MCP tool calls hang or timeout.

**Causes:**

1. **Container not started** - MCP waits for stdio
2. **Qdrant slow to start** - First query may timeout
3. **Large repository indexing** - Can take minutes

**Solution:**

For large operations, increase timeout or use async patterns.

---

## Performance Tuning

### Memory Usage

Typical memory consumption:
- **Qdrant:** ~100MB base + indexed data
- **FastEmbed:** ~200MB for model
- **ContextD:** ~50MB

**Reduce memory:**

```bash
# Use smaller embedding model
-e EMBEDDINGS_MODEL=BAAI/bge-small-en-v1.5

# Limit Qdrant memory (external Qdrant)
--env QDRANT_MEMORY_LIMIT=256MB
```

### Embedding Latency

| Operation | Typical Latency |
|-----------|-----------------|
| First embedding (cold) | 5-15s |
| Subsequent embeddings | 10-20ms |
| Memory search | 50-100ms |
| Checkpoint save | 100-200ms |

---

## Getting Help

If you're still stuck:

1. **Check logs** - Most issues have clear error messages
2. **Search issues** - https://github.com/fyrsmithlabs/contextd/issues
3. **Open an issue** - Include logs and reproduction steps

### Useful Information for Bug Reports

```bash
# System info
uname -a
docker version
go version  # if building from source

# Container info
docker inspect <container_id>

# Recent logs
docker logs --tail 200 <container_id> 2>&1
```
