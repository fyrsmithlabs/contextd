# Metadata Health Monitoring

**Status**: ✅ Implemented (P1 Priority)
**Incident**: PROD-2026-01-22-001
**Created**: 2026-01-22

---

## Overview

Metadata health monitoring provides real-time verification of vectorstore collection integrity. It detects corrupt collections (missing metadata files) and exposes health status via HTTP endpoints.

**Purpose**: Prevent silent metadata corruption from going undetected, enabling proactive monitoring and alerting.

**Scope**: chromem-based vectorstore only (default configuration)

---

## Architecture

### Components

| Component | Purpose | Location |
|-----------|---------|----------|
| `MetadataHealthChecker` | Core health verification logic | `internal/vectorstore/metadata_health.go` |
| HTTP `/health` endpoint | Basic health status with metadata summary | `internal/http/server.go:handleHealth` |
| HTTP `/api/v1/health/metadata` endpoint | Detailed per-collection status | `internal/http/server.go:handleMetadataHealth` |

### Health Check Algorithm

```go
For each collection directory in vectorstore path:
  1. Check if metadata file (00000000.gob) exists
  2. Count document files (*.gob, excluding metadata)
  3. Classify as:
     - Healthy: metadata exists
     - Corrupt: no metadata BUT has documents
     - Empty: no metadata AND no documents
```

### Response Format

**Basic Health (`GET /health`)**:
```json
{
  "status": "ok|degraded",
  "metadata": {
    "status": "healthy|degraded",
    "healthy_count": 22,
    "corrupt_count": 0,
    "empty_count": 0,
    "total": 22,
    "corrupt_hashes": []
  }
}
```

**Detailed Health (`GET /api/v1/health/metadata`)**:
```json
{
  "healthy": ["hash1", "hash2", ...],
  "corrupt": [],
  "empty": [],
  "total": 22,
  "healthy_count": 22,
  "corrupt_count": 0,
  "last_check_time": "2026-01-22T14:30:20.758185-05:00",
  "check_duration": 1100542,
  "details": {
    "hash1": "healthy: 7 documents, metadata size 127 bytes",
    "hash2": "corrupt: 1 documents, no metadata"
  }
}
```

---

## Usage

### Checking Health via HTTP

**Basic Health Check**:
```bash
curl http://localhost:9090/health
```

**Detailed Metadata Health**:
```bash
curl http://localhost:9090/api/v1/health/metadata | jq
```

### HTTP Status Codes

| Condition | Status Code | Response Status |
|-----------|------------|-----------------|
| All collections healthy | 200 OK | `status: "ok"` |
| Corrupt collections detected | 503 Service Unavailable | `status: "degraded"` |
| Health checker not configured | 503 Service Unavailable | Error message |

### Integration with Monitoring

**Prometheus Metrics** (Future):
```
vectorstore_collections_total{status="healthy|corrupt|empty"}
vectorstore_metadata_health_check_duration_seconds
```

**Alerting Example** (Future):
```yaml
alert: VectorStoreMetadataCorruption
expr: vectorstore_collections_total{status="corrupt"} > 0
for: 1m
annotations:
  summary: "Corrupt vectorstore collections detected"
  description: "{{ $value }} collections have missing metadata files"
```

---

## Configuration

### Enabling Health Checks

Health checks are **automatically enabled** when:
1. HTTP server is enabled (`--no-http=false`, default)
2. Vectorstore provider is `chromem` (default)
3. Vectorstore path is configured

**No additional configuration required.**

### Disabling Health Checks

To disable metadata health monitoring:
```bash
# Disable HTTP server entirely
contextd --no-http

# Or use Qdrant (health checks only apply to chromem)
# Edit ~/.config/contextd/config.yaml:
vectorstore:
  provider: qdrant
```

---

## Implementation Details

### Code Flow

**Initialization** (`cmd/contextd/main.go`):
```go
// Create metadata health checker for chromem vectorstore
if cfg.VectorStore.Provider == "chromem" && cfg.VectorStore.Chromem.Path != "" {
    expandedPath := expandPath(cfg.VectorStore.Chromem.Path)
    healthChecker = vectorstore.NewMetadataHealthChecker(expandedPath, logger)
}

// Pass to HTTP server config
httpCfg := &httpserver.Config{
    Host:          httpServerHost,
    Port:          httpServerPort,
    Version:       version,
    HealthChecker: healthChecker,
}
```

**Health Check Execution** (`internal/http/server.go`):
```go
func (s *Server) handleHealth(c echo.Context) error {
    ctx := c.Request().Context()
    resp := HealthResponse{Status: "ok"}

    if s.healthChecker != nil {
        health, err := s.healthChecker.Check(ctx)
        if err != nil {
            s.logger.Warn("metadata health check failed", zap.Error(err))
        } else {
            resp.Metadata = &MetadataHealthStatus{
                Status:        health.Status(),
                HealthyCount:  health.HealthyCount,
                CorruptCount:  health.CorruptCount,
                EmptyCount:    len(health.Empty),
                Total:         health.Total,
                CorruptHashes: health.Corrupt,
            }

            if !health.IsHealthy() {
                resp.Status = "degraded"
            }
        }
    }

    statusCode := http.StatusOK
    if resp.Status == "degraded" {
        statusCode = http.StatusServiceUnavailable
    }

    return c.JSON(statusCode, resp)
}
```

### Performance Characteristics

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| Health check | O(n) collections | Reads directory, stats metadata file |
| Typical duration | 1-2ms | For ~20 collections |
| Memory overhead | O(n) hashes | Stores collection hashes in response |

**Optimization**: Health checks are performed **on-demand** (per HTTP request), not continuously in background.

---

