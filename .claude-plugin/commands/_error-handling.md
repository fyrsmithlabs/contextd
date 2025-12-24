# Standard Error Handling for contextd Commands

## Server Health Check

If MCP tool call fails:

1. Check server: `curl -s http://localhost:9090/health`
   - Expected: `{"status":"ok"}`
   - If different or fails: Server not running

2. If server down:
   ```
   contextd server not responding.
   Start with: contextd &
   Verify with: ctxd health
   ```

3. If connection refused:
   ```
   Cannot connect to contextd.
   Check: Is contextd MCP server configured in Claude settings?
   Verify: claude mcp list | grep contextd
   ```

4. For other errors:
   - Display sanitized error (no internal paths)
   - Suggest: "Check contextd logs for details"
