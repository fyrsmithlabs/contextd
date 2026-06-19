---
name: status
description: Show contextd session info — memories, checkpoints, and tenant/project context.
---

# /contextd:status

Report the current contextd state for this session.

Steps:

1. Call `checkpoint_list` to get available checkpoints (count + most recent).
2. Run a broad `memory_search` for the current project to gauge how many relevant memories exist.
3. Summarize:
   - Tenant / project context contextd is operating under (auto-derived from the repository).
   - Number of checkpoints and the most recent one.
   - Whether relevant memories exist for this project.
4. If the contextd MCP server is unavailable, say so and suggest checking that `contextd --mcp` is running.

Keep the output to a compact status block.
