# Vector Storage Security

## Multi-Tenant Isolation

contextd supports two isolation strategies. **PayloadIsolation** is the default and recommended approach.

### Isolation Modes

| Mode | Strategy | Recommendation |
|------|----------|----------------|
| `PayloadIsolation` | Shared collection with metadata filtering | **Default, recommended** |
| `FilesystemIsolation` | Separate database per tenant/project | Legacy, migration path available |
| `NoIsolation` | No tenant filtering | **Testing only** |

### Security Guarantees

| Concern | Mitigation |
|---------|------------|
| Cross-tenant access | Tenant filters injected on all queries via `ApplyTenantFilters()` |
| Filter injection | User-provided `tenant_id`/`team_id`/`project_id` rejected with `ErrTenantFilterInUserFilters` |
| Metadata poisoning | Tenant fields overwritten from context on all documents via `InjectMetadata()` |
| Missing tenant context | Fail closed: returns `ErrMissingTenant`, never empty results |
| Context bypass | Isolation mode validates tenant before any operation |

### TenantFromContext Pattern

The `TenantFromContext()` function extracts tenant information from Go's context:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Setting tenant context (typically at request boundary)
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",      // Required
    TeamID:    "platform",     // Optional
    ProjectID: "contextd",     // Optional
})

// Extracting tenant context (used by isolation layer)
tenant, err := vectorstore.TenantFromContext(ctx)
if err != nil {
    // err is ErrMissingTenant - fail closed
    return nil, err
}
```

### ErrMissingTenant Fail-Closed Behavior

When tenant context is missing, operations return `ErrMissingTenant` rather than proceeding with no filtering:

```go
func (p *PayloadIsolation) InjectFilter(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
    tenant, err := TenantFromContext(ctx)
    if err != nil {
        return nil, err  // Returns ErrMissingTenant - fail closed
    }
    // ... proceed with tenant filtering
}
```

This ensures:
- **No silent failures** - Missing tenant is always an error
- **No data leakage** - Cannot accidentally query all tenants
- **Audit trail** - Errors are logged and traceable

### Defense-in-Depth Layers

```
Request Flow:
+------------------------+
|   Request Handler      |  Layer 1: Sets tenant context
+------------------------+
           |
           v
+------------------------+
|   IsolationMode        |  Layer 2: Validates tenant, injects filters
+------------------------+
           |
           v
+------------------------+
|   ApplyTenantFilters() |  Layer 3: Rejects user tenant fields
+------------------------+
           |
           v
+------------------------+
|   VectorStore          |  Layer 4: Executes filtered query
+------------------------+
```

### Threat Model

| Threat | Attack | Defense |
|--------|--------|---------|
| Cross-tenant data access | Attacker queries without tenant filter | Fail-closed: `ErrMissingTenant` |
| Filter injection | Attacker injects `tenant_id` in query filters | `ApplyTenantFilters()` rejects |
| Metadata poisoning | Attacker sets `tenant_id` in document metadata | `InjectMetadata()` overwrites |
| Privilege escalation | Attacker modifies tenant context mid-request | Context is immutable |
| Isolation mode bypass | Attacker attempts to disable isolation | Mode set at construction, not runtime |

### Legacy: FilesystemIsolation

For backward compatibility, `FilesystemIsolation` provides database-per-project isolation:

```go
// Legacy approach - separate database per tenant
config := vectorstore.ChromemConfig{
    Path:      "/data/vectorstore/{tenant_id}",
    Isolation: vectorstore.NewFilesystemIsolation(),
}
```

| Concern | Mitigation |
|---------|------------|
| Cross-tenant access | Physical filesystem separation |
| Database name injection | Validate: `^[a-z0-9_]{1,64}$` |
| Missing tenant context | Fail closed: `ErrMissingTenant` |

**Migration:** See `docs/migration/payload-filtering.md` for migration from filesystem to payload isolation.

## Credential Protection

| Concern | Mitigation |
|---------|------------|
| API key in logs | Use `config.Secret` type |
| API key in errors | Wrap: `fmt.Errorf("auth failed: %w", err)` |
| API key in transit | TLS required for non-localhost |

### Secret Handling

```go
type Config struct {
    Host   string        `koanf:"host"`
    Port   int           `koanf:"port"`
    APIKey config.Secret `koanf:"api_key"`  // Never logged
    UseTLS bool          `koanf:"use_tls"`
}
```

## Input Validation Layers

| Layer | Validation |
|-------|------------|
| Session middleware | Validate session_id, extract trusted tenant |
| VectorDB client | Validate database name (defense-in-depth) |
| Code indexer | Validate file paths, prevent traversal |

### File Path Validation

```go
func validateFilePath(path string) error {
    // Prevent path traversal
    if strings.Contains(path, "..") {
        return ErrInvalidPath
    }
    
    // Must be relative
    if filepath.IsAbs(path) {
        return ErrInvalidPath
    }
    
    return nil
}
```

## Required Security Tests

```go
func TestMultiTenantIsolation_CrossOrgQueryFails(t *testing.T) {
    ctx1 := WithTenant(ctx, &Tenant{OrgID: "org_a"})
    ctx2 := WithTenant(ctx, &Tenant{OrgID: "org_b"})
    
    // Insert into org_a
    err := client.Upsert(ctx1, &UpsertRequest{
        Collection: "memories",
        Points:     []*DocumentPoint{{ID: "1", Document: &Document{Text: "secret"}}},
    })
    require.NoError(t, err)
    
    // Search from org_b - MUST NOT find org_a data
    results, err := client.Search(ctx2, &SearchRequest{
        Collection: "memories",
        Document:   &Document{Text: "secret"},
    })
    require.NoError(t, err)
    
    if len(results) > 0 {
        t.Error("SECURITY VIOLATION: Cross-org data leakage")
    }
}

func TestMissingTenantContext_FailsClosed(t *testing.T) {
    ctx := context.Background()  // No tenant
    
    _, err := client.Search(ctx, &SearchRequest{
        Collection: "memories",
        Document:   &Document{Text: "query"},
    })
    
    if !errors.Is(err, ErrMissingTenant) {
        t.Error("SECURITY VIOLATION: Operation allowed without tenant")
    }
}

func TestDatabaseNameValidation_RejectsInjection(t *testing.T) {
    tests := []string{
        "../etc/passwd",
        "org; DROP TABLE",
        "org\x00hidden",
        strings.Repeat("a", 100),  // Too long
    }
    
    for _, name := range tests {
        if isValidDatabaseName(name) {
            t.Errorf("SECURITY VIOLATION: Invalid name accepted: %q", name)
        }
    }
}
```
