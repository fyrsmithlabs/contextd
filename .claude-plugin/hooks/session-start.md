## Session Start Protocol (CONTEXTD)

**MANDATORY: Before your first substantive response, you MUST:**

1. Run `mcp__contextd__checkpoint_list(tenant_id, project_path)` to check for existing checkpoints
2. If checkpoints exist, present the most recent relevant checkpoint to the user:
   - Show checkpoint name, summary, and when it was created
   - Ask: "Would you like to resume from this checkpoint?"
3. If user says yes, run `mcp__contextd__checkpoint_resume(checkpoint_id, tenant_id, level)` with level="context"
4. Run `mcp__contextd__memory_search(project_id, "current task context")` to retrieve relevant memories

**This protocol ensures continuity across sessions. Do not skip these steps.**
