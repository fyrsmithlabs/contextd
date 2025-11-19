# Checkpoint

Call the contextd MCP `checkpoint_save` tool.

**Workflow:**
1. Get current directory: `pwd`
2. Call MCP tool: `checkpoint_save`
   - `content`: Session summary (from ARGUMENTS or user prompt)
   - `project_path`: current directory
   - `metadata`: {} (optional)
3. Report: operation_id

**Note**: Checkpoint saves async in background.

ARGUMENTS: {summary}
