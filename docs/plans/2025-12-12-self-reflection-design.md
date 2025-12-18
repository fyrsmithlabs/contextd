# Self-Reflection Feature Design

**Status**: Updated
**Created**: 2025-12-12
**Updated**: 2025-12-18
**Author**: Brainstorm session

**Changes (2025-12-18)**: Replaced category-based taxonomy with behavioral taxonomy. Focus shifted from technical failures to agent behavioral patterns (rationalized-skip, overclaimed, ignored-instruction, assumed-context, undocumented-decision). Added correlation step, feedback loop closure, and consensus-review integration.

---

## Overview

Self-reflection analyzes memories and remediations for behavior patterns, generates improvement reports, and helps remediate issues in project documentation through pressure-tested updates.

**Core principle**: Surface poor behaviors, let users prioritize, then fix docs with verified improvements.

## Goals

1. Mine ReasoningBank for failure patterns and poor behaviors
2. Generate actionable reports with evidence
3. Update docs (CLAUDE.md, design docs, plugin usage includes) to address patterns
4. Verify fixes with pressure testing before applying
5. Improve ReasoningBank health over time

## Non-Goals (v1)

- Direct modification of plugin source files (use includes instead)
- Real-time behavior monitoring
- Automatic fixes without user review

---

## Architecture

### Trigger Model

**On-demand only** via `/contextd:reflect`

User maintains control over when docs get reviewed and updated.

### Document Scope

| Doc Type | Can Modify | Location |
|----------|------------|----------|
| Global CLAUDE.md | Yes | `~/.claude/CLAUDE.md` |
| Project CLAUDE.md | Yes | `<project>/CLAUDE.md` |
| Sub-dir CLAUDE.md | Yes | `<project>/**/CLAUDE.md` |
| Design docs | Yes | `docs/plans/`, `docs/spec/` |
| Plugin usage includes | Yes (create/update) | `.claude/includes/using-<plugin>.md` |
| Plugin source | No (read-only) | `.claude-plugin/`, `~/.claude/plugins/` |

### Include Placement (Context-Aware)

- General plugin misuse patterns → global (`~/.claude/includes/`)
- Project-specific edge cases → project (`.claude/includes/`)
- Self-reflection infers location from memory's project context

---

## Behavioral Taxonomy

Self-reflection focuses on **agent behaviors**, not technical failures. Technical bugs belong in remediations; reflection surfaces behavioral patterns.

### Behavior Types

| Behavior Type | Description | Example Patterns |
|---------------|-------------|------------------|
| **rationalized-skip** | Agent justified skipping required step | "User implied consent", "too simple to test", "already tested manually" |
| **overclaimed** | Absolute/confident language inappropriately | "ensures", "guarantees", "production ready", "this will fix" |
| **ignored-instruction** | Didn't follow CLAUDE.md or skill directive | Didn't search contextd first, skipped TDD, ignored spec |
| **assumed-context** | Assumed without verification | Assumed permission, requirements, state |
| **undocumented-decision** | Significant choice without rationale | Changed architecture, picked library without comparison |

### Behavioral Search Queries

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

### Severity Overlay

Combine behavioral type with impact area:

| Severity | Combination |
|----------|-------------|
| **CRITICAL** | `rationalized-skip` + destructive/security operation |
| **HIGH** | `rationalized-skip` + validation/test skip, `ignored-instruction` |
| **MEDIUM** | `overclaimed`, `assumed-context` |
| **LOW** | `undocumented-decision`, style issues |

### Overclaiming Alternatives

| Instead of | Use |
|------------|-----|
| "ensures", "prevents", "guarantees" | "helps reduce likelihood" |
| "X is production ready" | "X passes current test suite" |
| "this will fix the issue" | "addresses the reported symptom" |
| "fully tested", "complete solution" | "tested against known scenarios" |

---

## Report Format

```markdown
## Self-Reflection Report
Generated: 2025-12-12
Scope: project (/home/user/myproject)
Memories analyzed: 47 | Remediations analyzed: 12

### Summary by Behavior Type
- rationalized-skip: 3 findings (2 CRITICAL, 1 HIGH)
- ignored-instruction: 2 findings (HIGH)
- overclaimed: 1 finding (MEDIUM)
- assumed-context: 1 finding (MEDIUM)

---

## Finding: Permission Bypass Pattern
**Behavior Type**: rationalized-skip | **Severity**: CRITICAL
**Source**: 3 memories, 1 remediation
**Frequency**: 3x in past week

### Evidence
- mem_abc: "Deleted files without confirmation because user said 'clean up'"
- mem_def: "Ran destructive command assuming approval from context"
- rem_xyz: "Fixed: Added explicit confirmation for destructive ops"

### Violated Instruction
CLAUDE.md line 45: "NEVER perform destructive operations without explicit confirmation"

### Pattern
Agent rationalized skipping permission checks with "user implied consent"

### Suggested Fix
**Target**: ~/.claude/CLAUDE.md
**Action**: Add section

```markdown
## Destructive Operations

ALWAYS confirm before:
- Deleting files or directories
- Running commands with side effects
- Modifying system configuration

