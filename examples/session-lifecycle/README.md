# Session Lifecycle Example

This example demonstrates the complete session lifecycle pattern in contextd: **search → do → record**. This is the fundamental workflow for building AI agents that learn from experience across sessions.

## Overview

The session lifecycle pattern enables agents to:

1. **Search** for relevant past experiences before starting work
2. **Do** the actual task, applying learned strategies
3. **Record** new learnings for future sessions
4. **Feedback** on what worked or didn't
5. **Outcome** tracking for confidence adjustment

This pattern is how contextd enables cross-session memory and continuous improvement.

## The Pattern

```
┌─────────────────────────────────────────────────────────┐
│                    Session Start                         │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
         ┌──────────────────────┐
         │  1. Search Memories  │  ← memory_search
         │  (Past strategies)   │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │   2. Apply Learning  │  ← Use retrieved strategies
         │   (Perform task)     │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  3. Record Learning  │  ← memory_record
         │  (Save new strategy) │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  4. Provide Feedback │  ← memory_feedback (optional)
         │  (Rate usefulness)   │
         └──────────┬───────────┘
                    │
                    ▼
         ┌──────────────────────┐
         │  5. Record Outcome   │  ← memory_outcome
         │  (Success/failure)   │
         └──────────────────────┘
```

## Quick Start

### Prerequisites

- contextd running (either as MCP server or standalone)
- Go 1.25+ installed

### Run the Example

```bash
# From the examples/session-lifecycle directory
go run main.go

# Or build first
go build -o session-lifecycle
./session-lifecycle
```

### Expected Output

```
Session Lifecycle Example - Demonstrating search->do->record pattern
=====================================================================

Step 1: Searching for existing memories about "error handling in Go"...
Found 2 relevant memories:
  - [ID: abc123] Always wrap errors with context (confidence: 0.85)
  - [ID: def456] Use errors.Is for error comparison (confidence: 0.78)

Step 2: Performing task using retrieved strategies...
Task: Implementing error handling based on past learnings
✓ Applied strategy: Always wrap errors with context

Step 3: Recording new learning...
✓ Recorded memory: "Use %w verb for error wrapping" (ID: ghi789)

Step 4: Providing feedback on helpful memory...
✓ Marked memory abc123 as helpful (new confidence: 0.95)

Step 5: Recording successful outcome...
✓ Task succeeded using memory abc123 (confidence updated)

Session complete! New memories are available for future sessions.
```

## How It Works

### 1. Memory Search

Before starting work, search for relevant past experiences:

```go
// Search for memories related to the current task
results, err := service.Search(ctx, projectID, "error handling in Go", 5)
if err != nil {
    log.Fatal(err)
}

// Review and apply relevant strategies
for _, memory := range results {
    fmt.Printf("- [%s] %s (confidence: %.2f)\n",
        memory.ID, memory.Title, memory.Confidence)
    // Apply the strategy...
}
```

**Why search first?**
- Avoid repeating past mistakes
- Apply proven strategies immediately
- Build on previous learnings

### 2. Perform Task

Execute the actual work, applying retrieved strategies:

```go
// Simulate performing a task with learned strategies
fmt.Println("Applying strategy: Always wrap errors with context")

// Your actual task implementation here
result, err := doSomeWork()
if err != nil {
    // Apply learned strategy: wrap errors
    return fmt.Errorf("doing work: %w", err)
}
```

### 3. Record Learning

After completing work, record new insights for future sessions:

```go
// Create a new memory from this session
memory := &reasoningbank.Memory{
    ID:          uuid.New().String(),
    ProjectID:   projectID,
    Title:       "Use %w verb for error wrapping",
    Content:     "When wrapping errors in Go, use %w verb instead of %v to preserve error chain for errors.Is/As",
    Description: "Learned from session: error-handling-task",
    Outcome:     reasoningbank.OutcomeSuccess,
    Tags:        []string{"go", "errors", "best-practice"},
    Confidence:  0.8, // Initial confidence for explicit records
    State:       reasoningbank.MemoryStateActive,
    CreatedAt:   time.Now(),
    UpdatedAt:   time.Now(),
}

// Store the memory
err = service.Record(ctx, memory)
if err != nil {
    log.Fatal(err)
}
```

**What to record?**
- ✅ Strategies that worked
- ✅ Common pitfalls to avoid
- ✅ Performance insights
- ✅ Configuration patterns
- ❌ Don't record obvious facts
- ❌ Don't record session-specific details

### 4. Provide Feedback (Optional)

Rate the usefulness of retrieved memories:

```go
// If a memory was helpful
newConfidence, err := service.Feedback(ctx, memoryID, true, "")
if err != nil {
    log.Printf("Warning: feedback failed: %v", err)
}
fmt.Printf("Memory confidence updated to %.2f\n", newConfidence)

// If a memory was NOT helpful
newConfidence, err := service.Feedback(ctx, memoryID, false, "")
```

**Feedback effects:**
- Helpful: Confidence +0.1 (capped at 1.0)
- Not helpful: Confidence -0.15 (floored at 0.0)
- Memories below 0.7 confidence are filtered from searches

### 5. Record Outcome

Track whether the memory actually helped solve the task:

