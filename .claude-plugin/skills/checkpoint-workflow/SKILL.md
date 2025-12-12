---
name: checkpoint-workflow
description: Use when context approaching 70% capacity, during long-running tasks, or before risky operations - saves and resumes session state with checkpoint_save, checkpoint_list, and checkpoint_resume
---

# Checkpoint Workflow

## Overview

Checkpoints preserve your work when context gets full. Save before overflow, resume later with full context.

**CRITICAL**: Checkpoints are USELESS without meaningful summaries. Generic text like "Context at 70% threshold" tells you nothing when resuming. Always include what you did, what's in progress, and what's next.

## When to Use

**Proactive (recommended)**:
- Context at or above 70% capacity
- Long-running tasks (multi-hour sessions)
- Before attempting risky refactors
- End of work session

**Reactive**:
- Context overflow warning
- Session interrupted unexpectedly
- Switching to different task

## The Checkpoint Cycle

```
┌─────────────────────────────────────────┐
│  SAVE when context is high              │
│  checkpoint_save(session_id, tenant_id, │
│    project_path, summary, context...)   │
├─────────────────────────────────────────┤
│  LIST to find previous work             │
│  checkpoint_list(tenant_id,             │
│    project_path, limit)                 │
├─────────────────────────────────────────┤
│  RESUME at appropriate level            │
│  checkpoint_resume(checkpoint_id,       │
│    tenant_id, level)                    │
└─────────────────────────────────────────┘
```

## Tool Reference

### checkpoint_save

```json
{
  "session_id": "session_abc123",
  "tenant_id": "fyrsmithlabs",
  "project_path": "/home/user/projects/contextd",
  "name": "Feature implementation checkpoint",
  "description": "Implementing skills system for contextd",
  "summary": "Completed: spec, 2 of 4 skills. Next: checkpoint-workflow and error-remediation skills.",
  "context": "Working on contextd-marketplace repo...",
  "full_state": "Complete conversation and file state...",
  "token_count": 45000,
  "threshold": 0.7,
  "auto_created": false
}
```

**Summary tips**:
- What was accomplished
- What's in progress
- What's next
- Key decisions made

**Example Response:**
```json
{
  "checkpoint_id": "cp_xyz789",
  "status": "saved",
  "created_at": "2025-12-10T16:45:00Z",
  "token_count": 45000,
  "message": "Checkpoint saved successfully"
}
```

### checkpoint_list

```json
{
  "tenant_id": "fyrsmithlabs",
  "project_path": "/home/user/projects/contextd",
  "limit": 10
}
```

Returns checkpoints sorted by recency.

### checkpoint_resume

```json
{
  "checkpoint_id": "cp_xyz789",
  "tenant_id": "fyrsmithlabs",
  "level": "context"
}
```

**Resume levels**:
| Level | Tokens | Content |
|-------|--------|---------|
| `summary` | ~100-200 | Checkpoint name and summary field only |
| `context` | ~500-1000 | Summary + context field + key decisions |
| `full` | Complete | Entire conversation history and file states |

Use `summary` for quick orientation, `context` for daily resumption, `full` only when stuck.

## Writing Good Summaries

**Include**:
- Completed work (bullet points)
- Current state (what's in progress)
- Next steps (what to do next)
- Blockers (if any)

**Example**:
```
Completed:
- Fixed memory_feedback collection bug
- Added skills system spec

In progress:
- Creating contextd-marketplace repo

Next:
- Write remaining 2 skills
- Create 6 slash commands
- Initialize git and push
```

## Quick Reference

| Trigger | Action |
|---------|--------|
| Context >= 70% | `checkpoint_save` immediately |
| Starting new session | `checkpoint_list` then `checkpoint_resume` |
| End of work day | `checkpoint_save` with detailed summary |
| Before risky operation | `checkpoint_save` as safety net |

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Waiting until overflow | Save at 70%, not 95% |
| Vague summaries | Be specific about state and next steps |
| Always using full resume | Use summary/context for faster startup |
| Forgetting to checkpoint | Build the habit, or enable auto-checkpoint |
| Generic auto-checkpoint names | Hook instructs you to provide real summaries |

## Auto-Checkpoint (PreCompact Hook)

When the `.claude/hooks/precompact.sh` hook fires, it will output instructions for you to create a checkpoint with a meaningful summary. **You MUST follow these instructions.**

**DO NOT** create checkpoints with:
- Name: "Auto-checkpoint at 70%"
- Summary: "Context at 70% threshold"

These are useless for resumption. Instead, generate:
- Name: Brief description of current work (e.g., "Implementing auth middleware")
- Summary: What you accomplished, what's in progress, what's next
- Context: Key decisions, blockers, important details for resumption

**Example auto-checkpoint (GOOD)**:
```json
{
  "name": "Auth middleware implementation",
  "summary": "Completed: JWT validation, role checking. In progress: rate limiting middleware. Next: integration tests for auth flow.",
  "context": "Using RS256 for JWT signing. Rate limiter using token bucket at 100 req/min. Blocked on Redis connection pooling config."
}
```

**Example auto-checkpoint (BAD)**:
```json
{
  "name": "Auto-checkpoint at 70%",
  "summary": "Context at 70% threshold",
  "context": ""
}
```
