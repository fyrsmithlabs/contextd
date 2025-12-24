---
name: context-folding
description: Use when executing complex sub-tasks that need context isolation - creates branches with token budgets that auto-cleanup on return
---

# Context Folding

## Overview

Context folding creates isolated branches for complex sub-tasks. Each branch has its own token budget and automatically cleans up when you return, preventing context bloat in the main conversation.

## When to Use

**Use context folding when:**
- Executing a complex multi-step sub-task
- Investigating a problem that requires reading many files
- Running operations that would bloat the main context
- You need to isolate exploratory work

**Don't use when:**
- Task is simple (< 3 steps)
- Results need to stay in main context
- You're at the end of a session anyway

## Tools

### branch_create

Create an isolated branch with a token budget:
```json
{
  "session_id": "my-session",
  "description": "Brief description of the sub-task",
  "prompt": "Detailed instructions for what to do in the branch",
  "budget": 4096,
  "timeout_seconds": 300
}
```

**Parameters:**
- `session_id` (required): Session identifier
- `description` (required): Brief description (shown in status)
- `prompt`: Detailed instructions
- `budget`: Token budget (default: 8192)
- `timeout_seconds`: Auto-return timeout (default: 300)

### branch_return

Return from a branch with results (auto-scrubs secrets):
```json
{
  "branch_id": "br_abc123",
  "message": "Summary of what was found/accomplished"
}
```

The message is scrubbed for secrets before being returned to the parent context. Child branches are force-returned first.

### branch_status

Check branch state and budget usage:
```json
{
  "branch_id": "br_abc123"
}
```

Or check active branch for a session:
```json
{
  "session_id": "my-session"
}
```

**Response fields:**
- `status`: "none", "active", or "completed"
- `budget_total`: Allocated token budget
- `budget_used`: Tokens consumed
- `budget_remaining`: Tokens left
- `depth`: Nesting level (0 = top level)

## Workflow

```
1. branch_create(session_id, description, budget)
   → Get branch_id
   → Budget allocated

2. Do work in the branch
   → Read files, search, analyze
   → All within budget

3. branch_status(branch_id)
   → Check budget usage (optional)

4. branch_return(branch_id, message)
   → Summary returned to parent
   → Secrets scrubbed
   → Branch cleaned up
```

## Example

```
# Create branch for code analysis
branch_create(
  session_id: "main",
  description: "Analyze auth module structure",
  budget: 4096
)
→ branch_id: "br_abc123"

# Do analysis work...
semantic_search("authentication handlers")
Read files, analyze patterns...

# Return with summary
branch_return(
  branch_id: "br_abc123",
  message: "Auth module has 3 handlers: login, logout, refresh. Uses JWT with 15min expiry."
)
→ Summary available in main context
→ Branch cleaned up
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
