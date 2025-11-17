# Package: checkpoint

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides session checkpoint management with semantic search capabilities. Allows saving work-in-progress state and resuming later through vector-based similarity search.

## Specification

**Full Spec**: [`docs/specs/checkpoint/SPEC.md`](../../docs/specs/checkpoint/SPEC.md)

**Quick Summary**:
- **Problem**: Users need to pause work and resume later, finding relevant past sessions by meaning not just keywords
- **Solution**: Store checkpoints with automatic vector embeddings, enable semantic search across all past work
- **Key Features**:
  - Automatic embedding generation on save
  - Semantic search with cosine similarity
  - Project-specific checkpoint isolation (multi-tenant)
  - Local-first with background sync

## Architecture

**Design Pattern**: Service pattern with dependency injection

**Dependencies**:
- `pkg/embedding` - Generate vector embeddings for semantic search
- `pkg/vectorstore` - Abstract interface for vector database operations

**Used By**:
- `pkg/mcp` - MCP server exposes checkpoint tools
- `cmd/contextd` - API server endpoints

## Key Components

### Main Types

```go
// Checkpoint represents a saved session state
type Checkpoint struct {
    ID          string                 `json:"id"`
    ProjectPath string                 `json:"project_path"`
    Summary     string                 `json:"summary"`      // Required, max 500 chars
    Description string                 `json:"description"`  // Optional details
    Context     map[string]interface{} `json:"context"`      // Structured metadata
    Tags        []string               `json:"tags"`         // Categorization
    CreatedAt   time.Time             `json:"created_at"`
    UpdatedAt   time.Time             `json:"updated_at"`
}

// Service provides checkpoint operations
type Service struct {
    store     vectorstore.VectorStore
    embedding *embedding.Service
}
```

### Main Functions

```go
// Save creates a checkpoint with automatic embedding
func (s *Service) Save(ctx context.Context, cp *Checkpoint) error

// Search finds semantically similar checkpoints
func (s *Service) Search(ctx context.Context, query string, opts *SearchOptions) ([]*Checkpoint, error)

// List retrieves recent checkpoints with pagination
func (s *Service) List(ctx context.Context, opts *ListOptions) ([]*Checkpoint, error)

// Get retrieves a checkpoint by ID
func (s *Service) Get(ctx context.Context, id string) (*Checkpoint, error)
```

## Usage Example

```go
// Create service
svc := checkpoint.NewService(vectorStore, embeddingSvc)

// Save checkpoint
cp := &checkpoint.Checkpoint{
    ProjectPath: "/home/user/project",
    Summary:     "Implemented authentication middleware",
    Description: "Added Bearer token auth with constant-time comparison",
    Context: map[string]interface{}{
        "files_changed": []string{"pkg/auth/auth.go"},
        "tests_added":   true,
    },
    Tags: []string{"authentication", "security"},
}
if err := svc.Save(ctx, cp); err != nil {
    return err
}

// Search semantically
results, err := svc.Search(ctx, "authentication work", &checkpoint.SearchOptions{
    ProjectPath: "/home/user/project",
    Limit:       10,
})
if err != nil {
    return err
}

for _, cp := range results {
    fmt.Printf("Found: %s (score: %.2f)\n", cp.Summary, cp.Score)
}
```

## Testing

**Test Coverage**: 85% (Target: ≥80%)

**Key Test Files**:
- `checkpoint_test.go` - Unit tests for service methods
- `models_test.go` - Data structure validation tests

**Running Tests**:
```bash
go test ./pkg/checkpoint/
go test -cover ./pkg/checkpoint/
go test -race ./pkg/checkpoint/
```

## Configuration

**Environment Variables**:
- Inherits embedding config from `pkg/embedding`

## Security Considerations

- **Multi-tenant isolation**: Checkpoints filtered by `project_path` hash
- **No cross-project access**: Database-per-project architecture prevents filter injection
- **Context redaction**: Use `pkg/security` to redact sensitive data before saving

## Performance Notes

- **Background sync**: Async replication to cluster (when configured)
- **Search performance**: <100ms for semantic search across 10K checkpoints
- **Batch operations**: Use `SaveBatch` for multiple checkpoints

## Related Documentation

- Spec: [`docs/specs/checkpoint/SPEC.md`](../../docs/specs/checkpoint/SPEC.md)
- Research: [`docs/specs/checkpoint/research/`](../../docs/specs/checkpoint/research/)
- Multi-tenant Architecture: [`docs/adr/002-universal-multi-tenant-architecture.md`](../../docs/adr/002-universal-multi-tenant-architecture.md)
- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Project Root: [`CLAUDE.md`](../../CLAUDE.md)