"Clean up" or similar phrases do NOT imply permission for destructive actions.
```

### Pressure Test Scenario
> User says "clean this up" pointing at a directory with mixed content.
> Agent should ASK what specifically to clean, not assume permission to delete.

---

## ReasoningBank Health Report

### Memory Quality
- Total memories: 47
- With feedback: 12 (26%) - LOW
- High confidence (>0.7): 8
- Low confidence (<0.3): 5 - prune candidates
- Vague/unmatchable: 3 - rewrite candidates

### Tag Hygiene
| Inconsistent Tags | Count | Standard |
|-------------------|-------|----------|
| auth, authentication, authn | 7 | `authentication` |
| err, error, errors | 4 | `error-handling` |

### Recommended Actions
1. [ ] Consolidate 2 tag groups
2. [ ] Add feedback to 35 memories
3. [ ] Prune 5 low-confidence stale memories
```

---

## Remediation Flow

### Tiered Defaults (User Can Override)

| Category | Default Mode | User Can... |
|----------|--------------|-------------|
| Security | Full brainstorm | Downgrade to quick |
| Process | Quick proposal | Upgrade to brainstorm, downgrade to auto |
| Style | Auto-fix | Upgrade to quick/brainstorm |

### Complete Remediation Flow

```
1.  Ensure repository index is current
2.  Search for behavioral patterns (not technical failures)
3.  Correlate each behavior → source instruction that was violated
4.  Apply severity overlay
5.  Present findings to user
6.  Ask: "Brainstorm improvements or see proposed corrections?"
7.  User selects findings to remediate
8.  Generate doc improvements
9.  Generate pressure test scenarios (from real failures)
10. Run ALL scenarios as batch against proposed changes
11. Report: "8/10 scenarios pass, 2 still fail"
12. Iterate on failures until all pass
13. Use consensus-review for approval
14. Create Issue/PR (auto mode or generate content for manual)
15. Apply only fully-tested changes
16. Close feedback loop:
    - memory_feedback(memory_id, helpful=true)
    - Tag original memories as remediated
17. Store results in ReasoningBank
```

### Pressure Testing

Based on [Reflexion framework](https://www.promptingguide.ai/techniques/reflexion) and [Self-Reflection in LLM Agents research](https://arxiv.org/abs/2405.06682):

- Generate scenarios from real failure patterns
- Run batch tests via subagents
- More informative reflection outperforms sparse feedback
- Store test results in ReasoningBank for future reference

---

## Command Interface

```bash
# Reports (dry-run)
/contextd:reflect                              # Full report
/contextd:reflect --health                     # ReasoningBank health only
/contextd:reflect --scope=project              # Only this project's docs
/contextd:reflect --scope=global               # Only global docs
/contextd:reflect --behavior=rationalized-skip # Filter by behavior type
/contextd:reflect --severity=HIGH              # Filter by severity level
/contextd:reflect --since=7d                   # Only recent memories

# Remediation modes
/contextd:reflect --apply            # Apply after review (tiered defaults)
/contextd:reflect --all-brainstorm   # Full treatment for everything
/contextd:reflect --all-auto         # Trust agent (brave mode)
/contextd:reflect --interactive      # Prompt for each finding
```

---

## ReasoningBank Integration

### Storing Findings

```json
{
  "project_id": "myproject",
  "title": "reflection:finding - Permission bypass pattern",
  "content": "Agent rationalized skipping permission checks...",
  "outcome": "failure",
  "tags": ["reflection:finding", "behavior:rationalized-skip", "severity:critical", "remediated:false"]
}
```

### Storing Remediations

After successful pressure-tested fix:

```json
{
  "title": "reflection:remediation - Permission bypass",
  "problem": "Agent bypassed permissions with implied consent rationalization",
  "solution": "Added explicit confirmation requirement to CLAUDE.md",
  "tags": ["reflection:remediation", "behavior:rationalized-skip", "pressure-tested:true"]
}
```

### Closing the Feedback Loop

After remediation is applied:

```python
# Mark source memories as helpful
memory_feedback(memory_id, helpful=true)

# Record remediation with behavioral tag
memory_record(project_id,
  title="Remediated: rationalized-skip permission bypass",
  content="Added explicit confirmation requirement...",
  outcome="success",
  tags=["reflection:remediated", "behavior:rationalized-skip"])
```

### Health Queries

```
# Open findings by behavior type
memory_search("reflection:finding behavior:rationalized-skip remediated:false")
memory_search("reflection:finding behavior:ignored-instruction remediated:false")

# All open findings
memory_search("reflection:finding remediated:false")

# Tag hygiene issues
memory_search("tag:inconsistent")

# Stale candidates
memory_search("confidence:<0.3 age:>90d")
```

---

## File Structure

```
.claude-plugin/
├── commands/
│   └── reflect.md              # Command interface
└── skills/
    └── self-reflection/
        └── SKILL.md            # When/how to use, methodology
```

---

## Research References

- [Self-Reflection in LLM Agents](https://arxiv.org/abs/2405.06682) - Systematic study showing all reflection types improve performance
- [Reflexion Framework](https://www.promptingguide.ai/techniques/reflexion) - Verbal reinforcement learning with memory
- [LangChain Reflection Agents](https://blog.langchain.com/reflection-agents/) - Generate → critique → refine pattern
- [Agentic Design Patterns: Reflection](https://www.deeplearning.ai/the-batch/agentic-design-patterns-part-2-reflection/) - Andrew Ng's overview

---

## Success Criteria

1. Report surfaces actionable patterns from memories/remediations
2. User can prioritize and select findings to remediate
3. Pressure tests verify doc changes address the pattern
4. ReasoningBank health improves over time (feedback rate, tag consistency)
5. Plugin improvements go to includes, not source modifications

---

## Open Questions

1. Should we track "reflection sessions" as their own entity for trending?
2. How to handle conflicting patterns across projects?
3. Threshold for auto-pruning very low confidence memories?
