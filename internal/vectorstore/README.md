# vectorstore

Interface-based vector storage package for contextd with multi-tenant isolation.

## Overview

The vectorstore package provides an abstraction layer for vector storage operations with multiple provider implementations:

- **ChromemStore** (default) - Embedded chromem-go storage with local embeddings
- **QdrantStore** - External Qdrant service via gRPC

Both providers implement the `Store` interface and support multi-tenant isolation via payload filtering or filesystem isolation.

## Multi-Tenant Isolation

The package provides three isolation modes for multi-tenant deployments:

| Mode | Description | Use Case |
|------|-------------|----------|
| `PayloadIsolation` | Single collection with metadata filtering | **Default, recommended** |
| `FilesystemIsolation` | Separate database per tenant/project | Legacy, migration path available |
| `NoIsolation` | No tenant filtering | **Testing only** |

### PayloadIsolation (Default)

All documents stored in shared collections with automatic tenant filtering:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Configure with PayloadIsolation (default if not specified)
config := vectorstore.ChromemConfig{
    Path:              "/path/to/data",
    DefaultCollection: "memories",
    VectorSize:        384,
    Isolation:         vectorstore.NewPayloadIsolation(), // Optional - this is the default
}

store, err := vectorstore.NewChromemStore(config, embedder, logger)
if err != nil {
    return err
}
defer store.Close()

// REQUIRED: Add tenant context to all operations
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",      // Required
    TeamID:    "platform",     // Optional
    ProjectID: "contextd",     // Optional
})

// Documents automatically tagged with tenant metadata
docs := []vectorstore.Document{{ID: "doc1", Content: "example"}}
ids, err := store.AddDocuments(ctx, docs)

// Searches automatically filtered by tenant
results, err := store.Search(ctx, "query", 10)
```

### Security Model

| Aspect | Behavior |
|--------|----------|
| Missing tenant context | Returns `ErrMissingTenant` (fail-closed) |
| User filters with tenant fields | Rejected with `ErrTenantFilterInUserFilters` |
| Document metadata | Tenant fields always overwritten from context |
| Default isolation | `PayloadIsolation` for production safety |

### Threat Model

The isolation system defends against:

1. **Cross-Tenant Data Access** - Attacker injects `tenant_id` in query filters
   - Defense: `ApplyTenantFilters()` rejects user-provided tenant fields

2. **Metadata Poisoning** - Attacker sets `tenant_id` in document metadata
   - Defense: `InjectMetadata()` always overwrites tenant fields from context

3. **Context Bypass** - Code executes without tenant context
   - Defense: Fail-closed behavior returns error, not empty results

4. **Race Conditions** - Isolation mode changed during operation
   - Defense: Config-based isolation is immutable after construction

## Migration Guide

### From FilesystemIsolation to PayloadIsolation

If you're using `StoreProvider` with database-per-project isolation:

```go
// BEFORE: FilesystemIsolation (database per project)
provider, _ := vectorstore.NewChromemStoreProvider(providerConfig, embedder, logger)
store, _ := provider.GetProjectStore(ctx, "platform", "contextd")
// Each project gets separate database files

// AFTER: PayloadIsolation (shared database, metadata filtering)
config := vectorstore.ChromemConfig{
    Path:              "/path/to/shared/data",
    DefaultCollection: "memories",
    VectorSize:        384,
    Isolation:         vectorstore.NewPayloadIsolation(),
}
store, _ := vectorstore.NewChromemStore(config, embedder, logger)

// Add tenant context to all operations
ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    TeamID:    "platform",
    ProjectID: "contextd",
})
```

**Migration steps:**
1. Export data from per-project databases
2. Create new shared database with PayloadIsolation
3. Import data with tenant metadata added
4. Update all callers to use `ContextWithTenant()`
5. Verify tenant filtering works correctly

**No data format changes** - only metadata is added to documents.

## Provider Selection

```yaml
vectorstore:
  provider: chromem  # "chromem" (default) or "qdrant"
