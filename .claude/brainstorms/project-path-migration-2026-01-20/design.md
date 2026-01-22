# Project Path Migration - Design Document

**Date**: 2026-01-20
**Complexity Tier**: STANDARD (10/15)
**Status**: Draft

## Problem Statement

When a user moves a project directory (e.g., `~/projects/contextd` → `~/projects/fyrsmithlabs/contextd`), their contextd data (memories, codebase index, checkpoints, remediations) becomes orphaned because collection names are derived from the project path.

### Current Behavior

Collection naming convention: `{tenant}_{project}_{type}`
- `contextd_memories` → old path
- `fyrsmithlabs_contextd_memories` → new path

Result: User loses access to all historical data when moving a project.

### Desired Behavior

User runs `ctxd migrate-project` and all data seamlessly migrates to the new collection names while keeping backups for safety.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Trigger | CLI command (`ctxd migrate-project`) | Explicit > implicit; avoids session_start hook performance concerns |
| Cleanup | Keep backup for 30 days | Safe default; user can manually purge earlier |
| Scope | All collection types | Memories, codebase, checkpoints, remediations all migrate together |
| Detection | Git remote + path analysis | Use existing `tenant.GetTenantIDForPath()` logic |

## Architecture

### Component Overview

```
ctxd migrate-project [--from <old-path>] [--dry-run] [--force]
        │
        ├─► CollectionDetector
        │     • Scan vectorstore for collections matching old project
        │     • Use tenant ID derivation to identify candidates
        │
        ├─► MigrationPlanner
        │     • Generate migration plan (old name → new name)
        │     • Calculate estimated data size
        │     • Check for conflicts (target exists)
        │
        ├─► MigrationExecutor
        │     • Copy collection directory
        │     • Update collection metadata
        │     • Update document metadata (project_id, paths)
        │     • Mark source as "migrated" with timestamp
        │
        └─► BackupManager
              • Track migrated collections
              • Auto-cleanup after retention period
              • Manual cleanup via `ctxd cleanup-backups`
```

### Data Flow

```
1. User runs: ctxd migrate-project --from ~/old/path
2. Detector scans: ~/.config/contextd/vectorstore/*/00000000.gob
3. Matches collections where:
   - project_id matches old path
   - OR tenant_id matches old git remote
4. Planner generates:
   - Migration map: {old_collection: new_collection}
   - Conflict report: existing targets
5. User confirms (unless --force)
6. Executor for each collection:
   - Copy directory with new hash name
   - Update 00000000.gob (collection name)
   - Update all documents (project_id, file paths)
   - Add .migrated marker to source
7. Report: "Migrated 4 collections, 127 documents"
```

### Collection Types to Migrate

| Type | Collection Pattern | Metadata Updates |
|------|-------------------|------------------|
| Memories | `{tenant}_{project}_memories` | `project_id` |
| Codebase | `{tenant}_{project}_codebase` | `project_id`, file paths |
| Checkpoints | Checkpoint table/collection | `project_path` |
| Remediations | Remediation table/collection | `project_path`, `affected_files` |

## CLI Interface

### Commands

```bash
# Interactive migration (prompts for confirmation)
ctxd migrate-project

# Specify old path explicitly
ctxd migrate-project --from ~/projects/old-contextd

# Dry run - show what would be migrated
ctxd migrate-project --dry-run

# Skip confirmation
ctxd migrate-project --force

# List backup collections
ctxd list-backups

# Manual cleanup of old backups
ctxd cleanup-backups [--older-than 7d] [--collection <name>]
```

### Output Example

```
$ ctxd migrate-project --from ~/projects/contextd

Detecting collections for migration...

Found 4 collections to migrate:
  contextd_memories (7 documents, 12KB)
    → fyrsmithlabs_contextd_memories
  contextd_codebase (541 documents, 2.1MB)
    → fyrsmithlabs_contextd_codebase
  contextd_checkpoints (3 entries)
    → fyrsmithlabs_contextd_checkpoints
  contextd_remediations (12 entries)
    → fyrsmithlabs_contextd_remediations

Proceed with migration? [y/N]: y

Migrating contextd_memories... done (7 documents)
Migrating contextd_codebase... done (541 documents)
Migrating contextd_checkpoints... done (3 entries)
Migrating contextd_remediations... done (12 entries)

Migration complete!
  - 4 collections migrated
  - 563 total documents
  - Old collections retained as backup (30 day retention)

To remove backups now: ctxd cleanup-backups --collection contextd_*
```

## Backup Management

### Marker File Structure

When a collection is migrated, add `.migrated` marker:

```
~/.config/contextd/vectorstore/e9f85bf6/
├── 00000000.gob          # Collection metadata
├── *.gob                 # Documents
└── .migrated             # Migration marker
```

`.migrated` contents:
```json
{
  "migrated_at": "2026-01-20T14:30:00Z",
  "migrated_to": "fyrsmithlabs_contextd_memories",
  "new_directory": "000170b2",
  "retention_days": 30,
  "auto_cleanup_after": "2026-02-19T14:30:00Z"
}
```

### Auto-Cleanup

Two options (implement simpler first):

1. **CLI-triggered**: `ctxd cleanup-backups` checks markers and removes expired
2. **Background**: Session start hook checks and cleans (low priority, async)

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Target collection exists | Warn, skip (unless --force to merge) |
| Partial migration failure | Rollback: remove new, keep old |
| No old collections found | Helpful message: "No collections found for old path" |
| Old path doesn't exist | Still scan by derived tenant ID |

## Testing Strategy

1. **Unit Tests**:
   - CollectionDetector logic
   - MigrationPlanner conflict detection
   - Metadata update functions

2. **Integration Tests**:
   - End-to-end migration flow
   - Backup marker creation
   - Cleanup functionality

3. **Manual Testing**:
   - Actual project move scenario
   - Verify memories accessible after migration

## Implementation Plan

### Phase 1: Core Migration (STANDARD)
1. Add migration package under `internal/migration/`
2. Implement CollectionDetector
3. Implement MigrationExecutor (based on existing `cmd/migrate-collection/`)
4. Add `ctxd migrate-project` command

### Phase 2: Backup Management
1. Add backup marker logic
2. Implement `ctxd list-backups`
3. Implement `ctxd cleanup-backups`

### Phase 3: Polish
1. Add progress indicators
2. Improve error messages
3. Add --dry-run validation

## Open Questions

- [ ] Should we support partial migration (e.g., memories only)?
- [ ] Should migration auto-trigger on first contextd access after move?
- [ ] How to handle very large codebase collections (streaming vs batch)?

## Related Files

- `cmd/migrate-collection/main.go` - Existing manual migration tool
- `internal/tenant/defaults.go` - Tenant ID derivation logic
- `internal/vectorstore/chromem.go` - Collection handling
