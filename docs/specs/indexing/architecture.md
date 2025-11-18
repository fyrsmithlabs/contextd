# Repository Indexing Architecture

**Parent**: [../SPEC.md](../SPEC.md)

## System Components

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

---

## Data Flow

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

---

## Component Interactions

### MCP Tool Handler (`pkg/mcp/tools.go`)

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

### Repository Indexer (`pkg/mcp/tools.go`)

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

### Checkpoint Service (`pkg/checkpoint/service.go`)

**Responsibilities**:
- Embedding generation (via embedding service)
- Vector storage (via vector store)
- Project database isolation
- OpenTelemetry instrumentation

### Embedding Service (`pkg/embedding/service.go`)

**Responsibilities**:
- Generate embeddings for file contents
- Support OpenAI and TEI backends
- Batch processing (future optimization)

### Vector Store (`pkg/vectorstore/`)

**Responsibilities**:
- Store checkpoint vectors
- Project-specific database isolation
- Efficient vector search

---

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

---

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

---

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
