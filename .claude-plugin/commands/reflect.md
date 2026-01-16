---
name: reflect
description: Analyze memories and remediations for behavior patterns, evaluate policy compliance, and optionally remediate with pressure-tested doc updates. Use for self-improvement and quality assurance.
arguments:
  - name: flags
    description: "Flags: --health (bank report), --policies (compliance only), --apply (make changes), --scope=project|global, --behavior=<type>, --severity=CRITICAL|HIGH|MEDIUM|LOW"
    required: false
---

Analyze memories and remediations for behavior patterns, evaluate policy compliance, and optionally remediate with pressure-tested doc updates.

## Flags

| Flag | Description |
|------|-------------|
| `--health` | ReasoningBank health report only |
| `--policies` | Policy compliance report only |
| `--apply` | Apply changes after review using **tiered defaults**: security/destructive issues → full brainstorm, process/validation issues → quick proposals, style issues → auto-fix |
| `--scope=project\|global` | Limit to project or global docs |
| `--behavior=<type>` | Filter by behavior type (rationalized-skip, overclaimed, ignored-instruction, assumed-context, undocumented-decision) |
| `--severity=CRITICAL\|HIGH\|MEDIUM\|LOW` | Filter by severity level |
| `--since=<duration>` | Only analyze memories from timeframe (e.g., `7d`, `30d`) |
| `--all-brainstorm` | Full brainstorm treatment for all findings |
| `--all-auto` | Auto-fix all (trust mode) |
| `--interactive` | Prompt for each finding |

## Complete Workflow Examples

### Example 1: Safe Review (Recommended First Run)
```bash
# Generate report without making changes (dry-run)
/contextd:reflect --scope=project

# Review findings, then apply with tiered defaults
/contextd:reflect --apply
```

### Example 2: Policy Compliance Only
```bash
# Check policy compliance for current session
/contextd:reflect --policies

# Policy compliance with specific severity
/contextd:reflect --policies --severity=HIGH
```

### Example 3: High-Stakes Review
```bash
# Full brainstorm for every finding - maximum deliberation
/contextd:reflect --all-brainstorm --scope=global
```

### Example 4: Trust Mode (Experienced Users)
```bash
# Auto-fix all findings without prompts
/contextd:reflect --all-auto
```

### Example 5: Targeted Review
```bash
# Only review rationalized-skip behaviors from last 7 days
/contextd:reflect --behavior=rationalized-skip --since=7d --apply
```

### Tiered Defaults Explained

When using `--apply`, remediation approach is automatically selected by severity:

| Severity | Impact Area | Remediation Approach |
|----------|-------------|---------------------|
| **CRITICAL** | Security, destructive ops | Full brainstorm with root cause analysis |
| **HIGH** | Validation skips, ignored instructions | Quick proposals with approval |
| **MEDIUM** | Overclaims, assumed context | Auto-fix with summary |
| **LOW** | Style, undocumented decisions | Auto-fix silently |

Override with `--all-brainstorm` or `--all-auto` if needed.

## Flow

### 1. Ensure Repository Index is Current

Before searching, verify the repo index is up to date:

```
repository_index(project_path)
```

### 2. Load Active Policies

Search for all enabled policies:

```
memory_search(project_id: "global", query: "type:policy enabled:true", limit: 50)
```

Parse policy content to extract:
- `rule`: The MUST statement
- `category`: verification, process, security, quality
- `severity`: critical, high, medium
- `scope`: global, skill:{name}, project:{path}

### 3. Generate Report (Default: Dry-Run)

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

### 4. Evaluate Policy Compliance

For each enabled policy, check recent actions:

```
┌──────────────────────────────────────────────────────────────────┐
│  POLICY COMPLIANCE CHECK                                         │
├──────────────────────────────────────────────────────────────────┤
│  For each policy:                                                │
│  1. Search recent memories for policy-relevant actions           │
│  2. Determine if action followed or violated the rule            │
│  3. Track violation count and evidence                           │
│  4. Update policy stats (violations++/successes++)               │
└──────────────────────────────────────────────────────────────────┘
```

Example evaluation:
```
Policy: test-before-fix
Rule: "Always run tests before claiming a fix is complete"

Recent memory: "Fixed the null pointer exception by adding a null check"
Evidence: No test run mentioned before claiming fix
Verdict: VIOLATION

Action: Update policy violations count, record finding
```

