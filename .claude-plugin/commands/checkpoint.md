---
name: checkpoint
description: Save a checkpoint of the current session state including accomplishments, in-progress work, and next steps. Use before clearing context or ending a session.
---

Save a checkpoint of the current session.

Generate a summary of:
1. What was accomplished in this session
2. What's currently in progress
3. What should be done next

Then call `mcp__contextd__checkpoint_save` with:
- session_id: Current session identifier
- tenant_id: From git remote or default
- project_path: Current working directory
- name: Brief checkpoint name
- summary: The generated summary
- context: Recent conversation context
- token_count: Estimated tokens used

Confirm the checkpoint was saved and provide the checkpoint ID.

## Error Handling

@_error-handling.md
