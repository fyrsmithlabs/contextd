---
description: List contextd checkpoints and resume from one
argument-hint: "[checkpoint id]"
---

# /contextd:resume

Resume work from a previously saved contextd checkpoint.

Steps:

1. If `$ARGUMENTS` contains a checkpoint id, skip to step 3 with that id.
2. Otherwise call `checkpoint_list` and show the available checkpoints (id, summary, timestamp). Ask the user which one to resume.
3. Call `checkpoint_resume` with the chosen id. Default to the `context` level unless the user asks for `summary` (quick reorientation) or `full` (deep resumption after a long gap).
4. Summarize the restored state and state the immediate next step so work can continue.
