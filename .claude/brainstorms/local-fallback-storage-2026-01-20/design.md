# Local Fallback Storage for ContextD

**Status**: Draft v4 - Addressing Consensus Review Round 3
**Created**: 2026-01-20
**Tier**: STANDARD (11/15 complexity)

---

## Executive Summary

When a remote vector store (Qdrant) is configured but unavailable, ContextD should gracefully fall back to local storage at `.claude/contextd/store`, then automatically sync to the remote when connectivity is restored.

## Goals

1. **Resilience** - ContextD never fails when remote is down
2. **Disconnected usage** - Support offline work (flights, poor connectivity)
3. **Cloud cost optimization** - Potential for batching writes
4. **Development convenience** - Local dev without Qdrant running

## Constraints

- **No data loss** - Local writes must eventually reach remote
- **Backwards compatible** - Existing configs work unchanged (opt-in)
- **Transparent to callers** - Same Store interface, MCP tools unaware
- **Minimal dependencies** - Use existing chromem/gob, no new engines

---

## Architecture

### System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     FallbackStore                            │
│  ┌──────────────────┐     ┌──────────────────┐             │
│  │   Remote Store   │     │   Local Store    │             │
│  │    (Qdrant)      │     │   (chromem)      │             │
│  │                  │     │  .claude/contextd│             │
│  └────────┬─────────┘     └────────┬─────────┘             │
│           │                        │                        │
│  ┌────────▼────────────────────────▼────────────┐          │
│  │              Health Monitor                   │          │
│  │  - gRPC state watcher (primary)              │          │
│  │  - Periodic ping (fallback, 30s)             │          │
│  └────────────────────┬─────────────────────────┘          │
│                       │                                     │
│  ┌────────────────────▼─────────────────────────┐          │
│  │              Sync Manager                     │          │
│  │  - Write-ahead journal (.claude/contextd/wal)│          │
│  │  - Background sync goroutine                 │          │
│  │  - Local-wins conflict resolution            │          │
│  └──────────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### Components

#### 1. FallbackStore (Decorator)

**File:** `internal/vectorstore/fallback.go`

Wraps remote and local stores, implementing the `Store` interface. Intercepts all operations and routes based on health state.

```go
type FallbackStore struct {
    remote      Store          // Primary (Qdrant)
    local       Store          // Fallback (chromem at .claude/contextd/store)
    health      *HealthMonitor
    sync        *SyncManager
    projectPath string
    mu          sync.RWMutex
}
```

**Responsibilities:**
- Route operations to appropriate store based on health
- Ensure writes always go to local first (durability)
- Trigger sync when remote becomes healthy
- Implement full Store interface transparently

#### 2. HealthMonitor

**File:** `internal/vectorstore/health.go`

Monitors remote store connectivity using multiple strategies.

```go
// HealthChecker interface for dependency injection and testability
type HealthChecker interface {
    IsHealthy(ctx context.Context) bool
    WatchState(ctx context.Context, callback func(healthy bool)) error
}

type HealthMonitor struct {
    checker       HealthChecker     // Interface for DI (gRPC, HTTP, mock)
    healthy       atomic.Bool
    lastCheck     atomic.Value      // time.Time
    checkInterval time.Duration     // Configurable via FallbackConfig
    mu            sync.RWMutex      // Protects callbacks slice
    callbacks     []func(healthy bool)
    ctx           context.Context   // For graceful shutdown
    cancel        context.CancelFunc
}

// GRPCHealthChecker implements HealthChecker for Qdrant
type GRPCHealthChecker struct {
    conn *grpc.ClientConn
}

// MockHealthChecker for testing
type MockHealthChecker struct {
    healthy atomic.Bool
}
```

**Health Detection Strategy:**
1. **Primary: gRPC state watcher** - React to `connectivity.Ready` vs `connectivity.TransientFailure`
2. **Fallback: Periodic ping** - Configurable interval (default 30s) via `health_check_interval`
3. **Exponential backoff** - On failures, increase check interval

**Thread-Safe Callback Management:**
```go
// RegisterCallback adds a callback with mutex protection
func (h *HealthMonitor) RegisterCallback(cb func(healthy bool)) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.callbacks = append(h.callbacks, cb)
}

// notifyCallbacks fires all callbacks under read lock (allows concurrent reads)
func (h *HealthMonitor) notifyCallbacks(healthy bool) {
    h.mu.RLock()
    callbacks := make([]func(bool), len(h.callbacks))
    copy(callbacks, h.callbacks) // Copy to avoid holding lock during callbacks
    h.mu.RUnlock()

    for _, cb := range callbacks {
        cb(healthy)
    }
}

// Stop gracefully shuts down the health monitor
func (h *HealthMonitor) Stop() {
    h.cancel()
}
```

