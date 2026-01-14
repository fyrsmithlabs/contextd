---
name: context-folding
description: Use when executing complex sub-tasks that need context isolation - creates branches with token budgets that auto-cleanup on return
---

# Context Folding

Create isolated branches for complex sub-tasks. Each branch has its own token budget and cleans up on return, preventing context bloat.

## When to Use

**Use context folding when:**
- Investigating a problem requiring many file reads
- Running exploratory work that would bloat context
- Multi-step sub-tasks with verbose intermediate steps

**Don't use when:**
- Task is simple (< 3 steps)
- Results need to stay in main context
- You're at end of session anyway

## Tools

### branch_create

```json
{
  "session_id": "my-session",
  "description": "Brief description of the sub-task",
  "prompt": "Detailed instructions for the branch",
  "budget": 4096,
  "timeout_seconds": 300
}
```

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `session_id` | Yes | - | Session identifier |
| `description` | Yes | - | Brief description (shown in status) |
| `prompt` | No | - | Detailed instructions |
| `budget` | No | 8192 | Token budget |
| `timeout_seconds` | No | 300 | Auto-return timeout |

### branch_return

```json
{
  "branch_id": "br_abc123",
  "message": "Summary of findings"
}
```

Message is scrubbed for secrets before returning to parent context.

### branch_status

```json
{ "branch_id": "br_abc123" }
```

Or check active branch:
```json
{ "session_id": "my-session" }
```

Returns: `status`, `budget_total`, `budget_used`, `budget_remaining`, `depth`

## Workflow

```
1. branch_create(session_id, description, budget) -> branch_id
2. Do work in the branch (read files, search, analyze)
3. branch_status(branch_id) - optional budget check
4. branch_return(branch_id, message) - summary returns, branch cleaned up
```

## Example

```
# Create branch for code analysis
branch_create(
  session_id: "main",
  description: "Analyze auth module structure",
  budget: 4096
) -> branch_id: "br_abc123"

# Do analysis...
semantic_search("authentication handlers")
Read files, analyze patterns...

# Return with summary
branch_return(
  branch_id: "br_abc123",
  message: "Auth module has 3 handlers: login, logout, refresh. Uses JWT with 15min expiry."
)
# Summary in main context, branch cleaned up
```

## Best Practices

| Practice | Why |
|----------|-----|
| Keep summaries concise | Only essential info returns to parent |
| Set appropriate budget | 4096 for small tasks, 8192+ for larger |
| Return early if done | Don't waste budget |
| Use for exploratory work | Keeps main context clean |

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Not returning from branch | Always call `branch_return` |
| Huge return messages | Summarize, don't dump |
| Nesting too deep | Max depth is 3 |
| Ignoring timeout | Branch auto-returns on timeout |
