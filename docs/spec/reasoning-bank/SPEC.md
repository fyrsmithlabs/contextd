# ReasoningBank Specification

**Feature**: ReasoningBank (Layer 2)
**Status**: Implemented
**Created**: 2025-11-22
**Last Updated**: 2026-01-06

## Overview

ReasoningBank provides cross-session memory storage, learning from both successful and failed agent interactions. Memories are distilled strategies that improve agent efficiency over time.

The implementation uses a generic vectorstore abstraction (chromem by default, Qdrant optional) with database-per-project isolation via StoreProvider, and payload-based tenant filtering for security.

## Architecture

### Storage Model

- **VectorStore Abstraction**: Generic interface supporting chromem (embedded, default) or Qdrant (external)
- **Database-per-Project**: Each project gets its own vectorstore instance via `StoreProvider`
- **Tenant Isolation**: Payload-based filtering with fail-closed security (operations fail if tenant context missing)
- **Collection Naming**: Simple "memories" collection per project (no prefixes needed with per-project databases)

### Confidence System

The system uses a **Bayesian confidence model** that learns which signals predict memory usefulness:

- **Beta Distribution**: Each memory has `alpha` (positive evidence) and `beta` (negative evidence)
- **Confidence Score**: `alpha / (alpha + beta)` (Beta distribution mean)
- **Weighted Signals**: Project-specific learned weights for each signal type
- **Adaptive Learning**: System learns which signal types are reliable predictors per project

### Signal Types

| Signal | Source | When Recorded | Impact |
|--------|--------|---------------|--------|
| **Explicit** | `memory_feedback` tool | User rates memory helpful/unhelpful | Direct confidence adjustment with learned weight |
| **Usage** | `memory_search` tool | Memory retrieved in search results | Positive signal (memory was relevant enough to return) |
| **Outcome** | `memory_outcome` tool | Agent reports task success/failure after using memory | Strongest predictor of actual usefulness |

### Hybrid Storage Model

Signals are stored using a hybrid approach for efficiency:

- **Recent Signals** (last 30 days): Stored individually with full detail
- **Historical Signals** (older than 30 days): Rolled up into aggregated counts
- **Daily Rollup Process**: Background job migrates old signals to aggregates

## User Scenarios

### P1: Strategy Retrieval During Task

**Story**: As an agent working on a Go error handling task, I want relevant strategies automatically surfaced, so that I apply proven approaches.

**Acceptance Criteria**:
```gherkin
Given a ReasoningBank with 100 memories for the project
And 5 memories relate to "Go error handling"
When the agent calls memory_search with query "fix error handling in auth service"
Then the top 3-5 relevant memories are returned
And returned memories have confidence >= 0.7
And memories are ordered by similarity score
```

**Implementation**: ✅ `Service.Search()` with post-filtering by `MinConfidence = 0.7`

**Edge Cases**:
- No relevant memories exist → Returns empty array
- All memories below confidence threshold → Returns empty array after post-filter
- Multiple equally relevant memories → Returns up to requested limit

### P2: Learning from Success

**Story**: As a developer completing a successful debugging session, I want the successful approach captured, so that future sessions benefit.

**Acceptance Criteria**:
```gherkin
Given a completed session where agent fixed a bug
When the session ends with outcome="success"
Then the distiller extracts the successful strategy
And creates a memory item with title, description, content
And memory is tagged with relevant context (language, problem type)
And initial confidence is set to 0.6 (DistilledConfidence)
```

**Implementation**: ✅ `Distiller.DistillSession()` with `extractSuccessPatterns()`

### P3: Learning from Failure

**Story**: As a developer whose agent took a wrong approach, I want that anti-pattern captured, so that it's avoided in future.

**Acceptance Criteria**:
```gherkin
Given a session where agent tried approach X that failed
When explicit feedback marks the approach as unhelpful
Then the distiller extracts an anti-pattern
And creates memory with outcome="failure"
And memory includes "what went wrong" in formatted content
```

