---
name: using-contextd
description: Use when starting any session with contextd MCP server available - introduces cross-session memory, checkpoints, and error remediation tools for persistent AI agent learning
---

# Using contextd

## Overview

contextd provides cross-session memory and context management via MCP. Your learnings persist across sessions, errors get remembered, and context can be checkpointed and resumed.

## Available Tools

| Category | Tools | Purpose |
|----------|-------|---------|
| **Memory** | `memory_search`, `memory_record`, `memory_feedback` | Cross-session learning |
| **Checkpoint** | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` | Context preservation |
| **Remediation** | `remediation_search`, `remediation_record` | Error pattern tracking |
| **Troubleshoot** | `troubleshoot_diagnose` | AI-powered error diagnosis |
| **Repository** | `repository_index`, `repository_search` | Semantic code indexing and search |

## When to Use Other Skills

| Situation | Use Skill |
|-----------|-----------|
| Starting any task | `contextd:cross-session-memory` (search first) |
| Context approaching 70% | `contextd:checkpoint-workflow` |
| Encountering errors | `contextd:error-remediation` |

## Key Concepts

**Tenant ID**: Auto-derived from GitHub remote URL. No configuration needed.

**Project ID**: Passed to memory tools, scopes learnings to project.

**Confidence**: Memories have confidence scores (0-1) that adjust via feedback.

## Code Search Priority (CRITICAL)

**Always search contextd FIRST, fallback to Read/Grep:**

| Priority | Tool | When |
|----------|------|------|
| **1st** | `repository_search` | Semantic code search - finds by meaning |
| **2nd** | `memory_search` | Check past learnings |
| **3rd** | Read/Grep/Glob | Fallback for specific files or exact matches |

```
# CORRECT workflow
repository_search(query: "authentication handler", project_path: ".")
→ Found relevant code? Use it
→ Not indexed? repository_index(path: ".") then search

# WRONG workflow
grep "auth" **/*.go  ← Skipped contextd, wasted context
```

**Why:** Repository search is semantic (finds by meaning), preserves context (returns only relevant snippets), and improves over time. Raw file reads bloat context.

## Quick Start

```
1. repository_search - "Where is this in the code?"
2. memory_search - "Have I solved this before?"
3. Do the work
4. memory_record - "What did I learn?"
5. checkpoint_save - If session is long or context is high
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Using Read/Grep before contextd | `repository_search` FIRST, fallback to Read/Grep |
| Not searching at task start | Always `memory_search` first |
| Forgetting to record learnings | `memory_record` at task completion |
| Letting context overflow | `checkpoint_save` at 70% |
| Re-solving fixed errors | `remediation_search` when errors occur |
