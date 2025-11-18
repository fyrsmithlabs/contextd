# Remediation Architecture

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the system design and component interactions for error solution storage and search.

---

## System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    MCP Tools Layer                          │
│  remediation_save, remediation_search                       │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────┴───────────────────────────────────────┐
│                  Remediation Service                        │
│  - Create remediation                                       │
│  - Find similar errors                                      │
│  - Signature generation                                     │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
┌───────┴──────┐ ┌───┴──────┐ ┌───┴────────────┐
│   Matcher    │ │ Embedder │ │ VectorStore    │
│  - Hybrid    │ │ - OpenAI │ │ - Qdrant       │
│  - Fuzzy     │ │          │ │ - Multi-tenant │
│    matching  │ │          │ │                │
└──────────────┘ └──────────┘ └────────────────┘
```

---

## Data Flow

### Save Flow

```
1. Receive error + solution
2. Validate inputs (required fields, size limits)
3. Generate error signature:
   - Normalize error message
   - Extract error type
   - Extract stack signature
   - Generate SHA256 hash
4. Generate embedding for semantic search
5. Store in shared database with signature + embedding
6. Return remediation ID
```

### Search Flow

```
1. Receive error message + optional stack trace
2. Generate query signature (normalize, extract features)
3. Generate embedding for semantic search
4. Search vector store for top 2N candidates (semantic)
5. Calculate string similarity for each candidate
6. Compute hybrid score (0.7*semantic + 0.3*string)
7. Apply boost factors:
   - Error type match: +10%
   - Stack trace match: +15%
8. Filter by minimum thresholds
9. Re-rank by final hybrid score
10. Return top N results with match details
```

---

## Data Models

### Remediation

Complete remediation record with error and solution.

```go
type Remediation struct {
    // Core fields
    ID           string    `json:"id"`                    // UUID
    ErrorMessage string    `json:"error_message"`          // Original error
    ErrorType    string    `json:"error_type"`             // Error class
    Solution     string    `json:"solution"`               // Fix description

    // Context
    ProjectPath  string            `json:"project_path,omitempty"`   // Optional project
    Context      map[string]string `json:"context,omitempty"`        // Additional metadata
    Tags         []string          `json:"tags"`                      // Categorization
    Severity     string            `json:"severity,omitempty"`       // low, medium, high, critical

    // Debugging
    StackTrace   string           `json:"stack_trace,omitempty"`    // Full stack trace

    // Generated
    Timestamp    int64            `json:"timestamp"`                 // Unix timestamp
    Signature    ErrorSignature   `json:"signature,omitempty"`      // Generated signature
}
```

### ErrorSignature

Normalized error signature for matching.

```go
type ErrorSignature struct {
    NormalizedError string  // Error with variables removed
    ErrorType       string  // Extracted error type/class
    StackSignature  string  // Normalized stack trace signature
    Hash            string  // SHA256 hash of above fields
}
```

### MatchResult

Detailed match result with scores.

```go
type MatchResult struct {
    ID              string   // Remediation ID

    // Scores
    SemanticScore   float64  // Vector similarity (0.0-1.0)
    StringScore     float64  // Levenshtein similarity (0.0-1.0)
    HybridScore     float64  // Weighted combination (0.0-1.0)

    // Match details
    StackTraceMatch bool     // Stack traces match
    ErrorTypeMatch  bool     // Error types match
}
```

### SimilarError

Search result with remediation and match details.

```go
type SimilarError struct {
    Remediation  Remediation  // Full remediation record
    MatchScore   float64      // Final hybrid score (0.0-1.0)
    MatchDetails MatchResult  // Detailed match breakdown
}
```

---

## Vector Embedding Strategy

### Embedding Generation

**Model Selection**:
- **TEI (Recommended)**: BAAI/bge-large-en-v1.5
  - Dimension: 1024
  - Local deployment (no API costs)
  - No rate limits
- **OpenAI**: text-embedding-3-small
  - Dimension: 1536
  - API-based ($0.02/1M tokens)
  - Subject to rate limits

**Embedding Input Format**:
```
"{error_type}: {error_message}"

