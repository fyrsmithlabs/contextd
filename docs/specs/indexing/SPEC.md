# Feature: Repository Indexing System

## Overview

The repository indexing system provides semantic search capabilities over existing repositories and directories. It walks file trees, filters files based on include/exclude patterns and size limits, reads file contents, and creates searchable checkpoints with vector embeddings. This enables users to search their codebase and documentation semantically using natural language queries.

**Status**: Implemented (v0.x)

**MCP Tool**: `index_repository`

**Related Services**: Checkpoint service, embedding service, vector store

## Motivation

### Problem Statement

Developers often need to:
- Find code examples across large repositories
- Search documentation and markdown files
- Locate configuration files and patterns
- Discover where specific functionality is implemented
- Navigate unfamiliar codebases quickly

Traditional text-based search (grep, find) requires:
- Exact keyword matching
- Knowledge of specific terms
- Multiple iterations to find relevant code
- Manual filtering of irrelevant results

### Solution

Repository indexing enables:
- **Semantic search**: Find code by describing what it does, not just keywords
- **Natural language queries**: "authentication middleware" finds auth code
- **Cross-project search**: Search across multiple indexed repositories
- **Context preservation**: Search results include file paths and surrounding context
- **One-time indexing**: Index once, search repeatedly without re-scanning

### Use Cases

1. **Onboarding**: New developers quickly find relevant code examples
2. **Documentation search**: Find docs by topic without knowing exact filenames
3. **Code discovery**: Locate similar patterns across projects
4. **Architecture analysis**: Understand codebase structure through semantic exploration
5. **Knowledge base**: Index documentation, READMEs, and guides

## Requirements

### Functional Requirements

#### FR1: File Tree Traversal
- **FR1.1**: Recursively walk directory structure from root path
- **FR1.2**: Follow symlinks with cycle detection
- **FR1.3**: Skip directories that match exclude patterns
- **FR1.4**: Handle permission errors gracefully

#### FR2: Pattern Matching
- **FR2.1**: Support glob-style include patterns (e.g., `*.go`, `**/*.md`)
- **FR2.2**: Support glob-style exclude patterns (e.g., `node_modules/**`, `*.log`)
- **FR2.3**: Include patterns: whitelist files to index (empty = all files)
- **FR2.4**: Exclude patterns: blacklist files to skip (e.g., binaries, logs)
- **FR2.5**: Pattern matching on both basename and full path

#### FR3: File Size Filtering
- **FR3.1**: Skip files exceeding max file size limit
- **FR3.2**: Default max file size: 1MB
- **FR3.3**: Configurable max file size: 0 to 10MB
- **FR3.4**: Report skipped files in debug logs

#### FR4: File Content Reading
- **FR4.1**: Read text file contents as UTF-8
- **FR4.2**: Handle binary files gracefully (skip or error)
- **FR4.3**: Respect context cancellation during read operations
- **FR4.4**: Validate file paths against repository root (prevent traversal)

#### FR5: Checkpoint Creation
- **FR5.1**: Create one checkpoint per indexed file
- **FR5.2**: Checkpoint summary: `"Indexed file: <relative-path>"`
- **FR5.3**: Checkpoint description: Full file contents
- **FR5.4**: Checkpoint project_path: Repository root path
- **FR5.5**: Checkpoint context: `{"indexed_file": "<relative-path>"}`
- **FR5.6**: Checkpoint tags: `["indexed", "repository", "<file-extension>"]`
- **FR5.7**: Generate vector embeddings automatically via checkpoint service

#### FR6: Indexing Results
- **FR6.1**: Return total count of files indexed
- **FR6.2**: Return include patterns used
- **FR6.3**: Return exclude patterns used
- **FR6.4**: Return max file size applied
- **FR6.5**: Return timestamp when indexing completed

### Non-Functional Requirements

