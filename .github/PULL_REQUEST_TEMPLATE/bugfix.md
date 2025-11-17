## Bug Fix Summary

<!-- Brief description of the bug and fix -->

## Related Issues

<!-- Link bug report: Fixes #123 -->

Fixes #

## Bug Description

**Severity**: Critical / High / Medium / Low

**Impact**: <!-- Who/what is affected? -->

**Discovered**: <!-- How was the bug found? User report, automated test, etc. -->

## Root Cause Analysis

**Root Cause**: <!-- What caused the bug? -->

**Why It Wasn't Caught Earlier**: <!-- Testing gap, edge case, etc. -->

**Affected Versions**: <!-- Which versions have this bug? -->

## Fix Implementation

### Changes Made

<!-- Detailed list of changes -->

**Files Modified**:
- `[file-path]` - [what changed and why]
- `[file-path]` - [what changed and why]

**Packages Affected**:
- `pkg/[package-name]/` - [description of changes]

### Fix Approach

<!-- Explain the fix strategy -->

**Solution**: <!-- How does this fix address the root cause? -->

**Alternatives Considered**:
- Option 1: [why not chosen]
- Option 2: [why not chosen]

**Trade-offs**: <!-- Any compromises made? -->

## Regression Testing

**Regression test REQUIRED for all bugs**

- [ ] Regression test created in `tests/regression/bugs/`
- [ ] Bug record created: `tests/regression/bugs/BUG-YYYY-MM-DD-NNN.md`
- [ ] Regression test reproduces bug (verified on buggy code)
- [ ] Regression test passes with fix
- [ ] Regression test added to CI/CD pipeline

### Bug Record

**Bug Record Location**: `tests/regression/bugs/BUG-[date]-[number].md`

**Bug Record Contents**:
```markdown
# BUG-[date]-[number]: [title]

**Date**: [date]
**Severity**: [level]
**Status**: Fixed in v[version]

## Description
[description]

## Reproduction Steps
[steps]

## Expected Behavior
[expected]

## Actual Behavior
[actual]

## Root Cause
[cause]

## Fix
- Commit: [hash]
- PR: #[number]
- Fixed in: v[version]

## Regression Test
- Test file: [path]
- Test function: [name]
```

### Regression Test Details

**Test File**: `[test-file-path]`
**Test Function**: `[function-name]`

```go
// Example regression test structure
func TestBug_YYYYMMDD_NNN_Description(t *testing.T) {
    // Setup that reproduces bug conditions
    // Assert that bug is fixed
}
```

## Testing Performed

### Test Coverage

- [ ] **Regression test** created and passing
- [ ] Unit tests updated (if applicable)
- [ ] Integration tests updated (if applicable)
- [ ] Manual testing completed
- [ ] Edge cases verified
- [ ] Test coverage maintained ≥80%

### Test Results

```bash
# Paste test output showing all tests pass
go test -v -cover -race ./...

# Specifically show regression test
go test -v -run="TestBug_" ./...
```

**Test Summary**:
- Regression test: ✅ PASS
- Related unit tests: ✅ PASS
- Integration tests: ✅ PASS
- Total coverage: ___%

### Manual Verification

**Steps to verify fix**:
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Verified on**:
- [ ] Local development
- [ ] Test environment
- [ ] User reproduction scenario (if available)

## Code Quality

### Pre-Review Checklist

- [ ] Code builds successfully (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] No race conditions (`go test -race ./...`)
- [ ] Code formatted (`gofmt -w .`)
- [ ] Linting passes (`golint ./...`)
- [ ] Static analysis passes (`go vet ./...`, `staticcheck ./...`)

### Code Standards

- [ ] Follows `docs/standards/coding-standards.md`
- [ ] Minimal changes (only what's needed for fix)
- [ ] Proper error handling
- [ ] Input validation (if applicable)
- [ ] Comments explain why (not what)
- [ ] No unnecessary refactoring

## Security Impact

**Security implications** (if any)

- [ ] No security impact
- [ ] Security vulnerability fixed (describe below)
- [ ] Security implications reviewed

### Security Details

<!-- If this fixes a security issue -->

**CVE**: [if applicable]
**CVSS Score**: [if applicable]
**Attack Vector**: [description]

**Mitigation**:
- [How fix prevents exploit]

**Credit**: <!-- Acknowledge reporter if applicable -->

## Performance Impact

**Performance changes**

- [ ] No performance impact
- [ ] Performance improved (provide benchmarks)
- [ ] Performance impact minimal and acceptable

**Benchmark Results** (if applicable):
```bash
go test -bench=. -benchmem ./...
```

## Documentation

### Code Documentation

- [ ] Comments updated
- [ ] Error messages clear and actionable
- [ ] Godoc updated (if public API affected)

### User Documentation

- [ ] README.md updated (if user-facing bug)
- [ ] Known issues removed (if documented)
- [ ] Troubleshooting guide updated (if applicable)
- [ ] CHANGELOG.md entry added

### Bug Tracking Documentation

- [ ] Bug record complete
- [ ] Regression test documented
- [ ] Root cause documented
- [ ] Fix approach documented

## Breaking Changes

**Breaking Changes**: None / Yes (describe below)

<!-- Bug fixes should rarely break compatibility -->
<!-- If they do, provide migration guide -->

### Migration Guide (if breaking)

**What Changes**:
- [API/behavior change]

**Why Breaking**:
- [Justification]

**Migration Steps**:
1. [Step 1]
2. [Step 2]

## Rollback Plan

**Rollback procedure** (if fix causes issues):

1. Revert commit: `git revert [commit-hash]`
2. [Additional steps if needed]

**Known Risks**: <!-- Any risks from rolling back? -->

## Related Bugs

**Similar Issues**: <!-- Links to related bugs -->

- Related to: #___
- Similar to: #___
- May affect: #___

**Pattern Analysis**: <!-- Is this part of a larger issue? -->

## Additional Context

### Environment Details

**Originally Reported In**:
- OS: [operating system]
- Version: [contextd version]
- Go version: [if relevant]
- Installation method: [binary, source, Docker]

### Reproduction Scenario

<!-- Detailed reproduction if not in bug record -->

**Prerequisites**:
- [requirement 1]
- [requirement 2]

**Steps to Reproduce**:
1. [step]
2. [step]
3. [step]

**Expected Result**: [expected]
**Actual Result**: [actual]

### Logs/Error Messages

<!-- Include relevant logs (sanitized) -->

<details>
<summary>Error Logs</summary>

```
[paste relevant logs]
```

</details>

## Verification by Reporter

<!-- If possible, ask original reporter to verify -->

- [ ] Original reporter verified fix (if applicable)
- [ ] Community feedback positive (if applicable)

---

**Bug Fix Checklist** (Complete before requesting review):

- [ ] Regression test created and passing
- [ ] Bug record documented
- [ ] All existing tests pass
- [ ] Root cause understood and documented
- [ ] Fix is minimal and targeted
- [ ] No unrelated changes included
- [ ] Code self-reviewed
- [ ] Documentation updated
- [ ] Ready for maintainer review

---

**Severity Assessment**:

| Level | Description | Examples |
|-------|-------------|----------|
| **Critical** | Service crash, data loss, security breach | Crashes, data corruption |
| **High** | Major feature broken, workaround exists | API failures, core features |
| **Medium** | Minor feature broken, inconvenient | UI bugs, edge cases |
| **Low** | Cosmetic, edge case, minor annoyance | Typos, formatting |

---

🤖 **Generated with [Claude Code](https://claude.com/claude-code)**

<!-- Co-Authored-By: Claude <noreply@anthropic.com> -->
