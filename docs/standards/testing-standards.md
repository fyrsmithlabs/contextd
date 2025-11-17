# Testing Standards

## Overview

This document defines the testing requirements and standards for contextd. All code must follow Test-Driven Development (TDD) principles and meet the coverage requirements outlined below.

## Test-Driven Development (TDD)

### TDD Workflow

**REQUIRED**: All code must be developed using TDD:

1. **Write Test**: Write a failing test for the desired functionality
2. **Run Test**: Verify the test fails (red)
3. **Write Code**: Write minimal code to make the test pass
4. **Run Test**: Verify the test passes (green)
5. **Refactor**: Improve code while keeping tests green
6. **Repeat**: Continue for next piece of functionality

### TDD Benefits

- Ensures code is testable from the start
- Provides living documentation
- Reduces debugging time
- Increases confidence in refactoring
- Prevents over-engineering

## Coverage Requirements

### Minimum Coverage Targets

- **Overall Project**: ≥ 80% code coverage (enforced in CI)
- **Core Packages** (vectorstore, adapter): 100% code coverage
- **Service Packages** (checkpoint, remediation, skills): ≥ 80%
- **Infrastructure** (config, telemetry): ≥ 60%
- **Critical Paths**: 100% code coverage
- **Error Paths**: 100% code coverage

### Critical Paths for contextd

Critical paths include:
- **Authentication and authorization logic**: Bearer token validation, constant-time comparison
- **Multi-tenant isolation**: Database selection, project hashing
- **Vector store operations**: Upsert, search, delete with proper database routing
- **Security-sensitive operations**: Credential handling, file permissions
- **Core MCP tools**: All 9 tools must have comprehensive tests
- **Error handling and recovery**: Wrapper functions, graceful degradation

### Measuring Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out -covermode=atomic

# View coverage in browser
go tool cover -html=coverage.out

# View coverage summary
go tool cover -func=coverage.out

# Check specific package
go test ./pkg/checkpoint -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Test Organization

### File Structure

- Place tests in `_test.go` files
- Test files should be in the same package
- Use `package_name_test` for black-box testing

```
pkg/checkpoint/
├── checkpoint.go
├── checkpoint_test.go        # White-box tests (package checkpoint)
├── service.go
├── service_test.go
└── integration_test.go       # Black-box tests (package checkpoint_test)
```

### Test Naming

```go
// Format: Test[Function]_[Scenario]_[ExpectedResult]
func TestCheckpointService_Save_ValidCheckpoint_Success(t *testing.T) {}
func TestCheckpointService_Save_NilCheckpoint_ReturnsError(t *testing.T) {}
func TestCheckpointService_Save_EmptySummary_ReturnsValidationError(t *testing.T) {}
func TestCheckpointService_Search_MultipleResults_ReturnsSortedByScore(t *testing.T) {}
```

## Test Types

### Unit Tests

Test individual functions/methods in isolation:

```go
func TestHashProjectPath_ValidPath_ReturnsConsistentHash(t *testing.T) {
    path := "/home/user/projects/contextd"

    hash1 := HashProjectPath(path)
    hash2 := HashProjectPath(path)

    if hash1 != hash2 {
        t.Errorf("Hash not consistent: %s != %s", hash1, hash2)
    }

    if len(hash1) != 8 {
        t.Errorf("Hash length = %d, want 8", len(hash1))
    }
}
```

### Table-Driven Tests

For testing multiple scenarios:

```go
func TestValidateToken(t *testing.T) {
    tests := []struct {
        name      string
        provided  string
        expected  string
        wantValid bool
    }{
        {
            name:      "valid token",
            provided:  "abc123",
            expected:  "abc123",
            wantValid: true,
        },
        {
            name:      "invalid token",
            provided:  "wrong",
            expected:  "abc123",
            wantValid: false,
        },
        {
            name:      "empty token",
            provided:  "",
            expected:  "abc123",
            wantValid: false,
        },
        {
            name:      "timing attack attempt",
            provided:  "abc12", // One char short
            expected:  "abc123",
            wantValid: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := &Middleware{expectedToken: tt.expected}
            got := m.validateToken(tt.provided)

            if got != tt.wantValid {
                t.Errorf("validateToken() = %v, want %v", got, tt.wantValid)
            }
        })
    }
}
```

### Integration Tests

