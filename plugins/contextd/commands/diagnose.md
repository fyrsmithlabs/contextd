---
name: diagnose
description: Diagnose an error and search contextd for a known fix.
argument-hint: "<error message or description>"
---

# /contextd:diagnose

Diagnose the error in `$ARGUMENTS` (or, if empty, the most recent error in the conversation) using contextd.

Steps:

1. Call `troubleshoot_diagnose` with the error to get AI-powered analysis of the likely cause.
2. Call `remediation_search` with the stable part of the error signature to find any fix that worked before.
3. Present:
   - The diagnosis (category + likely cause).
   - Any matching known remediation, clearly marked as a prior fix.
   - A recommended next step.
4. After the user applies a fix and confirms it works, offer to record it with `remediation_record` so the fix is reused next time.
