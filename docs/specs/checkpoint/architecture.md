# Checkpoint Architecture

**Parent**: [../SPEC.md](../SPEC.md)

## Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     MCP Tools Layer                          │
│  checkpoint_save | checkpoint_search | checkpoint_list       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                  Checkpoint Service                          │
│  • Orchestration logic                                       │
│  • Embedding generation                                      │
│  • Database scoping                                          │
│  • OpenTelemetry instrumentation                            │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
┌──────────────────┐      ┌──────────────────┐
│  Vector Store    │      │  Embedding       │
│                  │      │  (OpenAI/TEI)    │
│  • Insert        │      │  • Generate      │
│  • Search        │      │  • Cache         │
│  • Get/Delete    │      │  • Cost tracking │
└──────────────────┘      └──────────────────┘
```

## Multi-Tenant Architecture

### Database Isolation Model (v2.0.0+)

```
├── project_abc123de (SHA256 hash of /home/user/project1)
│   └── checkpoints collection
│       ├── checkpoint_1
│       ├── checkpoint_2
│       └── ...
│
├── project_def456gh (SHA256 hash of /home/user/project2)
│   └── checkpoints collection
│       ├── checkpoint_1
│       └── ...
│
└── shared (global knowledge)
    ├── skills collection
    ├── remediations collection
    └── troubleshooting collection
```

**Benefits**:
- **Security**: Physical database boundary eliminates filter injection attacks
- **Performance**: 10-16x faster queries (no filter overhead)
- **Scalability**: Partition pruning at database level
- **Portability**: Easy project migration (copy database)
- **Compliance**: Complete data isolation for multi-tenant SaaS

**Database Naming**:
- Format: `project_<hash>` where hash = SHA256(project_path)[:16]
- Example: `/home/user/myproject` → `project_770a5f097cd8`
- Deterministic: Same project path always generates same database name
- Case-sensitive: Preserves original path casing in hash

## Service Architecture

### Layered Design

**Layer 1: Interface Layer** (`interfaces.go`)
- `VectorStore` - Universal vector database interface
- `EmbeddingGenerator` - Text embedding interface

**Layer 2: Service Layer** (`service.go`)
- Orchestrates between interfaces
- Handles database scoping (multi-tenant)
- Manages OpenTelemetry instrumentation
- Coordinates embedding generation + vector storage

**Layer 3: Domain Layer** (`models.go`)
- `Checkpoint` - Core domain entity
- `CheckpointSearchResult` - Search result with score
- `ListResult` - Paginated list response
- `SearchResult` - Semantic search response

**Layer 4: Package API** (`checkpoint.go`)
- Package-level documentation
- Usage examples
- Thread safety guarantees

## Vector Embedding Strategy

### Embedding Providers

**1. OpenAI API**
- Model: `text-embedding-3-small`
- Dimension: 1536
- Cost: $0.02 per 1M tokens
- Rate limit: 3,000 requests/minute (tier 1)
- Latency: 50-150ms (US region)

**2. Text Embeddings Inference (TEI)**
- Model: `BAAI/bge-small-en-v1.5`
- Dimension: 384
- Cost: Free (local Docker)
- Rate limit: Hardware dependent
- Latency: 10-50ms (local)

### Content Preparation

```
Embedding Input = summary + "\n\n" + description
```

**Rationale**:
- Summary contains key concepts (high weight)
- Description provides context (moderate weight)
- Separation with `\n\n` helps model distinguish sections
- Context JSON not included in embedding (too noisy)

### Caching Strategy

**Cache Implementation**:
- In-memory LRU cache (15-minute TTL)
- Key: SHA256 hash of input text
- Value: Embedding vector + metadata
- Eviction: LRU when cache full or TTL expired

**Cache Hit Ratio**:
- Typical: 30-40% (duplicate summaries common)
- Testing: 70-80% (repeated test fixtures)
- Production: Varies by workflow patterns

**Benefits**:
- Reduces API costs (skip duplicate calls)
- Improves latency (instant cache hits)
- Reduces rate limit pressure

### Token Counting

**Algorithm**:
```go
func CountTokens(text string) int {
    words := len(strings.Fields(text))
    return int(float64(words) * 0.75)
}
```

**Accuracy**:
- Simple approximation (not tiktoken)
- Typically within 10-20% of actual tokens
- Good enough for cost estimation
- Use actual token count from embedding API for billing

## Search Algorithm

### Vector Similarity

**Distance Metric**: Cosine Similarity
```
similarity = 1 - cosine_distance
           = dot(A, B) / (||A|| * ||B||)
```

**Score Range**: 0.0 - 1.0
- 1.0: Identical vectors (perfect match)
- 0.9-1.0: Very similar (highly relevant)
- 0.7-0.9: Somewhat similar (relevant)
- 0.5-0.7: Weakly similar (may be relevant)
- 0.0-0.5: Dissimilar (not relevant)

**No Threshold**: Returns topK results regardless of score

### Tag Filtering

**Filter Expression**:
```
tags like "%tag1%" && tags like "%tag2%"
```

**Logic**: AND (all tags must match)
**Case Sensitivity**: Case-insensitive (depends on database)
**Partial Match**: Yes (substring match)

**Filter Order**:
1. Apply tag filter (pre-filter vectors)
2. Perform vector search on filtered set
3. Rank by similarity score

### Result Ranking

**Primary Sort**: Descending similarity score
**Secondary Sort**: None (arbitrary order for ties)
**Limit**: Return exactly topK results (or fewer if not enough matches)

**Example**:
```
Query: "JWT authentication"
Results:
  1. "Implemented JWT authentication" (score: 0.95)
  2. "Added token-based auth" (score: 0.87)
  3. "Fixed authentication bug" (score: 0.73)
  4. "User login flow" (score: 0.65)
  5. "Session management" (score: 0.58)
```

## Storage Schema

### Vector Storage

**Collection Name**: `checkpoints`
**Vector Dimension**: 1536 (OpenAI) or 384 (TEI)
**Distance Metric**: Cosine similarity

**Fields**:
```json
{
  "id": "string (primary key)",
  "embedding": "float32[] (vector field)",
  "summary": "string",
  "content": "string (summary + description + context JSON)",
  "project": "string (absolute path)",
  "timestamp": "int64 (Unix timestamp)",
  "token_count": "int64",
  "tags": "string (comma-separated)"
}
```

**Indexes**:
- Primary: `id` (unique)
- Vector: HNSW index on `embedding` field
- Metadata: No secondary indexes (full scan for filters)

## Observability

### OpenTelemetry Integration

**Spans**:
- All operations instrumented
- Error details in span status
- Performance metrics attached as attributes

**Metrics**:
- `checkpoint.errors.total` - Counter by error type
- `checkpoint.error_rate` - Error rate per operation
- `checkpoint.timeout_count` - Timeout occurrences
- `checkpoint.latency` - Latency histogram

**Logging**:
- Error-level: All internal errors
- Warn-level: Validation errors, timeouts
- Info-level: Successful operations
- Debug-level: Detailed execution traces
