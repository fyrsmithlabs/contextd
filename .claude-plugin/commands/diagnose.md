---
name: diagnose
description: Diagnose an error message using AI analysis and past fixes. Searches remediations and provides root cause analysis with recommended next steps.
arguments:
  - name: error
    description: "The error message to diagnose"
    required: false
---

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

@_error-handling.md

**Partial failures:**
- If `troubleshoot_diagnose` fails: Fall back to manual analysis, still attempt `remediation_search`
- If `remediation_search` fails: Continue with diagnosis only, note "Proceeding with AI diagnosis only."

**Both fail:** Fall back to standard debugging approach without contextd assistance.