#### 3. SyncManager

**File:** `internal/vectorstore/sync.go`

Manages the write-ahead log and background synchronization.

```go
type SyncManager struct {
    wal      *WAL
    local    Store
    remote   Store
    health   *HealthMonitor
    syncCh   chan struct{}     // Bounded channel for backpressure
    ctx      context.Context   // For graceful shutdown
    cancel   context.CancelFunc
    wg       sync.WaitGroup    // Wait for goroutines on shutdown
}

// NewSyncManager creates a SyncManager with bounded channels and shutdown support
func NewSyncManager(ctx context.Context, wal *WAL, local, remote Store, health *HealthMonitor) *SyncManager {
    ctx, cancel := context.WithCancel(ctx)
    return &SyncManager{
        wal:    wal,
        local:  local,
        remote: remote,
        health: health,
        syncCh: make(chan struct{}, 100), // Bounded: backpressure after 100 pending
        ctx:    ctx,
        cancel: cancel,
    }
}

// Start begins background sync goroutine
func (s *SyncManager) Start() {
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.runSyncLoop()
    }()
}

// Stop gracefully shuts down the sync manager
func (s *SyncManager) Stop() error {
    s.cancel()
    s.wg.Wait() // Wait for goroutine to finish
    return nil
}
```

**Responsibilities:**
- Maintain WAL of pending writes
- Background goroutine for sync
- Local-wins conflict resolution
- Idempotent sync operations

#### 4. WAL (Write-Ahead Log)

**File:** `internal/vectorstore/wal.go`

Durable journal of operations pending sync to remote with integrity validation.

```go
type WAL struct {
    path      string              // .claude/contextd/wal/
    mu        sync.Mutex
    entries   []WALEntry
    hmacKey   []byte              // Cryptographically random key (NOT derived from path)
    scrubber  *secrets.Scrubber   // gitleaks integration
    keyPath   string              // .claude/contextd/wal/.hmac_key (0600 permissions)
}

// initHMACKey generates or loads HMAC key with secure file creation
func (w *WAL) initHMACKey() error {
    w.keyPath = filepath.Join(w.path, ".hmac_key")

    // Try to load existing key
    if key, err := w.loadKeySecure(); err == nil {
        w.hmacKey = key
        return nil
    }

    // Generate new 32-byte (256-bit) random key
    key := make([]byte, 32)
    if _, err := crypto_rand.Read(key); err != nil {
        return fmt.Errorf("failed to generate HMAC key: %w", err)
    }

    // Write key with secure permissions from creation (atomic)
    if err := w.writeKeySecure(key); err != nil {
        return err
    }

    w.hmacKey = key
    return nil
}

// writeKeySecure uses atomic write with secure permissions from creation
func (w *WAL) writeKeySecure(key []byte) error {
    // Create temp file with restricted permissions immediately
    tmpPath := w.keyPath + ".tmp." + randomSuffix()
    f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
    if err != nil {
        return fmt.Errorf("failed to create key file: %w", err)
    }

    if _, err := f.Write(key); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("failed to write key: %w", err)
    }

    if err := f.Sync(); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("failed to sync key file: %w", err)
    }
    f.Close()

    // Atomic rename (prevents partial read)
    if err := os.Rename(tmpPath, w.keyPath); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("failed to finalize key file: %w", err)
    }

    return nil
}

type WALEntry struct {
    ID           string
    Operation    string     // "add", "delete" - validated against whitelist
    Docs         []Document // For add operations (scrubbed before write)
    IDs          []string   // For delete operations
    Timestamp    time.Time
    Synced       bool
    Checksum     []byte     // HMAC-SHA256 of entry content
    RemoteState  string     // "unknown", "exists", "deleted" - tracks last known remote state
    SyncAttempts int        // Number of sync attempts
    LastAttempt  time.Time  // When last sync attempted
    SyncError    string     // Last error message (for debugging)
}

// ValidOperations whitelist for deserialization safety
var ValidOperations = map[string]bool{"add": true, "delete": true}
```

