---
name: cross-session-memory
description: Use when starting any task to search for relevant past learnings, and at task completion to record new insights - implements the learning loop with memory_search, memory_record, and memory_feedback
---

# Cross-Session Memory

## Overview

The learning loop: search before starting, record after completing. Your insights persist and improve with feedback.

## When to Use

**Task Start (ALWAYS)**:
- Before implementing any feature
- Before debugging any issue
- Before answering complex questions
- "Have I solved something like this before?"

**Task Completion**:
- After successfully fixing a bug
- After discovering a useful pattern
- After learning something non-obvious
- Both successes AND failures are valuable

## The Learning Loop

```
┌─────────────────────────────────────────┐
│  1. SEARCH at task start (MANDATORY)    │
│     semantic_search(query, path)        │  ← Code first (auto grep fallback)
│     memory_search(project_id, query)    │  ← Then memories
├─────────────────────────────────────────┤
│  2. DO the work                         │
│     (apply relevant code/memories)      │
├─────────────────────────────────────────┤
│  3. RECORD at completion                │
│     memory_record(project_id, title,    │
│                   content, outcome)     │
├─────────────────────────────────────────┤
│  4. FEEDBACK when memories helped       │
│     memory_feedback(memory_id, helpful) │
└─────────────────────────────────────────┘
```

## Tool Reference

### memory_search

```json
{
  "project_id": "contextd",
  "query": "fixing race conditions in goroutines",
  "limit": 5
}
```

Returns memories ranked by relevance with confidence scores.

### memory_record

```json
{
  "project_id": "contextd",
  "title": "Goroutine race condition fix",
  "content": "Use sync.Mutex for shared state, or channels for communication. Run with -race flag to detect.",
  "outcome": "success",
  "tags": ["go", "concurrency", "debugging"]
}
```

Outcome: `success` or `failure` - both are valuable learnings.

**Example Response:**
```json
{
  "memory_id": "mem_abc123",
  "status": "created",
  "confidence": 0.5,
  "message": "Memory recorded successfully"
}
```

New memories start at confidence 0.5 (neutral). Use `memory_feedback` to adjust.

### memory_feedback

```json
{
  "memory_id": "mem_abc123",
  "helpful": true
}
```

Adjusts confidence: helpful increases, unhelpful decreases.

## Understanding Confidence Scores

Memories have confidence scores (0-1) that affect search ranking:
- New memories start at 0.5 (neutral)
- `memory_feedback(helpful=true)` increases confidence toward 1.0
- `memory_feedback(helpful=false)` decreases confidence toward 0.0
- Higher confidence = ranks first in search results

Use `min_confidence` parameter to filter unreliable memories (recommend 0.6+ for critical decisions).

## What to Record

**Good memories**:
- Non-obvious solutions
- Patterns that apply broadly
- Mistakes and why they failed
- Workarounds for known issues
- **Design decisions with significant user input**

**Skip recording**:
- Trivial fixes (typos, syntax)
- One-off configurations
- Project-specific details (put in CLAUDE.md)

## Recording Design Decisions (The WHY)

**When design involves significant user input/discussion, always capture:**

1. **WHAT** was decided
2. **WHY** that approach (rejected alternatives, tradeoffs)
3. **CONSEQUENCES** (what changes, what to watch for)

**Rule: Lots of user input = needs a why.**

The discussion IS the value - don't just record the outcome.

**Format for design memories:**

```json
{
  "project_id": "contextd",
  "title": "ADR: Registry interface pattern for DI",
  "content": "DECISION: Use Registry interface with accessor methods.\n\nWHY: Idiomatic Go, single mock for tests, clear dependency boundary.\n\nREJECTED: Passing individual services (constructor bloat), concrete struct (can't mock), global singleton (untestable).\n\nCONSEQUENCES: All service access via registry.Service().Method(), new services require interface update.",
  "outcome": "success",
  "tags": ["adr", "architecture", "patterns", "design-decision"]
}
```

**Also write to docs:**
- Save design doc: `docs/plans/YYYY-MM-DD-<topic>-design.md`
- Save ADR: `docs/plans/YYYY-MM-DD-<topic>-adr.md`

Memory should be searchable by both the decision AND the reasoning.

## Quick Reference

| When | Action | Tool |
|------|--------|------|
| Starting task | Search for relevant memories | `memory_search` |
| Memory was helpful | Provide positive feedback | `memory_feedback(helpful=true)` |
| Memory was wrong | Provide negative feedback | `memory_feedback(helpful=false)` |
| Task succeeded | Record the learning | `memory_record(outcome=success)` |
| Task failed | Record what didn't work | `memory_record(outcome=failure)` |

## Common Mistakes & Symptoms

| Mistake | Symptom | Fix |
|---------|---------|-----|
| Skipping search "to save time" | Spending >10 min on familiar problems | Search takes seconds, saves hours |
| Only recording successes | Repeating same failed approaches | Failures prevent repeated mistakes |
| Vague memory content | Memories don't match searches | Be specific: what, why, how |
| Never giving feedback | Irrelevant memories rank high | Feedback improves future rankings |
| **Assuming without searching** | **Re-deriving existing solutions** | **Always search before re-deriving** |
| Recording WHAT without WHY | Can't understand past decisions | Include rejected alternatives

## CRITICAL: Search Before Assuming

**Before assuming something doesn't exist or needs to be designed from scratch:**

| Priority | Tool | Purpose |
|----------|------|---------|
| **1st** | `repository_search` | Find existing code by meaning |
| **2nd** | `memory_search` | Have we solved this before? |
| **3rd** | `checkpoint_list` | Is there recent work on this? |
| **4th** | `remediation_search` | Have we fixed related errors? |
| **Last** | Read/Grep/Glob | Fallback for exact matches |

```
# CORRECT: contextd first
repository_search(query: "user authentication", project_path: ".")
memory_search(project_id: "myproject", query: "auth patterns")

# WRONG: raw file search first
grep -r "auth" src/  ← Bloats context, misses semantic matches
```

**Why repository_search FIRST:**
- Semantic: finds code by meaning, not just keywords
- Efficient: returns relevant snippets, not entire files
- Branch-aware: knows what branch you're on

**You will waste significant time re-discovering things that are already documented.**

This conversation you're in right now? Previous sessions probably covered it. Search contextd first.
