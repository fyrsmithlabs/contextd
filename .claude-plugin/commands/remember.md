---
name: remember
description: Record a learning or insight from the current session. Captures what was learned, details, and outcome (success/failure) into contextd memory.
---

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

@_error-handling.md

If the memory wasn't saved, offer to retry or save the content locally as a fallback.