**Security Controls:**
1. **Secret Scrubbing**: All document content scrubbed via gitleaks before WAL write; scrub result validated
2. **Integrity Validation**: HMAC-SHA256 checksum per entry using `subtle.ConstantTimeCompare`
3. **Operation Whitelist**: Validated at entry AND deserialization time
4. **Secure File Creation**: `os.OpenFile` with 0600 permissions on create (no TOCTOU window)
5. **Atomic Writes**: Write temp + rename pattern prevents partial/corrupted reads
6. **Entry Size Limits**: Max 10MB per entry, max 10000 documents per entry

**Storage Format:** Gob-encoded files in `.claude/contextd/wal/` with per-entry checksums

**Secure File Creation Pattern:**
```go
// writeEntrySecure ensures no TOCTOU vulnerability
func (w *WAL) writeEntrySecure(entry WALEntry) error {
    // Validate entry size limits
    if err := w.validateEntrySize(entry); err != nil {
        return err
    }

    entryPath := filepath.Join(w.path, entry.ID+".wal")
    tmpPath := entryPath + ".tmp." + randomSuffix()

    // Create with secure permissions from the start (no TOCTOU window)
    f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
    if err != nil {
        return fmt.Errorf("WAL: failed to create entry file: %w", err)
    }

    encoder := gob.NewEncoder(f)
    if err := encoder.Encode(entry); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("WAL: failed to encode entry: %w", err)
    }

    if err := f.Sync(); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("WAL: failed to sync entry: %w", err)
    }
    f.Close()

    // Atomic rename - no window where file exists with wrong permissions
    if err := os.Rename(tmpPath, entryPath); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("WAL: failed to finalize entry: %w", err)
    }

    return nil
}

// validateEntrySize prevents DoS via oversized entries
func (w *WAL) validateEntrySize(entry WALEntry) error {
    const maxEntrySize = 10 * 1024 * 1024   // 10MB
    const maxDocsPerEntry = 10000

    if len(entry.Docs) > maxDocsPerEntry {
        return fmt.Errorf("WAL: entry exceeds max documents (%d > %d)", len(entry.Docs), maxDocsPerEntry)
    }

    // Estimate size (actual gob encoding may vary)
    estimatedSize := 0
    for _, doc := range entry.Docs {
        estimatedSize += len(doc.Content) + len(doc.ID)
    }
    if estimatedSize > maxEntrySize {
        return fmt.Errorf("WAL: entry exceeds max size (%d > %d bytes)", estimatedSize, maxEntrySize)
    }

    return nil
}
```

---

## Data Flow

### Write Path (AddDocuments) - Atomic with Rollback

```
1. Scrub document content ─────── gitleaks removes secrets
2. Check remote health
3. IF HEALTHY:
   a. Write to REMOTE first
   b. Write to LOCAL (for query consistency)
   c. Record in WAL as SYNCED
   d. Return success
4. IF UNHEALTHY:
   a. Record in WAL as PENDING (with checksum)
   b. Write to LOCAL
   c. Return success
5. ON ANY FAILURE:
   a. Rollback: Delete from stores where written
   b. Remove incomplete WAL entry
   c. Return error
```

**Atomicity Guarantees:**
- WAL entry written BEFORE local write when offline
- Remote write BEFORE local write when online (prevents stale reads)
- Rollback on partial failure prevents inconsistent state

### Read Path (Search) - Merge Strategy

```
1. Check remote health
2. IF HEALTHY:
   a. Search REMOTE (authoritative)
   b. Search LOCAL for pending (unsynced) documents
   c. Merge results (local wins for conflicts)
   d. Add metadata: {source: "merged", pending_count: N}
3. IF UNHEALTHY:
   a. Search LOCAL only
   b. Add metadata: {source: "local", last_sync: timestamp, stale_warning: true}
4. Return results with metadata
```

**Read Consistency:**
- When online: remote + pending local = complete view
- When offline: local only with staleness indicator
- Callers can inspect `source` metadata for data freshness

### Sync Path (Background)

```
1. Health monitor signals: Remote → Healthy
2. Read pending entries from WAL
3. For each entry:
   a. Apply to remote (upsert - local wins)
   b. Mark synced in WAL
4. Compact WAL (remove synced entries)
```

---

## Configuration

### Config Schema

```yaml
vectorstore:
  provider: qdrant
  fallback:
    enabled: true                           # Opt-in (default: false)
    local_path: .claude/contextd/store      # Default path
    sync_on_connect: true                   # Immediate sync (default: true)
    health_check_interval: 30s              # Periodic check interval
    wal_path: .claude/contextd/wal          # WAL directory
```

