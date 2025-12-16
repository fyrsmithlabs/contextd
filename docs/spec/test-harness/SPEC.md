# Integration Test Harness Specification

**Status**: Active
**Last Updated**: 2025-12-11

---

## Critical Rules

- **NEVER** use mock stores for semantic similarity validation
- **ALWAYS** test secret scrubbing with real gitleaks rules
- **NEVER** trust confidence scores from mock stores (they return 0.9 for everything)

---

## Overview

The integration test harness validates contextd's cross-session memory system by simulating developer workflows. It tests that memories persist, secrets stay scrubbed, and knowledge transfers between developers.

### What It Tests

| Capability | Confidence | Method |
|------------|------------|--------|
| Secret scrubbing | 95% | Real gitleaks scrubber |
| Checkpoint save/resume | 90% | Real checkpoint service |
| Cross-developer sharing | 85% | Shared store pattern |
| API contracts | 90% | Interface validation |
| Semantic relevance | 60% | Mock store (known gap) |

### What It Does NOT Test

- Whether queries find semantically relevant memories
- Whether confidence scores reflect actual similarity
- Whether unrelated queries return empty results

---

## Test Suites

### Suite A: Policy Compliance

Tests that team policies (TDD, conventional commits, no secrets) are stored and retrieved.

| Test | Purpose | Assertion Type |
|------|---------|----------------|
| A.1 TDD Enforcement | Records TDD policy, retrieves when starting feature work | Binary + Threshold |
| A.2 Conventional Commits | Records commit policy, retrieves for commit guidance | Binary + Threshold |
| A.3 No Secrets | Records security policy, retrieves for security guidance | Binary + Behavioral |
| A.4 Secret Scrubbing | Records content with secrets, verifies automatic redaction | Behavioral |
| A.5 Defense-in-Depth | Verifies scrubbing on both write and read paths | Behavioral |

### Suite C: Bug-Fix Learning

Tests that bug fixes are captured and surfaced when similar bugs occur.

| Test | Purpose | Assertion Type |
|------|---------|----------------|
| C.1 Same Bug Retrieval | Exact bug description retrieves previous fix | Threshold (>=0.7) |
| C.2 Similar Bug Adaptation | Related bug retrieves adaptable fix | Threshold (>=0.5) |
| C.3 False Positive Prevention | Unrelated query avoids returning bug fixes | Behavioral |
| C.4 Confidence Decay | Negative feedback reduces memory confidence | Threshold (< initial) |
| C.5 Knowledge Transfer | Junior retrieves senior's fix | Binary + Threshold |

### Suite D: Multi-Session

Tests checkpoint save, resume, and cross-session memory persistence.

| Test | Purpose | Assertion Type |
|------|---------|----------------|
| D.1 Clean Resume | Save checkpoint, resume in new session | Binary |
| D.2 Checkpoint Selection | List and select specific checkpoint | Binary |
| D.3 Partial Work Resume | Progress summary preserved in checkpoint | Behavioral |
| D.4 Memory Accumulation | Memories from session 1 accessible in session 2 | Binary + Threshold |
| D.5 Checkpoint Stats | Tool calls tracked in session statistics | Threshold |

---

## Assertion System

Three assertion types validate different aspects:

### Binary Assertions

Pass/fail checks for tool execution.

```go
type BinaryAssertion struct {
    Check  string // "tool_called", "search_has_results"
    Method string // Tool or method name
    Target string // Expected target
}
```

**Examples**:
- `tool_called` - Did the developer call `memory_record`?
- `search_has_results` - Did search return any results?

### Threshold Assertions

Numeric comparisons against expected values.

```go
type ThresholdAssertion struct {
    Check     string  // "confidence", "result_count"
    Method    string  // "first_result", "latest_search"
    Threshold float64 // Expected value
    Operator  string  // ">=", "<", "=="
}
```

**Examples**:
- Confidence >= 0.7 for exact matches
- Confidence >= 0.5 for similar matches
- Result count >= 1

### Behavioral Assertions

Pattern matching on content.

