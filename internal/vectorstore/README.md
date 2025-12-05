# vectorstore

Interface-based vector storage package for contextd.

## Overview

The vectorstore package provides an abstraction layer for vector storage operations with multiple provider implementations:

- **ChromaStore** (default) - Embedded SQLite-based storage with built-in embeddings
- **QdrantStore** - External Qdrant service via gRPC

Both providers implement the `Store` interface and support the hierarchical collection architecture defined in the contextd specification.

## Provider Selection

```yaml
vectorstore:
  provider: chroma  # "chroma" (default) or "qdrant"
```

| Provider | Use Case | Dependencies |
|----------|----------|--------------|
| Chroma | Local dev, simple setups | None (embedded) |
| Qdrant | Production, high scale | External Qdrant + ONNX |

## Collection Naming Convention

Collections follow a hierarchical naming pattern:

| Scope | Pattern | Example |
|-------|---------|---------|
| Organization | `org_{type}` | `org_memories` |
| Team | `{team}_{type}` | `platform_memories` |
| Project | `{team}_{project}_{type}` | `platform_contextd_memories` |

## Interface

The `Store` interface provides:

- **Document Operations**: `AddDocuments`, `DeleteDocuments`
- **Search Operations**: `Search`, `SearchWithFilters`, `SearchInCollection`, `ExactSearch`
- **Collection Management**: `CreateCollection`, `DeleteCollection`, `CollectionExists`, `ListCollections`, `GetCollectionInfo`
- **Resource Management**: `Close`

## Implementation

### ChromaStore (Default)

Embedded Chroma implementation with:

- **SQLite-based persistence** - stores at `~/.config/contextd/chroma.db`
- **Built-in embeddings** - sentence-transformers models (no ONNX needed)
- **768d default** - `all-mpnet-base-v2` for balanced performance
- **Model validation** - strict dimension/model compatibility checks
- **Telemetry** - search latency, query count, index size metrics
- **Zero external deps** - works out of the box

### QdrantStore

Qdrant gRPC client implementation with:

- **Native gRPC transport** (port 6334) - bypasses HTTP payload limits
- **Binary protobuf encoding** - no JSON size limits
- **Retry logic** - exponential backoff for transient failures
- **Circuit breaker** - protects against cascading failures
- **Collection caching** - reduces existence checks
- **Security validation** - collection name validation prevents path traversal

## Usage

### ChromaStore (Default - Zero Config)

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Configure Chroma (embedded)
config := vectorstore.ChromaConfig{
    Path:      "~/.config/contextd/chroma.db",
    Model:     "sentence-transformers/all-mpnet-base-v2",
    Dimension: 768,
    Distance:  "cosine",
}

// Create store (no external dependencies)
store, err := vectorstore.NewChromaStore(config, logger)
if err != nil {
    // Handle error
}
defer store.Close()

// Add documents
docs := []vectorstore.Document{
    {
        ID:      "doc1",
        Content: "example content",
        Metadata: map[string]interface{}{
            "owner": "alice",
            "project": "contextd",
        },
    },
}
ids, err := store.AddDocuments(ctx, docs)

// Search
results, err := store.Search(ctx, "query text", 10)
```

### QdrantStore (External Service)

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Configure Qdrant connection
config := vectorstore.QdrantConfig{
    Host:           "localhost",
    Port:           6334,
    CollectionName: "platform_contextd_memories",
    VectorSize:     384, // Match embedder output
    UseTLS:         false,
}

// Create store with embedder (requires ONNX runtime)
store, err := vectorstore.NewQdrantStore(config, embedder)
if err != nil {
    // Handle error
}
defer store.Close()

// Add documents
docs := []vectorstore.Document{
    {
        ID:      "doc1",
        Content: "example content",
        Metadata: map[string]interface{}{
            "owner": "alice",
            "project": "contextd",
        },
        Collection: "platform_contextd_codebase", // Optional: override default
    },
}
ids, err := store.AddDocuments(ctx, docs)

// Search in specific collection
results, err := store.SearchInCollection(ctx,
    "platform_contextd_memories",
    "query text",
    10,
    map[string]interface{}{"owner": "alice"})
```

### Provider Factory (Recommended)

```go
// Use factory for config-driven provider selection
store, err := vectorstore.NewStore(cfg.VectorStore, logger)
if err != nil {
    // Handle error
}
defer store.Close()
```

## Configuration

### Environment Variables

Not currently implemented - use direct config structs.

### Defaults

- `MaxRetries`: 3
- `RetryBackoff`: 1 second (exponential)
- `MaxMessageSize`: 50MB
- `CircuitBreakerThreshold`: 5 failures
- `Distance`: Cosine

## Security

- **Collection name validation**: `^[a-z0-9_]{1,64}$`
- **Query size limits**: Max 10,000 characters
- **Result limits**: Capped at 10,000 (k parameter)
- **Circuit breaker**: Prevents cascading failures

## Testing

```bash
# Unit tests only
go test ./internal/vectorstore/... -short

# Integration tests (requires Qdrant on localhost:6334)
go test ./internal/vectorstore/...
```

## Dependencies

### ChromaStore
- `github.com/amikos-tech/chroma-go` - Chroma Go client
- `go.opentelemetry.io/otel` - Observability instrumentation

### QdrantStore
- `github.com/qdrant/go-client` - Official Qdrant gRPC client
- `github.com/google/uuid` - UUID generation
- `go.opentelemetry.io/otel` - Observability instrumentation

## Architecture Notes

This package implements the collection architecture spec:
- Database-per-organization (future)
- Collection-per-scope (org/team/project)
- Physical isolation for multi-tenancy

See `/home/dahendel/projects/contextd-reasoning/docs/spec/collection-architecture/SPEC.md` for full details.
