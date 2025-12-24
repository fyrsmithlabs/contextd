---
name: using-contextd
description: Use when starting any session with contextd MCP server available - introduces cross-session memory, checkpoints, and error remediation tools for persistent AI agent learning
---

# Using contextd

## ⚠️ CRITICAL: Pre-Flight Checks (DO THIS FIRST)

**BEFORE touching Read, Grep, or Glob, you MUST:**

1. **`mcp__contextd__semantic_search(query, project_path: ".")`**
   - Semantic code search with automatic grep fallback
   - This is your PRIMARY tool for code lookup
   - NEVER skip this step

2. **`mcp__contextd__memory_search(project_id, query)`**
   - Check past learnings and solutions
   - Required at the start of every task

**Skipping semantic_search is a protocol violation. Always search contextd first.**

## Overview

contextd provides cross-session memory and context management via MCP. Your learnings persist across sessions, errors get remembered, and context can be checkpointed and resumed.

## Available Tools

| Category | Tools | Purpose |
|----------|-------|---------|
| **Memory** | `memory_search`, `memory_record`, `memory_feedback`, `memory_outcome` | Cross-session learning |
| **Checkpoint** | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` | Context preservation |
| **Remediation** | `remediation_search`, `remediation_record` | Error pattern tracking |
| **Troubleshoot** | `troubleshoot_diagnose` | AI-powered error diagnosis |
| **Repository** | `repository_index`, `repository_search`, `semantic_search` | Semantic code search |
| **Context Folding** | `branch_create`, `branch_return`, `branch_status` | Isolated sub-tasks with token budgets |

## When to Use Other Skills

| Situation | Use Skill |
|-----------|-----------|
| Starting any task | `contextd:cross-session-memory` (search first) |
| Context approaching 70% | `contextd:checkpoint-workflow` |
| Encountering errors | `contextd:error-remediation` |
| Setting up secret scrubbing | `contextd:secret-scrubbing` (PostToolUse hooks) |

## Key Concepts

**Tenant ID**: Derived from git remote URL as org/owner name (e.g., `github.com/fyrsmithlabs/contextd` -> `fyrsmithlabs`). For non-GitHub remotes or projects without remotes, provide tenant_id explicitly. Verify with: `git remote get-url origin | sed 's|.*github.com[:/]\([^/]*\).*|\1|'`

**Project ID**: Scopes memories to project. Use consistent format:
- Single-org: repository name only (e.g., `contextd`)
- Multi-org: `org_repo` format (e.g., `fyrsmithlabs_contextd`)

Changing project_id creates a new, separate memory space.

**Confidence**: Memories have confidence scores (0-1) that adjust via feedback.

**HTTP Server**: Required for `ctxd` CLI and PostToolUse hooks. Default port 9090. Check: `ctxd health`

## Code Search Priority (CRITICAL)

**Always search contextd FIRST, fallback to Read/Grep:**

| Priority | Tool | When |
|----------|------|------|
| **1st** | `semantic_search` | Smart search - auto-fallback to grep if not indexed |
| **2nd** | `repository_search` | Direct semantic code search (requires prior indexing) |
| **3rd** | `memory_search` | Check past learnings |
| **4th** | Read/Grep/Glob | Fallback for specific files or exact matches |

```
# BEST workflow (auto-handles indexing)
semantic_search(query: "authentication handler", project_path: ".")
→ Uses indexed semantic search if available
→ Falls back to grep automatically if not indexed

# ALTERNATIVE workflow (manual indexing)
repository_search(query: "authentication handler", project_path: ".")
→ Found relevant code? Use it
→ Not indexed? repository_index(path: ".") then search

# WRONG workflow
grep "auth" **/*.go  ← Skipped contextd, wasted context
```

**Why:** `semantic_search` is the preferred tool - it automatically chooses between semantic search (if indexed) and grep fallback. Repository search is semantic (finds by meaning), preserves context (returns only relevant snippets), and improves over time. Raw file reads bloat context.

## Quick Start

```
1. semantic_search - "Where is this in the code?" (auto-fallback to grep)
2. memory_search - "Have I solved this before?"
3. Do the work
4. memory_record - "What did I learn?"
5. checkpoint_save - If session is long or context is high
```

## Example Response: memory_search

```json
{
  "memories": [
    {
      "id": "mem_abc123",
      "title": "Goroutine race condition fix",
      "content": "Use sync.Mutex for shared state, or channels for communication. Run with -race flag to detect.",
      "confidence": 0.85,
      "outcome": "success",
      "tags": ["go", "concurrency", "debugging"],
      "created_at": "2025-12-10T14:30:00Z"
    },
    {
      "id": "mem_def456",
      "title": "Context cancellation pattern",
      "content": "Pass context.Context as first arg, check ctx.Done() in loops.",
      "confidence": 0.72,
      "outcome": "success",
      "tags": ["go", "concurrency", "context"],
      "created_at": "2025-12-09T09:15:00Z"
    }
  ],
  "total": 2
}
```

Higher confidence memories appear first. Use `memory_feedback` to improve rankings.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Using Read/Grep before contextd | `semantic_search` FIRST (auto-fallback to grep) |
| Not searching at task start | Always `memory_search` first |
| Forgetting to record learnings | `memory_record` at task completion |
| Letting context overflow | `checkpoint_save` at 70%, or use `branch_create` for sub-tasks |
| Re-solving fixed errors | `remediation_search` when errors occur |
| Long sub-tasks bloating context | Use context folding: `branch_create` → work → `branch_return` |
