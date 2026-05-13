---
name: review-pr-local
specializes: review-pr
description: Repo-specific review guidance for oz-for-oss. Only the categories declared overridable by the core review-pr skill may be specialized here.
---

# Repo-specific review guidance for `oz-for-oss`

This file is a companion to the core `review-pr` skill. It does not
redefine the review output schema, severity labels, safety rules, or
evidence rules. It only specializes the override categories the core
skill marks as overridable.

## User-facing strings

- Flag interpolated text that would read unnaturally at runtime (e.g. wrong casing after a sentence fragment like "The triage concluded that {summary}").
- Link text should be descriptive (e.g. "triage session on Warp"), not bare URLs or generic "click here" labels.
- Verify that terminology is consistent across related messages in the same PR.

## Graceful degradation

- When code renders optional dynamic data (URLs, session links, metadata), flag cases where a missing value would produce empty or broken output. The fix is usually to omit the element entirely and show a short fallback message.
- Prefer starting with generic, user-safe error messages over exposing internal details.

## Debugging and observability

- Do not suggest removing session links, workflow URLs, or other debugging context from error paths. These are valuable for post-incident investigation even when the operation failed.