Test multiple components together:

```go
func TestCheckpointService_Integration(t *testing.T) {
    // Setup test Qdrant
    qdrant := setupTestQdrant(t)
    store := qdrant.NewVectorStore()
    service := checkpoint.NewService(store, embedder)

    // Cleanup
    t.Cleanup(func() {
        qdrant.Teardown()
    })

    // Test save and search
    checkpoint := &Checkpoint{
        Summary:  "Test checkpoint",
        Project:  "/test/project",
        Content:  "Test content",
    }

    err := service.Save(context.Background(), checkpoint)
    if err != nil {
        t.Fatalf("Save() error = %v", err)
    }

    // Search for checkpoint
    results, err := service.Search(context.Background(), "Test checkpoint", 5)
    if err != nil {
        t.Fatalf("Search() error = %v", err)
    }

    if len(results) == 0 {
        t.Error("Expected search results, got none")
    }
}
```

### Benchmark Tests

For performance-sensitive code:

```go
func BenchmarkVectorStore_Search(b *testing.B) {
    store := setupBenchmarkStore(b)
    vector := make([]float32, 384)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := store.Search(context.Background(), "project_abc123", "checkpoints", vector, 10)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Run benchmarks
// go test -bench=. -benchmem ./pkg/vectorstore/
```

## Testing Patterns

### Test Fixtures

Create reusable test data:

```go
func newTestCheckpoint() *Checkpoint {
    return &Checkpoint{
        ID:        "test-id-123",
        Summary:   "Test checkpoint",
        Project:   "/home/user/projects/test",
        Content:   "Test content here",
        CreatedAt: time.Now(),
    }
}

func TestCheckpointService_Update(t *testing.T) {
    checkpoint := newTestCheckpoint()
    // Test with checkpoint
}
```

### Test Helpers

Extract common setup logic:

```go
func setupTestService(t *testing.T) *CheckpointService {
    t.Helper() // Mark as helper function

    store := setupTestVectorStore(t)
    embedder := setupTestEmbedder(t)

    t.Cleanup(func() {
        store.Cleanup()
    })

    return checkpoint.NewService(store, embedder)
}

func TestCheckpointService_Save(t *testing.T) {
    service := setupTestService(t)
    // Test with service
}
```

### Mocking

Use interfaces for mocking dependencies:

```go
// Define interface
type VectorStore interface {
    Upsert(ctx context.Context, database, collection string, points []Point) error
    Search(ctx context.Context, database, collection string, vector []float32, limit int) ([]SearchResult, error)
}

// Mock implementation
type mockVectorStore struct {
    upsertFunc func(ctx context.Context, database, collection string, points []Point) error
    searchFunc func(ctx context.Context, database, collection string, vector []float32, limit int) ([]SearchResult, error)
}

func (m *mockVectorStore) Upsert(ctx context.Context, database, collection string, points []Point) error {
    if m.upsertFunc != nil {
        return m.upsertFunc(ctx, database, collection, points)
    }
    return nil
}

func (m *mockVectorStore) Search(ctx context.Context, database, collection string, vector []float32, limit int) ([]SearchResult, error) {
    if m.searchFunc != nil {
        return m.searchFunc(ctx, database, collection, vector, limit)
    }
    return nil, nil
}

// Usage in tests
func TestCheckpointService_Save(t *testing.T) {
    store := &mockVectorStore{
        upsertFunc: func(ctx context.Context, database, collection string, points []Point) error {
            if len(points) == 0 {
                return errors.New("no points to upsert")
            }
            return nil
        },
    }

    service := checkpoint.NewService(store, embedder)
    // Test with mocked store
}
```

### Context Testing

Always test with proper context:

```go
func TestCheckpointService_Save_ContextCanceled(t *testing.T) {
    service := setupTestService(t)

    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    checkpoint := newTestCheckpoint()
    err := service.Save(ctx, checkpoint)

    if !errors.Is(err, context.Canceled) {
        t.Errorf("Expected context.Canceled error, got %v", err)
    }
}
```

## Race Condition Testing

**ALWAYS** test for race conditions:

```bash
# Run tests with race detector
go test -race ./...
```

Example concurrent test:

```go
func TestCheckpointService_ConcurrentSaves(t *testing.T) {
    service := setupTestService(t)
    var wg sync.WaitGroup

    // Spawn 100 goroutines
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            checkpoint := &Checkpoint{
                Summary: fmt.Sprintf("Checkpoint %d", id),
                Project: "/test/project",
            }

            err := service.Save(context.Background(), checkpoint)
            if err != nil {
                t.Errorf("Save() error = %v", err)
            }
        }(i)
    }

    wg.Wait()

    // Verify all checkpoints saved
    results, err := service.List(context.Background(), "/test/project", 200)
    if err != nil {
        t.Fatalf("List() error = %v", err)
    }

    if len(results) != 100 {
        t.Errorf("Expected 100 checkpoints, got %d", len(results))
    }
}
```

## Error Testing

Test all error paths:

```go
func TestCheckpointService_Save_InvalidCheckpoint(t *testing.T) {
    service := setupTestService(t)

    tests := []struct {
        name       string
        checkpoint *Checkpoint
        wantErr    string
    }{
        {
            name:       "nil checkpoint",
            checkpoint: nil,
            wantErr:    "checkpoint cannot be nil",
        },
        {
            name:       "empty summary",
            checkpoint: &Checkpoint{Project: "/test"},
            wantErr:    "summary required",
        },
        {
            name:       "empty project",
            checkpoint: &Checkpoint{Summary: "test"},
            wantErr:    "project required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.Save(context.Background(), tt.checkpoint)

            if err == nil {
                t.Fatal("Expected error, got nil")
            }

            if !strings.Contains(err.Error(), tt.wantErr) {
                t.Errorf("Error = %v, want to contain %q", err, tt.wantErr)
            }
        })
    }
}
```

## Test Assertion Helpers

Create custom assertion helpers:

```go
func assertEqual(t *testing.T, got, want interface{}) {
    t.Helper()
    if !reflect.DeepEqual(got, want) {
        t.Errorf("got %v, want %v", got, want)
    }
}

func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func assertError(t *testing.T, err error, msg string) {
    t.Helper()
    if err == nil {
        t.Fatalf("expected error: %s", msg)
    }
}

func assertContains(t *testing.T, haystack, needle string) {
    t.Helper()
    if !strings.Contains(haystack, needle) {
        t.Errorf("%q does not contain %q", haystack, needle)
    }
}
```

## Skill-First Testing (contextd-specific)

### Philosophy

**Every feature MUST have a test skill created.**

Test skills are stored in the contextd skills database and can be executed by persona agents (@qa-engineer, @developer-user, @security-tester, @performance-tester).

### Test Skills

```bash
# When implementing new feature
1. Create feature
2. Create test skill for feature
3. Have @qa-engineer execute test skill
4. Fix any issues found
5. Commit feature + test skill together
```

**Test Skills in contextd**:
- **MCP Tool Testing Suite** - All 9 MCP tools
- **API Testing Suite** - All REST endpoints
- **Integration Testing Suite** - End-to-end workflows
- **Regression Testing Suite** - All fixed bugs

### Persona Agent Testing

Use persona agents for comprehensive testing:

- **@qa-engineer** - Comprehensive testing, edge cases, security
- **@developer-user** - Workflow testing, developer experience
- **@security-tester** - Security focus, vulnerability testing
- **@performance-tester** - Load testing, performance benchmarks

**Location**: `.claude/agents/`

## Bug Tracking with Regression Tests

### Bug Lifecycle

```
Found → Documented → Reproduced → Regression Test Created → Fixed → Verified → Closed
```

### Bug Documentation

```bash
# When bug is found
1. Create bug record: tests/regression/bugs/BUG-YYYY-MM-DD-NNN.md
2. Document reproduction steps
3. Create regression test
4. Fix bug
5. Verify regression test passes
6. Commit fix + regression test together
```

### Bug Record Format

```
tests/regression/
├── bugs/
│   ├── BUG-2025-11-02-001.md
│   ├── BUG-2025-11-02-002.md
│   └── ...
├── security/
│   ├── SEC-2025-11-02-001.md
│   └── ...
└── performance/
    └── PERF-2025-11-02-001.md
```

### Regression Test Requirements

**Every bug MUST have**:
1. Bug documentation (markdown)
2. Regression test (executable script or Go test)
3. Test in CI/CD pipeline
4. Test passing before close