### Go Config Types

```go
type FallbackConfig struct {
    Enabled             bool          `koanf:"enabled"`
    LocalPath           string        `koanf:"local_path"`
    SyncOnConnect       bool          `koanf:"sync_on_connect"`
    HealthCheckInterval time.Duration `koanf:"health_check_interval"`
    WALPath             string        `koanf:"wal_path"`
}
```

---

## Error Handling

### Failure Scenarios

| Scenario | Behavior |
|----------|----------|
| Remote unavailable at startup | Use local store, start health monitor |
| Remote fails mid-operation | Rollback partial writes, retry locally |
| Local write fails | Return error (fatal - no fallback for fallback) |
| Sync fails | Retry with exponential backoff + circuit breaker |
| WAL entry corrupted | Skip entry, log with full context, continue with valid entries |
| WAL file corrupted | Attempt partial recovery, backup corrupted file, rebuild from local store |

### WAL Recovery Strategy (No Data Loss)

```go
func (w *WAL) RecoverCorrupted(ctx context.Context) (*RecoveryReport, error) {
    report := &RecoveryReport{}

    // 1. Backup corrupted WAL
    backupPath := w.path + ".corrupted." + time.Now().Format("20060102-150405")
    copyFile(w.path, backupPath)
    report.BackupPath = backupPath

    // 2. Read entries one by one, validate checksums
    validEntries := []WALEntry{}
    for entry := range w.readEntries() {
        if w.validateChecksum(entry) {
            validEntries = append(validEntries, entry)
            report.RecoveredCount++
        } else {
            report.CorruptedCount++
            report.CorruptedIDs = append(report.CorruptedIDs, entry.ID)
            logger.Warn("fallback: skipping corrupted WAL entry",
                zap.String("id", entry.ID),
                zap.Time("timestamp", entry.Timestamp))
        }
    }

    // 3. Rebuild WAL from valid entries
    w.entries = validEntries
    w.persist()

    // 4. Scan local store for orphans with BOUNDS CHECKING
    // Only recover documents where we have evidence they weren't deleted
    const maxOrphansToScan = 10000 // Prevent infinite loops on corrupted local store
    orphans, truncated := w.findOrphansWithLimit(ctx, maxOrphansToScan)
    if truncated {
        logger.Warn("fallback: orphan scan truncated, manual review recommended",
            zap.Int("limit", maxOrphansToScan))
        report.ScanTruncated = true
    }

    // 5. Filter orphans: only recover documents NOT known to be deleted
    deletedIDs := w.getDeletedDocumentIDs() // From WAL delete entries
    for _, doc := range orphans {
        // Skip documents we know were deleted (from WAL history)
        if deletedIDs[doc.ID] {
            logger.Debug("fallback: skipping orphan - known deleted",
                zap.String("id", doc.ID))
            report.SkippedDeleted++
            continue
        }

        // Mark as requiring remote verification before sync
        w.addEntry(doc, "add", false)
        w.entries[len(w.entries)-1].RemoteState = "unknown" // Requires verification
        report.OrphansRecovered++
    }

    return report, nil
}

// findOrphansStreaming uses a channel-based iterator to prevent memory exhaustion
// Only orphan metadata is kept in memory; document content is streamed
func (w *WAL) findOrphansStreaming(ctx context.Context, limit int) (<-chan OrphanResult, error) {
    results := make(chan OrphanResult, 100) // Bounded channel for backpressure

    go func() {
        defer close(results)
        count := 0

        // Use streaming iterator - documents not loaded into memory all at once
        iter, err := w.local.NewDocumentIterator(ctx)
        if err != nil {
            results <- OrphanResult{Err: err}
            return
        }
        defer iter.Close()

        for {
            select {
            case <-ctx.Done():
                results <- OrphanResult{Err: ctx.Err(), Truncated: true}
                return
            default:
            }

            if count >= limit {
                results <- OrphanResult{Truncated: true}
                return
            }

            doc, ok := iter.Next()
            if !ok {
                break // End of iteration
            }

            if !w.hasWALEntry(doc.ID) {
                // Only send minimal metadata, not full document content
                results <- OrphanResult{
                    DocID:    doc.ID,
                    Metadata: doc.Metadata, // Small
                    // Content NOT included - will be re-read during sync
                }
            }
            count++
        }
    }()

    return results, nil
}

type OrphanResult struct {
    DocID     string
    Metadata  map[string]string
    Err       error
    Truncated bool
}

// DocumentIterator interface for streaming iteration (memory-safe)
type DocumentIterator interface {
    Next() (Document, bool)
    Close() error
}

// getDeletedDocumentIDs returns IDs of documents deleted via WAL
func (w *WAL) getDeletedDocumentIDs() map[string]bool {
    deleted := make(map[string]bool)
    for _, entry := range w.entries {
        if entry.Operation == "delete" {
            for _, id := range entry.IDs {
                deleted[id] = true
            }
        }
    }
    return deleted
}
```

