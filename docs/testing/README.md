# Integration Test Framework

**Status**: Active Development
**Last Updated**: 2025-12-11

---

## Critical Rules

**ALWAYS use unique ProjectIDs per test** to avoid cross-contamination between subtests.

**ALWAYS use `NewSharedStore` for cross-developer scenarios** - the mock store filters by ProjectID.

**NEVER skip threshold assertions** - confidence >= 0.7 validates semantic search quality.

---

## What This Tests

The integration test framework validates contextd's core promise: **recorded knowledge helps future work**.

Three scenarios prove this:
1. Developer B follows policies that Developer A recorded
2. Developer B finds fixes for bugs that Developer A already solved
3. Developers resume work from checkpoints without losing context

---

## Test Suites

| Suite | Purpose | Tests | Runtime |
|-------|---------|-------|---------|
| A | Policy compliance & secrets | 12 subtests | ~1s |
| C | Bug-fix learning | 5 tests | <1s |
| D | Multi-session continuity | 6 tests | <1s |

---

## Quick Start

```bash
# Run all framework tests (no Docker required)
make test-integration-framework

# Run individual suites
make test-integration-policy       # Suite A
make test-integration-bugfix       # Suite C
make test-integration-multisession # Suite D

# Run all suites in sequence
make test-integration-all-suites
```

---

## Directory Structure

```
test/integration/framework/
├── developer.go              # Developer simulator (core component)
├── suite_a_policy_test.go    # Policy compliance tests
├── suite_a_secrets_test.go   # Secret scrubbing tests
├── suite_c_bugfix_test.go    # Bug-fix learning tests
├── suite_d_multisession_test.go # Checkpoint/resume tests
├── metrics.go                # OpenTelemetry observability
├── workflow.go               # Temporal workflow definitions
└── activities.go             # Temporal activities
```

---

## Pass Criteria

| Metric | Threshold | Rationale |
|--------|-----------|-----------|
| Memory search confidence | >= 0.7 | Semantic similarity must be meaningful |
| Checkpoint resume | Success | Context preservation is binary |
| Secret scrubbing | 100% removal | Security is non-negotiable |
| Cross-dev knowledge transfer | Results found | Proves shared learning works |

---

## Documentation

| Document | Purpose |
|----------|---------|
| [TEST_SUITES.md](./TEST_SUITES.md) | Detailed test descriptions |
| [RUNNING_TESTS.md](./RUNNING_TESTS.md) | Execution guide and troubleshooting |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Framework internals |

---

## Known Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| Empty search results | Mock store filters by confidence | Use SharedStore, set confidence metadata |
| Wrong checkpoint resumed | ID filter not matching | Mock store now filters by `id` field |
| Import path errors | Module is `fyrsmithlabs` | Use `github.com/fyrsmithlabs/contextd` |

---

## Related Files

- [Integration Test Design](../plans/2025-12-10-integration-test-framework-design.md) - Original design document
- [Makefile](../../Makefile) - Test targets and commands
