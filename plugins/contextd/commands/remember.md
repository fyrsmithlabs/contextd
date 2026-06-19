---
description: Record a learning from this session into contextd memory
argument-hint: "[what to remember]"
---

# /contextd:remember

Record a durable learning into the contextd ReasoningBank using the `memory_record` MCP tool.

Steps:

1. Determine the content to record:
   - If `$ARGUMENTS` is provided, record that.
   - Otherwise, distill the key insight from the recent conversation.
2. Capture the **why**, not just the what — include the approach that worked, rejected alternatives, the deciding tradeoff, and any consequences/gotchas.
3. Call `memory_record` with the distilled content.
4. Confirm what was stored in one or two lines.

Do not record secrets or credentials. Skip recording if the insight is already obvious from the code or docs.
