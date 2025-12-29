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

## Behavioral Taxonomy

Focus on **agent behaviors**, not technical failures.

| Behavior Type | Description | Example Patterns |
|---------------|-------------|------------------|
| **rationalized-skip** | Agent justified skipping required step | "User implied consent", "too simple to test", "already tested" |
| **overclaimed** | Absolute/confident language inappropriately | "ensures", "guarantees", "production ready", "this will fix" |
| **ignored-instruction** | Didn't follow CLAUDE.md or skill directive | Didn't search contextd, skipped TDD, ignored spec |
| **assumed-context** | Assumed without verification | Assumed permission, requirements, state |
| **undocumented-decision** | Significant choice without rationale | Changed architecture, picked library without comparison |

## Severity Overlay

Combine behavioral type with impact area:

| Severity | Combination |
|----------|-------------|
| **CRITICAL** | `rationalized-skip` + destructive/security operation |
| **HIGH** | `rationalized-skip` + validation/test skip, `ignored-instruction` |
| **MEDIUM** | `overclaimed`, `assumed-context` |
| **LOW** | `undocumented-decision`, style issues |

## The Report

For each finding, surface:

1. **Behavior Type** - Which taxonomy category (rationalized-skip, overclaimed, etc.)
2. **Severity** - CRITICAL/HIGH/MEDIUM/LOW
3. **Evidence** - Memory/remediation IDs with excerpts
4. **Violated Instruction** - The skill, command, or CLAUDE.md section that was ignored
5. **Suggested Fix** - Target doc and proposed change
6. **Pressure Scenario** - Test case from real failure

## Remediation Flow

```
Present findings
        ↓
Ask: "Brainstorm or Propose?"
        ↓
User selects findings to remediate
        ↓
Generate doc improvements
        ↓
Correlate behavior → source instruction
        ↓
Generate pressure scenarios (from real failures)
        ↓
Run batch tests via subagents
        ↓
    Pass? ──No──→ Iterate
        ↓ Yes
Use consensus-review for approval
        ↓
Create Issue/PR (auto or generate content)
        ↓
Apply changes
        ↓
Close feedback loop:
  - memory_feedback(memory_id, helpful=true)
  - Tag original memories as remediated
        ↓
Store in ReasoningBank (tags: behavior:<type>, remediated:true, pressure-tested:true)
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

## Behavioral Search Queries

Search for behavioral patterns, not technical errors:

```
# Rationalized skips
memory_search("skip OR skipped OR bypass OR ignored")
memory_search("too simple OR trivial OR obvious")

# User feedback indicating ignored instructions
memory_search("why did you OR should have OR forgot to")
memory_search("didn't you read OR didn't follow")

# Assumptions without verification
memory_search("assumed OR without checking OR without verification")

# Overclaiming
memory_search("ensures OR guarantees OR production ready")
```

**Filter out technical bugs**: Exclude memories tagged with `error:*` or containing stack traces.

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
| Filter by behavior | `/contextd:reflect --behavior=rationalized-skip` |
| Filter by severity | `/contextd:reflect --severity=HIGH` |

## Anti-Patterns

| Mistake | Why It Fails |
|---------|--------------|
| Skipping pressure tests | "Fixed" docs don't actually prevent behavior |
| Modifying plugin source | Breaks on plugin update; use includes |
| Auto-applying security fixes | High-stakes changes need review |
| Ignoring frequency | 10 TDD skips is systemic, not minor |
| Absolute claims in fixes | "This prevents X" → "This helps reduce X" |
