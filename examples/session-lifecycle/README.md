# Session Lifecycle Example

Learn the fundamental contextd workflow: **search → do → record**. This pattern enables AI agents to learn from past experiences and improve over time.

## Overview

The session lifecycle pattern is how contextd enables cross-session memory:

1. **Search** - Find relevant past strategies before starting work
2. **Do** - Execute the task, applying learned approaches
3. **Record** - Save new learnings for future sessions
4. **Feedback** - Rate what worked (optional but recommended)
5. **Outcome** - Track success/failure for confidence adjustment

## Quick Start

### Prerequisites

- contextd installed and configured with Claude Code
- See [QUICKSTART.md](../../QUICKSTART.md) for setup instructions

### Pattern in Action

```
User: "Implement error handling for the API"

Claude: "Let me search for past learnings about error handling..."

[Uses MCP tool: memory_search]
{
  "project_id": "my-api",
  "query": "error handling patterns API",
  "limit": 5
}

Response:
- Found: "Always wrap errors with context" (confidence: 0.85)
- Found: "Use custom error types for API responses" (confidence: 0.78)

Claude: "Based on past learnings, I'll implement error handling with context wrapping..."

[Implements the solution using retrieved strategies]

[Uses MCP tool: memory_record]
{
  "project_id": "my-api",
  "title": "API error handling with structured responses",
  "content": "Implemented error middleware that wraps all errors with: 1) HTTP status codes 2) Error codes for client logic 3) User-friendly messages 4) Debug context for logs",
  "outcome": "success",
  "confidence": 0.8,
  "tags": ["api", "errors", "middleware"]
}

Claude: "I've recorded this approach for future reference."

[Uses MCP tool: memory_feedback]
{
  "memory_id": "abc123",
  "helpful": true
}

Claude: "Marked the original memory as helpful (confidence boosted to 0.95)"
```

## Available MCP Tools

### memory_search

Find relevant past learnings before starting work.

**When to use**: At the start of any non-trivial task

**Input**:
```json
{
  "project_id": "my-project",
  "query": "what you're trying to do",
  "limit": 5,
  "min_confidence": 0.5
}
```

**Output**:
```json
{
  "memories": [
    {
      "id": "mem_abc123",
      "title": "Strategy name",
      "content": "Detailed approach...",
      "confidence": 0.85,
      "tags": ["tag1", "tag2"],
      "created_at": "2024-01-10T12:00:00Z"
    }
  ]
}
```

**Example prompts**:
- "Search for learnings about database migrations"
- "Find past strategies for handling rate limits"
- "Look for error handling patterns"

---

### memory_record

Save new learnings after completing a task.

**When to use**: After solving a problem or completing a task

**Input**:
```json
{
  "project_id": "my-project",
  "title": "Concise strategy name",
  "content": "Detailed explanation of what worked and why",
  "outcome": "success",
  "tags": ["relevant", "tags"]
}
```

**Output**:
```json
{
  "id": "mem_xyz789",
  "title": "Concise strategy name",
  "outcome": "success",
  "confidence": 0.7
}
```

**Best practices**:
- Record **why** something worked, not just **what** you did
- Include failure cases and gotchas
- Use specific, searchable tags
- Confidence starts at 0.7 and adjusts based on feedback/outcomes

---

### memory_feedback

Rate whether a retrieved memory was helpful.

**When to use**: After using a memory to complete a task

**Input**:
```json
{
  "memory_id": "mem_abc123",
  "helpful": true
}
```

**Effect**:
- `helpful: true` → Confidence increases (better ranking in future searches)
- `helpful: false` → Confidence decreases (less likely to appear)

---

### memory_outcome

Track task success/failure for confidence adjustment.

**When to use**: After completing a task that used a retrieved memory

**Input**:
```json
{
  "memory_id": "mem_abc123",
  "succeeded": true,
  "session_id": "optional-session-id"
}
```

**Output**:
```json
{
  "recorded": true,
  "new_confidence": 0.92,
  "message": "Outcome recorded successfully"
}
```

**Effect**:
- `succeeded: true` → Memory confidence increases
- `succeeded: false` → Memory confidence decreases

---

## Real-World Examples

### Example 1: Database Migration Task

```
User: "Add a new users table to the database"

# Step 1: Search
Claude uses: memory_search(project_id="api", query="database migration", limit=5)

Results:
- "Always create rollback migrations" (0.90)
- "Add indexes after data load" (0.82)

# Step 2: Apply learnings
Claude: "Based on past learnings, I'll create both up and down migrations..."

[Creates migration files]

# Step 3: Record new learning
Claude uses: memory_record(
  project_id="api",
  title="Users table with email uniqueness constraint",
  content="Created users table with: 1) UUID primary key 2) Unique index on email 3) Timestamps with defaults 4) Check constraint on email format",
  outcome="success"
)

# Step 4: Provide feedback
Claude uses: memory_feedback(memory_id="rollback-migration-memory", helpful=true)
```

---

### Example 2: API Error Handling

