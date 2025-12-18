# Store Provider Design: Database-per-Project Isolation

**Date**: 2025-12-17
**Status**: Design Review

## Problem

Current architecture has inconsistent collection naming:
- Checkpoint service uses `tenant.Router` (requires org + team + project)
- ReasoningBank uses `project.GetCollectionName(projectID)` (just project)
- MCP tools expose confusing parameters that don't match

This causes:
1. `checkpoint_list` fails with "invalid team ID" when team_id not provided
2. Potential data leakage through routing bugs
3. Confusion about what tenant_id/team_id/project_id mean

## Solution

**Database-per-project isolation** via filesystem paths.

### Directory Structure

```
~/.config/contextd/vectorstore/
├── {tenant}/
│   ├── {project}/              ← chromem.DB instance
│   │   ├── checkpoints/        ← collection
│   │   ├── memories/
│   │   ├── remediations/
│   │   └── codebase/
│   │
│   ├── memories/               ← org-level shared (optional)
│   └── remediations/           ← org-level shared
│
│   # Team level (optional):
│   ├── {team}/
│   │   ├── {project}/          ← chromem.DB instance
│   │   │   └── ...
│   │   ├── memories/           ← team-level shared
│   │   └── remediations/
```

### Path Formula

| Scope | Path | Collections |
|-------|------|-------------|
| Project (direct) | `{tenant}/{project}/` | checkpoints, memories, remediations, codebase |
| Project (team-scoped) | `{tenant}/{team}/{project}/` | checkpoints, memories, remediations, codebase |
| Team shared | `{tenant}/{team}/` | memories, remediations |
| Org shared | `{tenant}/` | memories, remediations |

### Key Insight

Each path level is its own `chromem.DB` instance. No routing logic needed - **path IS isolation**.

## Implementation

### 1. StoreProvider Interface

```go
// StoreProvider manages chromem.DB instances per scope path.
type StoreProvider interface {
    // GetProjectStore returns a store for project-level collections.
    // Path: {basePath}/{tenant}/{project}/ (direct)
    // Path: {basePath}/{tenant}/{team}/{project}/ (team-scoped)
    GetProjectStore(ctx context.Context, tenant, team, project string) (Store, error)

    // GetTeamStore returns a store for team-level shared collections.
    // Path: {basePath}/{tenant}/{team}/
    GetTeamStore(ctx context.Context, tenant, team string) (Store, error)

    // GetOrgStore returns a store for org-level shared collections.
    // Path: {basePath}/{tenant}/
    GetOrgStore(ctx context.Context, tenant string) (Store, error)

    // Close closes all managed stores.
    Close() error
}
```

### 2. Chromem Implementation

```go
type chromemStoreProvider struct {
    basePath  string
    embedder  Embedder
    logger    *zap.Logger

    stores    sync.Map  // path -> *ChromemStore
    mu        sync.Mutex
}

func (p *chromemStoreProvider) GetProjectStore(ctx context.Context, tenant, team, project string) (Store, error) {
    var path string
    if team != "" {
        path = filepath.Join(p.basePath, tenant, team, project)
    } else {
        path = filepath.Join(p.basePath, tenant, project)
    }
    return p.getOrCreateStore(path)
}

func (p *chromemStoreProvider) getOrCreateStore(path string) (Store, error) {
    // Check cache
    if store, ok := p.stores.Load(path); ok {
        return store.(*ChromemStore), nil
    }

    p.mu.Lock()
    defer p.mu.Unlock()

    // Double-check after lock
    if store, ok := p.stores.Load(path); ok {
        return store.(*ChromemStore), nil
    }

    // Create new store at path
    config := ChromemConfig{
        Path: path,
        // ... other config
    }
    store, err := NewChromemStore(config, p.embedder, p.logger)
    if err != nil {
        return nil, err
    }

    p.stores.Store(path, store)
    return store, nil
}
```

### 3. Service Changes

Services receive `StoreProvider` instead of `Store`:

```go
// Before
type checkpointService struct {
    store  vectorstore.Store
    router tenant.CollectionRouter  // REMOVE
}

// After
type checkpointService struct {
    stores vectorstore.StoreProvider
}

func (s *checkpointService) List(ctx context.Context, req *ListRequest) ([]*Checkpoint, error) {
    // Get project-scoped store - no routing logic needed!
    store, err := s.stores.GetProjectStore(ctx, req.TenantID, req.TeamID, req.ProjectID)
    if err != nil {
        return nil, err
    }

    // Collection name is just "checkpoints" - no prefix needed
    return s.listFromStore(ctx, store, "checkpoints", req)
}
```

### 4. MCP Tool Changes

```go
// Before (confusing)
type checkpointListInput struct {
    SessionID   string `json:"session_id,omitempty"`
    TenantID    string `json:"tenant_id" jsonschema:"required"`
    ProjectPath string `json:"project_path,omitempty"`
    // team_id missing but required by router!
}

// After (clear)
type checkpointListInput struct {
    TenantID  string `json:"tenant_id" jsonschema:"required,Org identifier"`
    TeamID    string `json:"team_id,omitempty" jsonschema:"Team identifier (optional)"`
    ProjectID string `json:"project_id" jsonschema:"required,Project identifier"`
    SessionID string `json:"session_id,omitempty" jsonschema:"Filter by session"`
}
```

### 5. Remove tenant.Router

The `internal/tenant/router.go` complexity is no longer needed:
- No `GetCollectionName()` with scope logic
- No `ValidateAccess()`
- No `GetSearchCollections()` hierarchy

Path construction IS the routing.

## Migration

Existing data in flat collections needs migration:
1. Detect old-style collections (hashed names in root)
2. Create new directory structure
3. Move documents to appropriate project directories
4. Clean up old collections

## Benefits

1. **No leakage** - Physical filesystem isolation
2. **Simple mental model** - Path = scope
3. **Clean collection names** - Just "checkpoints", not "{tenant}_{team}_{project}_checkpoints"
4. **Easy backup/delete** - `rm -rf {tenant}/{project}/`
5. **Future-proof** - Team tier is just another directory level

## Questions to Resolve

1. How to derive `project_id` from `project_path`? (basename or sanitized?)
2. Default `tenant_id` for local-first? (username? "default"?)
3. Migration strategy for existing data?

## Next Steps

1. [x] Implement `StoreProvider` interface - `internal/vectorstore/provider.go`
2. [x] Implement Registry package - `internal/registry/registry.go`
3. [x] Update checkpoint service to use `StoreProvider` (with backward-compatible `NewServiceWithStore`)
4. [ ] Update other services (remediation, repository, reasoningbank) to use `StoreProvider`
5. [ ] Simplify MCP tool inputs
6. [ ] Remove `tenant.Router`
7. [ ] Add migration tooling
8. [ ] Update tests