### Circuit Breaker for Sync

```go
type CircuitBreaker struct {
    failures    atomic.Int32
    threshold   int32         // Default: 5
    resetAfter  time.Duration // Default: 5m
    state       atomic.Uint32 // 0=closed, 1=open, 2=half-open
    lastFailure atomic.Int64  // Unix nano timestamp
}

const (
    circuitClosed   uint32 = 0
    circuitOpen     uint32 = 1
    circuitHalfOpen uint32 = 2
)

func (cb *CircuitBreaker) Allow() bool {
    for {
        state := cb.state.Load()
        switch state {
        case circuitOpen:
            lastFail := time.Unix(0, cb.lastFailure.Load())
            if time.Since(lastFail) > cb.resetAfter {
                // CAS: only one goroutine transitions to half-open
                if cb.state.CompareAndSwap(circuitOpen, circuitHalfOpen) {
                    return true // This goroutine gets the test request
                }
                continue // Another goroutine won, retry
            }
            return false
        case circuitHalfOpen:
            return false // Only one request allowed in half-open
        default: // circuitClosed
            return true
        }
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.failures.Store(0)
    cb.state.Store(circuitClosed)
}

func (cb *CircuitBreaker) RecordFailure() {
    // Atomic increment + CAS loop to prevent TOCTOU race
    for {
        currentFailures := cb.failures.Load()
        newFailures := currentFailures + 1

        // Try to increment atomically
        if !cb.failures.CompareAndSwap(currentFailures, newFailures) {
            continue // Another goroutine incremented, retry
        }

        // Check threshold with the value we successfully stored
        if newFailures >= cb.threshold {
            // CAS to open state (only one goroutine wins)
            if cb.state.CompareAndSwap(circuitClosed, circuitOpen) ||
               cb.state.CompareAndSwap(circuitHalfOpen, circuitOpen) {
                cb.lastFailure.Store(time.Now().UnixNano())
            }
        }
        return
    }
}
```

### Logging with Context

```go
// State transitions
logger.Info("fallback: remote unavailable, using local store",
    zap.String("project", projectPath),
    zap.Int("pending_entries", len(wal.pending())))

logger.Info("fallback: remote restored, starting sync",
    zap.Int("entries_to_sync", count),
    zap.Duration("offline_duration", time.Since(lastHealthy)))

logger.Info("fallback: sync complete",
    zap.Int("entries", count),
    zap.Duration("duration", elapsed))

// Errors with actionable context
logger.Warn("fallback: sync failed, will retry",
    zap.Error(err),
    zap.Int("attempt", attempt),
    zap.Int("remaining_entries", remaining),
    zap.Duration("next_retry", backoff))

logger.Error("fallback: local write failed",
    zap.Error(err),
    zap.String("operation", op),
    zap.Int("doc_count", len(docs)))
```

---

## Testing Strategy

### Unit Tests

| Component | Test Cases |
|-----------|------------|
| FallbackStore | Route to local when unhealthy, route to remote when healthy |
| HealthMonitor | State transitions, callback firing |
| SyncManager | WAL read/write, sync idempotency |
| WAL | Persistence, compaction, corruption recovery |

### Integration Tests

| Scenario | Test |
|----------|------|
| Qdrant down at startup | Verify local operations work |
| Qdrant fails mid-session | Verify seamless fallback |
| Qdrant restored | Verify sync completes |
| Conflict resolution | Verify local wins |

### Mocks

```go
type MockHealthMonitor struct {
    healthy atomic.Bool
}

func (m *MockHealthMonitor) SetHealthy(h bool) {
    m.healthy.Store(h)
    // Fire callbacks
}
```

---

## Security Considerations

### Tenant Isolation (CRITICAL)

