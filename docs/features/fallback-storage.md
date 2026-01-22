# Local Fallback Storage

**Status**: Implemented (v0.4.0)
**Epic**: #114
**Issues**: #115-120

---

## Overview

Local Fallback Storage provides graceful degradation when the remote vector store (Qdrant) is unavailable. Writes automatically fall back to local chromem storage with a write-ahead log (WAL), then sync to remote when connectivity is restored.

## Key Features

- **Automatic Failover**: Seamless switch to local storage when remote is down
- **Background Sync**: Automatic sync when connectivity restored
- **No Data Loss**: WAL ensures all writes reach remote eventually
- **Security**: HMAC-SHA256 checksums + gitleaks secret scrubbing
- **Circuit Breaker**: Prevents sync storms (5 failures → 5min backoff)

## Configuration

### Environment Variables

```bash
# Enable fallback storage (default: false)
CONTEXTD_FALLBACK_ENABLED=true

# Local storage path (default: .claude/contextd/store)
CONTEXTD_FALLBACK_LOCAL_PATH=".claude/contextd/store"

# WAL directory (default: .claude/contextd/wal)
CONTEXTD_FALLBACK_WAL_PATH=".claude/contextd/wal"

# Health check interval (default: 30s)
CONTEXTD_FALLBACK_HEALTH_INTERVAL="30s"

# Sync on reconnect (default: true)
CONTEXTD_FALLBACK_SYNC_ON_CONNECT=true

# WAL retention days (default: 7)
CONTEXTD_FALLBACK_WAL_RETENTION_DAYS=7
```

### Example Configuration

```yaml
vectorstore:
  provider: qdrant
  fallback:
    enabled: true
    local_path: .claude/contextd/store
    sync_on_connect: true
    health_check_interval: 30s
    wal_path: .claude/contextd/wal
    wal_retention_days: 7
```

## Architecture

### Components

| Component | Purpose | File |
|-----------|---------|------|
| FallbackStore | Decorator wrapping remote + local stores | `internal/vectorstore/fallback.go` |
| HealthMonitor | Connection state monitoring | `internal/vectorstore/health.go` |
| SyncManager | Background sync orchestration | `internal/vectorstore/sync.go` |
| WAL | Write-ahead log with checksums | `internal/vectorstore/wal.go` |

### Data Flow

**Write Path (Remote Healthy)**:
1. Write to remote (Qdrant) first
2. Write to local (chromem) for consistency
3. Record in WAL as SYNCED
4. Return success

**Write Path (Remote Unhealthy)**:
1. Record in WAL as PENDING with checksum
2. Write to local (chromem)
3. Return success
4. Background sync when remote recovers

**Read Path**:
- Remote healthy: Search remote (authoritative)
- Remote unhealthy: Search local with `stale_warning` metadata

## Security

### WAL Security Controls

| Control | Implementation |
|---------|----------------|
| Secret Scrubbing | All content scrubbed via gitleaks before WAL write |
| Integrity | HMAC-SHA256 checksum per entry using crypto/rand key |
| Operation Whitelist | Only "add"/"delete" accepted, validated at entry + deserialize |
| File Permissions | WAL files created with 0600 (owner-only access) |
| Atomic Writes | Temp file + rename pattern prevents partial reads |
| Entry Size Limits | Max 10MB per entry, max 10000 documents |

### Tenant Isolation

- Local and remote stores both use PayloadIsolation mode
- TenantInfo required in context for all operations (fail-closed)
- Cross-tenant queries impossible

## Usage Example

```go
// Configuration automatically creates FallbackStore when enabled
cfg := config.Load()
store, err := vectorstore.NewStore(cfg, embedder, logger)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Normal operations work transparently
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    ProjectID: "my-project",
})

// Writes automatically use fallback when Qdrant is down
ids, err := store.AddDocuments(ctx, docs)

// Searches return local results with metadata when remote down
results, err := store.Search(ctx, "query", 10)
```

## Monitoring

### Metrics (Future)

```
fallback_state{project="..."}                     gauge  # 0=remote, 1=local
fallback_wal_entries{state="pending|synced"}      gauge
fallback_sync_duration_seconds                    histogram
fallback_sync_errors_total                        counter
fallback_health_checks_total{result="healthy|unhealthy"} counter
```

### Logging

```
INFO  fallback: FallbackStore initialized local_path=... wal_path=...
INFO  fallback: using local store tenant_id=... doc_count=...
INFO  fallback: remote became healthy, triggering sync
INFO  fallback: sync complete synced=10 failed=0 duration=123ms
WARN  fallback: sync failed, will retry attempt=2 remaining=5
ERROR fallback: local write failed operation=add doc_count=5
```

## Troubleshooting

### Common Issues

| Symptom | Cause | Solution |
|---------|-------|----------|
| "fallback: local write failed" | Disk full or permissions | Check disk space, verify `.claude/contextd/store` writable |
| "fallback: sync failed, will retry" | Network connectivity | Verify Qdrant reachable, check firewall |
| "fallback: WAL corrupted, recovering" | Unexpected shutdown | Automatic recovery; check `*.corrupted.*` backups |
| Slow queries in fallback mode | Large local store | Check `fallback_wal_entries` metric |
| Sync never completes | Circuit breaker open | Check `fallback_sync_errors_total` |

### Manual WAL Operations

```bash
# View WAL status
ls -la .claude/contextd/wal/

# Compact WAL (remove synced entries)
# TODO: Add ctxd fallback compact command

# Reset WAL (CAUTION: loses pending data)
# TODO: Add ctxd fallback reset command
```

## Implementation Details

### Circuit Breaker

- Opens after 5 consecutive sync failures
- Half-open state after 5 minutes (allows one test request)
- Closes on successful sync
- Uses CAS operations for thread-safety

### Health Monitoring

- Primary: gRPC connection state watcher
- Fallback: Periodic ping (configurable interval)
- Exponential backoff on failures
- Copy-before-fire callback pattern (thread-safe)

### Sync Strategy

- FIFO ordering (preserves causal order)
- Local-wins conflict resolution (user intent)
- Bounded channel (100 pending syncs max)
- Graceful shutdown with WaitGroup

## See Also

- Design Document: `.claude/brainstorms/local-fallback-storage-2026-01-20/design.md`
- Architecture Spec: `docs/spec/vector-storage/architecture.md`
- Epic #114: Local Fallback Storage
- Issues #115-120: Individual implementation tasks
