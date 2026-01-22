# Chromem Metadata File Recovery Guide

**Status**: Production Operations Runbook
**Severity**: P0 - Complete Service Outage
**Last Updated**: 2026-01-22

---

## Quick Recovery

### Symptoms
```bash
$ contextd
{"level":"warn","msg":"vectorstore initialization failed","error":"creating chromem DB: collection metadata file not found: /path/to/vectorstore/<hash>"}
{"msg":"contextd initialized","services":["checkpoint:unavailable","remediation:unavailable",...]}
```

All services show `unavailable` due to vectorstore initialization failure.

### Immediate Fix

**Step 1**: Identify the corrupt collection
```bash
# Find collection with missing metadata
for dir in ~/.config/contextd/vectorstore/*/; do
    if [ ! -f "${dir}00000000.gob" ] && [ -n "$(ls -A $dir 2>/dev/null)" ]; then
        echo "CORRUPT: $dir"
        basename "$dir"
    fi
done
```

**Step 2**: Determine collection name from hash
```bash
# Run the hash reverse lookup
python3 << 'EOF'
import hashlib
import sys

hash_target = sys.argv[1] if len(sys.argv) > 1 else "e9f85bf6"

# Common contextd collection names
names = [
    "contextd_memories", "contextd_checkpoints", "contextd_remediations",
    "contextd_repository", "contextd_default", "contextd_conversations",
    "memories", "checkpoints", "remediations", "repository",
    "reasoningbank", "signals", "feedback",
]

for name in names:
    h = hashlib.sha256(name.encode()).hexdigest()[:8]
    if h == hash_target:
        print(f"MATCH: {name}")
        sys.exit(0)
    print(f"{h}  {name}")

print(f"\nNo match found for {hash_target}")
sys.exit(1)
EOF
```

**Step 3**: Run the recovery tool
```bash
# Option A: Using the recovery tool (recommended)
go run ./cmd/recover-metadata/main.go

# Option B: Manual recovery script
# Edit collection name and hash in the script below
cat > /tmp/recover.go << 'EOF'
package main
import ("encoding/gob"; "os"; "path/filepath")
func main() {
    home, _ := os.UserHomeDir()
    name := "contextd_memories"  // ← Change this
    hash := "e9f85bf6"            // ← Change this
    path := filepath.Join(home, ".config/contextd/vectorstore", hash, "00000000.gob")
    f, _ := os.Create(path)
    defer f.Close()
    pc := struct{Name string; Metadata map[string]string}{Name: name, Metadata: map[string]string{}}
    gob.NewEncoder(f).Encode(pc)
    f.Sync()
}
EOF
go run /tmp/recover.go && rm /tmp/recover.go
```

**Step 4**: Verify and restart
```bash
# Verify metadata file exists
ls -lh ~/.config/contextd/vectorstore/<hash>/00000000.gob

# Restart contextd
systemctl restart contextd  # or: pkill contextd && contextd
```

---

## Root Cause

### Chromem Architecture Flaw

Chromem's `NewPersistentDB()` uses **fail-deadly** validation:
```go
// If ANY collection has documents but no metadata → FAIL ENTIRE DB LOAD
if c.Name == "" {
    return nil, fmt.Errorf("collection metadata file not found: %s", collectionPath)
}
```

**Impact**: One corrupt collection → All services unavailable

### How Metadata Files Can Be Lost

#### 1. Process Crash During Collection Creation (Most Common)
```
Timeline:
1. newCollection() called
2. os.MkdirAll() creates directory ✅
3. Process crashes before metadata file write ❌
4. Directory exists but empty
5. Next startup: NewPersistentDB() fails
```

**Prevention**: Use atomic operations (see Implementation section)

#### 2. Disk Full During Write
- Directory created ✅
- Metadata file creation fails due to no space
- Later writes might succeed if space freed

**Prevention**: Monitor disk space, pre-allocate space

#### 3. File System Corruption
- Metadata file deleted by FS corruption
- Documents remain intact

**Prevention**: Regular backups, FS health checks

#### 4. Manual Deletion (Rare)
- User accidentally deletes `00000000.gob`

**Prevention**: File permissions, education

---

## Prevention Strategies

### 1. Graceful Degradation (High Priority)

**File**: `internal/vectorstore/resilient.go` (to be created)