#### NFR1: Performance
- **NFR1.1**: Index 1000 files in < 5 minutes (assuming 1KB average file size)
- **NFR1.2**: Respect 5-minute timeout for MCP tool operations
- **NFR1.3**: Batch embedding generation where possible
- **NFR1.4**: Minimal memory footprint (stream file contents, don't load all at once)

#### NFR2: Scalability
- **NFR2.1**: Support repositories up to 10,000 files
- **NFR2.2**: Support file sizes up to 10MB
- **NFR2.3**: Handle deeply nested directory structures (>100 levels)

#### NFR3: Reliability
- **NFR3.1**: Continue indexing if individual file fails
- **NFR3.2**: Log errors for skipped files
- **NFR3.3**: Return count of successfully indexed files
- **NFR3.4**: Idempotent: re-indexing creates duplicate checkpoints (by design)

#### NFR4: Security
- **NFR4.1**: Validate repository path exists and is accessible
- **NFR4.2**: Prevent path traversal attacks (validate all file paths)
- **NFR4.3**: Respect file system permissions
- **NFR4.4**: Don't index sensitive files (*.env, credentials.json, etc.)

#### NFR5: Observability
- **NFR5.1**: OpenTelemetry tracing for indexing operations
- **NFR5.2**: Metrics: files indexed, time taken, errors encountered
- **NFR5.3**: Log progress at regular intervals (every 100 files)
- **NFR5.4**: Structured logging with repository path and file count

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                     MCP Server                              │
│                                                             │
│  ┌───────────────────────────────────────────────────┐    │
│  │ handleIndexRepository                              │    │
│  │  - Validate inputs                                 │    │
│  │  - Call indexRepositoryFiles                       │    │
│  │  - Return IndexRepositoryOutput                    │    │
│  └─────────────────┬─────────────────────────────────┘    │
│                    │                                        │
│  ┌─────────────────▼─────────────────────────────────┐    │
│  │ indexRepositoryFiles                               │    │
│  │  - filepath.Walk to traverse tree                  │    │
│  │  - Apply include/exclude patterns                  │    │
│  │  - Check file size limits                          │    │
│  │  - Read file contents                              │    │
│  │  - Create checkpoints via service                  │    │
│  └─────────────────┬─────────────────────────────────┘    │
└────────────────────┼─────────────────────────────────────┘
                     │
                     │ Create checkpoint
                     │
         ┌───────────▼────────────┐
         │  Checkpoint Service    │
         │  - Generate embedding  │
         │  - Store in vector DB  │
         │  - Return checkpoint   │
         └───────────┬────────────┘
                     │
         ┌───────────▼────────────┐
         │   Vector Store         │
         │   - Project database   │
         │   - checkpoints coll.  │
         └────────────────────────┘
```

### Data Flow

1. **User invokes MCP tool**: `index_repository(path="/repo", include_patterns=["*.md"])`
2. **MCP handler validates inputs**:
   - Check path exists
   - Validate patterns (test with `filepath.Match`)
   - Validate max_file_size (0 < size <= 10MB)
3. **indexRepositoryFiles walks tree**:
   - For each file in directory tree:
     - Skip if directory
     - Skip if file size > max_file_size
     - Skip if matches exclude pattern
     - Skip if doesn't match include pattern (if specified)
     - Validate path within repository root
     - Read file contents
     - Create checkpoint via service
4. **Checkpoint service processes**:
   - Generate embedding from file contents
   - Store in project-specific database
   - Return checkpoint with ID
5. **Return results**: Files indexed count, patterns used, timestamp

### Component Interactions

#### MCP Tool Handler (`pkg/mcp/tools.go`)

```go
func (s *Server) handleIndexRepository(ctx context.Context, req *mcpsdk.CallToolRequest, input IndexRepositoryInput) (*mcpsdk.CallToolResult, IndexRepositoryOutput, error) {
    // Validation
    // Call indexRepositoryFiles
    // Return output
}
```

**Responsibilities**:
- Input validation (path, patterns, max_file_size)
- Context timeout management (5 minutes)
- Error handling and MCP error conversion
- OpenTelemetry span creation
- Metrics recording

#### Repository Indexer (`pkg/mcp/tools.go`)

```go
func (s *Server) indexRepositoryFiles(ctx context.Context, repoPath string, includePatterns, excludePatterns []string, maxFileSize int64) (int, error) {
    // filepath.Walk implementation
    // Pattern matching
    // File reading
    // Checkpoint creation
}
```

**Responsibilities**:
- File tree traversal
- Pattern matching (include/exclude)
- File size filtering
- Path traversal prevention
- File content reading
- Checkpoint creation delegation

#### Checkpoint Service (`pkg/checkpoint/service.go`)

**Responsibilities**:
- Embedding generation (via embedding service)
- Vector storage (via vector store)
- Project database isolation
- OpenTelemetry instrumentation

#### Embedding Service (`pkg/embedding/service.go`)

**Responsibilities**:
- Generate embeddings for file contents
- Support OpenAI and TEI backends
- Batch processing (future optimization)

#### Vector Store (`pkg/vectorstore/`)

**Responsibilities**:
- Store checkpoint vectors
- Project-specific database isolation
- Efficient vector search

## API Specification

### MCP Tool: `index_repository`

#### Description

Index an existing repository or directory for semantic search. Creates searchable checkpoints from files matching include patterns while respecting exclude patterns and file size limits.

#### Input Schema

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Path to repository or directory to index (required)",
      "minLength": 1
    },
    "include_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "File patterns to include (e.g., ['*.md', '*.txt'])",
      "default": []
    },
    "exclude_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "File patterns to exclude (e.g., ['*.log', 'node_modules/**'])",
      "default": []
    },
    "max_file_size": {
      "type": "integer",
      "description": "Maximum file size in bytes (default: 1MB, max: 10MB)",
      "default": 1048576,
      "minimum": 0,
      "maximum": 10485760
    }
  },
  "required": ["path"]
}
```

#### Output Schema

```json
{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Repository path that was indexed"
    },
    "files_indexed": {
      "type": "integer",
      "description": "Number of files successfully indexed"
    },
    "include_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Include patterns used"
    },
    "exclude_patterns": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Exclude patterns used"
    },
    "max_file_size": {
      "type": "integer",
      "description": "Maximum file size applied (bytes)"
    },
    "indexed_at": {
      "type": "string",
      "format": "date-time",
      "description": "Timestamp when indexing completed"
    }
  }
}
```

#### Error Responses

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `validation_error` | 400 | Invalid input (path doesn't exist, invalid pattern, size too large) |
| `timeout_error` | 504 | Indexing exceeded 5-minute timeout |
| `internal_error` | 500 | Unexpected error during indexing |

#### Examples

**Example 1: Index all markdown files**

Request:
```json
{
  "path": "/home/user/docs",
  "include_patterns": ["*.md"]
}
```

Response:
```json
{
  "path": "/home/user/docs",
  "files_indexed": 42,
  "include_patterns": ["*.md"],
  "exclude_patterns": [],
  "max_file_size": 1048576,
  "indexed_at": "2025-11-04T12:00:00Z"
}
```

**Example 2: Index code with exclusions**

Request:
```json
{
  "path": "/home/user/project",
  "include_patterns": ["*.go", "*.js", "*.py"],
  "exclude_patterns": ["*_test.go", "node_modules/**", "__pycache__/**"],
  "max_file_size": 524288
}
```

Response:
```json
{
  "path": "/home/user/project",
  "files_indexed": 157,
  "include_patterns": ["*.go", "*.js", "*.py"],
  "exclude_patterns": ["*_test.go", "node_modules/**", "__pycache__/**"],
  "max_file_size": 524288,
  "indexed_at": "2025-11-04T12:05:00Z"
}
```

**Example 3: Validation error**

Request:
```json
{
  "path": "/nonexistent/path"
}
```

Response (Error):
```json
{
  "error": "validation_error",
  "message": "path does not exist",
  "details": {
    "field": "path",
    "path": "/nonexistent/path"
  }
}
```

### CLI Command: `ctxd index`

The CLI provides a convenience wrapper for the MCP tool.

**Usage**:
```bash
ctxd index <path> [flags]
```

**Flags**:
- `--include <patterns>`: File patterns to include (repeatable)
- `--exclude <patterns>`: File patterns to exclude (repeatable)
- `--max-size <bytes>`: Maximum file size (default: 1048576)

**Examples**:
```bash
# Index all files
ctxd index /path/to/repo

# Index markdown files
ctxd index . --include "*.md"

# Index with exclusions
ctxd index /repo --include "*.go" --exclude "vendor/**" --exclude "*_test.go"

# Custom max size (512KB)
ctxd index /docs --max-size 524288
```

**Note**: The CLI currently only provides usage instructions and debugging commands. It does not directly invoke the MCP tool (requires contextd service running and MCP integration).

## Data Model

### Checkpoint Storage

Each indexed file becomes a checkpoint stored in the project-specific vector database.

#### Checkpoint Structure

```json
{
  "id": "uuid-generated",
  "summary": "Indexed file: docs/README.md",
  "description": "<full file contents>",
  "project_path": "/home/user/repo",
  "context": {
    "indexed_file": "docs/README.md"
  },
  "tags": ["indexed", "repository", ".md"],
  "token_count": 1523,
  "created_at": "2025-11-04T12:00:00Z",
  "updated_at": "2025-11-04T12:00:00Z"
}
```

#### Vector Storage

- **Database**: `project_<sha256(project_path)>`
- **Collection**: `checkpoints`
- **Vector dimension**: 1536 (OpenAI text-embedding-3-small) or 384 (TEI BAAI/bge-small-en-v1.5)
- **Metric**: Cosine similarity

#### Payload Fields

| Field | Type | Description | Indexed |
|-------|------|-------------|---------|
| `id` | string | UUID | Yes (primary key) |
| `summary` | string | "Indexed file: <path>" | No |
| `content` | string | Full file contents + context JSON | No |
| `project` | string | Repository root path | No (database boundary) |
| `timestamp` | int64 | Unix timestamp (seconds) | Yes |
| `token_count` | int | Embedding tokens used | No |
| `tags` | string | Comma-separated tags | Yes (filterable) |

### Search Integration

Indexed files are searchable via:

1. **`checkpoint_search`**: Semantic search across indexed files
   ```json
   {
     "query": "authentication middleware",
     "project_path": "/home/user/repo",
     "tags": ["indexed", ".go"]
   }
   ```

2. **`checkpoint_list`**: List all indexed files
   ```json
   {
     "project_path": "/home/user/repo",
     "limit": 50
   }
   ```

### Tag Strategy

Tags enable filtering indexed files:
- `"indexed"`: Marks checkpoint as indexed file (vs manual checkpoint)
- `"repository"`: Indicates repository indexing (vs single file)
- `".<extension>"`: File extension (e.g., `.go`, `.md`, `.py`)

**Filter examples**:
```javascript
// Find all indexed markdown files
tags.includes("indexed") && tags.includes(".md")

// Find all indexed code (exclude docs)
tags.includes("indexed") && !tags.includes(".md")
```

## File Pattern Matching

### Pattern Syntax

Uses Go's `filepath.Match` syntax (glob-style):

| Pattern | Matches |
|---------|---------|
| `*.go` | All Go files in current directory |
| `**/*.go` | All Go files recursively (requires custom implementation) |
| `test_*.go` | Go files starting with "test_" |
| `main.go` | Exact filename match |
| `pkg/*/service.go` | service.go in any immediate subdirectory of pkg/ |

**Note**: Go's `filepath.Match` does NOT support `**` (recursive). The implementation matches patterns against both basename and full relative path to achieve recursive behavior.

### Include Patterns

**Behavior**:
- If **empty**: Include ALL files (subject to exclude patterns and size limit)
- If **specified**: Include ONLY files matching at least one pattern

**Matching logic**:
```go
if len(includePatterns) > 0 {
    included := false
    for _, pattern := range includePatterns {
        if matched, _ := filepath.Match(pattern, basename); matched {
            included = true
            break
        }
    }
    if !included {
        return nil // Skip file
    }
}
```

**Examples**:
```javascript
// Include only markdown files
include_patterns: ["*.md"]

// Include multiple types
include_patterns: ["*.go", "*.py", "*.js"]

// Include all (default)
include_patterns: []
```

### Exclude Patterns

**Behavior**:
- Files matching ANY exclude pattern are skipped
- Matched against both basename and full path
- Takes precedence over include patterns

**Matching logic**:
```go
for _, pattern := range excludePatterns {
    if matched, _ := filepath.Match(pattern, basename); matched {
        return nil // Skip file
    }
    if matched, _ := filepath.Match(pattern, fullPath); matched {
        return nil // Skip file
    }
}
```

**Examples**:
```javascript
// Exclude vendor and test files
exclude_patterns: ["vendor/**", "*_test.go"]

// Exclude binaries and logs
exclude_patterns: ["*.exe", "*.bin", "*.log", "*.tmp"]

// Exclude common directories
exclude_patterns: ["node_modules/**", ".git/**", "__pycache__/**"]
```

### Pattern Validation

Patterns are validated before indexing:

```go
for _, pattern := range includePatterns {
    if _, err := filepath.Match(pattern, "test"); err != nil {
        return ValidationError("invalid include pattern")
    }
}
```

Invalid patterns cause immediate error (before any indexing).

### Common Patterns

#### Exclude Sensitive Files
```javascript
exclude_patterns: [
  "*.env",
  "*.key",
  "*.pem",
  "*credentials*",
  "*secret*",
  ".aws/**",
  ".ssh/**"
]
```

#### Index Documentation Only
```javascript
include_patterns: ["*.md", "*.txt", "*.rst"],
exclude_patterns: ["node_modules/**", "vendor/**"]
```

#### Index Source Code Only
```javascript
include_patterns: ["*.go", "*.js", "*.py", "*.java", "*.c", "*.cpp"],
exclude_patterns: ["*_test.*", "*.min.js", "vendor/**", "node_modules/**"]
```

## File Size Handling

### Size Limits

- **Default**: 1MB (1,048,576 bytes)
- **Minimum**: 0 bytes (no limit, not recommended)
- **Maximum**: 10MB (10,485,760 bytes)

### Rationale

**1MB default**:
- Balances indexing coverage and embedding costs
- Most source files are < 1MB
- Prevents indexing large generated files (minified JS, compiled binaries)

**10MB maximum**:
- Protects against runaway embedding costs
- OpenAI API has 8192 token limit (~32KB text)
- Large files would be truncated or fail anyway

### Size Filtering Logic

```go
if info.Size() > maxFileSize {
    return nil // Skip file silently
}
```

**Behavior**:
- Files larger than limit are silently skipped
- No error is returned (not considered failure)
- Skipped files NOT counted in `files_indexed`
- Logged at debug level (not user-visible)

### Size-Based Strategy

**Small repositories (< 100 files)**:
```javascript
max_file_size: 10485760  // 10MB - index everything
```

**Medium repositories (100-1000 files)**:
```javascript
max_file_size: 1048576  // 1MB - skip large generated files
```

**Large repositories (> 1000 files)**:
```javascript
max_file_size: 524288  // 512KB - aggressively filter
include_patterns: ["*.md", "*.txt"]  // Documentation only
```

## Indexing Workflow

### Phase 1: Validation

1. **Path validation**:
   ```go
   if _, err := os.Stat(input.Path); os.IsNotExist(err) {
       return ValidationError("path does not exist")
   }
   ```

2. **Pattern validation**:
   ```go
   for _, pattern := range input.IncludePatterns {
       if _, err := filepath.Match(pattern, "test"); err != nil {
           return ValidationError("invalid include pattern")
       }
   }
   ```

3. **Size validation**:
   ```go
   if input.MaxFileSize > 10*1024*1024 {
       return ValidationError("max_file_size too large")
   }
   ```

### Phase 2: Tree Traversal

Uses `filepath.Walk` for recursive directory traversal:

```go
err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
    // 1. Check for walk errors
    if err != nil {
        return err
    }

    // 2. Skip directories
    if info.IsDir() {
        return nil
    }

    // 3. Check file size
    if info.Size() > maxFileSize {
        return nil
    }

    // 4. Check exclude patterns
    for _, pattern := range excludePatterns {
        if matched, _ := filepath.Match(pattern, info.Name()); matched {
            return nil
        }
        if matched, _ := filepath.Match(pattern, path); matched {
            return nil
        }
    }

    // 5. Check include patterns (if specified)
    if len(includePatterns) > 0 {
        included := false
        for _, pattern := range includePatterns {
            if matched, _ := filepath.Match(pattern, info.Name()); matched {
                included = true
                break
            }
        }
        if !included {
            return nil
        }
    }

    // 6. Validate path within repository (security)
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("failed to resolve file path: %w", err)
    }
    absRepoPath, err := filepath.Abs(repoPath)
    if err != nil {
        return fmt.Errorf("failed to resolve repository path: %w", err)
    }
    if !strings.HasPrefix(absPath, absRepoPath) {
        return fmt.Errorf("path traversal detected: %s", path)
    }

    // 7. Read file contents
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    // 8. Create checkpoint
    relPath, _ := filepath.Rel(repoPath, path)
    req := &validation.CreateCheckpointRequest{
        Summary:     fmt.Sprintf("Indexed file: %s", relPath),
        Description: string(content),
        ProjectPath: repoPath,
        Context:     map[string]string{"indexed_file": relPath},
        Tags:        []string{"indexed", "repository", filepath.Ext(path)},
    }

    _, err = s.services.Checkpoint.Create(ctx, req)
    if err != nil {
        return err
    }

    filesIndexed++
    return nil
})
```

### Phase 3: Checkpoint Creation

For each file:

1. **Generate relative path**: `filepath.Rel(repoPath, path)`
2. **Build checkpoint request**:
   - Summary: `"Indexed file: <relative-path>"`
   - Description: Full file contents as string
   - Project path: Repository root (for database isolation)
   - Context: `{"indexed_file": "<relative-path>"}`
   - Tags: `["indexed", "repository", "<extension>"]`
3. **Call checkpoint service**: `s.services.Checkpoint.Create(ctx, req)`
4. **Increment counter**: `filesIndexed++`

### Phase 4: Return Results

```go
output := IndexRepositoryOutput{
    Path:            input.Path,
    FilesIndexed:    filesIndexed,
    IncludePatterns: input.IncludePatterns,
    ExcludePatterns: input.ExcludePatterns,
    MaxFileSize:     input.MaxFileSize,
    IndexedAt:       time.Now(),
}
```

## Performance Characteristics

### Time Complexity

**File traversal**: O(N) where N = total files in directory tree

**Pattern matching**: O(N × P) where P = number of patterns

**Embedding generation**: O(N × T) where T = average time per embedding
- OpenAI API: ~500ms per embedding
- TEI local: ~50ms per embedding

**Vector storage**: O(N × V) where V = time per vector insert

**Total time**: O(N × (P + T + V))

### Space Complexity

**Memory**: O(F) where F = largest file size
- Files read one at a time (streaming)
- Checkpoint service processes sequentially
- No bulk loading into memory

**Storage**: O(N × S) where S = average file size
- Each file stored as checkpoint
- Full file contents in description field
- Vector embeddings (1536 × 4 bytes = 6KB per file)

### Throughput Estimates

**TEI embedding backend** (recommended):
- 50ms per embedding
- 1KB average file size
- 10ms per vector insert
- **Throughput**: ~16 files/second
- **1000 files**: ~62 seconds

**OpenAI embedding backend**:
- 500ms per embedding (rate limited)
- 1KB average file size
- 10ms per vector insert
- **Throughput**: ~2 files/second
- **1000 files**: ~8.3 minutes

### Optimization Strategies

#### Current Implementation (Sequential)
```
For each file:
  Read → Embed → Store → Next
```

#### Future: Batch Embedding (10x faster)
```
Read 10 files → Embed batch → Store batch → Next 10
```

Reduces embedding overhead from N calls to N/10 calls.

#### Future: Parallel Processing (20x faster)
```
Worker pool (10 workers):
  Each worker: Read → Embed → Store
```

Parallelizes I/O, embedding, and storage operations.

#### Future: Incremental Indexing
```
Check last modified timestamp
Skip files not changed since last index
```

Reduces re-indexing overhead for large repositories.

## Error Handling

### Error Categories

#### 1. Validation Errors (User-facing)

**Path doesn't exist**:
```json
{
  "error": "validation_error",
  "message": "path does not exist",
  "details": {"field": "path", "path": "/invalid/path"}
}
```

**Invalid pattern**:
```json
{
  "error": "validation_error",
  "message": "invalid include pattern",
  "details": {"field": "include_patterns", "pattern": "[invalid"}
}
```

**Max file size too large**:
```json
{
  "error": "validation_error",
  "message": "max_file_size too large",
  "details": {"field": "max_file_size", "max_allowed": 10485760, "provided": 20971520}
}
```

#### 2. Timeout Errors

**Indexing timeout** (5 minutes exceeded):
```json
{
  "error": "timeout_error",
  "message": "repository indexing timed out",
  "details": {"timeout": "5m0s"}
}
```

#### 3. Internal Errors

**Embedding generation failure**:
```json
{
  "error": "internal_error",
  "message": "failed to generate embedding",
  "details": {"cause": "OpenAI API rate limit exceeded"}
}
```

**Vector store failure**:
```json
{
  "error": "internal_error",
  "message": "failed to insert checkpoint",
}
```

**Path traversal detected**:
```json
{
  "error": "internal_error",
  "message": "path traversal detected",
  "details": {"path": "../../etc/passwd"}
}
```

### Error Recovery

#### Continue on File Errors

Current behavior: **FAIL FAST** - first error aborts indexing

```go
if err := processFile(path); err != nil {
    return err // Abort entire indexing
}
```

**Future improvement**: Continue indexing, collect errors

```go
var errors []error
if err := processFile(path); err != nil {
    errors = append(errors, err)
    continue // Process next file
}
```

Return partial success with error details:
```json
{
  "files_indexed": 42,
  "files_failed": 3,
  "errors": [
    {"file": "path/to/file1.txt", "error": "permission denied"},
    {"file": "path/to/file2.md", "error": "encoding error"}
  ]
}
```

#### Retry Strategy

**Transient errors** (network, rate limits):
- Retry with exponential backoff
- Max 3 retries per file
- Skip file after max retries

**Permanent errors** (invalid UTF-8, permission denied):
- Log error and skip file
- Continue indexing

## Security Considerations

### Path Traversal Prevention

**Attack**: User provides path like `/repo/../../etc/passwd`

**Defense**:
```go
absPath, err := filepath.Abs(path)
absRepoPath, err := filepath.Abs(repoPath)

if !strings.HasPrefix(absPath, absRepoPath+string(filepath.Separator)) &&
    absPath != absRepoPath {
    return fmt.Errorf("path traversal detected: %s", path)
}
```

**Result**: Only files within repository root can be indexed

### Sensitive File Exclusion

**Recommended exclude patterns**:
```javascript
exclude_patterns: [
  "*.env",           // Environment variables
  "*.key",           // Private keys
  "*.pem",           // Certificates
  "*credentials*",   // Credential files
  "*secret*",        // Secret files
  ".aws/**",         // AWS credentials
  ".ssh/**",         // SSH keys
  ".git/config",     // Git config (may contain tokens)
  "*.p12",           // Certificate bundles
  "*.jks",           // Java keystores
]
```

**Note**: Automatic exclusion not implemented - users must specify patterns

### File System Permissions

**Respects OS permissions**:
- If user can't read file → `os.ReadFile` fails → indexing aborts
- No privilege escalation
- Runs with user's permissions

### Multi-Tenant Isolation

**Project-specific databases**:
- Each repository indexed into separate database
- Database name: `project_<sha256(repo_path)>`
- No cross-project contamination

**Search isolation**:
- Checkpoint search scoped to project database
- Cannot search other users' indexed files

### Input Sanitization

**Path sanitization**:
```go
absPath, err := filepath.Abs(input.Path)
if err != nil {
    return ValidationError("invalid path")
}
```

**Pattern sanitization**:
- Validated via `filepath.Match(pattern, "test")`
- Invalid patterns rejected before indexing

**Size validation**:
- Max file size: 10MB (prevents memory exhaustion)
- Max timeout: 5 minutes (prevents indefinite hanging)

## Testing Requirements

### Unit Tests

#### 1. Pattern Matching Tests

```go
func TestPatternMatching(t *testing.T) {
    tests := []struct {
        name            string
        filename        string
        includePatterns []string
        excludePatterns []string
        want            bool
    }{
        {"include match", "file.md", []string{"*.md"}, nil, true},
        {"exclude match", "test.log", nil, []string{"*.log"}, false},
        {"include + exclude", "test.md", []string{"*.md"}, []string{"test.*"}, false},
        {"no patterns", "file.txt", nil, nil, true},
    }
    // ...
}
```

#### 2. File Size Filtering Tests

```go
func TestFileSizeFiltering(t *testing.T) {
    tests := []struct {
        name        string
        fileSize    int64
        maxFileSize int64
        want        bool
    }{
        {"within limit", 500000, 1048576, true},
        {"exceeds limit", 2000000, 1048576, false},
        {"exact limit", 1048576, 1048576, true},
    }
    // ...
}
```

#### 3. Path Traversal Prevention Tests

```go
func TestPathTraversalPrevention(t *testing.T) {
    tests := []struct {
        name     string
        repoPath string
        filePath string
        wantErr  bool
    }{
        {"valid path", "/repo", "/repo/file.txt", false},
        {"traversal up", "/repo", "/repo/../etc/passwd", true},
        {"traversal sideways", "/repo", "/other/file.txt", true},
    }
    // ...
}
```

### Integration Tests

#### 1. End-to-End Indexing Test

```go
func TestIndexRepository_EndToEnd(t *testing.T) {
    // Setup: Create temp directory with test files
    tmpDir := t.TempDir()
    createTestFiles(tmpDir, map[string]string{
        "file1.md":  "# Documentation",
        "file2.txt": "Plain text",
        "large.bin": string(make([]byte, 2*1024*1024)), // 2MB
    })

    // Index repository
    output, err := indexRepository(ctx, tmpDir, []string{"*.md", "*.txt"}, nil, 1048576)

    // Verify: 2 files indexed (large.bin skipped)
    assert.NoError(t, err)
    assert.Equal(t, 2, output.FilesIndexed)

    // Verify checkpoints created
    checkpoints, err := checkpointService.Search(ctx, "documentation", 10, tmpDir, []string{"indexed"})
    assert.NoError(t, err)
    assert.Len(t, checkpoints.Results, 1)
}
```

#### 2. Pattern Exclusion Test

```go
func TestIndexRepository_ExcludePatterns(t *testing.T) {
    tmpDir := t.TempDir()
    createTestFiles(tmpDir, map[string]string{
        "main.go":      "package main",
        "main_test.go": "package main",
        "vendor/pkg.go": "package vendor",
    })

    output, err := indexRepository(ctx, tmpDir, []string{"*.go"}, []string{"*_test.go", "vendor/**"}, 1048576)

    assert.NoError(t, err)
    assert.Equal(t, 1, output.FilesIndexed) // Only main.go
}
```

#### 3. Large Repository Test

```go
func TestIndexRepository_LargeRepo(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping large repository test")
    }

    tmpDir := t.TempDir()
    // Create 1000 files
    for i := 0; i < 1000; i++ {
        createFile(tmpDir, fmt.Sprintf("file%d.txt", i), "content")
    }

    start := time.Now()
    output, err := indexRepository(ctx, tmpDir, nil, nil, 1048576)
    duration := time.Since(start)

    assert.NoError(t, err)
    assert.Equal(t, 1000, output.FilesIndexed)
    assert.Less(t, duration, 5*time.Minute) // Must complete within timeout
}
```

### Manual Testing Checklist

- [ ] Index small repository (< 10 files)
- [ ] Index medium repository (100-1000 files)
- [ ] Index large repository (> 1000 files)
- [ ] Verify checkpoint creation in database
- [ ] Search indexed files semantically
- [ ] Test with various include patterns
- [ ] Test with various exclude patterns
- [ ] Test with different max file sizes
- [ ] Verify path traversal prevention
- [ ] Test with non-existent path (validation error)
- [ ] Test with invalid patterns (validation error)
- [ ] Test timeout with very large repository
- [ ] Test OpenTelemetry traces and metrics
- [ ] Test CLI command (`ctxd index`)

## Implementation Plan

### Phase 1: Core Implementation (Complete)

**Status**: Implemented in v0.x

**Components**:
- [x] MCP tool handler (`handleIndexRepository`)
- [x] File tree traversal (`indexRepositoryFiles`)
- [x] Pattern matching (include/exclude)
- [x] File size filtering
- [x] Path traversal prevention
- [x] Checkpoint creation integration
- [x] CLI command (`ctxd index`)

**Commits**:
- Initial implementation with basic pattern matching
- Path traversal security fix (CVE-007)

### Phase 2: Optimizations (Future)

**Priority**: Medium

**Features**:
- [ ] Batch embedding generation (10x speedup)
- [ ] Parallel processing with worker pool (20x speedup)
- [ ] Incremental indexing (skip unchanged files)
- [ ] Progress reporting (callback for every N files)

**Estimated Effort**: 1-2 weeks

### Phase 3: Enhanced Error Handling (Future)

**Priority**: Low

**Features**:
- [ ] Continue on file errors (partial success)
- [ ] Retry with exponential backoff
- [ ] Detailed error reporting per file
- [ ] Binary file detection and skip

**Estimated Effort**: 1 week

### Phase 4: Advanced Features (Future)

**Priority**: Low

**Features**:
- [ ] AST-based code indexing (extract functions, classes)
- [ ] Chunking for large files (split into sections)
- [ ] Metadata extraction (author, license, etc.)
- [ ] Custom tag strategies (language-specific)
- [ ] De-duplication (avoid re-indexing identical files)

**Estimated Effort**: 2-3 weeks

## Open Questions

### Q1: Should indexing be idempotent?

**Current behavior**: Re-indexing creates duplicate checkpoints

**Options**:
1. **Keep current behavior** (simplest)
   - Pros: Simple implementation, no state tracking
   - Cons: Duplicate checkpoints clutter search results
2. **Delete old checkpoints before re-indexing**
   - Pros: Clean state, no duplicates
   - Cons: Loses historical checkpoints (e.g., if file was edited)
3. **Update existing checkpoints**
   - Pros: Preserves checkpoint IDs, clean state

**Recommendation**: Option 1 for MVP, Option 3 for future enhancement

### Q2: How to handle binary files?

**Current behavior**: Attempts to read as UTF-8, may fail

**Options**:
1. **Skip binary files automatically** (check MIME type or magic bytes)
2. **Extract metadata only** (filename, size, type)
3. **Convert to text** (e.g., PDF → text, DOCX → text)

**Recommendation**: Option 1 (skip) for MVP, Option 3 (convert) for future

### Q3: Should we support symbolic links?

**Current behavior**: `filepath.Walk` follows symlinks

**Risk**: Circular symlinks cause infinite loop

**Options**:
1. **Continue following symlinks** (current)
2. **Skip symlinks entirely**
3. **Follow with cycle detection**

**Recommendation**: Option 3 for security and robustness

### Q4: What about very large repositories (> 10,000 files)?

**Current limitation**: 5-minute timeout may be insufficient

**Options**:
1. **Increase timeout** (e.g., 30 minutes)
2. **Batch indexing** (index 1000 files at a time, multiple calls)
3. **Background job** (return immediately, index asynchronously)

**Recommendation**: Option 1 (increase timeout) for now, Option 3 (background job) for future

### Q5: How to exclude sensitive files automatically?

**Current behavior**: User must specify exclude patterns

**Options**:
1. **Keep current behavior** (explicit exclusions)
2. **Built-in exclusion list** (*.env, *.key, etc.)
3. **AI-powered detection** (classify files as sensitive)

**Recommendation**: Option 2 (built-in list) as default, allow override

## References

### Related Standards
- [docs/standards/architecture.md](../../standards/architecture.md) - Multi-tenant architecture
- [docs/standards/coding-standards.md](../../standards/coding-standards.md) - Go coding patterns
- [docs/standards/testing-standards.md](../../standards/testing-standards.md) - TDD requirements

### Related Specifications
- [docs/specs/config/SPEC.md](../config/SPEC.md) - Checkpoint system specification

### Related ADRs
- [docs/adr/002-universal-multi-tenant-architecture.md](../../architecture/adr/002-universal-multi-tenant-architecture.md) - Multi-tenant database isolation

### Implementation Files
- `pkg/mcp/tools.go` - MCP tool handler
- `pkg/mcp/types.go` - Input/output types
- `pkg/checkpoint/service.go` - Checkpoint creation
- `cmd/ctxd/index.go` - CLI command

### External Documentation
- [Go filepath package](https://pkg.go.dev/path/filepath) - File path utilities
- [Go filepath.Match](https://pkg.go.dev/path/filepath#Match) - Pattern matching
- [Go filepath.Walk](https://pkg.go.dev/path/filepath#Walk) - Directory traversal