```

| Provider | Use Case | Dependencies |
|----------|----------|--------------|
| Chromem | Local dev, simple setups | None (embedded) |
| Qdrant | Production, high scale | External Qdrant + embedder |

## Collection Naming Convention

Collections follow a hierarchical naming pattern:

| Scope | Pattern | Example |
|-------|---------|---------|
| Organization | `org_{type}` | `org_memories` |
| Team | `{team}_{type}` | `platform_memories` |
| Project | `{team}_{project}_{type}` | `platform_contextd_memories` |

**Note**: With PayloadIsolation, you can use a single collection (e.g., `memories`) and rely on metadata filtering for tenant separation. Collection naming is optional.

## Interface

The `Store` interface provides:

- **Document Operations**: `AddDocuments`, `DeleteDocuments`, `DeleteDocumentsFromCollection`
- **Search Operations**: `Search`, `SearchWithFilters`, `SearchInCollection`, `ExactSearch`
- **Collection Management**: `CreateCollection`, `DeleteCollection`, `CollectionExists`, `ListCollections`, `GetCollectionInfo`
- **Isolation**: `SetIsolationMode` (deprecated), `IsolationMode`
- **Resource Management**: `Close`

## Usage Examples

### ChromemStore with PayloadIsolation

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Configure store with isolation
config := vectorstore.ChromemConfig{
    Path:              "/data/vectorstore",
    DefaultCollection: "memories",
    VectorSize:        384,
    Compress:          true,
    Isolation:         vectorstore.NewPayloadIsolation(),
}

store, err := vectorstore.NewChromemStore(config, embedder, logger)
if err != nil {
    return err
}
defer store.Close()

// Create tenant context (required for all operations)
ctx := vectorstore.ContextWithTenant(context.Background(), &vectorstore.TenantInfo{
    TenantID:  "org-123",
    TeamID:    "platform",
    ProjectID: "contextd",
})

// Add documents (tenant metadata injected automatically)
docs := []vectorstore.Document{
    {
        ID:      "mem-1",
        Content: "User prefers dark mode",
        Metadata: map[string]interface{}{
            "category": "preference",
        },
    },
}
ids, err := store.AddDocuments(ctx, docs)

// Search (tenant filter injected automatically)
results, err := store.Search(ctx, "user preferences", 10)

// Search with additional filters
results, err := store.SearchWithFilters(ctx, "preferences", 10,
    map[string]interface{}{"category": "preference"})
```

### QdrantStore with PayloadIsolation

```go
config := vectorstore.QdrantConfig{
    Host:           "localhost",
    Port:           6334,
    CollectionName: "memories",
    VectorSize:     384,
    UseTLS:         false,
    Isolation:      vectorstore.NewPayloadIsolation(),
}

store, err := vectorstore.NewQdrantStore(config, embedder)
if err != nil {
    return err
}
defer store.Close()

// Same tenant context pattern
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID: "org-123",
})

results, err := store.Search(ctx, "query", 10)
```

### Testing with NoIsolation

```go
// For tests where tenant isolation is not relevant
config := vectorstore.ChromemConfig{
    Path:              t.TempDir(),
    DefaultCollection: "test",
    VectorSize:        384,
    Isolation:         vectorstore.NewNoIsolation(), // WARNING: Testing only!
}

store, _ := vectorstore.NewChromemStore(config, embedder, nil)
// No tenant context required
results, _ := store.Search(context.Background(), "query", 10)
```

## Configuration

### ChromemConfig

```go
type ChromemConfig struct {
    Path              string         // Database directory path
    Compress          bool           // Enable compression
    DefaultCollection string         // Default collection name
    VectorSize        int            // Embedding dimensions (e.g., 384)
    Isolation         IsolationMode  // PayloadIsolation (default), FilesystemIsolation, or NoIsolation
}
```

### QdrantConfig

```go
type QdrantConfig struct {
    Host                    string         // Qdrant server hostname
    Port                    int            // gRPC port (default: 6334)
    CollectionName          string         // Default collection
    VectorSize              uint64         // Embedding dimensions
    Distance                qdrant.Distance // Similarity metric (default: Cosine)
    UseTLS                  bool           // Enable TLS
    MaxRetries              int            // Retry attempts (default: 3)
    RetryBackoff            time.Duration  // Initial backoff (default: 1s)
    MaxMessageSize          int            // gRPC message size (default: 50MB)
    CircuitBreakerThreshold int            // Failures before open (default: 5)
    Isolation               IsolationMode  // PayloadIsolation (default), etc.
}
```

## Security

### Input Validation

- **Collection names**: `^[a-z0-9_]{1,64}$` (prevents path traversal)
- **Query length**: Max 10,000 characters (QdrantStore)
- **Result limits**: Capped at collection size or 10,000

### Tenant Isolation

- **Fail-closed**: Missing context returns error, never empty results
- **Filter injection blocked**: User cannot provide tenant_id/team_id/project_id
- **Metadata enforced**: Tenant fields always set from authenticated context

## Testing

```bash
# Unit tests only
go test ./internal/vectorstore/... -short

# Integration tests (requires Qdrant on localhost:6334)
go test ./internal/vectorstore/...
```

## Dependencies

### ChromemStore
- `github.com/philippgille/chromem-go` - Embedded vector store
- `go.opentelemetry.io/otel` - Observability

### QdrantStore
- `github.com/qdrant/go-client` - Official Qdrant gRPC client
- `github.com/google/uuid` - UUID generation
- `go.opentelemetry.io/otel` - Observability