```go
package vectorstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	chromem "github.com/philippgille/chromem-go"
	"go.uber.org/zap"
)

// NewResilientChromemDB creates a chromem DB with graceful degradation for corrupt collections.
func NewResilientChromemDB(path string, compress bool, logger *zap.Logger) (*chromem.DB, error) {
	// Try normal load
	db, err := chromem.NewPersistentDB(path, compress)
	if err == nil {
		logger.Info("ChromemDB loaded successfully", zap.String("path", path))
		return db, nil
	}

	// Check if error is due to missing metadata
	if !strings.Contains(err.Error(), "collection metadata file not found") {
		return nil, err // Different error, fail normally
	}

	// Find and quarantine corrupt collections
	corruptCollections, findErr := findCorruptCollections(path, logger)
	if findErr != nil {
		logger.Error("Failed to find corrupt collections", zap.Error(findErr))
		return nil, err // Return original error
	}

	if len(corruptCollections) == 0 {
		return nil, err // No corrupt collections found, return original error
	}

	// Quarantine corrupt collections
	quarantinePath := filepath.Join(path, ".quarantine")
	if err := os.MkdirAll(quarantinePath, 0755); err != nil {
		logger.Error("Failed to create quarantine directory", zap.Error(err))
		return nil, err
	}

	for _, hash := range corruptCollections {
		src := filepath.Join(path, hash)
		dst := filepath.Join(quarantinePath, hash)

		logger.Warn("Quarantining corrupt collection",
			zap.String("collection_hash", hash),
			zap.String("from", src),
			zap.String("to", dst))

		if err := os.Rename(src, dst); err != nil {
			logger.Error("Failed to quarantine collection",
				zap.String("collection", hash),
				zap.Error(err))
			continue
		}
	}

	// Retry DB load
	db, err = chromem.NewPersistentDB(path, compress)
	if err != nil {
		logger.Error("Failed to load DB even after quarantine", zap.Error(err))
		return nil, err
	}

	logger.Info("ChromemDB loaded successfully after quarantine",
		zap.Int("quarantined_count", len(corruptCollections)))

	return db, nil
}

// findCorruptCollections identifies collections with documents but no metadata file.
func findCorruptCollections(path string, logger *zap.Logger) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var corrupt []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue // Skip files and hidden directories
		}

		collectionPath := filepath.Join(path, entry.Name())
		metadataPath := filepath.Join(collectionPath, "00000000.gob")

		// Check if metadata exists
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			// Check if collection has any .gob files (documents)
			files, _ := os.ReadDir(collectionPath)
			hasDocuments := false
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".gob") {
					hasDocuments = true
					break
				}
			}

			if hasDocuments {
				logger.Warn("Found corrupt collection (missing metadata but has documents)",
					zap.String("collection_hash", entry.Name()),
					zap.String("path", collectionPath))
				corrupt = append(corrupt, entry.Name())
			}
		}
	}

	return corrupt, nil
}
```

**Integration** (`internal/vectorstore/chromem.go`):
```go
// In NewChromemStore(), replace:
db, err := chromem.NewPersistentDB(expandedPath, config.Compress)

// With:
db, err := NewResilientChromemDB(expandedPath, config.Compress, logger)
```

### 2. Health Monitoring

**File**: `internal/vectorstore/health.go` (to be created)

```go
package vectorstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// HealthChecker performs periodic health checks on the vectorstore.
type HealthChecker struct {
	store  *ChromemStore
	logger *zap.Logger
	ticker *time.Ticker
	done   chan struct{}
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(store *ChromemStore, logger *zap.Logger, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		store:  store,
		logger: logger,
		ticker: time.NewTicker(interval),
		done:   make(chan struct{}),
	}
}

// Start begins periodic health checks.
func (h *HealthChecker) Start() {
	go h.run()
}

// Stop stops the health checker.
func (h *HealthChecker) Stop() {
	close(h.done)
	h.ticker.Stop()
}

func (h *HealthChecker) run() {
	for {
		select {
		case <-h.ticker.C:
			if err := h.check(context.Background()); err != nil {
				h.logger.Error("Health check failed", zap.Error(err))
			}
		case <-h.done:
			return
		}
	}
}

func (h *HealthChecker) check(ctx context.Context) error {
	// Verify all collections have metadata files
	collections := h.store.db.ListCollections()
	for name, col := range collections {
		if col == nil {
			continue
		}

		// Access persist directory via reflection or getter (chromem doesn't expose this)
		// For now, we can check via file system
		basePath := h.store.config.Path

		// Note: This requires knowing the hash function chromem uses
		// In production, we'd need chromem to expose collection paths or add a getter

		h.logger.Debug("Checking collection health",
			zap.String("collection", name))
	}

	return nil
}
```

### 3. Atomic Metadata Writes (Upstream Fix)

**Contribute to chromem-go**:

```go
// In collection.go persistMetadata():
func (c *Collection) persistMetadata() error {
	metadataPath := filepath.Join(c.persistDirectory, metadataFileName)
	metadataPath += ".gob"
	if c.compress {
		metadataPath += ".gz"
	}

	// Write to temporary file first
	tempPath := metadataPath + ".tmp"
	pc := struct {
		Name     string
		Metadata map[string]string
	}{Name: c.Name, Metadata: c.metadata}

	err := persistToFile(tempPath, pc, c.compress, "")
	if err != nil {
		return err
	}

	// Sync to ensure data is on disk
	f, _ := os.Open(tempPath)
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tempPath)
		return fmt.Errorf("syncing temp metadata: %w", err)
	}
	f.Close()

	// Atomic rename
	if err := os.Rename(tempPath, metadataPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("renaming temp metadata: %w", err)
	}

	// Create backup for recovery
	backupPath := metadataPath + ".backup"
	persistToFile(backupPath, pc, c.compress, "")

	return nil
}
```

### 4. Collection Integrity Verification

**CLI Tool**: `cmd/ctxd/verify.go`

```go
func verifyCollections(basePath string) error {
	entries, _ := os.ReadDir(basePath)
	var issues []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		collectionPath := filepath.Join(basePath, entry.Name())
		metadataPath := filepath.Join(collectionPath, "00000000.gob")

		// Check metadata exists
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("Missing metadata: %s", entry.Name()))
			continue
		}

		// Verify metadata is readable
		type pc struct {
			Name     string
			Metadata map[string]string
		}
		var metadata pc
		f, _ := os.Open(metadataPath)
		if err := gob.NewDecoder(f).Decode(&metadata); err != nil {
			issues = append(issues, fmt.Sprintf("Corrupt metadata: %s", entry.Name()))
		}
		f.Close()

		// Count documents
		files, _ := os.ReadDir(collectionPath)
		docCount := 0
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".gob") && f.Name() != "00000000.gob" {
				docCount++
			}
		}

		fmt.Printf("✅ %s (%s): %d documents\n", entry.Name(), metadata.Name, docCount)
	}

	if len(issues) > 0 {
		fmt.Println("\n⚠️  Issues found:")
		for _, issue := range issues {
			fmt.Println("  -", issue)
		}
		return fmt.Errorf("found %d collection issues", len(issues))
	}

	return nil
}
```

**Usage**:
```bash
ctxd verify --vectorstore ~/.config/contextd/vectorstore
```

---

## Monitoring and Alerting

### Metrics to Track

```yaml
vectorstore_collections_total: Total number of collections
vectorstore_collections_healthy: Collections with valid metadata
vectorstore_collections_corrupt: Collections missing metadata
vectorstore_documents_total: Total documents across all collections
```

### Alert Rules

```yaml
- alert: VectorstoreCorruptCollection
  expr: vectorstore_collections_corrupt > 0
  for: 1m
  severity: critical
  summary: "Corrupt vectorstore collection detected"
  description: "{{ $value }} collection(s) missing metadata files"

- alert: VectorstoreInitializationFailure
  expr: vectorstore_initialization_failures_total > 0
  for: 1m
  severity: critical
  summary: "Vectorstore failed to initialize"
```

---

## Testing

### Simulate Metadata Loss

```bash
# Create test environment
TEST_DIR=$(mktemp -d)
cp -r ~/.config/contextd/vectorstore "$TEST_DIR/"

# Remove metadata from a collection
rm "$TEST_DIR/vectorstore/e9f85bf6/00000000.gob"

# Try to load (should fail)
CONTEXTD_VECTORSTORE_PATH="$TEST_DIR/vectorstore" contextd
# Expected: initialization failure

# Run recovery
go run ./cmd/recover-metadata/main.go --path "$TEST_DIR/vectorstore/e9f85bf6"

# Verify recovery
CONTEXTD_VECTORSTORE_PATH="$TEST_DIR/vectorstore" contextd
# Expected: successful initialization
```

---

## References

- **Root Cause Analysis**: `METADATA_LOSS_ROOT_CAUSE.md`
- **Recovery Tool Source**: `cmd/recover-metadata/main.go`
- **Chromem Source**: `github.com/philippgille/chromem-go@v0.7.0`
- **Related Issue**: (Link to GitHub issue once created)

---

## Incident History

| Date | Collection | Cause | Recovery Time |
|------|-----------|-------|---------------|
| 2026-01-22 | contextd_memories (e9f85bf6) | Unknown (suspected crash) | 2 hours |

---

## Future Work

1. **PR to chromem-go**: Implement graceful degradation and atomic writes
2. **Automated Recovery**: Add auto-recovery to contextd startup
3. **Backup Strategy**: Periodic metadata backups
4. **Monitoring**: Add health check endpoints
5. **Documentation**: User guide for recovery procedures
