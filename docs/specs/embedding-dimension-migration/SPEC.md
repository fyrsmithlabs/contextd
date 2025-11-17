# Embedding Dimension Migration & Multi-Provider Support

**Status**: Draft
**Created**: 2025-11-07
**Author**: Claude Code

## Problem Statement

### Current Issue

The contextd service has a **dimension mismatch** between the embedding provider and vector database collections:

- **TEI (BAAI/bge-small-en-v1.5)**: Produces **384-dimensional** embeddings
- **Result**: `vector dimension mismatch, expected vector size(byte) 1536, actual 6144`

### Root Causes

1. **Hardcoded Dimensions**: Vector store adapters hardcode dimension to 1536
   - `pkg/vectorstore/adapter/qdrant/adapter.go:86-91`

2. **Missing Configuration**: `EMBEDDING_DIM` environment variable not set
   - Config defaults to 1536 (`pkg/config/config.go:203`)
   - Service doesn't detect dimension from provider

3. **No Validation**: No dimension compatibility check before vector operations

### Impact

- ❌ **Checkpoint search fails** (dimension mismatch error)
- ❌ **Remediation search fails**
- ❌ **Skill search fails**
- ❌ **Cannot switch embedding providers** without data loss
- ✅ **Checkpoint data exists** and must be preserved

## Requirements

### Functional Requirements

1. **FR1**: Automatically detect embedding dimension from provider
2. **FR2**: Migrate existing collections to new dimension while preserving data
3. **FR3**: Support multiple embedding providers:
   - TEI (384-dim: BAAI/bge-small-en-v1.5)
   - OpenAI (1536-dim: text-embedding-3-small)
   - Custom providers with configurable dimensions
4. **FR4**: Validate dimension compatibility before vector operations
5. **FR5**: Provide migration CLI tool for user-initiated migration
6. **FR6**: Re-embed existing data with new provider during migration

### Non-Functional Requirements

1. **NFR1**: Zero downtime migration (optional offline mode)
2. **NFR2**: Preserve all checkpoint metadata during migration
3. **NFR3**: Support rollback if migration fails
4. **NFR4**: Migration progress reporting
5. **NFR5**: Validate data integrity after migration

### Security Requirements

1. **SR1**: No API keys exposed in migration logs
2. **SR2**: Backup created before destructive operations
3. **SR3**: Validate collection permissions before migration

## Solution Design

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Migration Workflow                                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Detect Provider Dimension                               │
│     ├─ Query embedding provider for test vector             │
│     └─ Validate dimension matches config (or auto-set)      │
│                                                              │
│  2. Analyze Existing Collections                            │
│     ├─ List all collections                                 │
│     ├─ Get schema and current dimension                     │
│     └─ Count vectors per collection                         │
│                                                              │
│  3. Export Data (if dimension mismatch)                     │
│     ├─ Export all vectors with metadata                     │
│     ├─ Save to JSON export format                           │
│     └─ Verify export completeness                           │
│                                                              │
│  4. Drop & Recreate Collections                             │
│     ├─ Create backup collections                            │
│     ├─ Drop old collections                                 │
│     └─ Create new collections with correct dimension        │
│                                                              │
│  5. Re-embed & Import Data                                  │
│     ├─ Re-generate embeddings with new provider             │
│     ├─ Batch insert vectors                                 │
│     └─ Verify vector count matches export                   │
│                                                              │
│  6. Validate Migration                                      │
│     ├─ Test search operations                               │
│     ├─ Compare sample results with pre-migration            │
│     └─ Delete backup collections                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Component Design

#### 1. Dimension Detection (`pkg/embedding/detect.go`)

```go
// DetectDimension queries the embedding provider to determine vector dimension
func (s *Service) DetectDimension(ctx context.Context) (int, error)

// ValidateDimension checks if provider dimension matches config
func (s *Service) ValidateDimension(ctx context.Context, expected int) error
```

#### 2. Migration Service (`pkg/migration/service.go`)

```go
type MigrationService struct {
    vectorStore vectorstore.VectorStore
    embedding   *embedding.Service
    backup      *backup.Service
}

// Migrate performs full collection migration
func (s *MigrationService) Migrate(ctx context.Context, opts MigrationOptions) error

// Export exports collection data to JSON
func (s *MigrationService) Export(ctx context.Context, collection string, path string) error

// Import imports collection data from JSON with re-embedding
func (s *MigrationService) Import(ctx context.Context, collection string, path string) error
```

#### 3. CLI Tool (`cmd/migrate/main.go`)

```bash
# Auto-detect and migrate all collections
ctxd migrate --auto

# Manual migration with specific dimension
ctxd migrate --dimension=384 --provider=tei

# Export only (no migration)
ctxd migrate --export --output=/tmp/checkpoint-export.json

# Import from export
ctxd migrate --import=/tmp/checkpoint-export.json --dimension=384

# Dry run (no changes)
ctxd migrate --dry-run
```

