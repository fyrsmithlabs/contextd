# Integration Test Harness Architecture

**Status**: Active
**Last Updated**: 2025-12-11

---

## Component Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Test Suite (Go)                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │   Suite A    │  │   Suite C    │  │   Suite D    │              │
│  │   Policy     │  │   Bug-fix    │  │ Multi-session│              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                 │                 │                       │
│         └─────────────────┼─────────────────┘                       │
│                           │                                         │
│                    ┌──────▼──────┐                                  │
│                    │  Developer  │                                  │
│                    │  Simulator  │                                  │
│                    └──────┬──────┘                                  │
│                           │                                         │
│         ┌─────────────────┼─────────────────┐                       │
│         │                 │                 │                       │
│  ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐               │
│  │ Reasoning   │   │ Checkpoint  │   │  Secrets    │               │
│  │    Bank     │   │   Service   │   │  Scrubber   │               │
│  └──────┬──────┘   └──────┬──────┘   └─────────────┘               │
│         │                 │                                         │
│         └────────┬────────┘                                         │
│                  │                                                  │
│           ┌──────▼──────┐                                           │
│           │ Vector Store │                                          │
│           │ (Mock/Real)  │                                          │
│           └─────────────┘                                           │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### Developer Simulator

Simulates a developer using contextd MCP tools.

```go
type Developer struct {
    id        string
    tenantID  string
    projectID string

    reasoningBank     *reasoningbank.Service
    checkpointService checkpoint.Service
    vectorStore       vectorstore.Store
    scrubber          secrets.Scrubber

    stats     SessionStats
    sessionID string
}
```

**Responsibilities**:
- Start/stop contextd services
- Record and search memories
- Save and resume checkpoints
- Track session statistics
- Apply secret scrubbing

### Shared Store

Enables cross-developer testing scenarios.

```go
type SharedStore struct {
    store  vectorstore.Store
    logger *zap.Logger
}
```

**Usage**:
- Multiple developers share one store instance
- Simulates team knowledge sharing
- Isolates tests via unique project IDs

### Mock Vector Store

In-memory store for deterministic testing.

```go
type mockVectorStore struct {
    mu          sync.RWMutex
    collections map[string][]vectorstore.Document
}
```

**Behavior**:
- Returns all documents matching filters
- Ignores query semantics (no embedding similarity)
- Returns `Score: 0.9` for all results
- Supports exact-match metadata filters only

**Limitation**: Cannot test semantic relevance. See [KNOWN-GAPS.md](KNOWN-GAPS.md).

### Test Embedder

Deterministic embedding generator for testing.

```go
type testEmbedder struct {
    vectorSize int
}
```

**Behavior**:
- Creates normalized vectors from text hash
- Same text produces same embedding
- Different texts produce different embeddings
- Used with real chromem store (non-shared scenarios)

---

## Service Integration

### Real Services Used

| Service | Package | Notes |
|---------|---------|-------|
| ReasoningBank | `internal/reasoningbank` | Full production service |
| Checkpoint | `internal/checkpoint` | Full production service |
| Secrets | `internal/secrets` | Real gitleaks rules |

### Mock Services Used

| Service | Package | Notes |
|---------|---------|-------|
| VectorStore | `test/integration/framework` | Mock for shared store scenarios |

---

## Data Flow

### Record Memory Flow

```
Developer.RecordMemory(title, content, tags)
    │
    ├─► Scrubber.Scrub(title)     ─► Redact secrets
    ├─► Scrubber.Scrub(content)   ─► Redact secrets
    │
    ├─► reasoningbank.NewMemory() ─► Create memory struct
    │
    └─► ReasoningBank.Record()    ─► Store in vector store
```

### Search Memory Flow

```
Developer.SearchMemory(query, limit)
    │
    ├─► ReasoningBank.Search()    ─► Query vector store
    │       │
    │       └─► Post-filter by confidence (>=0.7)
    │
    └─► Scrubber.Scrub(result)    ─► Defense-in-depth
```

### Checkpoint Flow

```
Developer.SaveCheckpoint(name, summary, context)
    │
    └─► checkpoint.Service.Save() ─► Store in vector store

Developer.ResumeCheckpoint(id)
    │
    └─► checkpoint.Service.Resume() ─► Retrieve and return
```

---

## Assertion Architecture

### Evaluation Pipeline

```
Test Function
    │
    ├─► Execute developer actions
    │
    ├─► Build AssertionSet
    │       ├─► Binary assertions
    │       ├─► Threshold assertions
    │       └─► Behavioral assertions
    │
    ├─► EvaluateAssertionSet(assertions, sessionResult)
    │       │
    │       ├─► EvaluateBinaryAssertion()
    │       ├─► EvaluateThresholdAssertion()
    │       └─► EvaluateBehavioralAssertion()
    │
    └─► AllPassed(results) ─► Test verdict
```

### Session Result

Captures test execution state for assertion evaluation.

```go
type SessionResult struct {
    Developer     DeveloperConfig
    MemoryIDs     []string
    SearchResults [][]MemoryResult
    Checkpoints   []CheckpointResult
}
```

---

## Temporal Workflow Integration

### Workflow Hierarchy

```
TestOrchestratorWorkflow
    │
    ├─► PolicyComplianceWorkflow (parallel)
    ├─► BugfixLearningWorkflow   (parallel)
    └─► MultiSessionWorkflow     (parallel)
            │
            └─► DeveloperSessionWorkflow
                    │
                    ├─► StartContextdActivity
                    ├─► RecordMemoryActivity
                    ├─► SearchMemoryActivity
                    └─► StopContextdActivity
```

### Activities

| Activity | Purpose |
|----------|---------|
| `StartContextdActivity` | Initialize developer services |
| `RecordMemoryActivity` | Record a memory |
| `SearchMemoryActivity` | Search for memories |
| `CheckpointSaveActivity` | Save a checkpoint |
| `CheckpointResumeActivity` | Resume from checkpoint |
| `StopContextdActivity` | Clean up services |

---

## Metrics and Observability

### Tracked Metrics

| Metric | Type | Purpose |
|--------|------|---------|
| `memory_search_hits` | Counter | Searches that found results |
| `memory_search_misses` | Counter | Searches with no results |
| `cross_developer_searches` | Counter | Searches finding other devs' memories |
| `checkpoint_saves` | Counter | Successful checkpoint saves |
| `checkpoint_failures` | Counter | Failed checkpoint operations |
| `test_passes` | Counter | Tests that passed |
| `test_failures` | Counter | Tests that failed |
| `suite_duration` | Histogram | Time to run test suite |

### Span Creation

```go
metrics.StartSuiteSpan(ctx, "SuiteA")
defer metrics.EndSuiteSpan()

metrics.StartTestSpan(ctx, "A4_SecretScrubbing")
defer metrics.EndTestSpan()
```

---

## Extension Points

### Adding New Test Suites

1. Create `suite_X_*.go` files
2. Use `NewDeveloperWithStore()` for isolation
3. Build `AssertionSet` with appropriate assertions
4. Add suite to `TestOrchestratorWorkflow` if using Temporal

### Adding New Assertion Types

1. Define struct in `assertions.go`
2. Implement `Evaluate*Assertion()` function
3. Add to `EvaluateAssertionSet()` dispatch

### Adding New Developer Actions

1. Add method to `Developer` struct
2. Update `SessionStats` tracking
3. Add corresponding Temporal activity if needed

---

## Related Documents

- [SPEC.md](SPEC.md) - Functional specification
- [KNOWN-GAPS.md](KNOWN-GAPS.md) - Known limitations
