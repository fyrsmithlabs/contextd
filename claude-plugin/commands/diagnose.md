Diagnose an error message using contextd.

Take the error message from the command argument or ask the user to provide it.

1. Call `mcp__contextd__troubleshoot_diagnose` with:
   - error_message: The provided error
   - error_context: Any additional context (stack trace, logs)

2. Call `mcp__contextd__remediation_search` with:
   - query: The error message
   - tenant_id: From git remote or default

3. Present findings:
   - Root cause analysis
   - Hypotheses ranked by likelihood
   - Past remediations if found
   - Recommended next steps

4. If a fix is found and applied successfully, offer to record it with `remediation_record`.

## Error Handling

If `troubleshoot_diagnose` fails:
- Fall back to manual analysis using the error message
- Still attempt `remediation_search` for past fixes

If `remediation_search` fails:
- Continue with diagnosis results only
- Note: "Could not search past remediations. Proceeding with AI diagnosis only."

If both fail:
1. Check if contextd server is running: `curl -s http://localhost:9090/health`
2. If server is down: "contextd server not responding. Falling back to manual debugging."
3. Proceed with standard debugging approach without contextd assistance.
