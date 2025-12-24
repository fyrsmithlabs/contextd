# ReasoningBank Specification

**Feature**: ReasoningBank (Layer 2)
**Status**: Draft
**Created**: 2025-11-22

## Overview

ReasoningBank provides cross-session memory storage, learning from both successful and failed agent interactions. Memories are distilled strategies that improve agent efficiency over time.

## User Scenarios

### P1: Strategy Retrieval During Task

**Story**: As an agent working on a Go error handling task, I want relevant strategies automatically surfaced, so that I apply proven approaches.

**Acceptance Criteria**:
```gherkin
Given a ReasoningBank with 100 memories for the project
And 5 memories relate to "Go error handling"
When the agent starts a task about "fix error handling in auth service"
Then the top 3 relevant memories are injected into context
And injected memories have confidence > 0.7
And total injection is <500 tokens
```

**Edge Cases**:
- No relevant memories exist (new project)
- All memories below confidence threshold
- Multiple equally relevant memories

### P2: Learning from Success

**Story**: As a developer completing a successful debugging session, I want the successful approach captured, so that future sessions benefit.

**Acceptance Criteria**:
```gherkin
Given a completed session where agent fixed a bug
When the session ends with outcome="success"
Then the distiller extracts the successful strategy
And creates a memory item with title, description, content
And memory is tagged with relevant context (language, problem type)
And initial confidence is set based on session signals
```

### P3: Learning from Failure

**Story**: As a developer whose agent took a wrong approach, I want that anti-pattern captured, so that it's avoided in future.

**Acceptance Criteria**:
```gherkin
Given a session where agent tried approach X that failed
When explicit feedback marks the approach as unhelpful
Then the distiller extracts an anti-pattern
And creates memory with outcome="failure"
And memory includes "what went wrong" and "what to do instead"
```

### P2: Explicit Memory Capture

**Story**: As a developer who discovered a useful pattern, I want to explicitly save it, so that my knowledge is preserved.

**Acceptance Criteria**:
```gherkin
Given a developer working in an agent session
When the developer calls memory_record with title, description, content
Then a new memory is created immediately
And memory bypasses distillation queue
And initial confidence is set to 0.8 (explicit capture boost)
```

## Functional Requirements

### FR-001: Memory Storage
The system MUST store memories in Qdrant with vector embeddings for semantic search.

### FR-002: Memory Schema
Memories MUST include: id, title, description, content, outcome, confidence, usage_count, tags, timestamps.

### FR-003: Semantic Search
The system MUST retrieve memories by semantic similarity to query, filtered by scope and confidence.

### FR-004: Scope Hierarchy
Memories MUST belong to a scope: project, team, or org. Search MUST cascade through hierarchy.

### FR-005: Confidence Tracking
Memory confidence MUST update based on feedback signals: explicit ratings, implicit success, code stability.

### FR-005a: Self-Improving Confidence (Bayesian)
The system MUST use Bayesian adaptive weighting to learn which signals predict memory usefulness:
- Each project maintains Beta distributions for signal weights (explicit, usage, outcome)
- Weights MUST update when explicit feedback validates/invalidates other signal predictions
- Initial priors: explicit=70% (7:3), usage=50% (5:5), outcome=50% (5:5)

### FR-005b: Multi-Signal Confidence
Memory confidence MUST be computed from multiple signal types:
- **Explicit signals**: User rates helpful/unhelpful via `memory_feedback`
- **Usage signals**: Memory retrieved in search results via `memory_search`
- **Outcome signals**: Agent reports task success/failure via `memory_outcome`

### FR-005c: Hybrid Signal Storage
The system MUST store signals using hybrid storage:
- Event log for recent signals (last 30 days) with full detail
- Aggregated counts for older signals (storage efficiency)
- Daily rollup process to migrate events to aggregates

### FR-005d: Outcome Reporting
The system MUST provide `memory_outcome` tool for agents to report task outcomes after using memories.

### FR-006: Distillation Pipeline
The system MUST extract memories from completed sessions asynchronously.

### FR-007: Explicit Capture
Users MUST be able to create memories explicitly via `memory_record` tool.

### FR-008: Feedback Loop
Users MUST be able to provide feedback via `memory_feedback` tool, affecting confidence.

### FR-009: Outcome Differentiation
Memories MUST distinguish between success patterns and failure anti-patterns.

## Success Criteria

### SC-001: Retrieval Relevance
>80% of injected memories should be rated "helpful" by users when feedback is collected.

### SC-002: Context Efficiency
Average memory injection should consume <500 tokens while providing actionable guidance.

### SC-003: Learning Rate
>30% of successful sessions should yield extractable memories.

### SC-004: Anti-Pattern Value
Teams using anti-pattern memories should see >20% reduction in repeated mistakes.

### SC-005: Adaptive Weight Convergence
After 50+ explicit feedback events per project, learned signal weights should stabilize (variance <10% over 7 days).

### SC-006: Confidence Calibration
Memories with confidence >0.8 should have >75% "helpful" rating when feedback is collected.
