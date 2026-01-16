---
name: consensus-review
description: Run a multi-agent consensus code review on specified files or directories. Dispatches 4 parallel agents (Security, Correctness, Architecture, UX/Docs) and synthesizes findings.
arguments:
  - name: path
    description: "File path, directory, PR reference, or scope description to review"
    required: true
---

# /consensus-review

Run a multi-agent consensus code review on specified files or directories.

## Usage

```
/contextd:consensus-review <path-or-scope>
```

## Arguments

- `<path-or-scope>`: File path, directory, or scope description
  - File: `/path/to/file.go`
  - Directory: `/path/to/package/`
  - PR: `PR #123` or commit range
  - Description: `"recent changes"` or `"unstaged files"`

## Examples

```
/contextd:consensus-review ./internal/sanitize/
/contextd:consensus-review .claude-plugin/
/contextd:consensus-review "unstaged changes"
/contextd:consensus-review "commit abc123"
```

## Behavior

1. **Parse scope**: Determine files to review
   - If directory: include all files recursively
   - If "unstaged": run `git diff` to get changed files
   - If commit: run `git show <commit>` to get diff

2. **Dispatch 4 parallel agents** using Task tool:
   - Security Reviewer
   - Correctness Reviewer
   - Architecture Reviewer
   - UX/Documentation Reviewer

3. **Wait for all agents** to complete (AgentOutputTool)

4. **Synthesize findings**:
   - Tally issues by severity
   - Identify consensus (issues flagged by 2+ agents)
   - De-duplicate similar findings
   - Prioritize: Critical > High > Medium > Low

5. **Present results** as summary table + recommendations:

```markdown
# Consensus Review: [scope]

## Summary
| Agent | Issues | Critical | High | Medium | Low |
|-------|--------|----------|------|--------|-----|

## Critical Issues (Must Fix)
...

## High Priority (Should Fix)
...

## Recommendations
1. ...
2. ...
3. ...
```

6. **Record memory** with key findings

## Agent Focus Areas

| Agent | Reviews |
|-------|---------|
| Security | Injection, secrets, supply chain, permissions |
| Correctness | Logic, schemas, edge cases, validation |
| Architecture | Structure, patterns, maintainability |
| UX/Docs | Clarity, examples, error guidance |

## Severity Guide

| Severity | Meaning | Action |
|----------|---------|--------|
| CRITICAL | Security hole, data loss, breaks build | Block until fixed |
| HIGH | Significant bug, poor UX, debt | Fix before merge |
| MEDIUM | Should improve | Plan to fix |
| LOW | Nice to have | Backlog |

## Error Handling

1. **If scope not found**:
   - Display: "Could not find files for: [scope]"
   - Suggest: Check path exists or clarify scope

2. **If agent fails**:
   - Continue with remaining agents
   - Note which agent failed in output
   - Partial results are still valuable

3. **If no issues found**:
   - Display: "No issues found by any reviewer"
   - Note: This is rare - consider if scope was too narrow

## Related

- **Skill**: `contextd:consensus-review` - Full workflow documentation
- **Single review**: Use Task tool directly for one reviewer type
- **PR review**: `/pr-review-toolkit:review-pr` for GitHub PR integration
