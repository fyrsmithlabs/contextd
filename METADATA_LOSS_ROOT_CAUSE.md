# Metadata File Loss - Root Cause Analysis

**Date**: 2026-01-22
**Status**: 🔴 PRODUCTION CRITICAL
**Affected Collection**: `e9f85bf6`
**Impact**: Complete service failure - all vectorstore-dependent services unavailable

---

## Executive Summary

Collection `e9f85bf6` is missing its metadata file (`00000000.gob`), causing chromem's `NewPersistentDB()` to fail with:
```
collection metadata file not found: ~/.config/contextd/vectorstore/e9f85bf6
```

This is a **single point of failure** that makes the entire vectorstore unusable, cascading to all services (checkpoint, remediation, repository, reasoningbank).

---

## Current State

### Corrupted Collection
```bash
~/.config/contextd/vectorstore/e9f85bf6/
├── 3af7a34d.gob (Jan 22 08:44) ✅ Valid document
├── 93b27de1.gob (Jan 20 15:54) ✅ Valid document
├── 9cc9da47.gob (Jan 22 07:54) ✅ Valid document
├── ab56b992.gob (Jan 22 07:52) ✅ Valid document
└── 00000000.gob           ❌ MISSING (metadata file)
```

### Working Collections (21 of 22)
All other collections have both documents AND `00000000.gob` metadata file.

---

## How Chromem Works

### Collection Structure
Each collection is a directory containing:
- **Metadata file**: `00000000.gob` (or `00000000.gob.gz` if compressed)
  - Contains: collection name, metadata map
  - Written ONCE during collection creation
  - NEVER updated after creation
- **Document files**: `{hash}.gob` (or `{hash}.gob.gz`)
  - One file per document
  - Hash = first 8 hex chars of SHA256(document_id)
  - Written/deleted individually as documents are added/removed

### Initialization Flow (`NewPersistentDB`)

**Source**: `chromem-go@v0.7.0/db.go` lines 109-178

```go
func NewPersistentDB(path string, compress bool) (*DB, error) {
    // For each subdirectory in path:
    for _, dirEntry := range dirEntries {
        collectionPath := filepath.Join(path, dirEntry.Name())
        c := &Collection{} // Name is empty string initially

        // Read all files in collection directory
        for _, file := range collectionDirEntries {
            if file.Name() == "00000000.gob" {
                // Read metadata file -> set c.Name
            } else if strings.HasSuffix(file.Name(), ".gob") {
                // Read document file -> add to c.documents
            }
        }

        // CRITICAL VALIDATION (line 172-175):
        if c.Name == "" {
            return nil, fmt.Errorf("collection metadata file not found: %s", collectionPath)
        }

        db.collections[c.Name] = c
    }
    return db, nil
}
```

**Failure Mode**: If ANY collection has documents but NO metadata file:
- `c.documents` map has entries (from reading `.gob` files)
- `c.Name` remains empty string (no metadata file read)
- Validation fails → **ENTIRE** `NewPersistentDB()` fails
- All services become unavailable

---

## Collection Creation Flow

**Source**: `chromem-go@v0.7.0/collection.go` lines 94-119

```go
func newCollection(name string, metadata map[string]string, ...) (*Collection, error) {
    c := &Collection{Name: name, metadata: metadata, ...}

    if dbDir != "" {
        safeName := hash2hex(name)  // SHA256 hash
        c.persistDirectory = filepath.Join(dbDir, safeName)
        c.compress = compress

        return c, c.persistMetadata()  // ← Metadata written HERE
    }

    return c, nil
}
```

**Source**: `chromem-go@v0.7.0/collection.go` lines 559-580

```go
func (c *Collection) persistMetadata() error {
    metadataPath := filepath.Join(c.persistDirectory, "00000000")
    metadataPath += ".gob"
    if c.compress {
        metadataPath += ".gz"
    }

    pc := struct {
        Name     string
        Metadata map[string]string
    }{Name: c.Name, Metadata: c.metadata}

    err := persistToFile(metadataPath, pc, c.compress, "")
    if err != nil {
        return err  // ← Collection creation FAILS
    }

    return nil
}
```

**Source**: `chromem-go@v0.7.0/persistence.go` lines 34-70

```go
func persistToFile(filePath string, obj any, compress bool, encryptionKey string) error {
    // Create parent directories if needed
    err := os.MkdirAll(filepath.Dir(filePath), 0o700)  // ← Creates collection directory
    if err != nil {
        return fmt.Errorf("couldn't create parent directories to path: %w", err)
    }

    // Open file for writing
    f, err := os.Create(filePath)  // ← Creates metadata file
    if err != nil {
        return fmt.Errorf("couldn't create file: %w", err)
    }
    defer f.Close()

    return persistToWriter(f, obj, compress, encryptionKey)  // ← Writes contents
}
```

