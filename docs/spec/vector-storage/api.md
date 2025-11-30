# Vector Storage API

## VectorDB Client Interface

```go
package vectordb

// Client provides vector database operations.
// Thread-safe. Tenant context MUST be set via WithTenant.
type Client interface {
    // Collection operations
    CreateCollection(ctx context.Context, req *CreateCollectionRequest) error
    DeleteCollection(ctx context.Context, name string) error
    CollectionExists(ctx context.Context, name string) (bool, error)

    // Point operations
    Upsert(ctx context.Context, req *UpsertRequest) error
    Get(ctx context.Context, req *GetRequest) ([]*Point, error)
    Delete(ctx context.Context, req *DeleteRequest) error

    // Search
    Search(ctx context.Context, req *SearchRequest) ([]*ScoredPoint, error)

    // Lifecycle
    Close() error
    HealthCheck(ctx context.Context) error
}
```

## Tenant Context

```go
// Tenant represents multi-tenant context.
type Tenant struct {
    OrgID     string
    TeamID    string
    ProjectID string
}

func (t *Tenant) Database() string { return t.OrgID }

func WithTenant(ctx context.Context, tenant *Tenant) context.Context
func TenantFromContext(ctx context.Context) *Tenant
```

## Search with Document Inference

```go
// SearchRequest uses Qdrant's built-in inference.
type SearchRequest struct {
    Collection     string
    Document       *Document  // Qdrant embeds this
    Limit          int
    Filter         *Filter
    ScoreThreshold float64
    WithPayload    bool
}

// Document for Qdrant inference.
type Document struct {
    Text  string  // Text to embed
    Model string  // e.g., "qdrant/bm25" or "sentence-transformers/all-minilm-l6-v2"
}
```

## Upsert with Document Inference

```go
// UpsertRequest with Document inference.
type UpsertRequest struct {
    Collection string
    Points     []*DocumentPoint
    Wait       bool
}

// DocumentPoint uses Qdrant inference for embedding.
type DocumentPoint struct {
    ID       string
    Document *Document
    Payload  map[string]any
}
```

## Code Indexer Interface

```go
package codeindex

// Indexer manages codebase indexing for a worktree.
type Indexer interface {
    // Index indexes changed files since last indexed commit.
    Index(ctx context.Context) (*IndexResult, error)
    
    // IndexAll forces full re-index.
    IndexAll(ctx context.Context) (*IndexResult, error)
    
    // NeedsReindex checks if index is stale.
    NeedsReindex(ctx context.Context) (bool, error)
    
    // Watch starts background watcher (ref changes + polling).
    Watch(ctx context.Context) error
    
    // Stop stops background watcher.
    Stop() error
}

// IndexResult contains statistics.
type IndexResult struct {
    FilesProcessed int
    UnitsExtracted int
    PointsUpserted int
    Duration       time.Duration
    LastIndexedSHA string
}
```

## Semantic Unit Types

```go
// SemanticUnit represents an extracted code unit.
type SemanticUnit struct {
    Name       string
    Signature  string
    UnitType   UnitType  // function, method, type, const
    Docstring  string
    Content    string    // Raw code
    FilePath   string
    LineStart  int
    LineEnd    int
    Context    UnitContext
    TokenCount int       // For BM25 fallback decision
}

type UnitType string

const (
    UnitFunction UnitType = "function"
    UnitMethod   UnitType = "method"
    UnitType     UnitType = "type"
    UnitConst    UnitType = "const"
)

// UnitContext provides surrounding context.
type UnitContext struct {
    Module     string  // Package name
    FileName   string
    StructName string  // For methods
}
```
