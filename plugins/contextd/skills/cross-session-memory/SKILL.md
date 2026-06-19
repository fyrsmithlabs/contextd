---
name: cross-session-memory
description: This skill should be used when starting a task to check for prior solutions, or when finishing one to record a learning. It triggers when the user says "have we solved this before", "remember this", "record what we learned", reuses a past approach, or captures a design decision worth surviving the session. Covers memory_search, memory_record, memory_feedback, and memory_outcome.
version: 0.5.0
---

# Cross-Session Memory

## Overview

contextd's ReasoningBank stores **reusable strategies and decisions** with confidence scoring. The loop is simple: search before solving, record after solving, and give feedback so confidence stays calibrated.

## The loop

### 1. Search before assuming (task start)

```
memory_search(project_id, query)
```

Ask "have I solved something like this before?" before re-deriving an approach. Always search before assuming a problem is novel.

### 2. Record after solving (task completion)

```
memory_record(project_id, content, ...)
```

Capture the **why**, not just the what. A good memory includes:
- The problem and the approach that worked
- Rejected alternatives and the tradeoff that decided it
- Consequences / gotchas to watch for

### 3. Report outcomes and feedback

- `memory_outcome` — after acting on a memory, report whether the task succeeded. This is the reinforcement signal.
- `memory_feedback` — rate a specific memory as helpful or not, adjusting its confidence.

### 4. Consolidate (periodically)

`memory_consolidate` merges similar memories into refined summaries so the bank stays sharp instead of accumulating near-duplicates.

## What makes a good memory

| Good | Avoid |
|------|-------|
| "Use payload isolation, not filesystem, for multi-tenant vectorstore — avoids N collections; rejected per-tenant DB because of open-file limits." | "Fixed the bug." |
| Decision + rejected alternative + consequence | Restating code that's already in the diff |

## When NOT to record

- Information already obvious from the code or docs.
- Secrets or credentials (contextd scrubs responses, but don't author them into memories).
