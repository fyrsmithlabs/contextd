# Checkpoints Example

Save and resume conversation state at any point. Checkpoints let you preserve context, handle interruptions, and recover from errors without losing progress.

## Overview

Checkpoints are snapshots of your conversation state:

- **Save**: Create checkpoint at key milestones
- **List**: Find available checkpoints to resume from
- **Resume**: Restore conversation to a previous point

Think of checkpoints like Git commits for conversations - you can always roll back to a known good state.

## Quick Start

### Prerequisites

- contextd installed and configured with Claude Code
- See [QUICKSTART.md](../../QUICKSTART.md) for setup instructions

### Pattern in Action

```
User: "I need to refactor the authentication system"

Claude: "Let me start this complex refactoring. First, I'll save a checkpoint..."

[Uses MCP tool: checkpoint_save]
{
  "session_id": "session_abc123",
  "project_path": "/Users/me/my-api",
  "name": "before-auth-refactor",
  "description": "Clean state before starting authentication refactoring",
  "summary": "All tests passing, authentication working with basic auth",
  "context": "Current implementation uses basic auth. Plan to migrate to OAuth2.",
  "token_count": 4523
}

Response: Checkpoint saved (ID: ckpt_xyz789)

[Claude works on refactoring]
[Makes changes to 5 files]
[Oops - introduced breaking change]

User: "This broke production! Roll back to the checkpoint"

Claude: "Let me restore to the checkpoint from before the refactor..."

[Uses MCP tool: checkpoint_list]
{
  "project_path": "/Users/me/my-api",
  "limit": 5
}

Response:
- "before-auth-refactor" (15 minutes ago, 4523 tokens)
- "api-baseline" (2 hours ago, 3200 tokens)

[Uses MCP tool: checkpoint_resume]
{
  "checkpoint_id": "ckpt_xyz789",
  "tenant_id": "org_123",
  "resume_level": "full"
}

Response: Context restored to: "All tests passing, authentication working with basic auth"

Claude: "Restored to the checkpoint. Let me try a different approach this time..."
```

## Available MCP Tools

### checkpoint_save

Create a snapshot of current conversation state.

**When to use**:
- Before starting complex refactoring
- After completing a major milestone
- Before trying experimental approaches
- When manually requested by user

**Input**:
```json
{
  "session_id": "unique-session-id",
  "project_path": "/path/to/project",
  "name": "checkpoint-name",
  "description": "Human-readable description",
  "summary": "Brief summary of current state",
  "context": "Additional context about what's happening",
  "full_state": "Complete conversation history",
  "token_count": 4500,
  "metadata": {
    "task": "refactoring",
    "status": "in-progress"
  }
}
```

**Output**:
```json
{
  "id": "ckpt_abc123",
  "session_id": "unique-session-id",
  "summary": "Brief summary",
  "token_count": 4500,
  "auto_created": false
}
```

**Best practices**:
- Use descriptive names: "before-auth-migration", not "checkpoint1"
- Include current status in summary
- Save before risky operations
- Set auto_created: false for manual checkpoints

---

### checkpoint_list

Find available checkpoints to resume from.

**When to use**:
- User wants to see available restore points
- Need to resume after interruption
- Looking for a specific earlier state

**Input**:
```json
{
  "project_path": "/path/to/project",
  "session_id": "optional-filter",
  "limit": 10,
  "auto_only": false
}
```

**Output**:
```json
{
  "checkpoints": [
    {
      "id": "ckpt_abc123",
      "name": "before-refactor",
      "summary": "All tests passing",
      "token_count": 4500,
      "created_at": "2024-01-15T10:30:00Z",
      "auto_created": false
    }
  ],
  "count": 1
}
```

**Filters**:
- `session_id`: Only checkpoints from specific session
- `auto_only: true`: Only show auto-created checkpoints
- `limit`: Max results (default: 20)

---

### checkpoint_resume

Restore conversation to a previous checkpoint.

**When to use**:
- Recovery from errors or bad decisions
- Resume interrupted work
- Restore known good state

**Input**:
```json
{
  "checkpoint_id": "ckpt_abc123",
  "tenant_id": "org_123",
  "resume_level": "full"
}
```

