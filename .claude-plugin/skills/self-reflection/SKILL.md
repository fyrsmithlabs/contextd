---
name: self-reflection
description: Use when reviewing agent behavior patterns, improving CLAUDE.md based on past failures, or checking ReasoningBank health - mines memories for poor behaviors and pressure-tests improvements
---

# Self-Reflection

Mine memories and remediations for behavior patterns, surface findings to user, remediate docs with pressure-tested improvements.

**Core loop:** Search -> Report -> User prioritizes -> Brainstorm -> Pressure test -> Apply

## When to Use

- Periodic review of agent behavior patterns
- After series of failures or poor outcomes
- Before major project milestones
- When CLAUDE.md feels stale or incomplete
- To check ReasoningBank health

## When NOT to Use

- Immediate error diagnosis -> use `contextd-workflow` skill
- Recording a single learning -> use `/remember`
- Checkpoint management -> use `contextd-workflow` skill

## Behavioral Taxonomy

Focus on **agent behaviors**, not technical failures:

| Behavior Type | Description | Examples |
|---------------|-------------|----------|
| **rationalized-skip** | Justified skipping required step | "too simple to test", "user implied consent" |
| **overclaimed** | Absolute language inappropriately | "ensures", "guarantees", "production ready" |
| **ignored-instruction** | Didn't follow CLAUDE.md/skill | Skipped contextd search, ignored TDD |
| **assumed-context** | Assumed without verification | Assumed permission, requirements, state |
| **undocumented-decision** | Significant choice without rationale | Changed architecture without comparison |

## Severity Overlay

| Severity | Combination |
|----------|-------------|
| **CRITICAL** | rationalized-skip + destructive/security operation |
| **HIGH** | rationalized-skip + validation skip, ignored-instruction |
| **MEDIUM** | overclaimed, assumed-context |
| **LOW** | undocumented-decision, style issues |

## The Report

For each finding, surface:

1. **Behavior Type** - Which taxonomy category
2. **Severity** - CRITICAL/HIGH/MEDIUM/LOW
3. **Evidence** - Memory/remediation IDs with excerpts
4. **Violated Instruction** - The skill/command/CLAUDE.md section ignored
5. **Suggested Fix** - Target doc and proposed change
6. **Pressure Scenario** - Test case from real failure

## Remediation Flow

```
Present findings
      |
User selects findings to remediate
      |
Generate doc improvements
      |
Generate pressure scenarios (from real failures)
      |
Run batch tests via subagents
      |
  Pass? --No--> Iterate
      | Yes
Create Issue/PR
      |
Apply changes
      |
Close feedback loop:
  memory_feedback(memory_id, helpful=true)
  Tag original memories as remediated
```

## Behavioral Search Queries

```
# Rationalized skips
memory_search("skip OR skipped OR bypass OR ignored")
memory_search("too simple OR trivial OR obvious")

# User feedback indicating ignored instructions
memory_search("why did you OR should have OR forgot to")

# Assumptions without verification
memory_search("assumed OR without checking")

# Overclaiming
memory_search("ensures OR guarantees OR production ready")
```

Filter out technical bugs: Exclude memories with `error:*` tags or stack traces.

## ReasoningBank Health

`--health` flag analyzes:

- **Memory quality**: feedback rate, confidence distribution
- **Tag hygiene**: inconsistent tags needing consolidation
- **Stale content**: old memories without feedback
- **Remediation completeness**: missing fields

## Quick Reference

| Action | Command |
|--------|---------|
| Full report | `/reflect` |
| Health only | `/reflect --health` |
| Apply fixes | `/reflect --apply` |
| Recent only | `/reflect --since=7d` |
| Filter by behavior | `/reflect --behavior=rationalized-skip` |
| Filter by severity | `/reflect --severity=HIGH` |

## Anti-Patterns

| Mistake | Why It Fails |
|---------|--------------|
| Skipping pressure tests | "Fixed" docs don't actually prevent behavior |
| Modifying plugin source | Breaks on update; use includes |
| Auto-applying security fixes | High-stakes changes need review |
| Ignoring frequency | 10 TDD skips is systemic, not minor |
| Absolute claims in fixes | "This prevents X" -> "This helps reduce X" |