```go
// Local store MUST use PayloadIsolation (same as remote)
localStore, err := NewChromemStore(chromemCfg, embedder, logger)
localStore.SetIsolationMode(NewPayloadIsolation())

// All operations validate tenant context
func (f *FallbackStore) AddDocuments(ctx context.Context, docs []Document) ([]string, error) {
    // Fail-closed: missing tenant = error
    tenant, err := TenantFromContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("fallback: %w", ErrMissingTenant)
    }
    // Both local and remote use same tenant filtering
}
```

**Tenant Guarantees:**
- `TenantFromContext()` called at start of every operation
- Local chromem store uses identical `PayloadIsolation` mode
- Cross-tenant queries impossible - fail-closed design
- Integration tests verify isolation in fallback mode

### WAL Security (CRITICAL)

| Control | Implementation |
|---------|----------------|
| **Secret Scrubbing** | gitleaks scrubber applied to all document content BEFORE WAL write |
| **Integrity Validation** | HMAC-SHA256 checksum per entry using project-derived key |
| **Operation Whitelist** | Only "add"/"delete" accepted during deserialization |
| **File Permissions** | WAL files created with 0600 (owner read/write only) |
| **Injection Prevention** | Gob deserialization validates all fields, rejects unknown operations |

```go
func (w *WAL) WriteEntry(ctx context.Context, entry WALEntry) error {
    // 1. Validate operation at entry point (defense in depth)
    if !ValidOperations[entry.Operation] {
        return fmt.Errorf("invalid WAL operation: %s", entry.Operation)
    }

    // 2. Scrub sensitive content with validation
    for i := range entry.Docs {
        scrubbed, findings := w.scrubber.ScrubWithReport(entry.Docs[i].Content)
        if findings.HasError {
            // Fail-safe: scrubber malfunction means we don't write potentially sensitive data
            return fmt.Errorf("WAL: scrubbing failed for doc %s: %w", entry.Docs[i].ID, findings.Error)
        }
        entry.Docs[i].Content = scrubbed

        // Log if secrets were found and redacted
        if findings.SecretsFound > 0 {
            logger.Warn("WAL: secrets redacted from document",
                zap.String("doc_id", entry.Docs[i].ID),
                zap.Int("secrets_found", findings.SecretsFound))
        }

        entry.Docs[i].Metadata = w.scrubber.ScrubMetadata(entry.Docs[i].Metadata)
    }

    // 3. Compute checksum using HMAC-SHA256
    entry.Checksum = w.computeHMAC(entry)

    // 4. Write with secure atomic pattern (no TOCTOU)
    return w.writeEntrySecure(entry)
}

// validateChecksum uses constant-time comparison to prevent timing attacks
func (w *WAL) validateChecksum(entry WALEntry) bool {
    expected := w.computeHMAC(entry)
    // subtle.ConstantTimeCompare returns 1 if equal, 0 otherwise
    return subtle.ConstantTimeCompare(entry.Checksum, expected) == 1
}

// ScrubReport tracks scrubbing results for validation
type ScrubReport struct {
    SecretsFound int
    HasError     bool
    Error        error
}
```

### Additional Controls

3. **No new attack surface** - Fallback is transparent to callers, same API
4. **gRPC TLS validation** - Health checks validate remote certificate
5. **Path traversal prevention** - Local paths validated via `filepath.Clean()` and base directory check

---

## Migration / Rollout

### Phase 1: Feature Flag (Week 1-2)
- Implement with `fallback.enabled: false` default
- Internal testing only

### Phase 2: Opt-in Beta (Week 3-4)
- Document in CHANGELOG
- Announce to early adopters
- Collect feedback

### Phase 3: Default On (Week 5+)
- Change default to `true` for Qdrant provider
- Monitor for issues

---

## Files to Create/Modify

### New Files

| File | Purpose |
|------|---------|
| `internal/vectorstore/fallback.go` | FallbackStore decorator |
| `internal/vectorstore/fallback_test.go` | Unit tests |
| `internal/vectorstore/health.go` | HealthMonitor |
| `internal/vectorstore/health_test.go` | Unit tests |
| `internal/vectorstore/sync.go` | SyncManager |
| `internal/vectorstore/sync_test.go` | Unit tests |
| `internal/vectorstore/wal.go` | Write-ahead log |
| `internal/vectorstore/wal_test.go` | Unit tests |

### Modified Files

