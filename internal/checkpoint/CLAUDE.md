# internal/checkpoint

Session checkpoint storage and resumption.

**Last Updated**: 2025-12-18

---

## What This Package Is

**Purpose**: Save/restore Claude session state for context recovery

**Spec**: @../../docs/spec/interface/SPEC.md (CheckpointService)

**Storage**: Vectorstore with payload-based tenant isolation

---

## Tenant Isolation

Checkpoints use context-based tenant isolation. Tenant context is required for all operations:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Set tenant context before checkpoint operations
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    ProjectID: "contextd",
})

// Save checkpoint (tenant metadata injected automatically)
id, err := checkpointService.Save(ctx, &checkpoint.SaveRequest{...})

// List checkpoints (filtered by tenant)
checkpoints, err := checkpointService.List(ctx, &checkpoint.ListRequest{...})
```

**Security**: Missing tenant context returns `ErrMissingTenant` (fail-closed behavior).

---

## Core Operations

| Operation | gRPC Method | Purpose |
|-----------|-------------|---------|
| Save | CheckpointService.Save | Snapshot current session |
| List | CheckpointService.List | Browse available checkpoints |
| Resume | CheckpointService.Resume | Restore session context |

---

## Checkpoint Schema

```go
type Checkpoint struct {
    ID          string
    SessionID   string
    Summary     string        // Human-readable description
    Context     string        // Optimized session context
    Tags        []string
    ProjectPath string
    CreatedAt   time.Time
}
```

---

## Resume Levels

| Level | Returns | Tokens |
|-------|---------|--------|
| `summary` | Brief description only | ~20 |
| `context` | Optimized context summary | ~200 |
| `full` | Complete session state (ref) | Lazy load |

**Default**: `context` (balance detail vs tokens)

---

## Auto-Checkpoint

**Trigger**: Context usage thresholds (configurable)
**Default**: 70%, 85% context usage
**Silent**: No user notification

---

## Testing

**Coverage Target**: >80%

**Key Tests**:
- Save/resume roundtrip
- Resume levels (token counts)
- Auto-checkpoint triggers
- Cleanup (old checkpoints)

---

## References

- Spec: @../../docs/spec/interface/SPEC.md#checkpointservice
- Collections: @../../docs/spec/collection-architecture/SPEC.md
