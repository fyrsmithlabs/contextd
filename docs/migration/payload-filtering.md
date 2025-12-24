# Migration Guide: Payload Filtering

This guide covers migrating from `FilesystemIsolation` (database-per-project) to `PayloadIsolation` (shared collection with metadata filtering).

## Overview

**PayloadIsolation** is the new default multi-tenant isolation strategy in contextd. It provides:

- **Simplified operations** - Single database, no per-project management
- **Better performance** - Shared indexes, reduced memory footprint
- **Same security guarantees** - Fail-closed behavior, injection protection

## Breaking Changes

**PR #47 is backward compatible.** No breaking changes for existing deployments.

| Behavior | Before | After |
|----------|--------|-------|
| Default isolation | `FilesystemIsolation` | `PayloadIsolation` |
| Tenant context | Optional (via StoreProvider) | Required (via ContextWithTenant) |
| Missing tenant | Empty results | `ErrMissingTenant` error |
| User tenant filters | Allowed | Rejected with `ErrTenantFilterInUserFilters` |

## Migration Steps

### Step 1: Update Configuration

**Before (FilesystemIsolation):**

```go
// Legacy: Using StoreProvider with database-per-project
provider, _ := vectorstore.NewChromemStoreProvider(providerConfig, embedder, logger)
store, _ := provider.GetProjectStore(ctx, "platform", "contextd")
// Each project gets separate database files at:
// /data/vectorstore/platform/contextd/
```

**After (PayloadIsolation):**

```go
// New: Single store with payload filtering
config := vectorstore.ChromemConfig{
    Path:              "/data/vectorstore",
    DefaultCollection: "memories",
    VectorSize:        384,
    // Isolation defaults to PayloadIsolation
}
store, _ := vectorstore.NewChromemStore(config, embedder, logger)
```

### Step 2: Add Tenant Context

All operations now require tenant context:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Create tenant-scoped context BEFORE any operation
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",      // Required: Organization identifier
    TeamID:    "platform",     // Optional: Team scope
    ProjectID: "contextd",     // Optional: Project scope
})

// Operations automatically filtered by tenant
docs := []vectorstore.Document{{ID: "doc1", Content: "example"}}
ids, err := store.AddDocuments(ctx, docs)  // Tenant metadata injected
results, err := store.Search(ctx, "query", 10)  // Tenant filter injected
```

### Step 3: Handle New Error Types

The fail-closed model introduces new error conditions:

```go
import "errors"

results, err := store.Search(ctx, "query", 10)
if err != nil {
    switch {
    case errors.Is(err, vectorstore.ErrMissingTenant):
        // Tenant context was not set - programming error
        log.Error("missing tenant context")
        return nil, err
    case errors.Is(err, vectorstore.ErrTenantFilterInUserFilters):
        // Caller tried to inject tenant fields in filters
        log.Warn("tenant filter injection attempt blocked")
        return nil, err
    default:
        return nil, err
    }
}
```

### Step 4: Data Migration (Optional)

If you need to migrate existing data from per-project databases to a shared database:

```go
// Export from legacy per-project database
legacyProvider, _ := vectorstore.NewChromemStoreProvider(legacyConfig, embedder, logger)
legacyStore, _ := legacyProvider.GetProjectStore(ctx, "platform", "contextd")

// Query all documents (using FilesystemIsolation - no tenant filter needed)
docs, err := legacyStore.Search(ctx, "*", 10000)  // Get all documents

// Import to new shared database with tenant metadata
newConfig := vectorstore.ChromemConfig{
    Path:              "/data/vectorstore-new",
    DefaultCollection: "memories",
    VectorSize:        384,
    // PayloadIsolation is default
}
newStore, _ := vectorstore.NewChromemStore(newConfig, embedder, logger)

// Set tenant context for import
ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    TeamID:    "platform",
    ProjectID: "contextd",
})

// Documents will have tenant metadata injected automatically
_, err = newStore.AddDocuments(ctx, docs)
```

**Note:** No data format changes are required. The only difference is that tenant metadata is added to documents.

## Explicit Isolation Mode

To keep using `FilesystemIsolation` explicitly:

```go
config := vectorstore.ChromemConfig{
    Path:              "/data/vectorstore/{tenant_id}",
    DefaultCollection: "memories",
    VectorSize:        384,
    Isolation:         vectorstore.NewFilesystemIsolation(),  // Explicit legacy mode
}
```

## Testing Migration

Verify the migration with these checks:

```go
func TestPayloadIsolationMigration(t *testing.T) {
    // 1. Verify tenant context is required
    ctx := context.Background()  // No tenant
    _, err := store.Search(ctx, "query", 10)
    require.ErrorIs(t, err, vectorstore.ErrMissingTenant)

    // 2. Verify tenant filtering works
    ctx1 := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{TenantID: "org-a"})
    ctx2 := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{TenantID: "org-b"})

    // Add document to org-a
    _, err = store.AddDocuments(ctx1, []vectorstore.Document{{ID: "1", Content: "secret"}})
    require.NoError(t, err)

    // Search from org-b should NOT find org-a data
    results, err := store.Search(ctx2, "secret", 10)
    require.NoError(t, err)
    require.Empty(t, results, "cross-tenant data leakage detected")

    // 3. Verify filter injection is blocked
    _, err = store.SearchWithFilters(ctx1, "query", 10, map[string]interface{}{
        "tenant_id": "org-b",  // Attempting to query another tenant
    })
    require.ErrorIs(t, err, vectorstore.ErrTenantFilterInUserFilters)
}
```

## Rollback

To rollback to `FilesystemIsolation`:

1. Set `Isolation: vectorstore.NewFilesystemIsolation()` in config
2. Data in the shared database will need to be exported per-tenant
3. Remove tenant context requirements from callers

## FAQ

### Q: Will my existing data work with PayloadIsolation?

Yes, but existing data won't have tenant metadata. You'll need to:
1. Export data
2. Re-import with tenant context set

### Q: Can I mix isolation modes?

No. Each store instance has one isolation mode. You can run multiple stores with different modes, but this adds complexity.

### Q: What happens to per-project databases?

They remain functional with `FilesystemIsolation`. You can migrate incrementally.

### Q: Is PayloadIsolation as secure as FilesystemIsolation?

Yes. Both provide the same security guarantees:
- Tenant isolation enforced on all operations
- Fail-closed behavior for missing tenant
- Defense-in-depth against injection attacks

PayloadIsolation uses query filters instead of filesystem separation, but the security model is equivalent.

## See Also

- Security spec: `docs/spec/vector-storage/security.md`
- Vectorstore README: `internal/vectorstore/README.md`
- Architecture: `docs/architecture.md`
