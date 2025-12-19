# Package: vectorstore

**Parent**: See [../../CLAUDE.md](../../CLAUDE.md) for project overview and package guidelines.

## Purpose

Interface-based vector storage package with multi-tenant isolation. Provides abstraction over chromem (embedded) and Qdrant (external) backends.

## Architecture

**Design Pattern**: Interface-based abstraction with configurable isolation modes

**Key Interfaces**:
- `Store` - Vector storage operations (add, search, delete)
- `IsolationMode` - Tenant isolation strategy

**Implementations**:
- `ChromemStore` - Embedded chromem-go storage (default)
- `QdrantStore` - External Qdrant via gRPC

## Multi-Tenant Isolation

**Default: PayloadIsolation** - All documents in shared collections, filtered by tenant metadata.

### Tenant Context (Required)

```go
// Set tenant context before any operation
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",      // Required
    TeamID:    "platform",     // Optional
    ProjectID: "contextd",     // Optional
})

// Operations automatically filtered
results, err := store.Search(ctx, "query", 10)
```

### Isolation Modes

| Mode | Strategy | File |
|------|----------|------|
| `PayloadIsolation` | Metadata filtering (default) | `isolation.go` |
| `FilesystemIsolation` | Separate database per tenant | `isolation.go` |
| `NoIsolation` | No filtering (testing only) | `isolation.go` |

### Security Guarantees

| Behavior | Implementation |
|----------|----------------|
| Fail-closed | `TenantFromContext()` returns `ErrMissingTenant` |
| Filter injection blocked | `ApplyTenantFilters()` rejects user tenant fields |
| Metadata enforced | `InjectMetadata()` overwrites tenant fields from context |

## Key Files

| File | Purpose |
|------|---------|
| `interface.go` | `Store` interface definition |
| `chromem.go` | chromem implementation |
| `qdrant.go` | Qdrant implementation |
| `isolation.go` | `IsolationMode` implementations |
| `tenant.go` | `TenantInfo`, context helpers |
| `filter.go` | `ApplyTenantFilters()`, filter builders |
| `provider.go` | `StoreProvider` (legacy) |
| `factory.go` | Provider factory |

## Configuration

```go
// ChromemStore with PayloadIsolation (default)
config := vectorstore.ChromemConfig{
    Path:              "/data/vectorstore",
    DefaultCollection: "memories",
    VectorSize:        384,
    // Isolation defaults to PayloadIsolation
}

// Explicit isolation mode
config.Isolation = vectorstore.NewPayloadIsolation()    // Default
config.Isolation = vectorstore.NewFilesystemIsolation() // Legacy
config.Isolation = vectorstore.NewNoIsolation()         // Testing only!
```

## Usage Example

```go
// Create store
store, err := vectorstore.NewChromemStore(config, embedder, logger)
if err != nil {
    return err
}
defer store.Close()

// Set tenant context (REQUIRED)
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    ProjectID: "my-project",
})

// Add documents (tenant metadata injected automatically)
docs := []vectorstore.Document{{ID: "1", Content: "example"}}
ids, err := store.AddDocuments(ctx, docs)

// Search (tenant filter injected automatically)
results, err := store.Search(ctx, "query", 10)
```

## Testing

```bash
# Unit tests
go test ./internal/vectorstore/... -short

# Integration tests (requires Qdrant)
go test ./internal/vectorstore/...

# Coverage
go test ./internal/vectorstore/... -cover
```

## Security Considerations

- **Tenant context required** - All operations fail without tenant
- **Filter injection blocked** - User cannot provide `tenant_id`/`team_id`/`project_id`
- **Metadata enforced** - Tenant fields always from context, never user input
- **Collection name validation** - `^[a-z0-9_]{1,64}$` pattern

## See Also

- README: `internal/vectorstore/README.md`
- Security spec: `docs/spec/vector-storage/security.md`
- Migration guide: `docs/migration/payload-filtering.md`