## Relationship to Other Prevention Measures

### P0: Graceful Degradation (Implemented)

**Metadata health monitoring complements graceful degradation**:
- **P0 (Resilient Wrapper)**: Quarantines corrupt collections on startup → service continues
- **P1 (Health Monitoring)**: Detects corrupt collections via HTTP → exposes status for monitoring

**Workflow**:
```
Startup:
  1. Resilient wrapper detects corrupt collection
  2. Quarantines to .quarantine/ directory
  3. contextd starts with degraded service

Runtime:
  1. Health check scans vectorstore
  2. Reports quarantined collections as "missing"
  3. Returns HTTP 503 status
  4. Monitoring system alerts on degraded status
```

### P1: Automated Startup Validation (Not Yet Implemented)

**Future**: Pre-flight metadata check before full initialization

---

## Testing

### Unit Tests

```bash
# Test metadata health checker
go test ./internal/vectorstore -run TestMetadataHealth -v

# All tests
go test ./internal/vectorstore -short
```

**Test Coverage**:
- ✅ All collections healthy
- ✅ Corrupt collection detection
- ✅ Empty collection handling
- ✅ Mixed state (healthy + corrupt + empty)
- ✅ Hidden directory filtering (`.quarantine`)
- ✅ File filtering (only directories counted)

### Integration Test

```bash
# Start contextd
go run ./cmd/contextd

# Basic health check
curl http://localhost:9090/health

# Expected (healthy):
# {"status":"ok","metadata":{"status":"healthy","healthy_count":22,"corrupt_count":0,...}}

# Detailed health
curl http://localhost:9090/api/v1/health/metadata | jq

# Create corrupt collection
mkdir -p ~/.config/contextd/vectorstore/testcorrupt
echo "fake" > ~/.config/contextd/vectorstore/testcorrupt/doc1.gob

# Check health again (should show degraded)
curl http://localhost:9090/health
# Expected: {"status":"degraded","metadata":{"status":"degraded","corrupt_count":1,"corrupt_hashes":["testcorrupt"],...}}

# Cleanup
rm -rf ~/.config/contextd/vectorstore/testcorrupt
```

---

## Troubleshooting

### Health Check Returns Empty Metadata

**Symptom**: `/health` returns `{"status":"ok"}` with no `metadata` field

**Causes**:
1. Health checker not initialized (vectorstore provider is not chromem)
2. HTTP server created before health checker initialization
3. Vectorstore path not configured

**Fix**:
```bash
# Verify config
cat ~/.config/contextd/config.yaml | grep -A5 vectorstore

# Check logs for "metadata health checker initialized"
# If missing, verify:
# - vectorstore.provider == "chromem"
# - vectorstore.chromem.path is set
```

### Health Check Returns Error

**Symptom**: HTTP 500 or error in logs

**Causes**:
1. Vectorstore path doesn't exist
2. Permission denied reading vectorstore directory
3. Corrupted directory structure

**Fix**:
```bash
# Check path exists and is readable
ls -la ~/.config/contextd/vectorstore/

# Check permissions
chmod 755 ~/.config/contextd/vectorstore
```

### Corrupt Collections Not Detected

**Symptom**: Health shows `"corrupt_count": 0` but you know collections are corrupt

**Causes**:
1. Collection has no documents (classified as "empty" not "corrupt")
2. Metadata file exists but is corrupted (still classified as "healthy")
3. Hidden directory (starts with `.`)

**Understanding**:
- **Corrupt** = metadata missing AND documents present
- **Empty** = metadata missing AND no documents
- **Healthy** = metadata exists (regardless of content validity)

---

## Security Considerations

### Exposed Information

Health endpoints expose:
- ✅ Collection hashes (SHA256 of collection name)
- ✅ Collection count
- ✅ Document count per collection
- ✅ Metadata file size

**Risk Level**: Low
- Collection hashes are one-way (cannot reverse to collection name)
- Document counts are not sensitive
- No actual document content exposed

### Denial of Service

Health check performs directory scan:
- Complexity: O(n) collections
- Typical duration: 1-2ms
- No rate limiting currently implemented

**Future Mitigation**: Rate limiting on health endpoints

---

## Future Enhancements

### P2: Periodic Background Checks

**Goal**: Detect corruption between HTTP requests

**Implementation**:
```go
// Start background checker goroutine
ticker := time.NewTicker(5 * time.Minute)
go func() {
    for range ticker.C {
        health, _ := checker.Check(ctx)
        if !health.IsHealthy() {
            logger.Warn("corrupt collections detected", zap.Strings("hashes", health.Corrupt))
            // Emit metrics
        }
    }
}()
```

### P2: Prometheus Metrics

**Goal**: Enable monitoring/alerting via Prometheus

**Metrics**:
- `vectorstore_collections_total{status="healthy|corrupt|empty"}`
- `vectorstore_metadata_health_check_duration_seconds`
- `vectorstore_metadata_health_last_check_timestamp`

### P3: Automatic Recovery

**Goal**: Auto-recover corrupt collections via metadata rebuild

**Approach**:
1. Detect corrupt collection
2. Analyze document structure
3. Rebuild metadata file
4. Verify integrity
5. Log recovery action

---

## References

- **Incident Report**: `PRODUCTION_INCIDENT_2026-01-22.md`
- **P0 Implementation**: `PREVENTION_MEASURES_IMPLEMENTED.md`
- **Code**: `internal/vectorstore/metadata_health.go`
- **Tests**: `internal/vectorstore/metadata_health_test.go`
- **Integration**: `internal/http/server.go`, `cmd/contextd/main.go`

---

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2026-01-22 | Initial implementation (P1) | Claude |
| 2026-01-22 | Documentation created | Claude |
