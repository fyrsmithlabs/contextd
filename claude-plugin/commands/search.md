Search across memories and remediations.

Take the search query from the command argument or ask the user.

1. Call `mcp__contextd__memory_search` with:
   - project_id: Current project
   - query: User's search query
   - limit: 5

2. Call `mcp__contextd__remediation_search` with:
   - query: User's search query
   - tenant_id: From git remote or default
   - limit: 5

3. Present combined results:

   **Memories Found:**
   - Title, confidence, outcome
   - Content preview
   - Tags

   **Remediations Found:**
   - Title, category, confidence
   - Problem summary
   - Solution preview

4. Offer to show full details for any result.

## Error Handling

If `memory_search` fails:
- Note: "Could not search memories."
- Continue with remediation search only.

If `remediation_search` fails:
- Note: "Could not search remediations."
- Continue with memory results only.

If both fail:
1. Check server: `curl -s http://localhost:9090/health`
2. If server is down: "contextd server not responding. Search unavailable."
3. Suggest: "Start contextd with `contextd serve` or verify MCP configuration."

If no results found:
- "No matches found for '[query]'."
- Suggest broader search terms or different keywords.
