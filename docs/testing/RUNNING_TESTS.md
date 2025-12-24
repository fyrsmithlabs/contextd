# Running Tests

**Status**: Active Development
**Last Updated**: 2025-12-11

---

## Prerequisites

| Requirement | Version | Check Command |
|-------------|---------|---------------|
| Go | 1.21+ | `go version` |
| Make | any | `make --version` |

**No Docker required** for framework tests. The framework uses an in-memory mock store.

---

## Commands

### All Framework Tests

```bash
make test-integration-framework
```

Runs all tests in `test/integration/framework/`. Expect ~1.5s runtime.

### Individual Suites

| Command | Suite | What It Tests |
|---------|-------|---------------|
| `make test-integration-policy` | A | Policy compliance |
| `make test-integration-secrets` | A | Secret scrubbing |
| `make test-integration-bugfix` | C | Bug-fix learning |
| `make test-integration-multisession` | D | Checkpoint/resume |

### All Suites Sequentially

```bash
make test-integration-all-suites
```

### Direct Go Commands

```bash
# All framework tests
go test -v -count=1 ./test/integration/framework/...

# Specific test function
go test -v -count=1 -run "TestSuiteA_Policy_TDDEnforcement" ./test/integration/framework/...

# Specific subtest
go test -v -count=1 -run "TestSuiteA_Policy_TDDEnforcement/searches_TDD_policy" ./test/integration/framework/...
```

---

## Output Interpretation

### Passing Test

```
=== RUN   TestSuiteA_Policy_TDDEnforcement
=== RUN   TestSuiteA_Policy_TDDEnforcement/searches_TDD_policy_with_confidence_>=_0.7
--- PASS: TestSuiteA_Policy_TDDEnforcement (0.00s)
    --- PASS: TestSuiteA_Policy_TDDEnforcement/searches_TDD_policy_with_confidence_>=_0.7 (0.00s)
```

### Failing Test

```
=== RUN   TestSuiteC_BugFix_SameBugSearch
    suite_c_bugfix_test.go:65:
        Error Trace:    suite_c_bugfix_test.go:65
        Error:          Not equal:
                        expected: true
                        actual  : false
        Test:           TestSuiteC_BugFix_SameBugSearch
        Messages:       should find at least one result
--- FAIL: TestSuiteC_BugFix_SameBugSearch (0.00s)
```

Key information:
- **File:Line** - Where assertion failed
- **expected vs actual** - What went wrong
- **Messages** - Human-readable explanation

---

## Common Failures

| Symptom | Cause | Solution |
|---------|-------|----------|
| `should find at least one result` | Empty search results | Check ProjectID matches between record and search |
| `confidence should be >= 0.7` | Low confidence score | Verify mock store sets confidence metadata |
| `checkpoint not found` | ID filter mismatch | Mock store filters by `id` field in metadata |
| Import errors | Wrong module path | Use `github.com/fyrsmithlabs/contextd` |

### Empty Search Results

The mock vector store filters by:
1. **ProjectID** - Collection name derived from ProjectID
2. **Confidence** - Must meet MinConfidence threshold (0.7)

Fix: Use `SharedStore` and unique ProjectIDs:

```go
// WRONG: Different ProjectIDs
dev.RecordMemory(ctx, record)  // Uses "project-a"
dev.SearchMemory(ctx, query, 5) // Searches "project-b"

// RIGHT: Same ProjectID
sharedStore, _ := NewSharedStore(SharedStoreConfig{
    ProjectID: "test_project_unique",
})
dev, _ := NewDeveloperWithStore(DeveloperConfig{
    ProjectID: "test_project_unique",
}, sharedStore)
```

### Checkpoint Not Found

The checkpoint service stores checkpoints with an `id` field in metadata. The mock store must filter by this field:

```go
// Mock store filter (already implemented)
if idFilter, ok := filters["id"].(string); ok {
    if doc.Metadata["id"] != idFilter {
        shouldInclude = false
    }
}
```

If checkpoints fail to resume, verify the mock store's `SearchInCollection` handles the `id` filter.

---

## Debugging

### Verbose Output

```bash
go test -v -count=1 ./test/integration/framework/...
```

### Single Test with Logging

```bash
go test -v -count=1 -run "TestSuiteD_MultiSession_CleanResume" ./test/integration/framework/... 2>&1 | tee test.log
```

### Race Detection

```bash
go test -race -v ./test/integration/framework/...
```

---

## Test Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./test/integration/framework/...

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

---

## CI Integration

The Makefile targets work in CI without modification:

```yaml
# GitHub Actions example
- name: Run integration tests
  run: make test-integration-framework
```

No Docker, no external services, no special setup required.
