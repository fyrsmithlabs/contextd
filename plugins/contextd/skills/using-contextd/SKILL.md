---
name: using-contextd
description: Use at the start of any coding or research session to establish the contextd workflow — cross-session memory, checkpoints, semantic search, and error remediation. Triggers when the user mentions remembering past work, resuming a session, "what did we do before", persistent memory, or when starting a non-trivial task that could benefit from prior learnings.
version: 0.5.0
---

# Using contextd

## Overview

contextd is an MCP server that gives Claude Code **persistent memory across sessions**. It learns from successes and failures, saves context for resumption, tracks error fixes, and provides semantic code search. Every response is scrubbed for secrets with gitleaks.

This skill establishes the mental model. Three companion skills cover the workflows:
- `cross-session-memory` — the learning loop (search before solving, record after)
- `checkpoint-workflow` — context preservation and resumption
- `error-remediation` — matching and recording error fixes

## The contextd tools

| Group | Tools | Use for |
|-------|-------|---------|
| Memory | `memory_search`, `memory_record`, `memory_feedback`, `memory_outcome`, `memory_consolidate` | Reusable strategies and design decisions |
| Checkpoint | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` | Saving/restoring session state |
| Remediation | `remediation_search`, `remediation_record`, `remediation_feedback` | Concrete error → fix pairs |
| Search | `semantic_search`, `repository_index`, `repository_search` | Finding code by meaning (with grep fallback) |
| Diagnosis | `troubleshoot_diagnose` | AI-powered analysis of an error |

## Pre-flight (do this first)

Before exploring a codebase or starting a task:

1. `semantic_search(query, project_path: ".")` — find relevant code by meaning before falling back to Read/Grep/Glob.
2. `memory_search(project_id, query)` — check whether this problem has been solved before.

These are cheap and usually save far more work than they cost.

## When NOT to use contextd

- Trivial one-line edits with no reusable insight.
- When the user explicitly wants a clean slate with no prior context.

## Tenant context

contextd derives the tenant/project automatically from the repository (e.g. the Git remote). No manual configuration is required for normal use.
