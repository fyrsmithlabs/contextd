# Integration Test Suite

Comprehensive integration tests for all contextd features in containerized environments.

## Overview

This test suite validates the complete contextd application including:
- **ReasoningBank**: Cross-session memory with confidence scoring
- **Checkpoint**: Context persistence and recovery
- **Remediation**: Error pattern tracking and matching
- **Repository**: Semantic code search with grep fallback
- **Context-Folding**: Active context management with branch isolation
- **End-to-End**: Complete multi-service workflows

All tests run in Docker containers with Qdrant vector database, ensuring consistency across environments.

---

## Quick Start

```bash
# Run all integration tests
./scripts/run-integration.sh

# Run specific feature
./scripts/run-integration.sh --feature reasoningbank
./scripts/run-integration.sh --feature checkpoint
./scripts/run-integration.sh --feature remediation
./scripts/run-integration.sh --feature repository
./scripts/run-integration.sh --feature folding
./scripts/run-integration.sh --feature e2e

# Verbose output
./scripts/run-integration.sh -v

# Keep containers running after tests
./scripts/run-integration.sh --no-cleanup
```

---

## Test Structure

### `reasoningbank_test.go`
**Tests**: ReasoningBank (cross-session memory)

| Test | Validates |
|------|-----------|
| `TestReasoningBank_MemoryCRUD` | Record, search, feedback, outcome tracking |
| `TestReasoningBank_MultiTenantIsolation` | Tenant-scoped memory isolation |
| `TestReasoningBank_ConfidenceScoring` | Success rate and confidence decay |

**Coverage**:
- ✅ Memory lifecycle (create → search → feedback → outcome)
- ✅ Multi-tenant payload filtering
- ✅ Confidence scoring algorithm
- ✅ Tag-based filtering
- ✅ Metadata preservation

### `checkpoint_test.go`
**Tests**: Checkpoint (context snapshots)

| Test | Validates |
|------|-----------|
| `TestCheckpoint_SaveAndResume` | Save/list/resume workflow |
| `TestCheckpoint_MultiSession` | Concurrent session handling |
| `TestCheckpoint_Pagination` | Limit-based result pagination |
| `TestCheckpoint_TenantIsolation` | Cross-tenant checkpoint isolation |

**Coverage**:
- ✅ Checkpoint CRUD operations
- ✅ Message history preservation
- ✅ Accomplishments/InProgress/NextSteps tracking
- ✅ Multi-tenant security
- ✅ Pagination behavior

### `remediation_test.go`
**Tests**: Remediation (error pattern tracking)

| Test | Validates |
|------|-----------|
| `TestRemediation_RecordAndSearch` | Error pattern matching |
| `TestRemediation_ErrorPatternMatching` | Fuzzy semantic matching |
| `TestRemediation_TenantIsolation` | Tenant-scoped remediation isolation |
| `TestRemediation_TagFiltering` | Language/framework tag filtering |

**Coverage**:
- ✅ Error pattern recording
- ✅ Semantic similarity search
- ✅ Code example preservation
- ✅ Tag-based categorization
- ✅ Multi-tenant isolation

### `repository_test.go`
**Tests**: Repository (semantic code search)

| Test | Validates |
|------|-----------|
| `TestRepository_IndexAndSearch` | Repository indexing and semantic search |
| `TestRepository_GrepFallback` | Grep fallback for exact matches |
| `TestRepository_FileTypeFiltering` | Extension-based filtering |
| `TestRepository_TenantIsolation` | Project-scoped code isolation |

**Coverage**:
- ✅ Codebase indexing
- ✅ Semantic code search
- ✅ Grep fallback for exact matches
- ✅ File type filtering (.go only)
- ✅ Multi-project isolation

### `folding_test.go`
**Tests**: Context-Folding (branch isolation)

| Test | Validates |
|------|-----------|
| `TestFolding_BasicBranchLifecycle` | Create → work → return workflow |
| `TestFolding_BudgetEnforcement` | Token budget limits |
| `TestFolding_NestedBranches` | Nested branch depth limits (max 3) |
| `TestFolding_SecretScrubbing` | Secret scrubbing on return |
| `TestFolding_ConcurrentBranches` | Multiple concurrent branches |
| `TestFolding_BranchContext` | Context propagation |

**Coverage**:
- ✅ Branch lifecycle (create/work/return)
- ✅ Token budget enforcement
- ✅ Nested branch limits
- ✅ Secret scrubbing (gitleaks)
- ✅ Concurrent branch handling
- ✅ Context propagation

### `e2e_test.go`
**Tests**: End-to-End workflows

| Test | Validates |
|------|-----------|
| `TestE2E_DevelopmentWorkflow` | Memory → Checkpoint → Remediation workflow |
| `TestE2E_CodebaseExploration` | Repository + Context-Folding integration |

