# Package: repository

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) and [../CLAUDE.md](../CLAUDE.md) for project overview and package guidelines.

## Purpose

Provides repository indexing functionality for semantic code search. Walks file trees, filters files by patterns and size, and creates searchable checkpoints for each indexed file.

## Specification

**Full Spec**: [`docs/specs/indexing/SPEC.md`](../../docs/specs/indexing/SPEC.md)

**Quick Summary**:
- **Problem**: Users need to search codebases semantically (natural language queries)
- **Solution**: Index repository files as checkpoints with vector embeddings
- **Key Features**:
  - Pattern-based file filtering (include/exclude glob patterns)
  - File size limits (1MB default, 10MB max)
  - Path traversal attack prevention
  - Async operation tracking via NATS

## Architecture

**Design Pattern**: Service pattern with dependency injection

**Dependencies**:
- `pkg/checkpoint` - Creates searchable checkpoints for indexed files

**Used By**:
- `pkg/mcp` - MCP server exposes index_repository tool
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
checkpointSvc := checkpoint.NewService(vectorStore, logger)
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

## MCP Integration

The `MCPAdapter` bridges repository service with MCP server types:

```go
// Create adapter for MCP integration
adapter := repository.NewMCPAdapter(svc)

// Use in MCP server
mcpServer := mcp.NewServer(
    echo, ops, nats,
    checkpointSvc, remediationSvc, skillsSvc, troubleshootSvc,
    adapter,  // Repository service adapter
    vectorStore, logger,
)
```

## Testing

**Test Coverage**: 81.1% (Target: ≥80%)

**Key Test Files**:
- `service_test.go` - Unit tests for indexing logic
- `mcp_adapter_test.go` - MCP integration tests

**Running Tests**:
```bash
go test ./pkg/repository/
go test -cover ./pkg/repository/
go test -race ./pkg/repository/
```

## Security Considerations

- **Path Traversal Prevention**: All paths cleaned with `filepath.Clean()`
- **File Size Limits**: Prevents indexing large files (max 10MB)
- **Pattern Validation**: Glob patterns validated before use
- **Multi-Tenant Isolation**: Checkpoints scoped to project_path
- **Input Validation**: All user inputs (path, patterns) validated

## Performance Notes

- **File Walking**: Sequential (no parallelization yet)
- **Checkpoint Creation**: One checkpoint per file
- **Throughput**: Depends on embedding service:
  - TEI: ~16 files/second
  - OpenAI API: ~2 files/second
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

## Related Documentation

- Spec: [`docs/specs/indexing/SPEC.md`](../../docs/specs/indexing/SPEC.md)
- Implementation: [`docs/specs/indexing/implementation.md`](../../docs/specs/indexing/implementation.md)
- Package Guidelines: [`pkg/CLAUDE.md`](../CLAUDE.md)
- Project Root: [`CLAUDE.md`](../../CLAUDE.md)

## Future Enhancements

See [`docs/specs/indexing/SPEC.md`](../../docs/specs/indexing/SPEC.md) Phase 2-4:
- Batch embedding generation (10x speedup)
- Parallel processing with worker pools (20x speedup)
- Incremental indexing (skip unchanged files)
- Progress reporting callbacks
- AST-based code indexing (extract functions/classes)
- Large file chunking
