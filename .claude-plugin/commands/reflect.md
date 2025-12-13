Analyze memories and remediations for behavior patterns, generate improvement reports, and optionally remediate with pressure-tested doc updates.

## Flags

| Flag | Description |
|------|-------------|
| `--health` | ReasoningBank health report only |
| `--apply` | Apply changes after review (uses tiered defaults) |
| `--scope=project\|global` | Limit to project or global docs |
| `--category=security\|process\|style` | Filter findings by category |
| `--since=<duration>` | Only analyze memories from timeframe (e.g., `7d`, `30d`) |
| `--all-brainstorm` | Full brainstorm treatment for all findings |
| `--all-auto` | Auto-fix all (trust mode) |
| `--interactive` | Prompt for each finding |

## Flow

### 1. Generate Report (Default: Dry-Run)

Search memories and remediations for patterns:

```
memory_search(project_id, "outcome:failure")
memory_search(project_id, "reflection:finding")
remediation_search(query, tenant_id)
```

Categorize findings:
- **Security**: secrets, permissions, destructive ops → always HIGH
- **Process**: TDD, specs, verification → priority compounds with frequency
- **Style**: formatting, naming → LOW regardless of frequency

### 2. Present Findings

For each finding, show:
- Category and priority
- Evidence (memory/remediation IDs with excerpts)
- Pattern description
- Suggested fix with target doc
- Pressure test scenario

### 3. User Prioritizes

User selects which findings to remediate. Respect user's choices.

### 4. Remediation (with `--apply`)

**Tiered defaults** (user can override):
- Security → Full brainstorm
- Process → Quick proposal + approval
- Style → Auto-fix with summary

**Auto-complete flow**:
1. Generate doc improvements
2. Generate pressure test scenarios from real failures
3. Run batch tests via subagents
4. Report pass/fail results
5. Iterate until scenarios pass
6. Present final changes for approval
7. Apply only fully-tested changes

### 5. Store Results

Record findings and remediations in ReasoningBank:

```json
{
  "tags": ["reflection:finding", "category:security", "remediated:true", "pressure-tested:true"]
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

## Common Patterns to Surface

- Permission bypass rationalizations
- Skipping TDD or verification steps
- Absolute language ("ensures", "guarantees", "production ready")
- Not reading specs before implementation
- Assuming without searching contextd first

## Error Handling

@_error-handling.md

If memory/remediation search fails, report partial results and note the gap.
