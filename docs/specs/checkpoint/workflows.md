# Checkpoint Workflows

**Parent**: [../SPEC.md](../SPEC.md)

## Create Checkpoint Workflow

### Function
`Service.Create(ctx, req) (*Checkpoint, error)`

### Step-by-Step Process

1. **Generate UUID** for checkpoint ID
2. **Combine text** summary + description for embedding
3. **Generate embedding** vector (1536 dimensions)
4. **Ensure database exists** project database (idempotent)
5. **Insert vector** with metadata to database
6. **Return checkpoint** with token count

### Embedding Strategy

**Input Text**: `summary + "\n\n" + description`
**Model**: text-embedding-3-small (OpenAI) or BAAI/bge-small-en-v1.5 (TEI)
**Dimension**: 1536 (OpenAI) or 384 (TEI)
**Cost**: ~$0.02 per 1M tokens (OpenAI) or free (TEI local)
**Caching**: Automatic via embedding service (15-minute TTL)

### Metadata Storage

```json
{
  "summary": "Brief checkpoint description",
  "content": "summary\n\ndescription\n\nContext: {json}",
  "project": "/absolute/path/to/project",
  "timestamp": 1699564800,
  "token_count": 42,
  "tags": "feature,bugfix,testing"
}
```

### Performance

- Typical latency: 50-200ms (dominated by embedding generation)
- Cached embeddings: <10ms
- Token count: Simple word-based approximation (0.75 tokens/word)

### Error Handling

- Embedding failures → wrapped error with context
- Database errors → retry with exponential backoff (future)
- Validation errors → rejected at API layer
- Timeout → context deadline exceeded (30s default)

## Search Checkpoint Workflow

### Function
`Service.Search(ctx, query, topK, projectPath, tags) (*SearchResult, error)`

### Step-by-Step Process

1. **Generate embedding** for search query
2. **Determine database** name (SHA256 hash of project path)
3. **Build filter** for tags (optional)
4. **Execute search** vector similarity search
5. **Convert results** to domain format
6. **Return ranked** results with scores

### Search Algorithm

**Distance Metric**: Cosine similarity (default) or L2/IP
**Index Type**: HNSW (Hierarchical Navigable Small World)
**Ranking**: Descending by similarity score (0.0 - 1.0)
**Threshold**: No minimum score (returns topK regardless)

### Filter Support

- **Tags**: `tags like "%tag1%" && tags like "%tag2%"` (AND logic)
- **Project Path**: Implicit via database boundary (no filter needed)
- **Date Range**: Not supported (use List + client-side filter)

### Query Optimization

- Embedding cached for 15 minutes (duplicate queries free)
- Database-level partition pruning (automatic)
- HNSW index enables sub-linear search time
- Typical latency: 20-100ms for cached queries

### Result Format

```json
{
  "results": [
    {
      "checkpoint": { /* full checkpoint object */ },
      "score": 0.92,
      "distance": 0.08
    }
  ],
  "query": "authentication implementation",
  "top_k": 5
}
```

## List Checkpoints Workflow

### Function
`Service.List(ctx, limit, offset, projectPath, sortBy) (*ListResult, error)`

### Step-by-Step Process

1. **Determine database** name from project path
2. **Fetch results** offset + limit + buffer
3. **Apply pagination** via slice operations
4. **Return paginated** results with total count

### Pagination Parameters

- **limit**: Results per page (default: 10, max: 100)
- **offset**: Starting position (default: 0)
- **sort_by**: Sort field (created_at, updated_at) - not yet implemented
- **projectPath**: Required for database scoping

### Limitations

- Sorting not implemented (returns in arbitrary order)
- Large offsets inefficient (must fetch offset+limit results)
- No total count optimization (must scan all records)
- Recommended: Use Search for finding specific checkpoints

### Performance

- Small offsets (<100): <50ms
- Large offsets (>1000): 100-500ms (scans many records)
- Database boundary prevents cross-project leakage

## Delete Checkpoint Workflow

### Function
`Service.Delete(ctx, id) error`

### Step-by-Step Process

1. **Determine database** name from project path
2. **Build filter** expression for ID
3. **Execute delete** operation
4. **Return error** if delete fails

### Performance

- Typical latency: <10ms
- Soft delete: Not supported (hard delete only)
- Cascade delete: Not applicable (no foreign keys)

## Usage Examples

### Example 1: Create Checkpoint

```go
import (
    "context"
    "github.com/axyzlabs/contextd/pkg/checkpoint"
    "github.com/axyzlabs/contextd/pkg/embedding"
    "github.com/axyzlabs/contextd/pkg/validation"
)

// Initialize dependencies
ctx := context.Background()
embeddingService, _ := embedding.NewService("http://localhost:8080/v1", "BAAI/bge-small-en-v1.5", "")

// Create checkpoint
req := &validation.CreateCheckpointRequest{
    Summary:     "Implemented user authentication",
    Description: "Added JWT-based authentication with refresh tokens",
    ProjectPath: "/home/user/myproject",
    Context: map[string]string{
        "branch": "feature/auth",
        "commit": "abc123",
    },
    Tags: []string{"auth", "security", "jwt"},
}

checkpoint, err := svc.Create(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created checkpoint: %s\n", checkpoint.ID)
fmt.Printf("Token count: %d\n", checkpoint.TokenCount)
```

### Example 2: Search Checkpoints

```go
// Search for authentication-related checkpoints
results, err := svc.Search(
    ctx,
    "authentication implementation",
    5,
    "/home/user/myproject",
    []string{"auth"},
)
if err != nil {
    log.Fatal(err)
}

for _, result := range results.Results {
    fmt.Printf("Score: %.2f - %s\n", result.Score, result.Checkpoint.Summary)
}
```

### Example 3: List Recent Checkpoints

```go
// List last 10 checkpoints
listResult, err := svc.List(ctx, 10, 0, "/home/user/myproject", "")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total checkpoints: %d\n", listResult.Total)
for _, cp := range listResult.Checkpoints {
    fmt.Printf("- %s (%s)\n", cp.Summary, cp.CreatedAt.Format(time.RFC3339))
}
```

### Example 4: MCP Tool Usage

**Save Checkpoint**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "checkpoint_save",
    "arguments": {
      "summary": "Completed user authentication feature",
      "description": "Implemented JWT tokens, refresh tokens, and password hashing",
      "project_path": "/home/user/myproject",
      "context": {
        "branch": "feature/auth",
        "files_changed": "5"
      },
      "tags": ["auth", "security"]
    }
  }
}
```

**Search Checkpoints**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "checkpoint_search",
    "arguments": {
      "query": "how did I implement authentication?",
      "project_path": "/home/user/myproject",
      "tags": ["auth"],
      "top_k": 3
    }
  }
}
```