**Implementation**: ✅ `Distiller.DistillSession()` with `extractFailurePatterns()`

### P4: Explicit Memory Capture

**Story**: As a developer who discovered a useful pattern, I want to explicitly save it, so that my knowledge is preserved.

**Acceptance Criteria**:
```gherkin
Given a developer working in an agent session
When the developer calls memory_record with title, description, content
Then a new memory is created immediately
And memory bypasses distillation queue
And initial confidence is set to 0.8 (ExplicitRecordConfidence)
```

**Implementation**: ✅ `Service.Record()` via `memory_record` MCP tool

## Functional Requirements

### FR-001: Memory Storage
The system MUST store memories in a vectorstore using the generic `vectorstore.Store` interface. Implementations include:
- chromem (embedded, default)
- Qdrant (external, optional)

**Implementation**: ✅ `Service` uses `vectorstore.Store` and optional `vectorstore.StoreProvider`

### FR-002: Memory Schema
Memories MUST include: id, project_id, title, description, content, outcome, confidence, usage_count, tags, timestamps.

**Implementation**: ✅ `Memory` type in `types.go` with all required fields

### FR-003: Semantic Search
The system MUST retrieve memories by semantic similarity to query, filtered by confidence threshold.

**Implementation**: ✅ `Service.Search()` with post-filter by `MinConfidence = 0.7`

### FR-004: Project Scoping
Memories MUST belong to a project identified by `project_id`. Each project has isolated storage.

**Implementation**: ✅ Database-per-project via `StoreProvider.GetProjectStore(tenant, "", projectID)`

**Note**: Team and org scoping mentioned in original spec is not implemented. This is future functionality.

### FR-005: Confidence Tracking
Memory confidence MUST update based on feedback signals: explicit ratings, implicit success, task outcomes.

**Implementation**: ✅ `Service.Feedback()` and `Service.RecordOutcome()` update confidence

### FR-005a: Self-Improving Confidence (Bayesian)
The system MUST use Bayesian adaptive weighting to learn which signals predict memory usefulness:
- Each project maintains Beta distributions for signal weights (explicit, usage, outcome)
- Weights MUST update when explicit feedback validates/invalidates other signal predictions
- Initial priors: explicit=70% (7:3), usage=50% (5:5), outcome=50% (5:5)

**Implementation**: ✅ `ProjectWeights` type with `LearnFromFeedback()` method

### FR-005b: Multi-Signal Confidence
Memory confidence MUST be computed from multiple signal types:
- **Explicit signals**: User rates helpful/unhelpful via `memory_feedback`
- **Usage signals**: Memory retrieved in search results via `memory_search`
- **Outcome signals**: Agent reports task success/failure via `memory_outcome`

**Implementation**: ✅ `ConfidenceCalculator.ComputeConfidence()` using `ComputeConfidenceFromHybrid()`

### FR-005c: Hybrid Signal Storage
The system MUST store signals using hybrid storage:
- Event log for recent signals (last 30 days) with full detail
- Aggregated counts for older signals (storage efficiency)
- Daily rollup process to migrate events to aggregates

**Implementation**: ✅ `SignalStore` interface with `GetRecentSignals()`, `GetAggregate()`, `RollupOldSignals()`

### FR-005d: Outcome Reporting
The system MUST provide `memory_outcome` tool for agents to report task outcomes after using memories.

**Implementation**: ✅ `Service.RecordOutcome()` via `memory_outcome` MCP tool

### FR-006: Distillation Pipeline
The system MUST extract memories from completed sessions asynchronously.

**Implementation**: ✅ `Distiller` type with `DistillSession()` method

**Note**: Integration with actual session lifecycle hooks is external to ReasoningBank service.

### FR-007: Explicit Capture
Users MUST be able to create memories explicitly via `memory_record` tool.

