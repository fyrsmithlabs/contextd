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

If the MCP call fails:
1. Check if contextd server is running: `curl -s http://localhost:9090/health`
2. If server is down: "contextd server not responding. Start it with `contextd serve` or check logs."
3. If connection refused: "Cannot connect to contextd. Verify MCP server is configured in Claude settings."
4. For other errors: Display the error message and suggest checking contextd logs.
