# Regression Test Suite

This directory contains a regression test suite (`regression_test.go`) for tracking and preventing bugs from recurring in the stdio MCP implementation.

## Purpose

**Regression tests ensure that once a bug is fixed, it never comes back.**

Every time a bug is discovered and fixed:
1. A regression test is added to `regression_test.go`
2. The test is named after the bug report: `TestRegression_BUG_YYYY_MM_DD_NNN_Description`
3. The test reproduces the bug conditions and verifies the fix works
4. The test serves as living documentation of the bug and its fix

## Quick Start

### Running All Regression Tests

```bash
# Run all regression tests
go test -v -run TestRegression ./pkg/mcp/stdio/

# Run with race detector (recommended)
go test -race -run TestRegression ./pkg/mcp/stdio/

# Run specific regression test
go test -v -run TestRegression_BUG_2025_11_20_001 ./pkg/mcp/stdio/
```

### Adding a New Regression Test

When a bug is filed (GitHub issue, Jira ticket, user report):

**Step 1: Create test function following the naming convention**

```go
func TestRegression_BUG_2025_11_20_004_ShortDescription(t *testing.T) {
    // Bug: [Brief description]
    // Root Cause: [What caused it]
    // Fix: [How it was fixed]
    // Issue: https://github.com/fyrsmithlabs/contextd/issues/123
    //
    // Reproduction:
    // 1. [Step to reproduce]
    // 2. [Step to reproduce]
    // Expected: [Expected behavior]
    // Actual (before fix): [Buggy behavior]

    // Test implementation
}
```

**Step 2: Write minimal reproduction**

The test should:
- Be as small as possible while still reproducing the bug
- FAIL before the fix is applied
- PASS after the fix is applied
- Use mock servers when possible (fast, no external dependencies)

**Step 3: Verify the test**

```bash
# Before applying fix: Test should FAIL
go test -v -run TestRegression_BUG_2025_11_20_004 ./pkg/mcp/stdio/

# Apply the fix to the code

# After applying fix: Test should PASS
go test -v -run TestRegression_BUG_2025_11_20_004 ./pkg/mcp/stdio/
```

**Step 4: Commit test with fix**

```bash
git add pkg/mcp/stdio/regression_test.go pkg/mcp/stdio/server.go
git commit -m "fix: [bug description] (BUG-2025-11-20-004)

Add regression test to prevent recurrence.

Fixes #123"
```

## Naming Convention

**Format**: `TestRegression_BUG_YYYY_MM_DD_NNN_ShortDescription`

- `YYYY-MM-DD`: Date bug was filed (ISO 8601)
- `NNN`: Sequential number (001, 002, 003, etc.)
- `ShortDescription`: CamelCase brief description (e.g., `EmptyResponseHandling`)

**Examples**:
- `TestRegression_BUG_2025_11_20_001_DaemonTimeoutNotHandled`
- `TestRegression_BUG_2025_11_20_002_EmptyResponseNilPanic`
- `TestRegression_BUG_2025_12_01_001_ContextCancellationLeak`

**Why this format?**
- Chronological ordering (easy to see bug history)
- Unique identifiers (date + number)
- Searchable (link to bug reports via issue number)
- Descriptive (know what the test is about)

## Template

Use this template for new regression tests:

```go
func TestRegression_BUG_YYYY_MM_DD_NNN_Description(t *testing.T) {
    // Bug: [Brief description of the bug]
    // Root Cause: [What caused the bug]
    // Fix: [How it was fixed]
    // Issue: [Link to GitHub issue or bug report]
    //
    // Reproduction:
    // 1. [Step to reproduce]
    // 2. [Step to reproduce]
    // Expected: [Expected behavior]
    // Actual (before fix): [Actual buggy behavior]

    // Create test scenario that reproduces the bug
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Simulate conditions that trigger the bug
    }))
    defer mockServer.Close()

    server, err := NewServer(mockServer.URL)
    if err != nil {
        t.Fatalf("Failed to create server: %v", err)
    }

    // Call the code that had the bug
    result, _, err := server.handleSomething(context.Background(), params)

    // Verify the fix works
    if err != nil {
        t.Errorf("Expected success after fix, got error: %v", err)
    }
}
```

## Test Categories

Regression tests are organized by bug category:

### 1. Timeout & Cancellation Bugs
Tests for context timeouts, cancellation handling, hung requests.

Example: `TestRegression_BUG_2025_11_20_001_DaemonTimeoutNotHandled`

### 2. Nil Pointer & Empty Response Bugs
Tests for nil checks, empty responses, missing fields.

Example: `TestRegression_BUG_2025_11_20_002_EmptyResponseNilPanicPrevention`

### 3. Concurrency & Race Condition Bugs
Tests for race conditions, concurrent access, goroutine leaks.

Example: `TestRegression_BUG_2025_11_20_003_ConcurrentRequestsSafety`

### 4. Protocol & Serialization Bugs
Tests for JSON encoding/decoding, MCP protocol violations, type mismatches.

Example: *(Add when bugs occur)*

### 5. Error Handling Bugs
Tests for error propagation, error wrapping, HTTP status codes.

Example: *(Add when bugs occur)*

## Current Regression Tests

| Test | Date | Issue | Description |
|------|------|-------|-------------|
| `BUG_2025_11_20_001` | 2025-11-20 | Preemptive | Daemon timeout handling |
| `BUG_2025_11_20_002` | 2025-11-20 | Preemptive | Empty response nil panic prevention |
| `BUG_2025_11_20_003` | 2025-11-20 | Preemptive | Concurrent requests safety |

**Preemptive tests**: Added before bugs occur to prevent common issues.

## Best Practices

### DO:
✅ Write minimal reproductions (small, focused tests)
✅ Use table-driven tests for related scenarios
✅ Link to bug reports in comments (GitHub issue, Jira ticket)
✅ Include clear "Expected" vs "Actual" in comments
✅ Use mock servers (fast, no dependencies)
✅ Run with `-race` flag for concurrency bugs
✅ Verify test fails before fix, passes after fix

### DON'T:
❌ Write regression tests for features (those go in unit tests)
❌ Skip bug documentation in comments
❌ Make tests depend on external services (use mocks)
❌ Combine multiple unrelated bugs in one test
❌ Forget to link to the bug report
❌ Write tests that can't reproduce the original bug

## Integration with CI/CD

Regression tests run automatically in CI:

```yaml
# .github/workflows/test.yml
- name: Run regression tests
  run: |
    go test -v -race -run TestRegression ./pkg/mcp/stdio/
```

**All regression tests must pass** before merging PRs.

## Maintenance

### Quarterly Review
Every quarter, review regression tests for:
- Tests that can be removed (code refactored, issue no longer applicable)
- Tests that need updates (API changes, protocol changes)
- Duplicate coverage (consolidate if possible)

### When to Remove
Remove a regression test only if:
- The code it tests has been completely removed
- The architecture changed such that the bug is impossible
- The test is consolidated into a broader test

**Never remove a regression test just because "the bug is old" or "unlikely to recur".**

## Questions?

For questions about the regression test suite:
1. Check this README first
2. Look at existing tests in `regression_test.go` for examples
3. Ask in team chat or open an issue

## Related Documentation

- **Testing Standards**: `docs/standards/testing-standards.md`
- **Bug Tracking**: Document in `docs/testing/regression/bugs/BUG-YYYY-MM-DD-NNN.md`
- **Main Test Suite**: `server_test.go`, `client_test.go`, `integration_test.go`
