---
name: session-lifecycle
description: Use at session start and before session end - manages contextd memory priming, checkpoint resume, and learning extraction using memory_search, checkpoint_list, checkpoint_save, and memory_record
---

# Session Lifecycle

## Overview

Manages session lifecycle: prime context at start, extract learnings at end, auto-checkpoint when context is high.

## When to Use

| Trigger | Action |
|---------|--------|
| Session begins | Search memories + check for checkpoints |
| After `git commit` | Re-index repository (captures branch + changes) |
| Context > 70% | Save checkpoint before `/clear` |
| Before `/clear` or end of work | Re-index repository + record learnings |

## Lifecycle Flow

```
1. SESSION START
   memory_search(project_id, "current task context")
   checkpoint_list(tenant_id, project_path)
   → Offer checkpoint resume, review relevant memories

2. DO THE WORK
   (apply memories, track progress)
   → Use repository_search FIRST for code lookups

3. AFTER GIT COMMIT
   repository_index(path: ".")
   → Captures changes + current branch

4. CONTEXT THRESHOLD (if >70%)
   checkpoint_save(..., auto_created: true)
   → Then /clear to reset context

5. SESSION END
   repository_index(path: ".")   → Re-index changes
   memory_record(...)            → Capture learnings
```

## Session Start Protocol

**Step 1: Search for relevant memories**
```json
{
  "project_id": "fyrsmithlabs_contextd",
  "query": "current task context",
  "limit": 5
}
```

**Step 2: Check for checkpoints**
```json
{
  "tenant_id": "fyrsmithlabs",
  "project_path": "/home/user/projects/contextd",
  "limit": 5
}
```

**Step 3: If checkpoint found**
- Ask user: "Found previous work: '[summary]'. Resume?"
- If yes: `checkpoint_resume(checkpoint_id, tenant_id, level)`
- Levels: `summary` (minimal), `context` (balanced), `full` (complete)

## Session End Protocol

**Before `/clear` or ending work:**

### Step 1: Re-index the repository
```json
{
  "path": "/home/user/projects/contextd"
}
```
This captures any code changes made during the session and updates branch metadata.

### Step 2: Record learnings
```json
{
  "project_id": "fyrsmithlabs_contextd",
  "title": "Implemented session lifecycle hooks",
  "content": "Used TDD with Registry pattern. Key insight: write handlers as methods on Registry for testability.",
  "outcome": "success",
  "tags": ["hooks", "lifecycle", "tdd", "registry-pattern"]
}
```

**Outcome values**: `success`, `failure`, or use content to describe partial progress

## Context Threshold Protocol

**When context > 70%, save checkpoint then clear:**

```json
{
  "session_id": "unique-session-id",
  "tenant_id": "fyrsmithlabs",
  "project_path": "/home/user/projects/contextd",
  "name": "Auto-checkpoint at 75%",
  "description": "Context threshold triggered",
  "summary": "Completed: X, Y. In progress: Z. Next: A, B.",
  "context": "Working on...",
  "full_state": "Complete state...",
  "token_count": 45000,
  "threshold": 0.75,
  "auto_created": true
}
```

Then run `/clear` to reset context.

## Writing Good Summaries

**Title**: Be specific - "Fixed memory search ranking bug" not "Fixed bug"

**Content**: Include the WHY - "Used mutex because goroutines shared state"

**Tags**: Use searchable keywords - `["concurrency", "mutex", "go"]` not `["code"]`

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Skipping memory search at start | Always search first - past sessions have answers |
| Forgetting to record before `/clear` | Call `memory_record` before clearing context |
| Vague memory content | Be specific: what, why, how |
| Not offering checkpoint resume | Always ask user if checkpoint found |
| Waiting until context overflow | Save checkpoint at 70%, not 95% |

## Git Commit Re-index Protocol

**After every `git commit`, re-index to capture changes:**

```json
{
  "path": "."
}
```

**Why:**
- Captures code changes for semantic search
- Updates branch metadata (useful for feature branches)
- Ensures `repository_search` returns current code

**Branch awareness:** `repository_index` auto-detects current branch. Working on `feature/auth`? The index knows.

## Quick Reference

| When | Tools | Purpose |
|------|-------|---------|
| Session start | `memory_search`, `checkpoint_list` | Prime context |
| Code lookup | `repository_search` | Semantic search (FIRST) |
| After git commit | `repository_index` | Update index with changes |
| Checkpoint found | `checkpoint_resume` | Resume previous work |
| Context > 70% | `checkpoint_save`, then `/clear` | Preserve and reset |
| Before `/clear` | `repository_index`, `memory_record` | Re-index + capture learnings |
| End of work | `repository_index`, `memory_record` | Save state |

## CRITICAL

**Before clearing context:**
1. `memory_record` - capture what you learned
2. `checkpoint_save` - if resuming later
3. `/clear` - reset context

The next session depends on what you record now.