**Resume levels**:
- `summary`: Just the brief summary (minimal context)
- `context`: Summary + context field (moderate detail)
- `full`: Complete conversation state (full restoration)

**Output**:
```json
{
  "checkpoint": {
    "id": "ckpt_abc123",
    "name": "before-refactor",
    "summary": "All tests passing, ready to start refactor",
    "context": "Authentication using basic auth...",
    "full_state": "[Complete conversation]",
    "token_count": 4500
  }
}
```

---

## Real-World Examples

### Example 1: Safe Refactoring

```
User: "Refactor the payment processing to use Stripe"

Claude: "This is a critical system. Let me save a checkpoint first..."

# Step 1: Save checkpoint
checkpoint_save(
  name="before-payment-refactor",
  summary="Payment processing working with legacy processor, all tests passing",
  context="Currently using internal payment processor. Migrating to Stripe API v2024."
)

Result: Checkpoint saved (ID: ckpt_safe001)

# Step 2: Start refactoring
[Makes changes to payment.go, payment_test.go, config.yaml]

# Step 3: Tests fail
User: "The tests are failing!"

# Step 4: Restore to safety
checkpoint_resume(
  checkpoint_id="ckpt_safe001",
  resume_level="full"
)

Claude: "Restored to working state. Let me try incremental changes this time..."
```

---

### Example 2: Interrupted Work

```
# Morning session
User: "Add user roles and permissions system"

Claude: "I'll implement RBAC..."

[Implements role model, database migrations]

# Auto-checkpoint triggered at 80% context threshold
checkpoint_save(
  name="auto-context-threshold-80",
  summary="RBAC: role model complete, migrations created, permissions next",
  auto_created=true
)

# Evening session
User: "Continue the RBAC work from this morning"

Claude: "Let me find the checkpoint from earlier..."

checkpoint_list(
  project_path="/Users/me/api",
  limit=5
)

Results:
- "auto-context-threshold-80" (6 hours ago) ← This one!
- "baseline-api" (yesterday)

checkpoint_resume(
  checkpoint_id="ckpt_auto123",
  resume_level="full"
)

Claude: "Restored! You had completed the role model and migrations. I'll continue with permissions..."
```

---

### Example 3: Experimental Approaches

```
User: "Optimize the database queries - they're too slow"

Claude: "Let me try adding indexes first..."

# Checkpoint 1
checkpoint_save(
  name="baseline-before-optimization",
  summary="Queries average 2.5s, no indexes beyond primary keys"
)

# Try approach 1: Indexes
[Adds indexes]
[Tests - still slow]

Claude: "Indexes didn't help enough. Let me try query restructuring..."

# Checkpoint 2
checkpoint_save(
  name="with-indexes",
  summary="Added 5 indexes, queries now 2.1s (16% improvement)"
)

# Try approach 2: Rewrite queries
[Rewrites queries with joins instead of N+1]
[Tests - much better!]

Claude: "Query restructuring worked! 300ms average now."

# Keep checkpoint 2 for reference, but main path is the rewrite
```

---

## Workflow Diagram

```
┌─────────────────────────────────────────────────┐
│          Start Complex Task                     │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  checkpoint_save     │  ← Save current state
        │  (Create safety net) │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  Do Risky Work       │  ← Refactor, experiment, etc.
        │  (Might break things)│
        └──────────┬───────────┘
                   │
           ┌───────┴────────┐
           │                │
           ▼                ▼
    ┌──────────┐      ┌──────────┐
    │ Success  │      │ Failed   │
    └────┬─────┘      └────┬─────┘
         │                 │
         │                 ▼
         │      ┌──────────────────────┐
         │      │  checkpoint_list     │  ← Find restore point
         │      └──────────┬───────────┘
         │                 │
         │                 ▼
         │      ┌──────────────────────┐
         │      │  checkpoint_resume   │  ← Restore safety
         │      └──────────┬───────────┘
         │                 │
         │                 ▼
         │      ┌──────────────────────┐
         │      │  Try Different Way   │
         │      └──────────────────────┘
         │
         ▼
    ┌─────────────────────────┐
    │  Continue Forward       │
    └─────────────────────────┘
```

## Auto-Checkpoints

