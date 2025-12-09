---
name: checkpoint-workflow
description: Use when context approaching 70% capacity, during long-running tasks, or before risky operations - saves and resumes session state with checkpoint_save, checkpoint_list, and checkpoint_resume
---

# Checkpoint Workflow

## Overview

Checkpoints preserve your work when context gets full. Save before overflow, resume later with full context.

## When to Use

**Proactive (recommended)**:
- Context approaching 70% capacity
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
| Level | Use When |
|-------|----------|
| `summary` | Quick orientation, minimal context |
| `context` | Normal resumption, balanced detail |
| `full` | Need complete state, maximum context |

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
| Context > 70% | `checkpoint_save` immediately |
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
