---
name: search
description: Search across memories, remediations, and code using semantic search. Combines results from contextd memory, remediations, and indexed code.
arguments:
  - name: query
    description: "The search query"
    required: false
---

Search across memories, remediations, and code.

Take the search query from the command argument or ask the user.

1. Call `mcp__contextd__semantic_search` with:
   - query: User's search query
   - project_path: Current working directory
   - limit: 5

   (This auto-uses semantic search if indexed, falls back to grep)

2. Call `mcp__contextd__memory_search` with:
   - project_id: Current project
   - query: User's search query
   - limit: 5

3. Call `mcp__contextd__remediation_search` with:
   - query: User's search query
   - tenant_id: From git remote or default
   - limit: 5

4. Present combined results:

   **Code Found:**
   - File path, line number
   - Code snippet preview
   - Relevance score

   **Memories Found:**
   - Title, confidence, outcome
   - Content preview
   - Tags

   **Remediations Found:**
   - Title, category, confidence
   - Problem summary
   - Solution preview

5. Offer to show full details for any result.

## Error Handling

@_error-handling.md

**Partial failures:**
- If `semantic_search` fails: Continue with memory/remediation search, note "Could not search code."
- If `memory_search` fails: Continue with other results, note "Could not search memories."
- If `remediation_search` fails: Continue with other results, note "Could not search remediations."

**No results:**
- "No matches found for '[query]'."
- Suggest broader search terms or different keywords.
