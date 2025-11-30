# internal/checkpoint

Session checkpoint storage and resumption.

**Last Updated**: 2025-11-25

---

## What This Package Is

**Purpose**: Save/restore Claude session state for context recovery

**Spec**: @../../docs/spec/interface/SPEC.md (CheckpointService)

**Storage**: Qdrant (project_checkpoints collection)

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
