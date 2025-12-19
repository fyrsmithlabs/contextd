# Package: repository

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides repository indexing functionality for semantic code search. Walks file trees, filters files by patterns and size, and creates searchable checkpoints for each indexed file.

## Architecture

**Design Pattern**: Service pattern with dependency injection

**Dependencies**:
- `internal/checkpoint` - Creates searchable checkpoints for indexed files

**Used By**:
- `internal/mcp` - MCP server exposes index_repository tool (future)
- `cmd/contextd` - API server initialization

## Key Components

### Main Types

```go
// Service provides repository indexing functionality
type Service struct {
    checkpointService CheckpointService
}

// IndexOptions configures indexing behavior
type IndexOptions struct {
    IncludePatterns []string  // e.g., ["*.go", "*.md"]
    ExcludePatterns []string  // e.g., ["vendor/**", "*_test.go"]
    MaxFileSize     int64     // Default: 1MB, Max: 10MB
}

// IndexResult contains indexing statistics
type IndexResult struct {
    Path            string
    FilesIndexed    int
    IncludePatterns []string
    ExcludePatterns []string
    MaxFileSize     int64
    IndexedAt       time.Time
}
```

### Main Functions

```go
// IndexRepository indexes all files matching the given options
func (s *Service) IndexRepository(ctx context.Context, path string, opts IndexOptions) (*IndexResult, error)
```

## Usage Example

```go
// Create service (uses checkpoint service for storage)
checkpointSvc, _ := checkpoint.NewService(cfg, qdrantClient, logger)
svc := repository.NewService(checkpointSvc)

// Index repository
opts := repository.IndexOptions{
    IncludePatterns: []string{"*.go", "*.md"},
    ExcludePatterns: []string{"vendor/**", "*_test.go"},
    MaxFileSize:     1024 * 1024, // 1MB
}

result, err := svc.IndexRepository(ctx, "/path/to/repo", opts)
if err != nil {
    return err
}

fmt.Printf("Indexed %d files\n", result.FilesIndexed)
```

## Testing

**Running Tests**:
```bash
go test ./internal/repository/
go test -cover ./internal/repository/
go test -race ./internal/repository/
```

## Security Considerations

- **Path Traversal Prevention**: All paths cleaned with `filepath.Clean()`
- **File Size Limits**: Prevents indexing large files (max 10MB)
- **Pattern Validation**: Glob patterns validated before use
- **Multi-Tenant Isolation**: Context-based tenant filtering via `ContextWithTenant()`
- **Input Validation**: All user inputs (path, patterns) validated

## Tenant Isolation

Repository indexing uses context-based tenant isolation:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Set tenant context before indexing
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    ProjectID: "contextd",
})

// Index repository (files tagged with tenant metadata)
result, err := svc.IndexRepository(ctx, "/path/to/repo", opts)

// Search indexed code (filtered by tenant)
results, err := svc.Search(ctx, "authentication logic", 10)
```

**Security**: Missing tenant context returns `ErrMissingTenant` (fail-closed behavior).

## Performance Notes

- **File Walking**: Sequential (no parallelization yet)
- **Checkpoint Creation**: One checkpoint per file
- **Future Optimization**: Batch embedding generation (10x speedup planned)

## Pattern Matching

### Include Patterns

If empty: include all files (subject to exclude patterns)
If specified: include ONLY matching files

```go
IncludePatterns: []string{"*.go", "*.md"}  // Only Go and Markdown files
```

### Exclude Patterns

Takes precedence over include patterns.

```go
ExcludePatterns: []string{
    "vendor/**",        // Exclude vendor directory
    "*_test.go",        // Exclude test files
    "*.log",            // Exclude log files
    ".git/**",          // Exclude git directory
}
```

### Pattern Syntax

Uses Go's `filepath.Match` (glob-style):
- `*.go` - All Go files in current directory
- `*_test.go` - Go test files
- `vendor/**` - Vendor directory (custom recursive matching)

## Future Enhancements

- Batch embedding generation (10x speedup)
- Parallel processing with worker pools (20x speedup)
- Incremental indexing (skip unchanged files)
- Progress reporting callbacks
- AST-based code indexing (extract functions/classes)
- Large file chunking
