---
name: error-remediation
description: This skill should be used whenever an error, exception, failed build, failing test, stack trace, compiler/linter error, CI failure, or panic appears, or when the user asks "why is this failing" or "how do I fix this". Covers troubleshoot_diagnose, remediation_search, remediation_record, and remediation_feedback to match errors to past fixes and record new ones.
version: 0.5.0
---

# Error Remediation

## Overview

contextd tracks **error → fix patterns**. When something breaks, check whether this exact failure was fixed before, diagnose it, fix it, then record the fix so the next occurrence is instant.

## The flow

### 1. Diagnose

```
troubleshoot_diagnose(error)
```

Get AI-powered analysis of the error first — it categorizes the failure and suggests likely causes.

### 2. Search for a known fix

```
remediation_search(query)
```

Pass the salient part of the error message. If contextd has seen it, it returns the fix that worked before — do not re-derive it.

### 3. Record the fix

After resolving it:

```
remediation_record(error, fix, ...)
```

Capture:
- The **error signature** (the stable part of the message, not volatile paths/timestamps).
- The **fix** that actually worked.
- Any root cause worth noting.

### 4. Feedback

`remediation_feedback` — rate whether a suggested fix actually helped, so its confidence stays accurate.

## Good vs. weak remediations

| Good | Weak |
|------|------|
| `ErrMissingTenant` on Search → wrap ctx with `ContextWithTenant` before calling the store; fail-closed is intentional. | "It works now." |
| Stable error signature + concrete fix + cause | Logging the full transient stack trace as the signature |

## When NOT to record

- One-off typos with no reusable pattern.
- Environment-specific noise unlikely to recur.
