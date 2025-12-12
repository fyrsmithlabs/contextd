List and resume from a previous checkpoint.

1. Call `mcp__contextd__checkpoint_list` with:
   - tenant_id: From git remote or default
   - project_path: Current working directory
   - limit: 10

2. Present checkpoints in a numbered list showing:
   - Name and description
   - When created
   - Summary preview
   - Token count

3. Ask user which checkpoint to resume (or "none" to cancel)

4. Call `mcp__contextd__checkpoint_resume` with:
   - checkpoint_id: Selected checkpoint
   - tenant_id: From git remote or default
   - level: Ask user preference (summary/context/full)

5. Display restored context and summarize where we left off.

## Error Handling

@_error-handling.md

**Resume-specific errors:**
- Checkpoint not found: "Checkpoint may have been deleted. Try listing again."
- Invalid level: "Invalid resume level. Use: summary, context, or full."
- Offer to try a different checkpoint or resume level.

**No checkpoints:**
- "No checkpoints found for this project. Use `/contextd:checkpoint` to create one."
