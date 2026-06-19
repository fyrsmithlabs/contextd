---
description: Search contextd memories, remediations, and code
argument-hint: "<query>"
---

# /contextd:search

Search contextd for anything relevant to `$ARGUMENTS`.

Steps:

1. Run these searches for the query:
   - `memory_search` — past strategies and decisions.
   - `remediation_search` — known error fixes.
   - `semantic_search` (with `project_path: "."`) — relevant code in this repository.
2. Merge and present the most relevant hits, grouped by source (Memories / Remediations / Code), each with a one-line relevance note.
3. If nothing relevant is found, say so plainly rather than padding with weak matches.