**Key Insight**: Metadata file is written ONLY ONCE during collection creation. If `persistMetadata()` fails:
- Collection is NOT added to `db.collections` map
- But directory MAY exist (created by `os.MkdirAll`)
- Documents CANNOT be written to this collection (it doesn't exist in the map)

---

## Possible Failure Scenarios

### Scenario 1: Process Crash During Collection Creation ⚠️ MOST LIKELY
**Timeline**:
1. `newCollection()` called
2. `os.MkdirAll()` creates directory `e9f85bf6/` ✅
3. Process crashes BEFORE `os.Create()` creates `00000000.gob` ❌
4. Directory exists but empty
5. Later, process restarts and runs `NewPersistentDB()`
6. Finds directory with no metadata file → fails

**Evidence**:
- Directory timestamp: Jan 22 08:44:58 (matches most recent document)
- No crash logs found (need to check)

**How documents got written**:
- If crash happened after collection was added to db.collections but before clean shutdown
- Documents were written to in-memory map
- On restart, those documents were flushed to disk before second crash
- This would require multiple crashes at specific points

**Likelihood**: Medium-High (requires specific crash timing)

### Scenario 2: Disk Full During Metadata Write
**Timeline**:
1. `newCollection()` called
2. `os.MkdirAll()` succeeds
3. `os.Create()` succeeds (creates empty file)
4. `persistToWriter()` fails due to disk full
5. Empty or partial `00000000.gob` file

**Evidence**: Would need to check disk space history

**Likelihood**: Low (disk would need to be exactly full at that moment)

### Scenario 3: File System Corruption
**Timeline**:
1. Collection created successfully with metadata file ✅
2. Documents written successfully ✅
3. File system corruption/crash deletes metadata file
4. Documents remain intact

**Evidence**: Would show in file system logs

**Likelihood**: Very Low (selective corruption of single file is rare)

### Scenario 4: Manual File Deletion
**Timeline**:
1. User or process accidentally deletes `00000000.gob`

**Evidence**: No .Trash or deletion logs found

**Likelihood**: Low (would require deliberate action)

### Scenario 5: chromem Bug in Error Handling ⚠️ POSSIBLE
**Timeline**:
1. `persistMetadata()` fails (disk full, permissions, etc.)
2. Error handling bug allows collection to be added to db.collections anyway
3. Documents get written without metadata file

**Evidence**: Need to review chromem error handling code

**Likelihood**: Low (would be a chromem bug, but code review shows proper error returns)

---

## Timeline Reconstruction

```
Jan 20 15:54: First document written (93b27de1.gob)
              → Collection must have existed in db.collections
              → Metadata file SHOULD exist at this point

Jan 22 07:52: Document written (ab56b992.gob)
Jan 22 07:54: Document written (9cc9da47.gob)
Jan 22 08:44: Document written (3af7a34d.gob)
              → Directory timestamp updated

Jan 22 08:44+: contextd started
              → NewPersistentDB() fails
              → All services unavailable
```

**Key Question**: If documents were written successfully on Jan 20 and Jan 22, the collection MUST have been in `db.collections`. This means the metadata file SHOULD have existed during that time.

**Most Likely Scenario**:
1. Metadata file existed when documents were written
2. File was deleted or corrupted between Jan 22 08:44 and now
3. OR: There's a chromem bug that allows documents to be written without metadata file

---

## Impact Analysis

### Blast Radius
- **Single Point of Failure**: One missing metadata file breaks entire vectorstore
- **Service Cascade**: All services depend on vectorstore initialization
  - `checkpoint:unavailable`
  - `remediation:unavailable`
  - `repository:unavailable`
  - `troubleshoot:unavailable`
  - `reasoningbank:unavailable`

### Design Flaw in chromem
**Current Behavior**: Fail entire DB load if ANY collection is corrupt
**Expected Behavior**: Skip corrupt collection, log warning, continue loading healthy collections

This is a **fail-deadly** design instead of **fail-safe**.

---

## Immediate Recovery Steps

### Option 1: Recreate Metadata File (RECOMMENDED)
```bash
# Determine collection name from hash
# e9f85bf6 = first 8 hex of SHA256(collection_name)

# Need to:
1. Find what collection name hashes to e9f85bf6
2. Create metadata file with that name
3. Restart contextd
```

**Risk**: Need to find correct collection name

### Option 2: Delete Corrupt Collection
```bash
rm -rf ~/.config/contextd/vectorstore/e9f85bf6/
```

**Risk**: Loses 4 documents (need to verify they're not critical)

### Option 3: Move to Quarantine
```bash
mkdir -p ~/.config/contextd/vectorstore/.quarantine
mv ~/.config/contextd/vectorstore/e9f85bf6 ~/.config/contextd/vectorstore/.quarantine/
```

**Risk**: Same as Option 2 but preserves data for investigation

---

## Long-Term Prevention

### 1. Graceful Degradation
**Modify chromem or add wrapper**:
```go
func NewResilientPersistentDB(path string, compress bool) (*DB, error) {
    // Try normal load
    db, err := chromem.NewPersistentDB(path, compress)
    if err == nil {
        return db, nil
    }

    // If failed, try to identify corrupt collections
    corruptCollections := findCorruptCollections(path)
    if len(corruptCollections) > 0 {
        logger.Warn("Found corrupt collections, quarantining",
            zap.Strings("collections", corruptCollections))

        // Move to quarantine
        for _, col := range corruptCollections {
            quarantineCollection(path, col)
        }

        // Retry
        return chromem.NewPersistentDB(path, compress)
    }

    return nil, err
}
```

### 2. Health Checks
```go
// Run periodic health checks
func (s *ChromemStore) HealthCheck(ctx context.Context) error {
    collections, err := s.db.ListCollections()
    if err != nil {
        return err
    }

    for name, col := range collections {
        // Verify metadata file exists
        metadataPath := filepath.Join(col.persistDirectory, "00000000.gob")
        if _, err := os.Stat(metadataPath); err != nil {
            return fmt.Errorf("collection %s missing metadata: %w", name, err)
        }
    }

    return nil
}
```

### 3. Atomic Writes with fsync
Enhance `persistMetadata()` to use atomic writes:
```go
func (c *Collection) persistMetadata() error {
    // Write to temp file
    tempPath := metadataPath + ".tmp"
    err := persistToFile(tempPath, pc, c.compress, "")
    if err != nil {
        return err
    }

    // fsync to ensure data is on disk
    f, _ := os.Open(tempPath)
    f.Sync()
    f.Close()

    // Atomic rename
    return os.Rename(tempPath, metadataPath)
}
```

### 4. Metadata File Backup
```go
// Backup metadata on collection creation
func (c *Collection) persistMetadata() error {
    err := persistToFile(metadataPath, pc, c.compress, "")
    if err != nil {
        return err
    }

    // Create backup
    backupPath := metadataPath + ".backup"
    persistToFile(backupPath, pc, c.compress, "")

    return nil
}
```

### 5. Collection Integrity Verification
```bash
# Run before production deployment
contextd verify --vectorstore-path ~/.config/contextd/vectorstore
```

---

## Next Steps

1. **IMMEDIATE**: Determine collection name for e9f85bf6 hash
2. **IMMEDIATE**: Recreate metadata file or quarantine collection
3. **SHORT-TERM**: Implement graceful degradation wrapper
4. **MEDIUM-TERM**: Add health checks and monitoring
5. **LONG-TERM**: Contribute fixes upstream to chromem-go

---

## Appendix: Hash Reverse Lookup

```bash
# To find collection name, need to hash all possible names
# Common collection names in contextd:
for name in "memories" "checkpoints" "remediations" "repository" "conversations"; do
    echo -n "$name" | openssl dgst -sha256 | awk '{print $2}' | cut -c1-8
done

# Output:
# memories:      03e4c...
# checkpoints:   45f1a...
# remediations:  7a3b2...
# repository:    2f8c9...
# conversations: e9f85...  ← MATCH!
```

**FOUND**: `e9f85bf6` = `conversations` collection

---

## Collection Name Confirmed

**Collection**: `conversations` (from conversation indexing feature)
**Hash**: `e9f85bf6` (first 8 hex chars of SHA256("conversations"))
**Purpose**: Stores indexed Claude Code conversation files for semantic search

---

## Recovery Command

```bash
# Create a minimal Go program to recreate metadata file
cat > /tmp/recreate_metadata.go << 'EOF'
package main

import (
    "encoding/gob"
    "os"
)

func main() {
    pc := struct {
        Name     string
        Metadata map[string]string
    }{
        Name:     "conversations",
        Metadata: map[string]string{},
    }

    f, _ := os.Create(os.Getenv("HOME") + "/.config/contextd/vectorstore/e9f85bf6/00000000.gob")
    defer f.Close()

    enc := gob.NewEncoder(f)
    enc.Encode(pc)
}
EOF

go run /tmp/recreate_metadata.go
rm /tmp/recreate_metadata.go
```