Example: "ImportError: No module named 'requests'"
```

**Why This Format?**
- Error type provides strong semantic signal
- Error message contains detailed context
- Matches how developers think about errors

### Vector Storage

**Database**: Shared database (global knowledge)
- Collection: `remediations`
- Multi-tenant mode: Always enabled (v2.0+)
- Physical isolation: Project data separate from remediation data

**Vector Schema**:
```
{
  "id": "uuid",
  "vector": [0.123, -0.456, ...],  // 1536 dimensions
  "payload": {
    "error_message": "...",
    "error_type": "...",
    "solution": "...",
    "project": "...",
    "tags": ["..."],
    "severity": "...",
    "stack_trace": "...",
    "timestamp": 1699012800,
    "signature": "sha256hash"
  }
}
```

### Indexing

**Index Configuration**:
- Algorithm: HNSW (Hierarchical Navigable Small World)
- Metric: Cosine similarity
- M: 16 (number of connections per layer)
- ef_construct: 100 (construction time accuracy)

**Performance**:
- Insert latency: ~10-50ms
- Search latency: ~5-20ms (P95)
- Throughput: ~1000 inserts/sec, ~5000 searches/sec

---

## Service Interface

```go
type ServiceInterface interface {
    // Create creates a new remediation with embedding
    Create(ctx context.Context, req *CreateRemediationRequest) (*Remediation, error)

    // FindSimilarErrors finds remediations for similar errors using hybrid matching
    FindSimilarErrors(ctx context.Context, req *SearchRequest) ([]SimilarError, error)

    // Get retrieves a remediation by ID
    Get(ctx context.Context, id string) (*Remediation, error)

    // List retrieves all remediations with pagination
    List(ctx context.Context, limit, offset int) ([]Remediation, error)

    // Delete removes a remediation by ID
    Delete(ctx context.Context, id string) error

    // Update updates a remediation
    Update(ctx context.Context, id string, req *CreateRemediationRequest) (*Remediation, error)
}
```

---

## Performance Characteristics

### Latency

**Operation Latencies (P50/P95/P99)**:
- `Create`: 50ms / 150ms / 300ms
  - Embedding generation: ~30ms
  - Vector insert: ~20ms
  - Signature generation: ~5ms

- `FindSimilarErrors`: 100ms / 250ms / 500ms
  - Embedding generation: ~30ms
  - Vector search (2N candidates): ~50ms
  - Hybrid matching (N results): ~20ms
  - Re-ranking: ~5ms

### Throughput

**Expected Load**:
- Creates: ~10/min (low volume, ad-hoc)
- Searches: ~100/min (developer workflows)

**System Capacity**:
- Creates: ~200/min (20x headroom)
- Searches: ~600/min (6x headroom)

### Scalability

**Collection Size**:
- Current: ~1,000 remediations
- Target: ~100,000 remediations (100x growth)
- Vector search: Sub-linear complexity (HNSW)

**Resource Usage**:
- Memory: ~50MB per 10,000 vectors
- Disk: ~200MB per 10,000 vectors (compressed)
- CPU: Minimal (HNSW is cache-efficient)

---

## Thread Safety

**Service Methods**:
- All methods are thread-safe
- Context-aware (respects cancellation)
- No shared mutable state

**Matcher**:
- Stateless (configuration only)
- Safe for concurrent use
- No synchronization required

---

## Dependencies

**Required**:
- `github.com/google/uuid` - UUID generation
- `github.com/lithammer/fuzzysearch/fuzzy` - Fuzzy string matching
- `go.opentelemetry.io/otel` - Observability

**Interfaces**:
- `VectorStore` - UniversalVectorStore interface
- `EmbeddingGenerator` - Embedding service interface
