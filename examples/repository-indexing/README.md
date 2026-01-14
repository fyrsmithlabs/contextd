# Repository Indexing Example

This example demonstrates how to use contextd's repository indexing and semantic code search capabilities. Index your codebase once, then search it using natural language queries instead of regex patterns.

## Overview

contextd's repository indexing enables:

1. **Semantic Code Search** - Find code using natural language ("error handling patterns") instead of regex
2. **Intelligent Filtering** - Include/exclude patterns for selective indexing
3. **Branch Awareness** - Track which branch code comes from
4. **Grep Fallback** - Fall back to regex search when needed
5. **Multi-Tenant Isolation** - Keep each project's index separate

## The Pattern

```
┌──────────────────────────────────────────────────────┐
│              Repository Indexing Flow                 │
└───────────────────┬──────────────────────────────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │  1. Index Repository  │  ← repository_index
        │  (Parse & embed code) │
        └───────────┬───────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │  2. Semantic Search   │  ← semantic_search
        │  (Natural language)   │
        └───────────┬───────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │  3. Grep Fallback     │  ← (automatic or manual)
        │  (Regex patterns)     │
        └───────────────────────┘
```

## Quick Start

### Prerequisites

- contextd running (either as MCP server or standalone)
- Go 1.25+ installed
- A code repository to index (defaults to contextd project)

### Run the Example

```bash
# From the examples/repository-indexing directory
go run main.go

# Or specify a different repository
REPO_PATH=/path/to/your/project go run main.go

# Or build first
go build -o repository-indexing
./repository-indexing
```

### Expected Output

```
Repository Indexing Example - Semantic Code Search
===================================================

Repository Path: /Users/you/projects/contextd

Step 1: Indexing repository...
This may take a moment for large repositories.

✓ Indexed 487 files from branch 'main'
  Collection: demo-user_contextd_codebase
  Include patterns: [*.go *.md]
  Exclude patterns: [*_test.go vendor/** .git/** ...]
  Max file size: 1048576 bytes

Step 2: Performing semantic searches...

Query: "vector database implementation with embeddings"
Found 5 results:

  1. internal/vectorstore/chromem.go (score: 0.856)
     Branch: main
     Preview: package vectorstore implements the vector store interface using chromem...

  2. internal/vectorstore/interface.go (score: 0.823)
     Branch: main
     Preview: Store defines the interface for vector database operations...

  ... and 3 more results

Query: "error wrapping and handling patterns"
Found 4 results:

  1. internal/repository/service.go (score: 0.801)
     Branch: main
     Preview: return fmt.Errorf("walking file tree: %w", err)...

  ... and 3 more results

Step 3: Demonstrating grep fallback for exact matches...

Grep pattern: func.*Index.*Repository
Found 12 matches:

  File: internal/repository/service.go:370
  Code: func (s *Service) IndexRepository(ctx context.Context, path string, ...

  ... and 11 more matches

✓ Repository indexed and searchable!
```

## How It Works

### 1. Index Repository

Before searching, index your codebase:

```go
// Create repository service
service := repository.NewService(store)

// Configure indexing options
opts := repository.IndexOptions{
    TenantID: "myusername",
    // Only index specific file types
    IncludePatterns: []string{"*.go", "*.py", "*.js", "*.md"},
    // Skip test files and dependencies
    ExcludePatterns: []string{
        "*_test.go",
        "vendor/**",
        "node_modules/**",
        ".git/**",
    },
    MaxFileSize: 1024 * 1024, // 1MB limit
}

// Index the repository
result, err := service.IndexRepository(ctx, "/path/to/repo", opts)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Indexed %d files from branch '%s'\n",
    result.FilesIndexed, result.Branch)
```

**What gets indexed:**
- ✅ All files matching include patterns (if specified)
- ✅ Files under max size limit
- ✅ UTF-8 text files (binary files skipped)
- ✅ Files not matching exclude patterns
- ❌ Directories in default skip list (node_modules, vendor, .git, etc.)

