# Vector Storage Testing

## Unit Tests

| Component | Test Cases |
|-----------|------------|
| `vectordb.Client` | Collection CRUD, point ops, search |
| `vectordb.Context` | WithTenant, TenantFromContext, missing |
| `codeindex.Parser` | Go/TS/Python/Rust AST extraction |
| `codeindex.Textify` | Code → natural language |
| `codeindex.Git` | Diff detection, ref watching |

## Integration Tests

| Test | Description |
|------|-------------|
| `TestQdrantIntegration` | Real Qdrant (docker-compose) |
| `TestCodeIndexing_E2E` | Parse → textify → upsert → search |
| `TestDeltaIndexing` | Only changed files re-indexed |
| `TestMultiTenantIsolation` | Cross-org MUST fail |

## Mock Client

```go
// MockClient for testing without Qdrant.
type MockClient struct {
    collections map[string]map[string]*mockCollection  // db -> collection -> data
    mu          sync.RWMutex
}

func NewMockClient() *MockClient {
    return &MockClient{
        collections: make(map[string]map[string]*mockCollection),
    }
}

func (m *MockClient) Search(ctx context.Context, req *SearchRequest) ([]*ScoredPoint, error) {
    tenant := TenantFromContext(ctx)
    if tenant == nil {
        return nil, ErrMissingTenant  // Same behavior as real client
    }
    
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    db := tenant.Database()
    if _, ok := m.collections[db]; !ok {
        return nil, nil  // Empty results
    }
    
    // In-memory search simulation...
    return m.searchInMemory(db, req)
}
```

## Test Fixtures

```go
// TestTenant returns a tenant for testing.
func TestTenant(orgID string) *Tenant {
    return &Tenant{
        OrgID:     orgID,
        TeamID:    "test-team",
        ProjectID: "test-project",
    }
}

// TestContext returns a context with tenant for testing.
func TestContext(orgID string) context.Context {
    return WithTenant(context.Background(), TestTenant(orgID))
}
```

## Coverage Requirements

| Category | Target |
|----------|--------|
| Unit tests | ≥80% |
| Integration tests | Critical paths |
| Security tests | All isolation scenarios |
| gosec | No findings |

## Docker Compose for Integration Tests

```yaml
# docker-compose.test.yml
services:
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
      - "6334:6334"
    environment:
      - QDRANT__SERVICE__GRPC_PORT=6334
```

## Running Tests

```bash
# Unit tests
go test ./internal/vectordb/... ./internal/codeindex/...

# Integration tests (requires docker-compose)
docker-compose -f docker-compose.test.yml up -d
go test -tags=integration ./internal/vectordb/... ./internal/codeindex/...
docker-compose -f docker-compose.test.yml down

# Security scan
gosec ./internal/vectordb/... ./internal/codeindex/...

# Coverage report
go test -coverprofile=coverage.out ./internal/vectordb/... ./internal/codeindex/...
go tool cover -html=coverage.out
```
