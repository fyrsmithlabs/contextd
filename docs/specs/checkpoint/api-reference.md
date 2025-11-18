# Checkpoint API Reference

**Parent**: [../SPEC.md](../SPEC.md)

## MCP Tools

### checkpoint_save

**Description**: Save session checkpoint with summary, description, project path, context metadata, and tags with automatic vector embeddings

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "summary": {
      "type": "string",
      "description": "required,Brief summary of checkpoint (max 500 chars)"
    },
    "description": {
      "type": "string",
      "description": "Detailed description (optional)"
    },
    "project_path": {
      "type": "string",
      "description": "required,Absolute path to project directory"
    },
    "context": {
      "type": "object",
      "description": "Additional context metadata",
      "additionalProperties": true
    },
    "tags": {
      "type": "array",
      "description": "Tags for categorization",
      "items": {"type": "string"}
    }
  },
  "required": ["summary", "project_path"]
}
```

**Output Schema**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "summary": "Implemented user authentication",
  "description": "Added JWT-based authentication...",
  "project_path": "/home/user/myproject",
  "context": {"branch": "feature/auth", "commit": "abc123"},
  "tags": ["auth", "security"],
  "token_count": 42,
  "created_at": "2024-11-04T12:00:00Z",
  "updated_at": "2024-11-04T12:00:00Z"
}
```

**Error Codes**:
- `VALIDATION_ERROR` - Invalid input (empty summary, missing project_path)
- `INTERNAL_ERROR` - Embedding generation failed, database error
- `TIMEOUT_ERROR` - Operation exceeded 30s timeout

**Rate Limits**:
- Default: 10 requests/minute per project
- Burst: 20 requests/minute
- Configurable via `RATE_LIMIT_CHECKPOINT_SAVE`

### checkpoint_search

**Description**: Search checkpoints using semantic similarity with optional filtering by project path and tags

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "required,Search query (semantic search)"
    },
    "project_path": {
      "type": "string",
      "description": "Filter by project path"
    },
    "tags": {
      "type": "array",
      "description": "Filter by tags",
      "items": {"type": "string"}
    },
    "top_k": {
      "type": "integer",
      "description": "Number of results to return (default: 5, max: 100)"
    }
  },
  "required": ["query"]
}
```

**Output Schema**:
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

**Search Behavior**:
- Returns results even if score is low (no minimum threshold)
- Results sorted by descending score (highest first)
- Empty results array if no matches found
- Tags filter uses AND logic (all tags must match)

### checkpoint_list

**Description**: List recent checkpoints with pagination, filtering by project path, sorting by creation/update time

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "description": "Number of results (default: 10, max: 100)"
    },
    "offset": {
      "type": "integer",
      "description": "Offset for pagination (default: 0)"
    },
    "project_path": {
      "type": "string",
      "description": "Filter by project path"
    },
    "sort_by": {
      "type": "string",
      "description": "Sort field (created_at, updated_at)"
    }
  }
}
```

**Output Schema**:
```json
{
  "checkpoints": [ /* array of checkpoint objects */ ],
  "total": 42,
  "limit": 10,
  "offset": 0
}
```

**Pagination Strategy**:
- Client-side: Iterate through all pages using offset
- Server-side: Fetch buffer beyond requested range
- Total count: Accurate (scans all records)

## Internal Service API

### Constructor

```go
func NewService(
    vectorStore VectorStore,
    embedder EmbeddingGenerator,
    projectPath string,
) (*Service, error)
```

### Core Methods

```go
// Create checkpoint with automatic embedding
Create(ctx context.Context, req *validation.CreateCheckpointRequest) (*Checkpoint, error)

// Semantic search
Search(ctx context.Context, query string, topK int, projectPath string, tags []string) (*SearchResult, error)

// Paginated list
List(ctx context.Context, limit, offset int, projectPath string, sortBy string) (*ListResult, error)

// Get by ID
GetByID(ctx context.Context, id string) (*Checkpoint, error)

// Update (not implemented)
Update(ctx context.Context, id string, fields *UpdateFields) (*Checkpoint, error)

// Delete by ID
Delete(ctx context.Context, id string) error

// Health check
Health(ctx context.Context) error
```

## Data Models

### Checkpoint Domain Model

```go
type Checkpoint struct {
    ID          string            `json:"id"`           // UUID v4
    Summary     string            `json:"summary"`      // Max 500 chars
    Description string            `json:"description"`  // Optional, no limit
    ProjectPath string            `json:"project_path"` // Absolute path
    Context     map[string]string `json:"context"`      // Custom metadata
    Tags        []string          `json:"tags"`         // Categorization
    TokenCount  int               `json:"token_count"`  // Embedding tokens
    CreatedAt   time.Time         `json:"created_at"`   // Creation timestamp
    UpdatedAt   time.Time         `json:"updated_at"`   // Update timestamp
}
```

**Field Constraints**:
- `ID`: Generated UUID v4 (36 chars with hyphens)
- `Summary`: Required, 1-500 characters
- `Description`: Optional, max 10,000 characters
- `ProjectPath`: Required, absolute path, must exist
- `Context`: Optional, max 50 key-value pairs, keys max 50 chars, values max 500 chars
- `Tags`: Optional, max 20 tags, each max 50 characters
- `TokenCount`: Auto-calculated, read-only
- Timestamps: Auto-generated, read-only

## Error Response Format

### Validation Error

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "invalid summary",
    "details": {
      "field": "summary",
      "error": "summary must be 1-500 characters"
    }
  }
}
```

### Internal Error

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "failed to generate embedding",
    "details": {
      "provider": "openai",
      "error": "rate limit exceeded"
    }
  }
}
```

### Timeout Error

```json
{
  "error": {
    "code": "TIMEOUT_ERROR",
    "message": "operation exceeded timeout",
    "details": {
      "timeout": "30s",
      "elapsed": "31.2s"
    }
  }
}
```

### Not Found Error

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "checkpoint not found",
    "details": {
      "id": "550e8400-e29b-41d4-a716-446655440000"
    }
  }
}
```