**Indexing includes:**
- File content (embedded as vectors)
- File path relative to repo root
- Git branch name (auto-detected)
- File size and extension
- Indexed timestamp

### 2. Semantic Search

Search indexed code using natural language:

```go
// Search for code related to "authentication"
searchOpts := repository.SearchOptions{
    ProjectPath: "/path/to/repo",
    TenantID:    "myusername",
    Branch:      "main",  // Optional: filter by branch
    Limit:       10,      // Max results
}

results, err := service.Search(ctx, "JWT token validation", searchOpts)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("File: %s (score: %.3f)\n", result.FilePath, result.Score)
    fmt.Printf("Content: %s\n\n", result.Content)
}
```

**How semantic search works:**
1. Query is embedded into a vector
2. Vector similarity search finds closest matches
3. Results are ranked by similarity score (0-1)
4. Metadata filters (branch, file type) applied
5. Top K results returned

**Advantages over grep:**
- Finds conceptually related code, not just exact matches
- Works with natural language queries
- No need to know exact function/variable names
- Understands synonyms and related concepts

### 3. Grep Fallback

When you need exact pattern matching, use grep:

```go
// Find all function definitions matching a pattern
grepOpts := repository.GrepOptions{
    ProjectPath:     "/path/to/repo",
    IncludePatterns: []string{"*.go"},
    ExcludePatterns: []string{"*_test.go", "vendor/**"},
    CaseSensitive:   false,
}

results, err := service.Grep(ctx, `func\s+(\w+)\(`, grepOpts)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("%s:%d: %s\n",
        result.FilePath, result.LineNumber, result.Content)
}
```

**When to use grep:**
- ✅ Need exact pattern matching
- ✅ Searching for specific syntax
- ✅ Code generation or refactoring
- ✅ Finding all occurrences of a symbol

**When to use semantic search:**
- ✅ Don't know exact names
- ✅ Looking for concepts or patterns
- ✅ Exploring unfamiliar codebases
- ✅ Finding similar implementations

## Real-World Usage

### Example 1: Exploring a New Codebase

```go
// Index the new project
result, _ := service.IndexRepository(ctx, "/path/to/new-project", opts)

// Search for key concepts
queries := []string{
    "database connection and pooling",
    "authentication middleware",
    "error handling patterns",
    "configuration loading",
}

for _, query := range queries {
    results, _ := service.Search(ctx, query, searchOpts)
    // Review results to understand architecture...
}
```

### Example 2: Finding Similar Code

```go
// You implemented a feature and want to find similar patterns
query := "rate limiting with token bucket algorithm"

results, _ := service.Search(ctx, query, searchOpts)
// Found 3 other rate limiter implementations!
// Can learn from existing patterns instead of starting from scratch
```

### Example 3: Refactoring Assistant

```go
// Step 1: Find all error handling code
semanticResults, _ := service.Search(ctx,
    "error handling and wrapping", searchOpts)

// Step 2: Use grep to find exact patterns to refactor
grepResults, _ := service.Grep(ctx,
    `fmt\.Errorf\([^%]*%v`, grepOpts)
// Found all error wrapping using %v instead of %w

// Step 3: Refactor the matches to use %w
```

### Example 4: Branch-Specific Search

```go
// Index multiple branches for comparison
mainResult, _ := service.IndexRepository(ctx, "/repo", repository.IndexOptions{
    Branch: "main",
    // ... other opts
})

devResult, _ := service.IndexRepository(ctx, "/repo", repository.IndexOptions{
    Branch: "develop",
    // ... other opts
})

// Search only in main branch
mainResults, _ := service.Search(ctx, "feature implementation", repository.SearchOptions{
    Branch: "main",
    // ... other opts
})

// Search only in develop branch
devResults, _ := service.Search(ctx, "feature implementation", repository.SearchOptions{
    Branch: "develop",
    // ... other opts
})
```

## Integration with MCP

When running as an MCP server, these operations are exposed as tools:

| Go Method | MCP Tool | Purpose |
|-----------|----------|---------|
| `service.IndexRepository()` | `repository_index` | Index a codebase |
| `service.Search()` | `repository_search` | Semantic code search |
| N/A | `semantic_search` | High-level search with automatic grep fallback |

### Using MCP Tools

```json
{
  "tool": "repository_index",
  "arguments": {
    "path": "/path/to/repo",
    "tenant_id": "myusername",
    "include_patterns": ["*.go", "*.md"],
    "exclude_patterns": ["*_test.go", "vendor/**"],
    "max_file_size": 1048576
  }
}
```

```json
{
  "tool": "repository_search",
  "arguments": {
    "project_path": "/path/to/repo",
    "query": "authentication middleware",
    "tenant_id": "myusername",
    "limit": 10
  }
}
```

```json
{
  "tool": "semantic_search",
  "arguments": {
    "query": "error handling patterns",
    "project_path": "/path/to/repo"
  }
}
```

**Note:** The `semantic_search` tool automatically tries repository search first, then falls back to grep if no semantic results are found. This is the recommended tool for AI agents.

## Performance Considerations

### Indexing Performance

| Repository Size | Files | Indexing Time | Storage Size |
|----------------|-------|---------------|--------------|
| Small (<100 files) | 50 | ~5 seconds | ~2 MB |
| Medium (100-1000 files) | 500 | ~30 seconds | ~20 MB |
| Large (1000-10000 files) | 5000 | ~5 minutes | ~200 MB |
| Very Large (>10000 files) | 50000 | ~1 hour | ~2 GB |

**Optimization tips:**
1. Use include/exclude patterns to limit indexing scope
2. Set appropriate max_file_size (default 1MB)
3. Skip non-code files (images, binaries, etc.)
4. Consider indexing only specific directories
5. Use Qdrant instead of chromem for large repos (see `examples/qdrant-config/`)

### Search Performance

- **Semantic search**: 10-100ms per query (depending on index size)
- **Grep fallback**: 100-1000ms per query (depends on repo size)
- **Network latency**: Add 20-100ms if using remote vector store

**Tips:**
- Limit results (default 10 is usually sufficient)
- Use branch filters to reduce search space
- Cache frequently used queries
- Consider using Qdrant for production deployments

## Troubleshooting

### Error: "path does not exist" during indexing

**Cause:** Invalid or non-existent repository path

**Solution:**
```go
// Use absolute paths
absPath, err := filepath.Abs("/path/to/repo")
if err != nil {
    log.Fatal(err)
}

// Verify path exists and is a directory
info, err := os.Stat(absPath)
if err != nil || !info.IsDir() {
    log.Fatal("path must be a directory")
}

result, err := service.IndexRepository(ctx, absPath, opts)
```

### Error: "max_file_size cannot exceed 10MB"

**Cause:** Tried to set MaxFileSize > 10MB

**Solution:**
```go
opts := repository.IndexOptions{
    MaxFileSize: 10 * 1024 * 1024, // 10MB max (limit)
    // For larger files, use Qdrant with large-repos.yaml config
}
```

For files >10MB, see `examples/qdrant-config/large-repos.yaml`.

### No results from semantic search

**Cause:** Repository not indexed, or query doesn't match indexed content

**Solution:**
```bash
# 1. Verify repository was indexed
result, err := service.IndexRepository(ctx, repoPath, opts)
fmt.Printf("Indexed %d files\n", result.FilesIndexed)

# 2. Check if files match include/exclude patterns
# List what files would be indexed:
# - Review include_patterns and exclude_patterns
# - Ensure file types are not excluded

# 3. Try different queries
queries := []string{
    "authentication",      // Too broad
    "JWT token validation", // More specific
    "error handling",      // Generic
    "error wrapping with fmt.Errorf", // Very specific
}

# 4. Fall back to grep for exact matches
grepResults, _ := service.Grep(ctx, "pattern", grepOpts)
```

### Slow indexing performance

**Cause:** Large repository or slow embeddings provider

**Solutions:**

1. **Use selective patterns:**
   ```go
   opts := repository.IndexOptions{
       // Index only specific file types
       IncludePatterns: []string{"*.go"},
       // Exclude unnecessary directories
       ExcludePatterns: []string{"vendor/**", "node_modules/**"},
   }
   ```

2. **Use TEI for faster embeddings:**
   ```go
   embedder, err := embeddings.NewProvider(embeddings.ProviderConfig{
       Provider: "tei",
       Endpoint: "http://localhost:8080", // Text-Embeddings-Inference server
   })
   ```

3. **Use Qdrant for large repos:**
   ```bash
   # See examples/qdrant-config/large-repos.yaml
   contextd --config examples/qdrant-config/large-repos.yaml
   ```

### Error: "tenant_id is required for tenant context"

**Cause:** Missing tenant ID in multi-tenant mode

**Solution:**
```go
opts := repository.IndexOptions{
    TenantID: "myusername", // Required for isolation
    // ... other options
}

searchOpts := repository.SearchOptions{
    TenantID: "myusername", // Must match indexing tenant
    // ... other options
}
```

**Note:** Tenant IDs are used for multi-tenant isolation. Each tenant's indexed repositories are kept separate.

### Grep fails with "invalid pattern"

**Cause:** Invalid regex pattern

**Solution:**
```go
// Test your regex before using
pattern := `func\s+(\w+)\(`
_, err := regexp.Compile(pattern)
if err != nil {
    log.Printf("Invalid pattern: %v", err)
    return
}

// Use raw strings for regex patterns
grepResults, err := service.Grep(ctx, `func\s+\w+`, grepOpts)
```

## Vector Store Options

contextd supports two vector storage backends:

### chromem (Default)

- **Type:** Embedded in-process database
- **Best for:** Single-user, local development, small-to-medium repositories
- **Setup:** No external dependencies
- **Performance:** Fast for <1000 files

```go
store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
    Path:              "/tmp/contextd-vectors",
    DefaultCollection: "codebase",
    VectorSize:        384,
}, embedder, logger)
```

### Qdrant (Production)

- **Type:** External gRPC server
- **Best for:** Multi-user, production, large repositories (>1000 files)
- **Setup:** Requires Qdrant server
- **Performance:** Scales to millions of files

See `examples/qdrant-config/` for detailed setup instructions.

## Best Practices

### Do's ✅

1. **Index incrementally** - Re-index only changed branches
2. **Use specific patterns** - Narrow down file types and directories
3. **Set appropriate file size limits** - Default 1MB works for most code
4. **Try semantic search first** - Fall back to grep when needed
5. **Filter by branch** - Search only relevant branches
6. **Use descriptive queries** - "JWT validation" not just "auth"

### Don'ts ❌

1. **Don't index everything** - Exclude tests, vendor, generated code
2. **Don't re-index unchanged repos** - Index once, search many times
3. **Don't use semantic search for exact matches** - Use grep instead
4. **Don't forget tenant ID** - Required for multi-tenant isolation
5. **Don't index binary files** - They're skipped anyway
6. **Don't exceed 10MB file size** - Use Qdrant for larger files

## Next Steps

- **Session Lifecycle**: See `examples/session-lifecycle/` for memory management
- **Checkpoints**: See `examples/checkpoints/` for saving/resuming context
- **Remediation**: See `examples/remediation/` for error pattern tracking
- **Context-Folding**: See `examples/context-folding/` for isolated subtask execution
- **Qdrant Setup**: See `examples/qdrant-config/` for production deployment

## See Also

- Repository service: `internal/repository/service.go`
- MCP handlers: `internal/mcp/handlers/repository.go`
- Vector store interface: `internal/vectorstore/interface.go`
- Embeddings: `internal/embeddings/provider.go`
