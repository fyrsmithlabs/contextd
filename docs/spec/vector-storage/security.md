# Vector Storage Security

## Multi-Tenant Isolation

| Concern | Mitigation |
|---------|------------|
| Cross-tenant access | Database-per-org, tenant from validated session |
| Database name injection | Validate: `^[a-z0-9_]{1,64}$` |
| Missing tenant context | Fail closed: `ErrMissingTenant` |
| Context override | Set at session boundary, immutable downstream |

### Validation Code

```go
var validDBName = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

func isValidDatabaseName(name string) bool {
    return validDBName.MatchString(name)
}

func (c *Client) Search(ctx context.Context, req *SearchRequest) ([]*ScoredPoint, error) {
    tenant := TenantFromContext(ctx)
    if tenant == nil {
        return nil, ErrMissingTenant  // Fail closed
    }
    
    db := tenant.Database()
    if !isValidDatabaseName(db) {  // Defense-in-depth
        return nil, ErrInvalidDatabase
    }
    // Proceed...
}
```

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
