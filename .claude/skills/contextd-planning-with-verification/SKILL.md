---
name: contextd-planning-with-verification
description: Use when creating TodoWrite for major work (features, bugs, refactoring, security, multi-file changes) - automatically adds verification subtasks to prevent unverified completion claims and forgotten evidence requirements
---

# Planning with Verification

## Overview

**Core Principle**: Every TodoWrite for major work MUST include verification subtasks. Verification is not optional, not implicit, and not "at the end."

**Following the spirit without the letter IS a violation.** Use exact templates, all required subtasks, no shortcuts.

This skill enforces systematic verification by automatically adding verification subtasks when creating todos.

## When to Use This Skill

Use when creating TodoWrite with ANY of these characteristics:
- Features, bug fixes, refactoring
- Security changes, performance improvements
- Multi-file changes, new file creation
- Documentation for complex features
- Anything affecting APIs, multi-tenancy, or core functionality

**DO NOT use for**:
- Pure reading/analysis (no implementation, no deliverable - don't create TodoWrite)
- Pure exploration research (no deliverable - don't create TodoWrite)
- Conversational requests (no work output)

**DO use for research with deliverables**:
- Research producing specs, ADRs, or decisions → Add verification subtask for deliverable
- SDK/library evaluation → Add verification of evaluation completeness

## Task Classification

### Major Tasks → Use completing-major-task

**Triggers**:
- "implement", "fix", "refactor", "add", "update" in task description
- Multiple files affected
- Security or multi-tenant implications
- New functionality or behavior changes
- Complex documentation (guides, specs, API docs with examples)

**Examples**: Implement auth, fix bug, refactor service, add MCP tool, update API, write DEVELOPMENT-WORKFLOW.md guide

### Minor Tasks → Use completing-minor-task

**Triggers**:
- "typo", "comment", "format", "whitespace" in task description
- Single file, cosmetic only
- No functional impact

**Examples**: Fix typo, update comment, format code

## Mandatory Verification Subtasks

### Pattern: Feature Implementation

**WITHOUT skill** (baseline):
```json
[
  {"content": "Implement user authentication", "status": "pending", "activeForm": "Implementing user authentication"},
  {"content": "Write tests for authentication", "status": "pending", "activeForm": "Writing tests for authentication"}
]
```

**WITH skill** (required):
```json
[
  {"content": "Implement user authentication", "status": "pending", "activeForm": "Implementing user authentication"},
  {"content": "Write tests for authentication (≥80% coverage)", "status": "pending", "activeForm": "Writing tests for authentication"},
  {"content": "Verify authentication implementation (completing-major-task)", "status": "pending", "activeForm": "Verifying authentication implementation"},
  {"content": "Run security checks (gosec, multi-tenant isolation)", "status": "pending", "activeForm": "Running security checks"},
  {"content": "Update CHANGELOG.md", "status": "pending", "activeForm": "Updating CHANGELOG.md"}
]
```

### Pattern: Bug Fix

**WITHOUT skill**:
```json
[
  {"content": "Fix JWT validation for special characters", "status": "pending", "activeForm": "Fixing JWT validation"},
  {"content": "Add regression test", "status": "pending", "activeForm": "Adding regression test"}
]
```

**WITH skill**:
```json
[
  {"content": "Fix JWT validation for special characters", "status": "pending", "activeForm": "Fixing JWT validation"},
  {"content": "Add regression test for special characters", "status": "pending", "activeForm": "Adding regression test"},
  {"content": "Verify bug fix (completing-major-task)", "status": "pending", "activeForm": "Verifying bug fix"},
  {"content": "Run security checks (input validation)", "status": "pending", "activeForm": "Running security checks"},
  {"content": "Update CHANGELOG.md (Fixed section)", "status": "pending", "activeForm": "Updating CHANGELOG.md"}
]
```

### Pattern: Refactoring

**WITHOUT skill**:
```json
[
  {"content": "Rename CheckpointSvc to CheckpointService", "status": "pending", "activeForm": "Renaming CheckpointSvc"},
  {"content": "Update tests", "status": "pending", "activeForm": "Updating tests"}
]
```

**WITH skill**:
```json
[
  {"content": "Rename CheckpointSvc to CheckpointService", "status": "pending", "activeForm": "Renaming CheckpointSvc"},
  {"content": "Update all references in tests", "status": "pending", "activeForm": "Updating test references"},
  {"content": "Verify refactoring (completing-major-task)", "status": "pending", "activeForm": "Verifying refactoring"},
  {"content": "Run full test suite (≥80% coverage maintained)", "status": "pending", "activeForm": "Running full test suite"},
  {"content": "Update CHANGELOG.md (Changed section)", "status": "pending", "activeForm": "Updating CHANGELOG.md"}
]
```

### Pattern: Minor Task

**WITHOUT skill**:
```json
[
  {"content": "Fix typo in README.md", "status": "pending", "activeForm": "Fixing typo"}
]
```

**WITH skill**:
```json
[
  {"content": "Fix typo in README.md", "status": "pending", "activeForm": "Fixing typo"},
  {"content": "Verify typo fix (completing-minor-task)", "status": "pending", "activeForm": "Verifying typo fix"}
]
```

## Verification Subtask Templates

### Always Add (Major Tasks):
```json
{"content": "Verify [task name] (completing-major-task)", "status": "pending", "activeForm": "Verifying [task name]"}
```

### Always Add (Major Tasks):
```json
{"content": "Run build and tests (≥80% coverage)", "status": "pending", "activeForm": "Running build and tests"}
```

### Always Add (Major Tasks):
```json
{"content": "Update CHANGELOG.md", "status": "pending", "activeForm": "Updating CHANGELOG.md"}
```

### Add for Security-Sensitive (Auth, Multi-Tenant, Input Validation):
```json
{"content": "Run security checks (gosec, multi-tenant isolation)", "status": "pending", "activeForm": "Running security checks"}
```

### Add for Bug Fixes:
```json
{"content": "Add regression test for [bug description]", "status": "pending", "activeForm": "Adding regression test"}
```

### Always Add (Minor Tasks):
```json
{"content": "Verify [task name] (completing-minor-task)", "status": "pending", "activeForm": "Verifying [task name]"}
```

## Detection Rules

### Detect Major Tasks:
- Contains: "implement", "add", "create", "fix", "refactor", "update", "modify"
- AND affects functionality (not cosmetic)

### Detect Security-Sensitive:
- Contains: "auth", "token", "session", "credential", "password", "multi-tenant", "isolation"
- OR file paths: `pkg/auth/`, `pkg/session/`, `pkg/mcp/auth`

### Detect Minor Tasks:
- Contains: "typo", "comment", "format", "whitespace", "cosmetic"
- AND single file only
- AND no functional impact

## Integration with Completion Skills

**Verification subtask ordering** (dependencies):
1. Implementation tasks (feature code, bug fix, refactoring)
2. Test tasks (unit tests, regression tests, integration tests)
3. Build/test execution ("Run build and tests")
4. Security checks ("Run security checks") - if applicable
5. Verification task ("Verify X using completing-*-task") - uses evidence from above
6. CHANGELOG update

**After completing implementation todos, the verification todo triggers**:

1. **Major task verification todo** → Invoke `contextd:completing-major-task`
   - Provides comprehensive template (build, tests, security, functionality, risk)
   - Evidence required before marking complete

2. **Minor task verification todo** → Invoke `contextd:completing-minor-task`
   - Provides self-interrogation checklist
   - Evidence required before marking complete

3. **Before PR creation** → Invoke `contextd:code-review`
   - Validates all verification evidence
   - Blocks merge if verification insufficient

## Common Rationalizations (Don't Fall For These)

| Excuse | Reality |
|--------|---------|
| "Verification todos add clutter" | Clutter prevents forgotten verification. Explicit todos = accountability. |
| "I'll verify at the end" | Batch verification = forgotten verification. Per-task verification catches issues early. |
| "Task too simple for verification" | Simple tasks still need evidence. No task exempt from verification. |
| "Verification is implicit in my workflow" | Implicit = forgotten. Explicit todos enforce discipline. |
| "Minor tasks don't need verification subtasks" | Minor tasks need completing-minor-task. ALL tasks need verification. |
| "This is urgent, skip verification" | Urgent tasks that fail verification waste MORE time. Always verify. |
| "I've done this before, verification unnecessary" | Past success ≠ future guarantee. Always verify. |
| "Batching verification is more efficient" | Batching = forgetting. Per-task verification is the discipline. |
| "I added completing-major-task subtask, that's enough" | Partial verification = incomplete verification. ALL required subtasks mandatory. |
| "This is research, not implementation" | Research with deliverables (specs, ADRs) needs verification of completeness. |
| "Just updating docs, only need minor verification" | Complex docs (guides, specs) = major task. Simple typos = minor task. |

## Red Flags - STOP and Add Verification Subtasks

If you find yourself thinking:
- "This is straightforward, don't need verification todos"
- "I'll remember to verify"
- "Verification is obvious, doesn't need a todo"
- "Too urgent for verification overhead"
- "Verification at the end is sufficient"

**All of these mean: Add verification subtasks NOW.**

## Enforcement Rules

### Rule 1: No Major Task Without completing-major-task Subtask
Every major task MUST have verification subtask that invokes completing-major-task skill.

### Rule 2: No Minor Task Without completing-minor-task Subtask
Every minor task MUST have verification subtask that invokes completing-minor-task skill.

### Rule 3: Security-Sensitive Tasks MUST Have Security Verification
Tasks involving auth, multi-tenant, input validation MUST have security check subtask.

### Rule 4: All Tasks MUST Update CHANGELOG (with Exemptions)
**Require CHANGELOG update for**:
- Features, bug fixes, refactoring, performance improvements
- User-facing documentation (README, API docs, guides)
- Breaking changes, security fixes

**Exempt from CHANGELOG**:
- Internal documentation only (CLAUDE.md, code comments)
- Simple typo fixes in docs
- Test-only changes (no production code affected)

### Rule 5: Verification is Per-Task, Not Batched
Each implementation task gets its own verification subtask. No "verify everything at end" shortcut.

## Quick Reference

| Task Type | Verification Subtasks to Add |
|-----------|------------------------------|
| Feature | completing-major-task + build/test + security (if applicable) + CHANGELOG |
| Bug Fix | completing-major-task + regression test + build/test + CHANGELOG |
| Refactoring | completing-major-task + full test suite + CHANGELOG |
| Security Change | completing-major-task + security checks + build/test + CHANGELOG |
| Minor Task | completing-minor-task only |

## Examples in Context

### Example 1: Urgent Production Bug

**User request**: "JWT validation is broken in production! Fix ASAP!"

**TodoWrite (WRONG - no skill)**:
```json
[
  {"content": "Fix JWT validation bug", "status": "pending", "activeForm": "Fixing JWT validation bug"},
  {"content": "Deploy hotfix", "status": "pending", "activeForm": "Deploying hotfix"}
]
```

**TodoWrite (RIGHT - with skill)**:
```json
[
  {"content": "Fix JWT validation bug", "status": "pending", "activeForm": "Fixing JWT validation bug"},
  {"content": "Add regression test for JWT validation", "status": "pending", "activeForm": "Adding regression test"},
  {"content": "Verify bug fix (completing-major-task)", "status": "pending", "activeForm": "Verifying bug fix"},
  {"content": "Run security checks (input validation, token handling)", "status": "pending", "activeForm": "Running security checks"},
  {"content": "Update CHANGELOG.md (Fixed section)", "status": "pending", "activeForm": "Updating CHANGELOG.md"},
  {"content": "Deploy hotfix", "status": "pending", "activeForm": "Deploying hotfix"}
]
```

### Example 2: Simple Refactoring

**User request**: "Rename svc to service for consistency"

**TodoWrite (WRONG - no skill)**:
```json
[
  {"content": "Rename svc to service", "status": "pending", "activeForm": "Renaming svc to service"}
]
```

**TodoWrite (RIGHT - with skill)**:
```json
[
  {"content": "Rename svc to service in all files", "status": "pending", "activeForm": "Renaming svc to service"},
  {"content": "Update tests to match new naming", "status": "pending", "activeForm": "Updating tests"},
  {"content": "Verify refactoring (completing-major-task)", "status": "pending", "activeForm": "Verifying refactoring"},
  {"content": "Run full test suite (≥80% coverage)", "status": "pending", "activeForm": "Running full test suite"},
  {"content": "Update CHANGELOG.md (Changed section)", "status": "pending", "activeForm": "Updating CHANGELOG.md"}
]
```

### Example 3: Documentation Typo (Minor Task)

**User request**: "Fix typo in README - 'Installtion' should be 'Installation'"

**TodoWrite (WRONG - no skill)**:
```json
[
  {"content": "Fix typo in README", "status": "pending", "activeForm": "Fixing typo"}
]
```

**TodoWrite (RIGHT - with skill)**:
```json
[
  {"content": "Fix typo 'Installtion' → 'Installation' in README.md", "status": "pending", "activeForm": "Fixing typo"},
  {"content": "Verify typo fix (completing-minor-task)", "status": "pending", "activeForm": "Verifying typo fix"}
]
```

### Example 4: Complex Documentation (Major Task)

**User request**: "Write VERIFICATION-POLICY.md guide explaining verification requirements"

**TodoWrite (WRONG - no skill)**:
```json
[
  {"content": "Write VERIFICATION-POLICY.md", "status": "pending", "activeForm": "Writing VERIFICATION-POLICY.md"}
]
```

**TodoWrite (RIGHT - with skill)**:
```json
[
  {"content": "Write VERIFICATION-POLICY.md with policy sections", "status": "pending", "activeForm": "Writing VERIFICATION-POLICY.md"},
  {"content": "Add examples for major and minor task templates", "status": "pending", "activeForm": "Adding examples"},
  {"content": "Verify documentation (completing-major-task)", "status": "pending", "activeForm": "Verifying documentation"},
  {"content": "Run markdown linting and link checks", "status": "pending", "activeForm": "Running linting and checks"},
  {"content": "Update CHANGELOG.md (Added section)", "status": "pending", "activeForm": "Updating CHANGELOG.md"}
]
```

## The Bottom Line

**Every TodoWrite for work MUST include verification subtasks.**

- Major tasks → completing-major-task verification
- Minor tasks → completing-minor-task verification
- Security tasks → security check verification
- All tasks → evidence before completion

No shortcuts. No "I'll remember." No "too simple." No "too urgent."

Verification subtasks = verification accountability = verification happens.
