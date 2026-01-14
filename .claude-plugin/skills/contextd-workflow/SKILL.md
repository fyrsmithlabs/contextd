---
name: contextd-workflow
description: Use for session lifecycle management - covers session start/end protocols, checkpoint workflow, and error remediation flow
---

# contextd Workflow

## Session Start Protocol

**Step 1: Search for relevant memories**
```json
{
  "project_id": "contextd",
  "query": "current task context",
  "limit": 5
}
```

**Step 2: Check for checkpoints**
```json
{
  "tenant_id": "fyrsmithlabs",
  "project_path": "/path/to/project",
  "limit": 5
}
```

**Step 3: If checkpoint found**
- Ask user: "Found previous work: '[summary]'. Resume?"
- If yes: `checkpoint_resume(checkpoint_id, tenant_id, level)`
- Levels: `summary` (minimal), `context` (balanced), `full` (complete)

## Checkpoint Workflow

**When to checkpoint:**
- Context >= 70% capacity
- End of work session
- Before risky operations
- Switching tasks

### checkpoint_save

```json
{
  "session_id": "session_abc123",
  "tenant_id": "fyrsmithlabs",
  "project_path": "/path/to/project",
  "name": "Feature implementation checkpoint",
  "summary": "Completed: spec, 2 of 4 skills. Next: remaining skills.",
  "context": "Working on plugin consolidation...",
  "full_state": "Complete conversation state...",
  "token_count": 45000,
  "threshold": 0.7,
  "auto_created": false
}
```

**Summary must include:**
- What was accomplished
- What's in progress
- What's next
- Key decisions made

### Resume Levels

| Level | Tokens | Content |
|-------|--------|---------|
| `summary` | ~100-200 | Name + summary only |
| `context` | ~500-1000 | Summary + context + decisions |
| `full` | Complete | Entire conversation history |

## Error Remediation Flow

**When encountering ANY error:**

```
1. DIAGNOSE the error
   troubleshoot_diagnose(error_message, error_context)

2. SEARCH for past fixes
   remediation_search(query, tenant_id)

3. APPLY fix (use diagnosis + history)

4. RECORD the solution
   remediation_record(title, problem, root_cause, solution, category)
```

### troubleshoot_diagnose

```json
{
  "error_message": "cannot use vectorStore as vectorstore.Store value",
  "error_context": "Running go test after adding new interface method"
}
```

Returns: `root_cause`, `hypotheses`, `recommendations`, `confidence`

### remediation_record

```json
{
  "title": "Mock missing interface method after interface change",
  "problem": "Test fails with 'missing method X'",
  "symptoms": ["go test fails", "cannot use X as Y value"],
  "root_cause": "Mock implementations don't get new interface methods",
  "solution": "Add new method to all mock implementations",
  "category": "syntax",
  "tenant_id": "fyrsmithlabs",
  "scope": "org"
}
```

**Categories:** `syntax`, `runtime`, `logic`, `config`, `dependency`, `network`, `auth`, `data`, `performance`

**Scopes:** `project` (this project), `team` (team members), `org` (entire organization)

## Session End Protocol

**Before `/clear` or ending work:**

1. **Re-index repository**
   ```json
   { "path": "." }
   ```
   Captures code changes and updates branch metadata.

2. **Record learnings**
   ```json
   {
     "project_id": "contextd",
     "title": "Implemented session lifecycle hooks",
     "content": "Used TDD with Registry pattern...",
     "outcome": "success",
     "tags": ["hooks", "lifecycle", "tdd"]
   }
   ```

3. **Checkpoint if resuming later**
   ```json
   { "auto_created": true, "threshold": 0.7, ... }
   ```

## Git Commit Re-index

**After every `git commit`:**
```json
{ "path": "." }
```

Why: Captures changes for semantic search, updates branch metadata.

## Quick Reference

| When | Action |
|------|--------|
| Session start | `memory_search` + `checkpoint_list` |
| After git commit | `repository_index` |
| Context >= 70% | `checkpoint_save` then `/clear` |
| Error encountered | `troubleshoot_diagnose` -> `remediation_search` -> fix -> `remediation_record` |
| Before `/clear` | `memory_record` + `checkpoint_save` |

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Skipping memory search at start | Always search first |
| Vague checkpoint summaries | Include completed/in-progress/next |
| Waiting until context overflow | Save at 70%, not 95% |
| Not recording before `/clear` | Call `memory_record` first |
| Skipping error diagnosis | `troubleshoot_diagnose` first, always |