**Example Regression Test**:
```bash
#!/bin/bash
# Regression test for BUG-2025-11-02-001

# This test FAILS if bug is reintroduced
go test -run TestFilterInjectionPrevention ./pkg/vectorstore/

# Or for API/MCP
./test-bug-scenario.sh
```

## Pre-Commit Testing Script

Create `.scripts/pre-task-complete.sh`:

```bash
#!/bin/bash

set -e

echo "Running pre-task completion checks..."

echo "1. Building..."
go build ./...

echo "2. Running tests..."
go test ./... -coverprofile=coverage.out -covermode=atomic

echo "3. Checking coverage..."
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 80" | bc -l) )); then
    echo "❌ Coverage is $COVERAGE%, required 80%"
    exit 1
fi
echo "✅ Coverage is $COVERAGE%"

echo "4. Running race detector..."
go test -race ./...

echo "5. Formatting code..."
gofmt -w .

echo "6. Running linters..."
golint ./...
go vet ./...
staticcheck ./...

echo "✅ All checks passed! Ready for commit."
```

Make it executable:

```bash
chmod +x .scripts/pre-task-complete.sh
```

## Continuous Integration

### GitHub Actions

contextd uses GitHub Actions for CI/CD with comprehensive testing:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...

      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 80" | bc -l) )); then
            echo "Coverage is $COVERAGE%, required 80%"
            exit 1
          fi

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
          token: ${{ secrets.CODECOV_TOKEN }}
```

### Codecov Integration

contextd uses Codecov for coverage tracking:

- **Dashboard**: https://codecov.io/gh/axyzlabs/contextd
- **Coverage badge** in README
- **PR comments** with coverage diff
- **Historical trends** for coverage monitoring

**See**: `docs/CODECOV-SETUP.md` for setup instructions

## Test Documentation

### Test Comments

Document complex test setups:

```go
// TestCheckpointService_MultiTenantIsolation verifies that checkpoints
// are properly isolated between projects using database-per-project
// architecture. This prevents filter injection attacks (Issue #60).
func TestCheckpointService_MultiTenantIsolation(t *testing.T) {
    // Test implementation
}
```

### Example Tests

Provide example tests as documentation:

```go
func ExampleCheckpointService_Save() {
    service := checkpoint.NewService(store, embedder)

    checkpoint := &Checkpoint{
        Summary: "Implemented user authentication",
        Project: "/home/user/projects/myapp",
        Content: "Added JWT-based authentication with refresh tokens",
    }

    err := service.Save(context.Background(), checkpoint)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Checkpoint saved")
    // Output: Checkpoint saved
}
```

## Testing Checklist

Before submitting a PR:

- [ ] All new code has unit tests
- [ ] All error paths are tested
- [ ] Critical paths have 100% coverage
- [ ] Overall coverage ≥ 80%
- [ ] Race detector passes (`go test -race ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] Integration tests added for new features
- [ ] Edge cases are tested (nil, empty, boundary values)
- [ ] Test names clearly describe scenarios
- [ ] Mocks are used appropriately
- [ ] Tests are independent and can run in parallel
- [ ] Test helpers use `t.Helper()`
- [ ] Context propagation tested
- [ ] Security-sensitive operations fully tested
- [ ] Multi-tenant isolation verified

## Best Practices

1. **Write tests first** (TDD)
2. **Test behavior, not implementation**
3. **Keep tests simple and readable**
4. **Use table-driven tests for multiple scenarios**
5. **Test error cases thoroughly**
6. **Use t.Helper() for test helpers**
7. **Clean up resources with t.Cleanup()**
8. **Run tests with race detector**
9. **Maintain high coverage**
10. **Document complex test scenarios**
11. **Create test skills for new features**
12. **Write regression tests for bugs**

## Summary

**Testing is not optional**. All code must:

1. ✅ Be developed using TDD
2. ✅ Have ≥ 80% coverage (100% for critical paths)
3. ✅ Pass race detector
4. ✅ Test all error paths
5. ✅ Use appropriate mocking
6. ✅ Be well-documented
7. ✅ Run in CI/CD pipeline
8. ✅ Include test skills for features
9. ✅ Include regression tests for bugs

## Related Standards

- **Architecture**: `docs/standards/architecture.md`
- **Coding Standards**: `docs/standards/coding-standards.md`
- **Package Guidelines**: `docs/standards/package-guidelines.md`