### 5. Correlate Behavior → Source

For each finding, identify which instruction was violated:

```
repository_search(project_path, "<behavior description>")
→ Returns: skill file, command, or CLAUDE.md section that was ignored
```

If a policy exists for the violated behavior, link to that policy.

### 6. Apply Severity Overlay

Combine behavioral type with impact area for priority:

- **CRITICAL**: `rationalized-skip` + destructive/security operation, critical policy violation
- **HIGH**: `rationalized-skip` + validation/test skip, `ignored-instruction`, high policy violation
- **MEDIUM**: `overclaimed`, `assumed-context`, medium policy violation
- **LOW**: `undocumented-decision`, style issues

### 7. Present Findings

For each finding, show:
- **Behavior Type**: Which taxonomy category
- **Severity**: CRITICAL/HIGH/MEDIUM/LOW
- **Evidence**: Memory/remediation IDs with excerpts
- **Policy Violated**: If applicable, the specific policy rule
- **Violated Instruction**: The skill, command, or CLAUDE.md section that was ignored
- **Suggested Fix**: Proposed doc improvement or policy creation

### Policy Compliance Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ POLICY COMPLIANCE REPORT                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ SUMMARY                                                                     │
│ ───────                                                                     │
│ Policies Checked: 6                                                         │
│ Compliance Rate: 83% (5/6 policies followed)                                │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ VIOLATIONS                                                                  │
│ ──────────                                                                  │
│                                                                             │
│ [HIGH] test-before-fix                                                      │
│        Rule: "Always run tests before claiming a fix is complete"           │
│        Evidence: Memory mem_abc123 - "Fixed bug without running tests"      │
│        Suggestion: Run tests before marking task complete                   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ COMPLIANT                                                                   │
│ ─────────                                                                   │
│ ✓ no-secrets-in-context (critical) - No violations detected                 │
│ ✓ contextd-first (high) - 3 searches before grep observed                   │
│ ✓ verify-before-complete (high) - Verification commands run                 │
│ ✓ consensus-binary (medium) - No "APPROVE WITH RESERVATIONS"                │
│ ✓ no-force-push-main (critical) - No force pushes detected                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 8. User Interaction

Ask user: **"Would you like to brainstorm improvements or see proposed corrections?"**

- **Brainstorm**: Opens full exploration with root cause analysis, multiple solution proposals, trade-off discussion
- **Propose**: Shows quick fix proposals for each finding with approve/reject options

**Selecting Findings:**

User can respond with:
- `"all"` - Fix everything
- `"1, 3, 5"` - Fix specific findings by number
- `"only HIGH"` - Filter by severity (CRITICAL, HIGH, MEDIUM, LOW)
- `"skip 2"` - Skip specific findings
- `"none"` - Exit without changes

User selects which findings to remediate. Respect user's choices.

### 9. Pressure Test Proposed Changes

> **STATUS: v2 Roadmap Item**
>
> Full automated pressure testing is planned for v2. Currently, use the manual
> process below. The v2 implementation will include:
> - Automated scenario generation from memory/remediation context
> - LLM-based counterfactual analysis
> - Integration with the reflection feedback loop

For each proposed fix, manually validate using the following process:

#### Step 1: Extract the Original Context

From the memory/remediation that surfaced the issue, identify:
- **Trigger**: What prompted the problematic behavior?
- **Rationalization**: How did the agent justify it?
- **Outcome**: What negative result occurred?

```
# Example extraction
Memory: "Skipped TDD because function seemed trivial"
Trigger: Simple getter function implementation
Rationalization: "Too simple to need tests"
Outcome: Bug in edge case caught late in review
```

#### Step 2: Generate Test Scenarios

Create 2-3 scenarios that would test the proposed instruction:

| Scenario Type | Description | Purpose |
|---------------|-------------|---------|
| **Direct replay** | Exact situation from original failure | Verify basic fix |
| **Boundary case** | Similar but slightly different situation | Test instruction specificity |
| **Adversarial** | Agent trying to rationalize around instruction | Test instruction robustness |

```
# Example scenarios for "No function is too trivial for tests"
1. Direct replay: "Implement getId() getter"
2. Boundary case: "Implement toString() that just returns a field"
3. Adversarial: "The test would be identical to the implementation"
```

