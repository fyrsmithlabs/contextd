# Self-Reflection Feature Design

**Status**: Draft
**Created**: 2025-12-12
**Author**: Brainstorm session

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

## Finding Categories & Priority

### Categories

| Category | Description | Examples |
|----------|-------------|----------|
| **Security** | Secrets, permissions, destructive ops | Credentials in context, permission bypass |
| **Process** | Workflow discipline violations | Skipping TDD, not reading specs, no verification |
| **Style** | Formatting, naming, minor consistency | Inconsistent naming, formatting drift |

### Priority Formula

```
priority = base_impact × frequency_multiplier(category)

frequency_multiplier:
  security: 1.0        # Always urgent, frequency irrelevant
  process:  log(f + 1) # Compounds - 10 skips >> 1 skip
  style:    0.1        # Low priority regardless of frequency
```

### Common Agent Absolutes to Flag

Poor language patterns that indicate overconfidence:

- "ensures", "prevents", "guarantees"
- "X is production ready"
- "this will fix the issue"
- "fully tested", "complete solution"

Better alternatives:

- "helps reduce likelihood"
- "X passes current test suite"
- "addresses the reported symptom"
- "tested against known scenarios"

---

## Report Format

```markdown
## Self-Reflection Report
Generated: 2025-12-12
Scope: project (/home/user/myproject)
Memories analyzed: 47 | Remediations analyzed: 12

### Summary
- Security findings: 1 (HIGH)
- Process findings: 4 (3 HIGH, 1 MEDIUM)
- Style findings: 2 (LOW)

---

## Finding: Permission Bypass Pattern
**Category**: Security | **Priority**: HIGH
**Source**: 3 memories, 1 remediation
**Frequency**: 3x in past week

### Evidence
- mem_abc: "Deleted files without confirmation because user said 'clean up'"
- mem_def: "Ran destructive command assuming approval from context"
- rem_xyz: "Fixed: Added explicit confirmation for destructive ops"

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

### Auto-Complete Flow

```
1. User selects findings to remediate
2. Agent generates doc improvements
3. Agent generates pressure test scenarios (from real failures)
4. Run ALL scenarios as batch against proposed changes
5. Report: "8/10 scenarios pass, 2 still fail"
6. Iterate on failures until all pass
7. Present final changes for approval
8. Apply only fully-tested changes
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
/contextd:reflect                    # Full report
/contextd:reflect --health           # ReasoningBank health only
/contextd:reflect --scope=project    # Only this project's docs
/contextd:reflect --scope=global     # Only global docs
/contextd:reflect --category=security  # Filter by category
/contextd:reflect --since=7d         # Only recent memories

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
  "tags": ["reflection:finding", "category:security", "remediated:false"]
}
```

### Storing Remediations

After successful pressure-tested fix:

```json
{
  "title": "reflection:remediation - Permission bypass",
  "problem": "Agent bypassed permissions with implied consent rationalization",
  "solution": "Added explicit confirmation requirement to CLAUDE.md",
  "tags": ["reflection:remediation", "pressure-tested:true"]
}
```

### Health Queries

```
memory_search("reflection:finding remediated:false")  # Open findings
memory_search("tag:inconsistent")                     # Tag hygiene issues
memory_search("confidence:<0.3 age:>90d")            # Stale candidates
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