```go
type BehavioralAssertion struct {
    Check            string   // "content_pattern"
    Method           string   // "regex_match"
    Patterns         []string // Patterns that SHOULD match
    NegativePatterns []string // Patterns that should NOT match
}
```

**Examples**:
- Content contains "nil check" (positive pattern)
- Content does NOT contain API keys (negative pattern)

---

## Developer Simulator

The `Developer` struct simulates a developer using contextd tools.

### Capabilities

| Method | Maps To | Purpose |
|--------|---------|---------|
| `StartContextd()` | Session start | Initialize services |
| `RecordMemory()` | `memory_record` | Store memory with auto-scrubbing |
| `SearchMemory()` | `memory_search` | Retrieve memories with auto-scrubbing |
| `GiveFeedback()` | `memory_feedback` | Update memory confidence |
| `SaveCheckpoint()` | `checkpoint_save` | Persist session state |
| `ListCheckpoints()` | `checkpoint_list` | List available checkpoints |
| `ResumeCheckpoint()` | `checkpoint_resume` | Restore session state |

### Shared Store Pattern

For cross-developer tests, multiple developers share one store:

```go
shared, _ := NewSharedStore(SharedStoreConfig{ProjectID: "team-project"})
devA, _ := NewDeveloperWithStore(DeveloperConfig{ID: "alice"}, shared)
devB, _ := NewDeveloperWithStore(DeveloperConfig{ID: "bob"}, shared)
```

Alice's recorded memories become searchable by Bob.

---

## Secret Scrubbing

The harness validates two-layer defense-in-depth:

### Layer 1: Write Scrubbing

`RecordMemory()` scrubs content before storage:

```go
scrubbedTitle := d.scrubber.Scrub(record.Title).Scrubbed
scrubbedContent := d.scrubber.Scrub(record.Content).Scrubbed
```

### Layer 2: Read Scrubbing

`SearchMemory()` scrubs results before returning:

```go
scrubbedTitle := d.scrubber.Scrub(r.Title).Scrubbed
scrubbedContent := d.scrubber.Scrub(r.Content).Scrubbed
```

### Tested Secret Types

| Secret Type | Pattern | Test Coverage |
|-------------|---------|---------------|
| AWS Access Key | `AKIA...` | Suite A.4 |
| GitHub PAT | `ghp_...` | Suite A.4 |
| JWT Token | `eyJ...` | Suite A.5 |
| Anthropic API Key | `sk-ant-...` | Suite A.5 |
| Database URI | `postgres://...` | Suite A.5 |

---

## Running Tests

```bash
# All integration tests
go test ./test/integration/framework/... -v

# Specific suite
go test ./test/integration/framework/... -run TestSuiteA -v
go test ./test/integration/framework/... -run TestSuiteC -v
go test ./test/integration/framework/... -run TestSuiteD -v

# With coverage
go test ./test/integration/framework/... -cover
```

---

## Success Criteria

### SC-001: Secret Detection Rate

>99% of known secret patterns must be detected and redacted.

### SC-002: Checkpoint Integrity

100% of checkpoint save/resume cycles must preserve summary and context.

### SC-003: Cross-Developer Retrieval

Memories recorded by Developer A must be retrievable by Developer B on the same project.

### SC-004: Confidence Decay

Negative feedback must reduce confidence by at least 0.05.

### SC-005: Assertion Coverage

Every test must use at least one assertion from the three-tier system.

---

## Files

| File | Purpose |
|------|---------|
| `developer.go` | Developer simulator and mock store |
| `assertions.go` | Three-tier assertion system |
| `metrics.go` | Test metrics and tracing |
| `suite_a_*.go` | Policy compliance tests |
| `suite_c_*.go` | Bug-fix learning tests |
| `suite_d_*.go` | Multi-session tests |
| `workflow.go` | Temporal workflow orchestration |

---

## Related Documents

- [ARCH.md](ARCH.md) - Architecture and component design
- [KNOWN-GAPS.md](KNOWN-GAPS.md) - Known limitations and future work
- [reasoning-bank/SPEC.md](../reasoning-bank/SPEC.md) - Memory system specification