| File | Change |
|------|--------|
| `internal/config/types.go` | Add FallbackConfig |
| `internal/vectorstore/factory.go` | Wrap with FallbackStore when enabled |
| `docs/spec/vector-storage/architecture.md` | Document fallback |

---

## Resolved Design Decisions

### Conflict Resolution Strategy

**Strategy: Local Wins with Timestamp Tracking**

```go
type ConflictResolution struct {
    Strategy  string // "local_wins"
    Timestamp time.Time
}

func (s *SyncManager) resolveConflict(local, remote Document) Document {
    // Local always wins - user's recent changes take precedence
    // Remote is overwritten with local version (upsert)
    return local
}
```

**Conflict Detection:**
- Same document ID exists in both stores
- Detected during sync when upserting to remote
- No version vectors needed - local is always "newer" (user intent)

### WAL Size Limits

| Setting | Default | Description |
|---------|---------|-------------|
| `wal_max_entries` | 10000 | Max pending entries before warning |
| `wal_max_size_mb` | 100 | Max WAL size before rotation |
| `wal_retention_days` | 7 | Keep synced entries for N days (debugging) |

### Sync Priority

**FIFO (First In, First Out)** - Entries synced in order received.

Rationale: Preserves causal ordering, simpler implementation, predictable behavior.

### Partial Sync Recovery

**Continue from last successful entry** - Track sync progress in WAL.

```go
type WALEntry struct {
    // ... existing fields
    SyncAttempts  int       // Number of sync attempts
    LastAttempt   time.Time // When last attempted
    SyncError     string    // Last error (for debugging)
}
```

### Metrics (Required)

```go
// Prometheus metrics exposed via OTEL
fallback_state{project="..."} gauge  // 0=remote, 1=local
fallback_wal_entries{state="pending|synced"} gauge
fallback_sync_duration_seconds histogram
fallback_sync_errors_total counter
fallback_health_checks_total{result="healthy|unhealthy"} counter
```

## Troubleshooting Guide

### Common Symptoms and Solutions

| Symptom | Cause | Solution |
|---------|-------|----------|
| "fallback: local write failed" | Disk full or permissions | Check disk space, verify `.claude/contextd/store` is writable |
| "fallback: sync failed, will retry" | Network connectivity | Verify Qdrant is reachable, check firewall rules |
| "fallback: WAL corrupted, recovering" | Unexpected shutdown | Automatic recovery runs; check `*.corrupted.*` backup files |
| Slow queries in fallback mode | Large local store | Check `fallback_wal_entries` metric, consider manual cleanup |
| Sync never completes | Circuit breaker open | Check `fallback_sync_errors_total`, investigate root cause |

### Manual WAL Cleanup

```bash
# View WAL status
ls -la .claude/contextd/wal/

# Check synced vs pending entries (via ctxd CLI)
ctxd fallback status

# Force compact (remove synced entries)
ctxd fallback compact

# Reset WAL (CAUTION: loses pending data)
ctxd fallback reset --confirm
```

### Monitoring Alerts

| Metric | Alert Threshold | Action |
|--------|-----------------|--------|
| `fallback_state == 1` | > 5 minutes | Check Qdrant connectivity |
| `fallback_wal_entries{state="pending"}` | > 1000 | Investigate sync issues |
| `fallback_sync_errors_total` rate | > 10/min | Check circuit breaker, Qdrant logs |

### Recovery Procedures

**WAL Corruption:**
1. Automatic recovery attempts partial restore
2. Check `.claude/contextd/wal/*.corrupted.*` backups
3. Review logs for `fallback: skipping corrupted WAL entry`
4. If data loss suspected, restore from backup or re-index

**Circuit Breaker Open:**
1. Circuit opens after 5 consecutive sync failures
2. Resets automatically after 5 minutes
3. Check Qdrant connectivity and logs
4. Manual reset: `ctxd fallback circuit-reset`

---

## Remaining Open Questions

1. **WAL encryption at rest** - Add AES-256-GCM encryption? (Deferred to v2)
2. **Multi-project WAL** - Shared WAL or per-project? (Currently: per-project)

---

## Acceptance Criteria

- [ ] Writes succeed when Qdrant is unavailable
- [ ] Searches return local results when Qdrant is unavailable
- [ ] Sync occurs automatically when Qdrant becomes available
- [ ] No data loss across offline/online transitions
- [ ] Existing configs (without fallback) work unchanged
- [ ] Unit test coverage > 80%
- [ ] Integration tests pass