### Migration Algorithm

```
1. PRE-FLIGHT CHECKS
   ├─ Detect embedding provider dimension
   ├─ List all collections
   ├─ Check if dimension mismatch exists
   └─ Estimate migration time

2. BACKUP PHASE (if --backup)
   ├─ Create timestamped backup directory
   ├─ Export all collections to JSON
   └─ Verify backup completeness

3. MIGRATION PHASE
   For each collection:
   ├─ Create new collection with correct dimension
   ├─ Export old collection data (JSON)
   ├─ Re-embed all text content with new provider
   ├─ Batch insert into new collection
   ├─ Verify vector count matches
   └─ Drop old collection (rename: collection_old_TIMESTAMP)

4. VALIDATION PHASE
   ├─ Test search on each collection
   ├─ Compare results with pre-migration samples
   └─ Report success/failure

5. CLEANUP PHASE
   ├─ Delete old collections (if --cleanup)
   └─ Delete backup files (if --no-keep-backup)
```

### Configuration Updates

```bash
# Auto-detect dimension from provider (recommended)
EMBEDDING_DIM_AUTO=true

# Manual dimension override
EMBEDDING_DIM=384

# Migration options
MIGRATION_BATCH_SIZE=1000        # Batch size for re-embedding
MIGRATION_CONCURRENCY=4           # Parallel embedding requests
MIGRATION_BACKUP_DIR=~/.local/share/contextd/migration-backups
```

### Data Export Format

```json
{
  "version": "1.0",
  "collection": "checkpoints",
  "dimension": 1536,
  "exported_at": "2025-11-07T12:00:00Z",
  "vector_count": 1523,
  "vectors": [
    {
      "id": "checkpoint-uuid-1",
      "text": "Original text content for re-embedding",
      "metadata": {
        "summary": "Completed feature X",
        "project_path": "/home/user/project",
        "created_at": "2025-11-06T10:00:00Z"
      },
      "embedding": [0.123, 0.456, ...] // Original embedding (optional)
    }
  ]
}
```

## Implementation Plan

### Phase 1: Dimension Detection

1. Add `DetectDimension()` to embedding service
2. Add dimension validation at startup
3. Add warning if dimension mismatch detected
4. Update config to support `EMBEDDING_DIM_AUTO=true`

### Phase 2: Export/Import Tooling

1. Create migration package (`pkg/migration/`)
2. Implement export to JSON
3. Implement import from JSON with re-embedding
4. Add batch processing for large collections

### Phase 3: CLI Migration Tool

1. Create `cmd/migrate/main.go`
2. Implement dry-run mode
3. Implement backup/restore
4. Add progress reporting

### Phase 4: Testing

1. Integration tests with TEI (384-dim)
2. Integration tests with OpenAI (1536-dim)
3. Migration tests with sample data
4. Rollback tests

### Phase 5: Documentation

1. Migration guide in docs/
2. Update GETTING-STARTED.md
3. Add troubleshooting section
4. Update CHANGELOG.md

## Migration Risks & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Data loss during migration | High | Medium | Automatic backup before migration, rollback support |
| API rate limits (OpenAI re-embedding) | Medium | High | Batch processing, retry logic, TEI recommended |
| Dimension detection fails | Medium | Low | Manual override via EMBEDDING_DIM env var |
| Migration takes too long | Low | Medium | Progress reporting, resumable migration |
| Vector search quality degrades | Medium | Low | Validate results before cleanup, A/B testing |

## Testing Strategy

### Unit Tests

- Dimension detection from provider
- Export to JSON format
- Import from JSON with re-embedding
- Batch processing logic

### Integration Tests

- Full migration TEI → TEI (dimension fix)
- Full migration OpenAI → TEI
- Full migration TEI → OpenAI
- Rollback after failure

### Manual Testing

- Migrate production-like data (1000+ checkpoints)
- Verify search quality before/after
- Test with rate-limited API (OpenAI)
- Test resumable migration

## Success Criteria

- ✅ All checkpoint search operations succeed
- ✅ Zero data loss during migration
- ✅ Dimension automatically detected from provider
- ✅ Migration completes in < 5 minutes for 1000 vectors
- ✅ Users can switch providers without data loss
- ✅ Clear documentation for manual migration

## Future Enhancements

1. **Incremental Migration**: Migrate collections one at a time
2. **Multi-Tenant Migration**: Migrate per-project databases separately
3. **Dimension Auto-Scaling**: Support variable-length embeddings
4. **Provider Fallback**: Auto-fallback if primary provider fails
5. **Cost Estimation**: Estimate re-embedding cost before migration

## References

- Qdrant Collections: `pkg/qdrant/collections.go`
- Embedding Service: `pkg/embedding/embedding.go`
- Config: `pkg/config/config.go`
- Regression Test: `pkg/embedding/embedding_regression_test.go:178-235`