**Coverage**:
- ✅ Multi-service integration
- ✅ Complete development workflows
- ✅ Service interaction patterns
- ✅ Real-world usage scenarios

---

## Architecture

```
test/integration/
├── reasoningbank_test.go    # ReasoningBank tests
├── checkpoint_test.go        # Checkpoint tests
├── remediation_test.go       # Remediation tests
├── repository_test.go        # Repository tests
├── folding_test.go           # Context-Folding tests
├── e2e_test.go               # End-to-end workflows
├── helpers.go                # Test infrastructure
└── README.md                 # This file

docker-compose.integration.yml  # Container orchestration
Dockerfile.integration          # Test container definition
scripts/run-integration.sh      # Automated test runner
```

---

## Container Stack

| Service | Purpose |
|---------|---------|
| `qdrant` | Vector database (v1.7.4) |
| `test-reasoningbank` | ReasoningBank test runner |
| `test-checkpoint` | Checkpoint test runner |
| `test-remediation` | Remediation test runner |
| `test-repository` | Repository test runner |
| `test-folding` | Context-Folding test runner |
| `test-e2e` | End-to-end test runner |
| `test-all` | All tests (default) |

---

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `VECTOR_STORE` | `chromem` | Vector store provider (`chromem` or `qdrant`) |
| `QDRANT_HOST` | `qdrant` | Qdrant host (container name in Docker) |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port |
| `GO_TEST_FLAGS` | `-v -race` | Go test flags |

---

## Running Tests Locally (Without Docker)

```bash
# Use chromem (embedded, no external dependencies)
go test ./test/integration -v -race

# Use Qdrant (requires running Qdrant instance)
export VECTOR_STORE=qdrant
export QDRANT_HOST=localhost
export QDRANT_PORT=6334
go test ./test/integration -v -race
```

---

## Adding New Tests

1. **Create test file**: `test/integration/myfeature_test.go`
2. **Add helper if needed**: Update `helpers.go` with common test infrastructure
3. **Add Docker profile**: Update `docker-compose.integration.yml` with new profile
4. **Update script**: Add feature option to `scripts/run-integration.sh`
5. **Document**: Update this README with test coverage

**Test Template**:
```go
package integration

import (
    "context"
    "testing"

    "github.com/fyrsmithlabs/contextd/internal/vectorstore"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/zap"
)

func TestMyFeature_BasicOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    ctx := context.Background()
    logger := zap.NewNop()

    store, cleanup := createTestVectorStore(t)
    defer cleanup()

    tenant := &vectorstore.TenantInfo{TenantID: "test-org"}
    tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

    // Test implementation
}
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run integration tests
        run: ./scripts/run-integration.sh
```

### GitLab CI

```yaml
integration-tests:
  image: docker:latest
  services:
    - docker:dind
  script:
    - ./scripts/run-integration.sh
```

---

## Troubleshooting

### Qdrant fails to start

```bash
# Check Qdrant logs
docker-compose -f docker-compose.integration.yml logs qdrant

# Restart Qdrant
docker-compose -f docker-compose.integration.yml restart qdrant
```

### Tests timeout

Increase timeout in test file:
```go
go test ./test/integration -timeout 30m
```

### Port conflicts

Change Qdrant ports in `docker-compose.integration.yml`:
```yaml
ports:
  - "16333:6333"
  - "16334:6334"
```

### Out of memory

Reduce concurrent tests or increase Docker memory:
```bash
# Docker Desktop → Settings → Resources → Memory
```

---

## Production Hardening Tests

For vector store-specific production hardening tests (health callbacks, path injection, etc.), see:
- **Tests**: `internal/vectorstore/validation_test.go`
- **Script**: `scripts/run-validation.sh`
- **Compose**: `docker-compose.validation.yml`

These are complementary to integration tests and focus on security, concurrency, and resilience.

---

## Test Coverage Summary

| Component | Unit Tests | Integration Tests | Coverage |
|-----------|------------|-------------------|----------|
| ReasoningBank | ✅ | ✅ | 82% |
| Checkpoint | ✅ | ✅ | High |
| Remediation | ✅ | ✅ | 82% |
| Repository | ✅ | ✅ | High |
| Context-Folding | ✅ | ✅ | High |
| VectorStore | ✅ | ✅ | High |
| Secrets | ✅ | ✅ | 97% |
| Embeddings | ✅ | ✅ | High |

**Total Test Files**: 119 unit test files + 6 integration test files

---

## References

- [CLAUDE.md](../../CLAUDE.md) - Project overview
- [Vector Store README](../../internal/vectorstore/README.md) - Multi-tenancy architecture
- [Context-Folding Spec](../../docs/spec/context-folding/SPEC.md) - Branch isolation design
- [Security Spec](../../docs/spec/vector-storage/security.md) - Multi-tenant security