**Implementation**: ✅ `Service.Record()` via `memory_record` MCP tool

### FR-008: Feedback Loop
Users MUST be able to provide feedback via `memory_feedback` tool, affecting confidence.

**Implementation**: ✅ `Service.Feedback()` via `memory_feedback` MCP tool

### FR-009: Outcome Differentiation
Memories MUST distinguish between success patterns (`outcome="success"`) and failure anti-patterns (`outcome="failure"`).

**Implementation**: ✅ `Outcome` type with `OutcomeSuccess` and `OutcomeFailure` constants

### FR-010: Tenant Isolation
All operations MUST enforce tenant context for security, using fail-closed approach.

**Implementation**: ✅ `vectorstore.ContextWithTenant()` required, operations fail if missing

## MCP Tools

### memory_search

**Purpose**: Search for relevant memories from past sessions

**Input**:
```json
{
  "project_id": "string (required)",
  "query": "string (required)",
  "limit": "int (optional, default: 5)"
}
```

**Output**:
```json
{
  "memories": [
    {
      "id": "uuid",
      "title": "string",
      "content": "string (scrubbed)",
      "outcome": "success|failure",
      "confidence": "float64",
      "tags": ["string"]
    }
  ],
  "count": "int"
}
```

**Behavior**:
- Performs semantic search using vectorstore
- Filters results to confidence >= 0.7
- Records usage signals for returned memories
- Scrubs content for secrets before returning

### memory_record

**Purpose**: Explicitly record a new memory/learning

**Input**:
```json
{
  "project_id": "string (required)",
  "title": "string (required)",
  "content": "string (required)",
  "outcome": "success|failure (required)",
  "tags": ["string (optional)"]
}
```

**Output**:
```json
{
  "id": "uuid",
  "title": "string",
  "outcome": "success|failure",
  "confidence": "float64 (0.8 for explicit records)"
}
```

**Behavior**:
- Sets confidence to 0.8 (ExplicitRecordConfidence)
- Bypasses distillation pipeline
- Creates memory immediately

### memory_feedback

**Purpose**: Rate a memory as helpful/unhelpful

**Input**:
```json
{
  "memory_id": "uuid (required)",
  "helpful": "bool (required)"
}
```

**Output**:
```json
{
  "memory_id": "uuid",
  "new_confidence": "float64",
  "helpful": "bool"
}
```

**Behavior**:
- Records explicit signal
- Learns from feedback (updates project weights)
- Recalculates confidence using Bayesian system

### memory_outcome

**Purpose**: Report task success/failure after using a memory

**Input**:
```json
{
  "memory_id": "uuid (required)",
  "succeeded": "bool (required)",
  "session_id": "string (optional)"
}
```

**Output**:
```json
{
  "recorded": "bool",
  "new_confidence": "float64",
  "message": "string"
}
```

**Behavior**:
- Records outcome signal
- Recalculates confidence using Bayesian system
- Strongest signal for learning actual usefulness

## Constants

| Constant | Value | Purpose |
|----------|-------|---------|
| `MinConfidence` | 0.7 | Minimum confidence for search results |
| `ExplicitRecordConfidence` | 0.8 | Initial confidence for explicitly recorded memories |
| `DistilledConfidence` | 0.6 | Initial confidence for distilled memories |
| `DefaultSearchLimit` | 10 | Default maximum search results |

## Success Criteria

### SC-001: Retrieval Relevance
>80% of injected memories should be rated "helpful" by users when feedback is collected.

**Status**: Aspirational (requires usage telemetry)

### SC-002: Context Efficiency
Average memory injection should consume <500 tokens while providing actionable guidance.

**Status**: Aspirational (requires token counting at MCP layer)

### SC-003: Learning Rate
>30% of successful sessions should yield extractable memories.

**Status**: Aspirational (depends on external distillation triggers)

### SC-004: Anti-Pattern Value
Teams using anti-pattern memories should see >20% reduction in repeated mistakes.

