Record a learning or insight from the current session.

Ask the user (if not provided):
1. What did you learn? (title)
2. Details of the learning (content)
3. Was this a success or failure?

Then call `mcp__contextd__memory_record` with:
- project_id: Current project identifier
- title: Brief title for the memory
- content: Detailed learning content
- outcome: "success" or "failure"
- tags: Relevant tags extracted from content

Confirm the memory was recorded with its ID and initial confidence.

## Error Handling

If the MCP call fails:
1. Check if contextd server is running: `curl -s http://localhost:9090/health`
2. If server is down: "contextd server not responding. Start it with `contextd serve` or check logs."
3. If connection refused: "Cannot connect to contextd. Verify MCP server is configured in Claude settings."
4. For other errors: Display the error message and suggest checking contextd logs.

If the memory wasn't saved, offer to retry or save the content locally as a fallback.
