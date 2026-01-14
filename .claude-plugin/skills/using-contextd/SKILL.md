---
name: using-contextd
description: Use when starting any session with contextd - introduces core tools for cross-session memory, semantic code search, and error remediation
---

# Using contextd

## Pre-Flight Protocol (MANDATORY)

**BEFORE any filesystem operation (Read, Grep, Glob), you MUST:**

1. **`semantic_search(query, project_path: ".")`** - Semantic code search with auto grep fallback
2. **`memory_search(project_id, query)`** - Check past learnings and solutions

Skipping this is a protocol violation.

## Core Tools

| Category | Tools | Purpose |
|----------|-------|---------|
| **Search** | `semantic_search`, `repository_search`, `repository_index` | Code lookup by meaning |
| **Memory** | `memory_search`, `memory_record`, `memory_feedback`, `memory_outcome` | Cross-session learning |
| **Checkpoint** | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` | Context preservation |
| **Remediation** | `remediation_search`, `remediation_record`, `troubleshoot_diagnose` | Error pattern tracking |
| **Context Folding** | `branch_create`, `branch_return`, `branch_status` | Isolated sub-tasks |

## Search Priority

| Priority | Tool | When |
|----------|------|------|
| **1st** | `semantic_search` | Auto-selects best method (indexed or grep fallback) |
| **2nd** | `memory_search` | Have I solved this before? |
| **3rd** | Read/Grep/Glob | Fallback for exact matches only |

## The Learning Loop

```
1. SEARCH at task start (MANDATORY)
   semantic_search(query, project_path)
   memory_search(project_id, query)

2. DO the work
   (apply relevant memories)

3. RECORD at completion
   memory_record(project_id, title, content, outcome)

4. FEEDBACK when memories helped
   memory_feedback(memory_id, helpful)
```

## Key Concepts

**Tenant ID**: Derived from git remote (e.g., `github.com/fyrsmithlabs/contextd` -> `fyrsmithlabs`). Verify with: `git remote get-url origin | sed 's|.*github.com[:/]\([^/]*\).*|\1|'`

**Project ID**: Scopes memories. Use repository name (e.g., `contextd`) or `org_repo` format for multi-org.

**Confidence**: Memories have scores (0-1) that adjust via feedback. Higher = ranks first.

## What to Record

**Good memories:**
- Non-obvious solutions
- Patterns that apply broadly
- Design decisions with rationale (the WHY)
- Mistakes and why they failed

**Skip recording:**
- Trivial fixes (typos, syntax)
- Project-specific details (put in CLAUDE.md)

## Recording Design Decisions

When design involves significant discussion, capture the WHY:

```json
{
  "project_id": "contextd",
  "title": "ADR: Registry pattern for DI",
  "content": "DECISION: Use Registry interface.\nWHY: Idiomatic Go, single mock for tests.\nREJECTED: Passing individual services (constructor bloat).",
  "outcome": "success",
  "tags": ["adr", "architecture", "design-decision"]
}
```

## Optional: Conversation Indexing

Index past Claude Code sessions to pre-warm contextd:

```bash
# Via /init command
/init --conversations

# What it extracts:
# - Error -> fix patterns (remediations)
# - Learnings (memories)
# - User corrections (policies)
```

Conversations are scrubbed for secrets before processing.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Using Read/Grep before contextd | `semantic_search` FIRST |
| Not searching at task start | Always `memory_search` first |
| Forgetting to record learnings | `memory_record` at task completion |
| Re-solving fixed errors | `remediation_search` when errors occur |
| Context bloat from sub-tasks | Use `branch_create` for isolation |