#### Step 3: Evaluate Each Scenario

For each scenario, ask:

1. **Would the instruction apply?** (Yes/No)
   - If No: Instruction may be too narrow
2. **Would it be clear what to do?** (Yes/No)
   - If No: Instruction may be ambiguous
3. **Could it be reasonably bypassed?** (Yes/No)
   - If Yes: Instruction may need stronger language

#### Step 4: Record Results

```
Pressure Test Results:
├── Scenario 1 (Direct): PASS - Instruction clearly applies
├── Scenario 2 (Boundary): PASS - "Trivial" definition is clear
└── Scenario 3 (Adversarial): FAIL - "identical to implementation" creates loophole

Recommendation: Strengthen to "No function is too trivial. If testing seems
pointless, that's a sign the function design may need review."
```

#### Pass/Fail Criteria

- **PASS**: Instruction would clearly prevent the original behavior in >= 2/3 scenarios
- **CONDITIONAL PASS**: Works for direct replay but needs refinement for edge cases
- **FAIL**: Instruction is too vague, narrow, or easily rationalized around

If a pressure test fails, iterate on the proposed instruction before applying.

### 10. Review Summary

Present consolidated findings with:
- Total findings by behavior type
- Policy compliance summary
- Proposed changes with pressure test results
- Files to be modified

Use `consensus-review` for approval of changes.

### 11. Update Policy Stats

After compliance evaluation, update policy statistics:

```
# For each policy violation
# Update the policy memory with incremented violation count

# For each policy followed
# Update the policy memory with incremented success count
```

### 12. Issue/PR Creation

After approval:
- **Auto mode**: Create issue/PR with generated content
- **Manual mode**: Generate content for user to copy

Include in PR/issue:
- Behavioral pattern addressed
- Policy violations (if any)
- Evidence from memories/remediations
- Pressure test results

### 13. Close Feedback Loop

After remediation:

```
memory_feedback(memory_id, helpful=true)  # For memories that led to improvements
memory_record(project_id, title, content, outcome="success", tags=["reflection:remediated", "behavior:<type>"])
```

Tag original memories as addressed to prevent re-surfacing.

### 14. Store Results

Record findings and remediations in ReasoningBank:

```json
{
  "tags": ["reflection:finding", "behavior:rationalized-skip", "remediated:true", "pressure-tested:true"]
}
```

Store reflection report:
- Full report to `.claude/reflections/{timestamp}.md` (git-ignored)
- Summary via `memory_record` with `type:reflection` tag

## Health Report (`--health`)

Analyze ReasoningBank quality:

- **Memory quality**: feedback rate, confidence distribution, vague entries
- **Tag hygiene**: inconsistent tags, suggested consolidations
- **Remediation coverage**: completeness of fields
- **Stale content**: old unfeedback'd memories, outdated remediations
- **Policy health**: enabled/disabled ratio, violation rates, unused policies

Suggest actions: consolidate tags, prune stale, add feedback, complete partials.

## Policy Report (`--policies`)

Policy-focused compliance analysis:

```
/contextd:reflect --policies
```

Outputs:
- List of all enabled policies
- Compliance rate for each
- Recent violations with evidence
- Suggestions for new policies based on behavioral patterns

## Doc Targets

| Doc Type | Modifiable | Location |
|----------|------------|----------|
| Global CLAUDE.md | Yes | `~/.claude/CLAUDE.md` |
| Project CLAUDE.md | Yes | `<project>/CLAUDE.md` |
| Sub-dir CLAUDE.md | Yes | `<project>/**/CLAUDE.md` |
| Design docs | Yes | `docs/plans/`, `docs/spec/` |
| Plugin usage includes | Yes | `.claude/includes/using-<plugin>.md` |
| Plugin source | **No** | Use includes instead |
| Policies | Yes | Via `project-setup` skill |

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

**When searches fail:**

1. **Partial Results**: Show findings from successful searches
   - Example: "3 findings from memories, remediations unavailable"

2. **Gap Communication**: Display warning
   - `⚠️ Remediation search failed. Results may be incomplete.`

3. **User Action**: Continue with available data or retry
   - Use `--verbose` for debugging failed searches
   - Partial results are still valuable - don't block on failures