```
User: "Fix the error handling in the authentication endpoint"

# Step 1: Search
Claude uses: memory_search(project_id="api", query="API error handling auth", limit=3)

Results:
- "Return 401 for invalid credentials, not 403" (0.88)
- "Log failed auth attempts for security monitoring" (0.75)

# Step 2: Apply learnings
Claude: "I'll update the error responses to use proper status codes..."

[Updates code]

# Step 3: Record refinement
Claude uses: memory_record(
  project_id="api",
  title="Auth error responses with rate limit headers",
  content="Updated auth endpoint to: 1) Return 401 for invalid creds 2) Include Retry-After header on rate limit 3) Log attempts with IP and user-agent 4) Return generic message to prevent user enumeration",
  outcome="success"
)
```

---

### Example 3: Performance Optimization

```
User: "The API is slow under load"

# Step 1: Search
Claude uses: memory_search(project_id="api", query="performance optimization slow API", limit=5)

Results:
- "Add connection pooling for database" (0.92)
- "Cache frequently accessed data in Redis" (0.85)

# Step 2: Try first approach
Claude: "Let me try adding connection pooling first..."

[Implements connection pooling]
[Tests - still slow]

Claude uses: memory_outcome(memory_id="connection-pooling", succeeded=false)

# Step 3: Try second approach
[Adds Redis caching]
[Tests - much better]

Claude uses: memory_outcome(memory_id="redis-caching", succeeded=true)

# Step 4: Record combined strategy
Claude uses: memory_record(
  project_id="api",
  title="Connection pooling + Redis caching for read-heavy APIs",
  content="For read-heavy endpoints: 1) Use connection pool (max 20 conns) 2) Cache with Redis (TTL 5min) 3) Invalidate cache on writes 4) Use cache-aside pattern",
  outcome="success"
)
```

---

## Workflow Diagram

```
┌─────────────────────────────────────────────────┐
│              New Task Starts                    │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  memory_search       │  ← "Find past learnings"
        │  (What worked before?)│
        └──────────┬───────────┘
                   │
                   ├─ No results found ──► Start fresh
                   │
                   ├─ Results found ──►┐
                   │                    │
                   ▼                    │
        ┌──────────────────────┐       │
        │  Apply Learnings     │ ◄─────┘
        │  (Do the task)       │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  memory_record       │  ← "Save new learning"
        │  (What did you learn?)│
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  memory_feedback     │  ← "Was old memory helpful?"
        │  (Optional)          │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  memory_outcome      │  ← "Did it work?"
        │  (Success/failure)   │
        └──────────────────────┘
```

## Best Practices

### ✅ DO

- **Search first**: Always search before starting non-trivial tasks
- **Record learnings**: Save strategies after completing tasks
- **Be specific**: Use clear titles and detailed content
- **Tag appropriately**: Use consistent, searchable tags
- **Give feedback**: Mark memories as helpful/not helpful
- **Track outcomes**: Report success/failure for confidence tuning

### ❌ DON'T

- **Don't skip search**: Missing past learnings means repeating work
- **Don't record trivia**: Only save reusable strategies, not one-off fixes
- **Don't use vague titles**: "Fixed bug" is useless, "Added null check for API response" is helpful
- **Don't set high confidence for untested approaches**: Start at 0.6-0.7, let outcomes adjust it
- **Don't forget feedback**: It's how the system learns what works

## Troubleshooting

### "No memories found" but I know there should be

**Cause**: Query doesn't match stored content semantically

**Fix**:
```
# Instead of:
memory_search(query="auth problem")

# Try:
memory_search(query="authentication error handling OAuth")
```

Use specific domain terms that likely appear in stored memories.

---

### Confidence scores seem wrong

**Cause**: Not providing feedback or outcome tracking

**Fix**: Always call `memory_feedback` and `memory_outcome` after using memories. The system learns from your feedback.

---

### Too many irrelevant results

**Cause**: min_confidence too low

**Fix**:
```json
{
  "project_id": "my-project",
  "query": "...",
  "min_confidence": 0.7  // Raise this threshold
}
```

---

### Memory not appearing in future searches

**Possible causes**:
1. **Low confidence**: Set at least 0.6 when recording
2. **Poor tags**: Add relevant, searchable tags
3. **Vague content**: Include specific technical details in content
4. **Wrong project_id**: Make sure you're searching the same project

## Integration with Other Features

- **Checkpoints**: Combine with [checkpoint_save](../checkpoints/) to preserve entire conversation state
- **Remediation**: Use [remediation_record](../remediation/) for error-specific patterns
- **Repository**: Enhance with [semantic_search](../repository-indexing/) for code-level context

## Next Steps

- Try [checkpoints](../checkpoints/) to save conversation snapshots
- Learn [remediation](../remediation/) for error pattern reuse
- Explore [context-folding](../context-folding/) for token-efficient subtask execution

---

**Remember**: The session lifecycle is your foundation. Master search → do → record, and contextd becomes more useful with every task you complete.
