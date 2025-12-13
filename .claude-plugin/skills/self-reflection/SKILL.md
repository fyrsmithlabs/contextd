---
name: self-reflection
description: Use when reviewing agent behavior patterns, improving CLAUDE.md or design docs based on past failures, or checking ReasoningBank health - mines memories for poor behaviors and pressure-tests doc improvements
---

# Self-Reflection

## Overview

Mine memories and remediations for behavior patterns, surface findings to user, remediate docs with pressure-tested improvements.

**Core loop**: Search → Report → User prioritizes → Brainstorm → Pressure test → Apply

## When to Use

- Periodic review of agent behavior patterns
- After series of failures or poor outcomes
- Before major project milestones
- When CLAUDE.md feels stale or incomplete
- To check ReasoningBank health (tag hygiene, stale memories)

## When NOT to Use

- For immediate error diagnosis → use `error-remediation` skill
- For recording a single learning → use `/contextd:remember`
- For checkpoint management → use `checkpoint-workflow` skill

## Finding Categories

| Category | Impact Model | Examples |
|----------|--------------|----------|
| **Security** | High always | Credentials in context, permission bypass |
| **Process** | Compounds with frequency | Skipping TDD, not reading specs |
| **Style** | Low regardless | Formatting, naming inconsistency |

**Priority formula**: `base_impact × frequency_multiplier(category)`

## The Report

For each finding, surface:

1. **Evidence** - Memory/remediation IDs with excerpts
2. **Pattern** - What went wrong and why
3. **Suggested fix** - Target doc and proposed change
4. **Pressure scenario** - Test case from real failure

## Remediation Flow

```
User selects findings
        ↓
Generate doc improvements
        ↓
Generate pressure scenarios (from real failures)
        ↓
Run batch tests via subagents
        ↓
    Pass? ──No──→ Iterate
        ↓ Yes
Present for approval
        ↓
Apply changes
        ↓
Store in ReasoningBank (tags: reflection:remediation, pressure-tested:true)
```

## Tiered Defaults

| Category | Default | Override Options |
|----------|---------|------------------|
| Security | Full brainstorm | Downgrade to quick |
| Process | Quick proposal | Up to brainstorm, down to auto |
| Style | Auto-fix | Up to quick/brainstorm |

User always has final control.

## Doc Targets

**Can modify**: CLAUDE.md (global, project, sub-dir), design docs, plugin usage includes

**Cannot modify**: Plugin source → create `.claude/includes/using-<plugin>.md` instead

**Placement**: General patterns → global includes. Project-specific → project includes.

## Common Patterns to Catch

- Permission bypass ("user implied consent")
- Absolute language ("ensures", "guarantees", "production ready")
- Skipping verification ("already tested manually")
- Not searching contextd first
- TDD shortcuts ("too simple to test")

## ReasoningBank Health

`--health` flag analyzes:

- **Memory quality**: feedback rate, confidence distribution
- **Tag hygiene**: inconsistent tags needing consolidation
- **Stale content**: old memories without feedback
- **Remediation completeness**: missing fields

## Pressure Testing Methodology

Based on [Reflexion](https://www.promptingguide.ai/techniques/reflexion) research:

1. Generate scenarios from **real failures** (not hypotheticals)
2. Run batch tests - don't skip "because it's obvious"
3. More informative feedback outperforms sparse ("retry")
4. Store results for future reflection sessions

## Quick Reference

| Action | Command |
|--------|---------|
| Full report | `/contextd:reflect` |
| Health only | `/contextd:reflect --health` |
| Apply fixes | `/contextd:reflect --apply` |
| Project scope | `/contextd:reflect --scope=project` |
| Recent only | `/contextd:reflect --since=7d` |
| Full brainstorm | `/contextd:reflect --all-brainstorm` |

## Anti-Patterns

| Mistake | Why It Fails |
|---------|--------------|
| Skipping pressure tests | "Fixed" docs don't actually prevent behavior |
| Modifying plugin source | Breaks on plugin update; use includes |
| Auto-applying security fixes | High-stakes changes need review |
| Ignoring frequency | 10 TDD skips is systemic, not minor |
| Absolute claims in fixes | "This prevents X" → "This helps reduce X" |
