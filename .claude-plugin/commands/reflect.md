Analyze memories and remediations for behavior patterns, generate improvement reports, and optionally remediate with pressure-tested doc updates.

## Flags

| Flag | Description |
|------|-------------|
| `--health` | ReasoningBank health report only |
| `--apply` | Apply changes after review (uses tiered defaults) |
| `--scope=project\|global` | Limit to project or global docs |
| `--behavior=<type>` | Filter by behavior type (rationalized-skip, overclaimed, ignored-instruction, assumed-context, undocumented-decision) |
| `--severity=CRITICAL\|HIGH\|MEDIUM\|LOW` | Filter by severity level |
| `--since=<duration>` | Only analyze memories from timeframe (e.g., `7d`, `30d`) |
| `--all-brainstorm` | Full brainstorm treatment for all findings |
| `--all-auto` | Auto-fix all (trust mode) |
| `--interactive` | Prompt for each finding |

## Flow

### 1. Ensure Repository Index is Current

Before searching, verify the repo index is up to date:

```
repository_index(project_path)
```

### 2. Generate Report (Default: Dry-Run)

Search for **behavioral patterns**, not technical failures:

```
# Behavioral pattern searches (primary)
memory_search(project_id, "skip OR skipped OR bypass OR ignored")
memory_search(project_id, "why did you OR should have OR forgot to")
memory_search(project_id, "rationalized OR justified OR implied consent")
memory_search(project_id, "assumed OR without verification OR without checking")

# Semantic search for instruction violations
repository_search(project_path, "skills commands instructions requirements")
semantic_search(project_path, "agent behavior patterns violations")
```

**Behavioral Taxonomy:**

| Behavior Type | Description | Example Patterns |
|---------------|-------------|------------------|
| **rationalized-skip** | Agent justified skipping required step | "User implied consent", "too simple to test" |
| **overclaimed** | Absolute/confident language inappropriately | "ensures", "guarantees", "production ready" |
| **ignored-instruction** | Didn't follow CLAUDE.md or skill directive | Didn't search contextd, skipped TDD |
| **assumed-context** | Assumed without verification | Assumed permission, requirements, state |
| **undocumented-decision** | Significant choice without rationale | Changed architecture, picked library |

### 3. Correlate Behavior → Source

For each finding, identify which instruction was violated:

```
repository_search(project_path, "<behavior description>")
→ Returns: skill file, command, or CLAUDE.md section that was ignored
```

### 4. Apply Severity Overlay

Combine behavioral type with impact area for priority:

- **CRITICAL**: `rationalized-skip` + destructive/security operation
- **HIGH**: `rationalized-skip` + validation/test skip, `ignored-instruction`
- **MEDIUM**: `overclaimed`, `assumed-context`
- **LOW**: `undocumented-decision`, style issues

### 5. Present Findings

For each finding, show:
- **Behavior Type**: Which taxonomy category
- **Severity**: CRITICAL/HIGH/MEDIUM/LOW
- **Evidence**: Memory/remediation IDs with excerpts
- **Violated Instruction**: The skill, command, or CLAUDE.md section that was ignored
- **Suggested Fix**: Proposed doc improvement

### 6. User Interaction

Ask user: **"Would you like to brainstorm improvements or see proposed corrections?"**

- **Brainstorm**: Full exploration of root causes and solutions
- **Propose**: Quick proposals for approval

User selects which findings to remediate. Respect user's choices.

### 7. Pressure Test Proposed Changes

For each proposed fix:

1. Generate test scenarios from the original failure
2. Simulate agent behavior with proposed instruction
3. Verify the instruction would have prevented the original behavior
4. Report pass/fail for each scenario

```
# Example pressure test
Scenario: "Agent skipped TDD because 'function is trivial'"
Proposed fix: Add to CLAUDE.md: "No function is too trivial for tests. Write test first."
Test: Would this instruction have prevented the skip? → PASS/FAIL
```

### 8. Review Summary

Present consolidated findings with:
- Total findings by behavior type
- Proposed changes with pressure test results
- Files to be modified

Use `consensus-review` for approval of changes.

### 9. Issue/PR Creation

After approval:
- **Auto mode**: Create issue/PR with generated content
- **Manual mode**: Generate content for user to copy

Include in PR/issue:
- Behavioral pattern addressed
- Evidence from memories/remediations
- Pressure test results

### 10. Close Feedback Loop

After remediation:

```
memory_feedback(memory_id, helpful=true)  # For memories that led to improvements
memory_record(project_id, title, content, outcome="success", tags=["reflection:remediated", "behavior:<type>"])
```

Tag original memories as addressed to prevent re-surfacing.

### 11. Store Results

Record findings and remediations in ReasoningBank:

```json
{
  "tags": ["reflection:finding", "behavior:rationalized-skip", "remediated:true", "pressure-tested:true"]
}
```

## Health Report (`--health`)

Analyze ReasoningBank quality:

- **Memory quality**: feedback rate, confidence distribution, vague entries
- **Tag hygiene**: inconsistent tags, suggested consolidations
- **Remediation coverage**: completeness of fields
- **Stale content**: old unfeedback'd memories, outdated remediations

Suggest actions: consolidate tags, prune stale, add feedback, complete partials.

## Doc Targets

| Doc Type | Modifiable | Location |
|----------|------------|----------|
| Global CLAUDE.md | Yes | `~/.claude/CLAUDE.md` |
| Project CLAUDE.md | Yes | `<project>/CLAUDE.md` |
| Sub-dir CLAUDE.md | Yes | `<project>/**/CLAUDE.md` |
| Design docs | Yes | `docs/plans/`, `docs/spec/` |
| Plugin usage includes | Yes | `.claude/includes/using-<plugin>.md` |
| Plugin source | **No** | Use includes instead |

## Behavioral Patterns to Surface

| Behavior Type | Indicators in Memories |
|---------------|------------------------|
| **rationalized-skip** | "too simple", "user implied", "already tested", "obvious", "trivial" |
| **overclaimed** | "ensures", "guarantees", "production ready", "this will fix", "definitely" |
| **ignored-instruction** | User asked "why did you", "should have", "forgot to", "didn't you read" |
| **assumed-context** | "assumed", "figured", "seemed like", "probably", "I thought" |
| **undocumented-decision** | Architectural changes without rationale, library choices without comparison |

**Search queries that surface behavioral issues:**
```
"why did you" OR "should have" OR "forgot to"
"skip" OR "skipped" OR "bypass" OR "ignored"
"assumed" OR "without checking" OR "without verification"
"too simple" OR "trivial" OR "obvious"
```

## Error Handling

@_error-handling.md

If memory/remediation search fails, report partial results and note the gap.
