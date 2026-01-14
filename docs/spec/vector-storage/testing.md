# Vector Storage Testing

## Overview

contextd's vector storage testing includes both unit tests for individual components and **semantic similarity tests** for validating real search quality. This ensures both correctness and effectiveness of semantic search.

**See Also:** [Semantic Similarity Testing Guide](../testing/semantic-similarity.md) - Complete guide to semantic search quality validation with test fixtures, quality metrics, and regression detection.

## Unit Tests

| Component | Test Cases |
|-----------|------------|
| `vectordb.Client` | Collection CRUD, point ops, search |
| `vectordb.Context` | WithTenant, TenantFromContext, missing |
| `codeindex.Parser` | Go/TS/Python/Rust AST extraction |
| `codeindex.Textify` | Code → natural language |
| `codeindex.Git` | Diff detection, ref watching |
| **`vectorstore.TestFixtures`** | **5 semantic test cases with known query-document pairs** |
| **`vectorstore.QualityMetrics`** | **NDCG (12), MRR (10), Precision@K (12) calculation tests** |

## Integration Tests

| Test | Description |
|------|-------------|
| `TestQdrantIntegration` | Real Qdrant (docker-compose) |
| `TestCodeIndexing_E2E` | Parse → textify → upsert → search |
| `TestDeltaIndexing` | Only changed files re-indexed |
| `TestMultiTenantIsolation` | Cross-org MUST fail |
| **`TestSemanticSimilarity`** | **End-to-end semantic search with fake or real embeddings** |

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

## Semantic Similarity Testing

contextd includes comprehensive semantic similarity testing to validate that vector search returns **actually relevant results** with proper ranking, not just hardcoded scores.

### Test Fixtures

Test fixtures are curated query-document pairs with known expected behavior. See [Test Fixtures Documentation](../testing/semantic-similarity.md#test-fixtures) for details.

**Available Fixtures:**
- `HighSimilarityPair` - Synonym recognition (Go vs Golang)
- `LowSimilarityPair` - Topic separation (programming vs cooking)
- `SynonymHandling` - Semantic equivalence (tutorial/guide)
- `MultiTopicDocuments` - Multi-topic ranking (ML + Python)
- `GradualRelevanceDecay` - Score distribution (5 docs)

**Location:** `internal/vectorstore/testdata/fixtures.go`

### Quality Metrics

Three standard information retrieval metrics validate search effectiveness:

| Metric | Measures | Range | Formula |
|--------|----------|-------|---------|
| **NDCG** | Ranking quality | 0.0 - 1.0 | DCG / IDCG (position-weighted relevance) |
| **MRR** | First relevant position | 0.0 - 1.0 | 1 / rank_of_first_relevant |
| **Precision@K** | Top-K accuracy | 0.0 - 1.0 | relevant_in_top_K / K |

**Implementation:** `internal/vectorstore/quality_metrics.go`

**Example Usage:**
```go
metrics := vectorstore.CalculateAllMetrics(
    results,          // Search results
    expectedRanking,  // Ideal document order
    relevantDocs,     // Relevant document IDs
    k,                // Top K cutoff
)
// Returns: NDCG, MRR, Precision@K
```

### Regression Detection

Tests automatically **fail** if search quality degrades below baseline thresholds:

```go
// TestChromemStore_SemanticReal_RegressionDetection
// Loads baseline_metrics.json
// Runs all 5 fixtures with real FastEmbed embeddings
// Fails if: actual_metric < baseline * 0.95 (5% tolerance)
```

**Baseline:** `internal/vectorstore/testdata/baseline_metrics.json`

**Current Thresholds:**
- Average NDCG ≥ 0.90
- Average MRR ≥ 0.95
- Average Precision ≥ 0.65

See [Regression Detection Guide](../testing/semantic-similarity.md#regression-detection) for updating baselines.

## Test Fixtures (Tenant Context)

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
| **Semantic similarity tests** | **All 5 fixtures validated with real embeddings** |
| **Quality metrics** | **NDCG ≥ 0.90, MRR ≥ 0.95, Precision ≥ 0.65** |
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

### Fast Tests (Mock/Fake Embeddings)

```bash
# Unit tests
go test ./internal/vectordb/... ./internal/codeindex/...

# Semantic similarity test fixtures
go test ./internal/vectorstore/testdata

# Quality metrics unit tests
go test ./internal/vectorstore -run QualityMetrics

# Integration tests with fake embeddings (fast)
go test ./test/integration/framework -run SemanticSimilarity
```

### Real Embedding Tests (Quality Validation)

**Requirements:** ONNX runtime (`/usr/lib/libonnxruntime.so` or `ONNX_PATH` env var)

```bash
# Semantic similarity tests with real FastEmbed embeddings
go test -v ./internal/vectorstore -run SemanticReal

# Regression detection (fails if quality degrades)
go test -v ./internal/vectorstore -run RegressionDetection

# Integration tests with real embeddings
USE_REAL_EMBEDDINGS=1 go test -v ./test/integration/framework -run SemanticSimilarity

# Makefile target (if available)
make test-semantic-real
```

**Note:** Real embedding tests are automatically skipped in short mode (`go test -short`) and when ONNX runtime is unavailable.

### Integration Tests (Qdrant)

```bash
# Integration tests (requires docker-compose)
docker-compose -f docker-compose.test.yml up -d
go test -tags=integration ./internal/vectordb/... ./internal/codeindex/...
docker-compose -f docker-compose.test.yml down
```

### Security and Coverage

```bash
# Security scan
gosec ./internal/vectordb/... ./internal/codeindex/...

# Coverage report
go test -coverprofile=coverage.out ./internal/vectordb/... ./internal/codeindex/...
go tool cover -html=coverage.out
```

## CI/CD Recommendations

```yaml
# Fast tests (always run on every commit)
- name: Unit Tests
  run: |
    go test ./internal/vectorstore/testdata
    go test ./internal/vectorstore -run QualityMetrics
    go test ./test/integration/framework -run SemanticSimilarity

# Real embedding tests (pre-merge, nightly, or on main branch)
- name: Semantic Quality Tests
  run: |
    go test -v ./internal/vectorstore -run SemanticReal
    go test -v ./internal/vectorstore -run RegressionDetection
```
