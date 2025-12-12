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

@_error-handling.md

**Partial failures:**
- If `memory_search` fails: Continue with remediation search only, note "Could not search memories."
- If `remediation_search` fails: Continue with memory results only, note "Could not search remediations."

**No results:**
- "No matches found for '[query]'."
- Suggest broader search terms or different keywords.