contextd can automatically create checkpoints when context usage crosses thresholds:

### Configuration

```yaml
# ~/.config/contextd/config.yaml
checkpoint:
  auto_checkpoint:
    enabled: true
    thresholds:
      - 0.6  # Save at 60% context usage
      - 0.8  # Save at 80% context usage
      - 0.95 # Save at 95% context usage (emergency)
```

### Behavior

When context usage hits a threshold:
1. Auto-checkpoint created with name: `auto-context-threshold-{percentage}`
2. Checkpoint marked with `auto_created: true`
3. Summary includes token count and state at that point

### Benefits

- **Automatic safety net**: Never lose progress due to context limits
- **Resume interrupted sessions**: Pick up where you left off
- **Emergency recovery**: Always have a recent restore point

### Example

```
Context: 55,000 / 200,000 tokens (27%) - No auto-checkpoint yet

[Work continues...]

Context: 120,000 / 200,000 tokens (60%) - Auto-checkpoint triggered!
  → checkpoint_save(name="auto-context-threshold-60", auto_created=true)

[More work...]

Context: 160,000 / 200,000 tokens (80%) - Auto-checkpoint triggered!
  → checkpoint_save(name="auto-context-threshold-80", auto_created=true)

[Work continues...]

Context: 190,000 / 200,000 tokens (95%) - Emergency auto-checkpoint!
  → checkpoint_save(name="auto-context-threshold-95", auto_created=true)
```

## Best Practices

### ✅ DO

- **Save before risks**: Checkpoint before major refactors or experiments
- **Use descriptive names**: Clear names help you find the right checkpoint later
- **Include context**: Add details about state and next steps
- **Resume with appropriate level**: Use `summary` for quick context, `full` for complete restoration
- **Clean up old checkpoints**: Remove obsolete checkpoints to reduce noise

### ❌ DON'T

- **Don't skip checkpoints for risky work**: You'll regret it when things break
- **Don't use generic names**: "checkpoint1", "temp", "test" are useless later
- **Don't resume without listing first**: Review available checkpoints to pick the right one
- **Don't forget to continue**: After restoring, make progress or you'll loop forever

## Troubleshooting

### "Checkpoint not found" error

**Cause**: Checkpoint ID doesn't exist or wrong tenant_id

**Fix**:
```
# Step 1: List available checkpoints
checkpoint_list(project_path="/path/to/project")

# Step 2: Use correct ID from results
checkpoint_resume(checkpoint_id="ckpt_xyz789", ...)
```

---

### Restored checkpoint has too little context

**Cause**: Used `resume_level: "summary"` when you needed more

**Fix**:
```json
{
  "checkpoint_id": "same-id",
  "resume_level": "full"  // Use full instead of summary
}
```

---

### Too many auto-checkpoints cluttering results

**Cause**: Low thresholds creating frequent checkpoints

**Fix**:
```yaml
# Adjust thresholds to reduce frequency
checkpoint:
  auto_checkpoint:
    thresholds:
      - 0.75  # Only at 75%
      - 0.95  # And emergency at 95%
```

Or filter them out:
```json
{
  "auto_only": false,  // Exclude auto-created checkpoints
  "limit": 10
}
```

---

### Can't resume from checkpoint - "tenant mismatch"

**Cause**: Using wrong tenant_id

**Fix**: Ensure tenant_id matches the checkpoint's tenant (usually derived from git remote):
```
# tenant_id is automatically derived from project_path's git remote
# If manual override needed, use the same tenant that created the checkpoint
```

## Integration with Other Features

- **Session Lifecycle**: Combine with [memory_record](../session-lifecycle/) to preserve learnings across checkpoints
- **Context-Folding**: Use [branch_return](../context-folding/) for temporary branches, checkpoints for longer-term saves
- **Remediation**: Save checkpoints before applying [remediation fixes](../remediation/) to have rollback option

## Next Steps

- Learn [remediation](../remediation/) for error pattern reuse
- Try [context-folding](../context-folding/) for isolated subtask execution
- Explore [session-lifecycle](../session-lifecycle/) for cross-session memory

---

**Remember**: Checkpoints are your safety net. When in doubt, save a checkpoint before making changes you might want to undo.