```go
// Record successful outcome
newConfidence, err := service.RecordOutcome(ctx, memoryID, true, sessionID)
if err != nil {
    log.Printf("Warning: outcome recording failed: %v", err)
}

// Record failure outcome
newConfidence, err := service.RecordOutcome(ctx, memoryID, false, sessionID)
```

**Why track outcomes?**
- Confidence scoring learns which memories are truly useful
- Failed outcomes reduce confidence (the memory didn't help)
- Successful outcomes increase confidence (the memory was valuable)

## Real-World Usage

### Example 1: API Error Handling

```go
// Session 1: First encounter with API error
query := "handling API rate limits"
results, _ := service.Search(ctx, projectID, query, 5)
// No existing memories found

// Implement solution through trial and error
// ... implementation ...

// Record what worked
memory := &reasoningbank.Memory{
    Title:   "Use exponential backoff for rate limits",
    Content: "When API returns 429, wait exponentially: 1s, 2s, 4s, 8s...",
    Outcome: reasoningbank.OutcomeSuccess,
    Tags:    []string{"api", "rate-limit", "retry"},
}
service.Record(ctx, memory)

// Session 2: Same API, different endpoint
results, _ = service.Search(ctx, projectID, "handling API rate limits", 5)
// Found 1 memory: "Use exponential backoff for rate limits"
// Apply immediately - no trial and error needed!
```

### Example 2: Learning from Failures

```go
// Record an anti-pattern (what NOT to do)
antiPattern := &reasoningbank.Memory{
    Title:       "Don't use time.Sleep for retries",
    Content:     "Fixed delays cause thundering herd. Use jittered exponential backoff instead.",
    Description: "Anti-pattern learned from session: api-retry-disaster",
    Outcome:     reasoningbank.OutcomeFailure, // Mark as anti-pattern
    Tags:        []string{"api", "retry", "anti-pattern"},
}
service.Record(ctx, antiPattern)

// Future searches will retrieve this as a warning
```

## Integration with MCP

When running as an MCP server, these operations are exposed as tools:

| Go Method | MCP Tool | Purpose |
|-----------|----------|---------|
| `service.Search()` | `memory_search` | Find relevant memories |
| `service.Record()` | `memory_record` | Save new memory |
| `service.Feedback()` | `memory_feedback` | Rate memory usefulness |
| `service.RecordOutcome()` | `memory_outcome` | Track task success/failure |

Example MCP tool call:

```json
{
  "tool": "memory_search",
  "arguments": {
    "project_id": "myproject",
    "query": "error handling in Go",
    "limit": 5
  }
}
```

## Confidence Scoring

Memories start with initial confidence based on how they were created:

| Source | Initial Confidence |
|--------|-------------------|
| Explicit record (`memory_record`) | 0.8 |
| Distilled from sessions | 0.6 |
| Manual creation | 0.5 (default) |

Confidence adjusts based on signals:

- **Positive feedback**: +0.1
- **Negative feedback**: -0.15
- **Successful outcome**: Bayesian update (typically +0.05 to +0.15)
- **Failed outcome**: Bayesian update (typically -0.05 to -0.10)

Search results are:
- Filtered to confidence ≥ 0.7
- Sorted by similarity × confidence
- Consolidated memories get 20% boost

## Best Practices

### Do's ✅

1. **Search before every task** - Don't reinvent the wheel
2. **Record insights immediately** - Capture while fresh
3. **Use descriptive titles** - Make memories searchable
4. **Tag appropriately** - Enable filtering and discovery
5. **Provide feedback** - Help the system learn what's valuable
6. **Record outcomes** - Track actual success/failure

### Don'ts ❌

1. **Don't record trivial facts** - "Go uses semicolons" is not useful
2. **Don't record session-specific data** - File paths, user names, etc.
3. **Don't skip outcome recording** - It's how confidence learns
4. **Don't over-tag** - 3-5 relevant tags is plenty
5. **Don't record before validating** - Only save what actually worked

## Troubleshooting

### "No memories found" on first search

**Expected behavior.** The memory base is empty on first run. After recording memories in this session, future sessions will find them.

### "Tenant ID not configured"

**Cause:** ReasoningBank service requires a tenant ID for multi-tenant isolation.

**Solution:**
```go
service, err := reasoningbank.NewServiceWithStoreProvider(
    stores,
    "myusername", // Tenant ID (typically git username)
    logger,
)
```

### Memories have low confidence scores

**Cause:** Memories without positive feedback/outcomes decay in confidence.

**Solution:** Consistently provide feedback and record outcomes when using memories. The system learns which memories are valuable based on actual usage patterns.

### Search returns irrelevant results

**Cause:** Query is too broad or memories lack good titles/tags.

**Solutions:**
1. Use more specific search queries
2. Improve memory titles (they're used for semantic search)
3. Add relevant tags for filtering
4. Mark unhelpful memories with negative feedback

## Next Steps

- **Checkpoints**: See `examples/checkpoints/` for saving/resuming context
- **Remediation**: See `examples/remediation/` for error pattern tracking
- **Repository Indexing**: See `examples/repository-indexing/` for semantic code search
- **Context-Folding**: See `examples/context-folding/` for isolated subtask execution

## See Also

- ReasoningBank spec: `docs/spec/reasoningbank/`
- Confidence scoring: `internal/reasoningbank/confidence.go`
- MCP tools: `internal/mcp/handlers/memory.go`
- Integration tests: `test/integration/framework/workflow.go`
