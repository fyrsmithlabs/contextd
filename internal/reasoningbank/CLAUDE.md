# Package: reasoningbank

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) for project overview and package guidelines.

## Purpose

Cross-session memory system storing learnings with confidence scores. Enables AI agents to remember strategies, track outcomes, and improve over time.

## Architecture

**Design Pattern**: Service pattern with dependency injection

**Dependencies**:
- `internal/vectorstore` - Vector storage with tenant isolation
- `internal/embeddings` - Text embedding generation

**Used By**:
- `internal/mcp` - MCP server exposes memory tools

## Tenant Isolation

ReasoningBank uses context-based tenant isolation. Tenant context is required for all operations:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Set tenant context before memory operations
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    ProjectID: "contextd",
})

// Search memories (filtered by tenant)
results, err := reasoningBank.Search(ctx, "error handling strategies", 10)

// Record memory (tenant metadata injected automatically)
id, err := reasoningBank.Record(ctx, &Memory{
    Title:   "Use retry with exponential backoff",
    Content: "When dealing with rate limits...",
})
```

**Security**: Missing tenant context returns `ErrMissingTenant` (fail-closed behavior).

## Key Components

### Main Types

```go
// Memory represents a stored learning or strategy
type Memory struct {
    ID          string
    Title       string
    Content     string
    Outcome     Outcome         // success, failure, unknown
    Confidence  float64         // 0.0 - 1.0
    Tags        []string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Service provides memory operations
type Service struct {
    store      vectorstore.Store
    embedder   embeddings.Provider
    collection string
}
```

### Main Operations

| Operation | Purpose |
|-----------|---------|
| `Search(ctx, query, limit)` | Semantic search for relevant memories |
| `Record(ctx, memory)` | Store new memory with embedding |
| `Feedback(ctx, memoryID, helpful)` | Adjust confidence based on usefulness |
| `Get(ctx, memoryID)` | Retrieve specific memory |
| `Outcome(ctx, memoryID, outcome)` | Record success/failure of strategy |

## Confidence Scoring

Memories have confidence scores (0.0 - 1.0) that adjust based on:

| Event | Adjustment |
|-------|------------|
| Positive feedback | +0.1 |
| Negative feedback | -0.1 |
| Success outcome | +0.05 |
| Failure outcome | -0.05 |

**Decay**: Confidence decays over time for unused memories.

## Testing

```bash
go test ./internal/reasoningbank/... -v
go test ./internal/reasoningbank/... -cover
```

**Coverage Target**: >80%

## See Also

- Vectorstore: `internal/vectorstore/CLAUDE.md`
- Security spec: `docs/spec/vector-storage/security.md`