**Status**: Aspirational (requires long-term usage tracking)

### SC-005: Adaptive Weight Convergence
After 50+ explicit feedback events per project, learned signal weights should stabilize (variance <10% over 7 days).

**Status**: Aspirational (requires weight history tracking)

### SC-006: Confidence Calibration
Memories with confidence >0.8 should have >75% "helpful" rating when feedback is collected.

**Status**: Aspirational (requires correlation analysis)

## Implementation Notes

### Database-per-Project Isolation

The service supports two modes:

1. **Legacy Single Store**: All projects share one vectorstore with prefixed collection names
   - Constructor: `NewService(store, logger, opts...)`
   - Collection naming: `{projectID}_memories`
   - Operations: `Get()`, `Delete()` (enumerate all collections)

2. **StoreProvider (Recommended)**: Each project has its own database instance
   - Constructor: `NewServiceWithStoreProvider(stores, defaultTenant, logger, opts...)`
   - Collection naming: Simple `"memories"` (no prefix needed)
   - Operations: `GetByProjectID()`, `DeleteByProjectID()` (direct access)

### Tenant Context

All operations require tenant context via Go's `context.Context`:

```go
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",
    ProjectID: "contextd",
})

// Operations automatically filtered by tenant
results, err := service.Search(ctx, "contextd", "query", 10)
```

**Security**: Operations fail with `ErrMissingTenant` if context is missing (fail-closed).

### Signal Store

The `SignalStore` interface abstracts signal persistence:

- **Production**: Would use SQL database or vectorstore
- **Testing**: `InMemorySignalStore` provided for tests
- **Methods**: `StoreSignal`, `GetRecentSignals`, `GetAggregate`, `StoreProjectWeights`, `RollupOldSignals`

### Distillation Integration

The `Distiller` is a standalone component that can be called when sessions end:

```go
summary := reasoningbank.SessionSummary{
    SessionID:   "session-123",
    ProjectID:   "contextd",
    Outcome:     reasoningbank.SessionSuccess,
    Task:        "Fix authentication bug",
    Approach:    "Added JWT validation middleware",
    Result:      "Bug fixed, tests pass",
    Tags:        []string{"go", "auth", "jwt"},
    Duration:    15 * time.Minute,
    CompletedAt: time.Now(),
}

err := distiller.DistillSession(ctx, summary)
```

**Note**: Integration with session lifecycle is handled externally (e.g., via hooks or workflow orchestration).

## Testing

Key test coverage:

- **service_test.go**: 82% coverage
  - Memory CRUD operations
  - Search with confidence filtering
  - Feedback and outcome recording
  - StoreProvider integration
  - Tenant context enforcement

- **signals_test.go**: 100% coverage
  - Signal creation and validation
  - Aggregate rollup
  - Weight learning
  - Confidence calculation

- **confidence_test.go**: 100% coverage
  - Bayesian confidence computation
  - Hybrid signal integration
  - Weight normalization

## Migration Notes

When migrating from single-store to StoreProvider:

1. Use `NewServiceWithStoreProvider()` instead of `NewService()`
2. Replace `Get()` → `GetByProjectID()` for memory retrieval
3. Replace `Delete()` → `DeleteByProjectID()` for memory deletion
4. Ensure tenant context is set on all contexts before calling service methods

## Future Enhancements

1. **Team/Org Scoping**: Extend beyond project-level to support team and org-wide memories
2. **Metrics Dashboard**: Implement SC-001 through SC-006 with actual telemetry
3. **Persistent SignalStore**: Replace in-memory implementation with SQL/vectorstore backend
4. **Automatic Distillation**: Hook into session lifecycle to trigger distillation automatically
5. **Memory Merging**: Detect duplicate/similar memories and merge them
6. **Decay Function**: Lower confidence of unused memories over time
7. **Context Injection**: Automatically inject top memories into agent context at session start
