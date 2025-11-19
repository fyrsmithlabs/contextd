---
name: auto-checkpoint
description: Auto-save checkpoint when context approaching limits
---

# Auto-Checkpoint

Call the contextd MCP `checkpoint_save` tool with auto-generated summary.

**Workflow:**
1. Generate summary from conversation history:
   - Tasks completed
   - Key decisions
   - Current state
   - Issues/findings
2. Get current directory: `pwd`
3. Call MCP tool: `checkpoint_save`
   - `content`: Generated summary
   - `project_path`: current directory
4. Report: operation_id, summary
5. If context > 90%: Recommend `/clear`

**Note**: Execute immediately, no user prompt for summary.
