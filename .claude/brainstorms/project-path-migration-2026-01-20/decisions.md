# Project Path Migration - Decision Log

**Date**: 2026-01-20

## Decision 1: Trigger Mechanism

**Question**: How should migration be triggered?

**Options Considered**:
1. Auto-detect on session_start hook
2. MCP tool call (`mcp__contextd__migrate_project`)
3. Explicit CLI command (`ctxd migrate-project`)

**Decision**: **Option 3 - Explicit CLI command**

**Rationale**:
- User expressed concern about hook performance affecting session startup
- Explicit trigger gives user control over when migration happens
- Avoids unexpected behavior during normal sessions
- CLI can provide rich interactive feedback (progress, confirmation)

## Decision 2: Cleanup Strategy

**Question**: What happens to old collections after migration?

**Options Considered**:
1. Delete immediately after successful migration
2. Keep as backup with configurable retention (30 days)
3. Rename with `.old` suffix, manual cleanup

**Decision**: **Option 2 - Keep as backup for 30 days**

**Rationale**:
- Safety net if migration has issues
- User can verify new collections work before cleanup
- Automatic cleanup reduces manual maintenance
- Manual cleanup available via `ctxd cleanup-backups`

## Decision 3: Migration Scope

**Question**: What data types should migrate?

**Decision**: All collection types migrate together:
- Memories (ReasoningBank)
- Codebase index (semantic search)
- Checkpoints
- Remediations

**Rationale**:
- Consistent behavior - user expects all data to move
- Reduces confusion about partial state
- Single command handles everything

## Decision 4: Detection Mechanism

**Question**: How to find old collections belonging to moved project?

**Decision**: Use existing tenant ID derivation logic:
1. Derive tenant ID from old path's git remote
2. Scan all collections for matching `project_id` or tenant pattern
3. Match collection name prefix (e.g., `contextd_*`)

**Rationale**:
- Leverages existing `tenant.GetTenantIDForPath()` function
- No new configuration needed
- Works even if old directory no longer exists

## Decision 5: User Confirmation

**Decision**: Require explicit confirmation (unless `--force` flag)

**Rationale**:
- Data migration is significant operation
- User should review what will be migrated
- `--dry-run` allows preview without commitment
- `--force` available for scripting/automation
