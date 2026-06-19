---
description: Save a resumable context checkpoint of this session
argument-hint: "[optional summary]"
---

# /contextd:checkpoint

Save a checkpoint of the current session using the contextd `checkpoint_save` MCP tool.

Steps:

1. Build a resumable summary. If the user supplied text in `$ARGUMENTS`, use it as the summary; otherwise auto-generate one from the recent conversation covering:
   - **What was done** — the concrete state reached so far.
   - **What's next** — the immediate next step(s).
   - **Open questions / blockers** — anything unresolved.
2. Call `checkpoint_save` with that summary.
3. Report the checkpoint id and a one-line confirmation of what was saved.

If the contextd MCP server is unavailable, tell the user the checkpoint could not be saved and suggest verifying the `contextd` MCP server is running (`contextd --mcp`).
