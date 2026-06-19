---
name: checkpoint-workflow
description: Use to preserve and restore session state — when context is getting full (approaching ~70%), before a long-running task, before /clear, or when resuming earlier work. Triggers on "save my progress", "pick up where we left off", "checkpoint this", or running low on context. Covers checkpoint_save, checkpoint_list, and checkpoint_resume.
version: 0.5.0
---

# Checkpoint Workflow

## Overview

Checkpoints snapshot the working context so a session can be resumed later — after `/clear`, in a new session, or after a long-running task. This is context **preservation**, distinct from memory (reusable strategies).

## When to checkpoint

- Context usage approaching ~70%.
- Before a long or risky operation that might exhaust context.
- Before `/clear` or ending a session with unfinished work.
- When the user asks to save progress.

## Saving

```
checkpoint_save(summary, ...)
```

Write a summary that lets a future session resume cold:
- **What was done** — the concrete state reached.
- **What's next** — the immediate next step(s).
- **Open questions / blockers** — anything unresolved.

A vague summary ("working on the feature") defeats the purpose.

## Resuming

```
checkpoint_list()                  # find the relevant checkpoint
checkpoint_resume(id, level)       # restore it
```

Choose the resume **level** to match the need:

| Level | Restores | Use when |
|-------|----------|----------|
| `summary` | Just the summary | Quick reorientation |
| `context` | Summary + key context | Continuing the same task |
| `full` | Everything captured | Deep resumption after a long gap |

## Tips

- Pair with `cross-session-memory`: checkpoints capture *this* session's state; memories capture *reusable* insight. Record durable learnings as memories before they're lost to a checkpoint you may never reopen.
- Auto-checkpoint on `/clear` and auto-resume on start can be enabled via contextd hooks/config (`CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR`, `CONTEXTD_AUTO_RESUME_ON_START`, `CONTEXTD_CHECKPOINT_THRESHOLD`).
