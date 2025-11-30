# vectorstore

Interface-based vector storage package for contextd-v2.

## Overview

The vectorstore package provides an abstraction layer for vector storage operations, with a Qdrant gRPC implementation. It supports the hierarchical collection architecture defined in the contextd specification.

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

### QdrantStore

Qdrant gRPC client implementation with:

- **Native gRPC transport** (port 6334) - bypasses HTTP payload limits
- **Binary protobuf encoding** - no JSON size limits
- **Retry logic** - exponential backoff for transient failures
- **Circuit breaker** - protects against cascading failures
- **Collection caching** - reduces existence checks
- **Security validation** - collection name validation prevents path traversal

## Usage

```go
import "github.com/fyrsmithlabs/contextd-v2/internal/vectorstore"

// Configure Qdrant connection
config := vectorstore.QdrantConfig{
    Host:           "localhost",
    Port:           6334,
    CollectionName: "platform_contextd_memories",
    VectorSize:     384, // Match embedder output
    UseTLS:         false,
}

// Create store with embedder
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

- `github.com/qdrant/go-client` - Official Qdrant gRPC client
- `github.com/google/uuid` - UUID generation
- `go.opentelemetry.io/otel` - Observability instrumentation

## Architecture Notes

This package implements the collection architecture spec:
- Database-per-organization (future)
- Collection-per-scope (org/team/project)
- Physical isolation for multi-tenancy

See `/home/dahendel/projects/contextd-reasoning/docs/spec/collection-architecture/SPEC.md` for full details.
