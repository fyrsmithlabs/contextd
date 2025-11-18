---
name: contextd-pre-pr-verification
description: Use when about to create PR or request code review, before invoking contextd:code-review - runs comprehensive pre-PR verification checks (build, tests, coverage, security, docs) to catch issues locally and prevent wasting review cycles
---

# Pre-PR Verification

## Overview

**Catch issues BEFORE code review**, not during. This skill runs comprehensive verification checks before PR creation, saving review cycles and CI time.

**Core principle**: Local verification prevents wasted context in review loops.

## When to Use This Skill

**Triggers**:
- About to create pull request
- About to invoke `contextd:code-review` skill
- Before requesting human code review
- After completing implementation work

**Do NOT skip because**:
- "CI will catch it" (catch locally, save CI cycles)
- "Already manually tested" (automated verification required)
- "Just a small change" (all changes need verification)

## The Iron Rule

```
NEVER create PR without running ALL verification checks.
NO EXCEPTIONS.
```

This applies regardless of:
- Who approved proceeding (authority doesn't override standards)
- How small the change is (even typos need verification)
- Time pressure (broken PR wastes more time)
- Confidence level (verification required regardless)

## Mandatory Pre-PR Checklist

Run ALL sections in order. Cannot skip any section.

### 0. Pre-commit Hooks (FIRST - Security Critical)

```bash
# Check if installed
pre-commit --version

# Run all hooks
pre-commit run --all-files

# Verify no --no-verify in recent commits
git log -5 --oneline | grep -v "no-verify"
```

**Requirements**:
- ✅ Pre-commit installed and functional
- ✅ All hooks pass (no failures)
- ✅ No `--no-verify` used in git history
- ✅ TruffleHog passed (no secrets detected)
- ✅ gosec passed (no security vulnerabilities)

**If hooks fail**: Fix issues, do NOT bypass with `--no-verify`

### 1. Build & Test Verification

```bash
# Build
go build ./...

# All tests
go test ./...

# Race detector
go test -race ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

**Requirements**:
- ✅ Build succeeds with no errors
- ✅ All tests pass (no failures, no skips)
- ✅ No race conditions detected
- ✅ Coverage ≥ 80% (critical paths 100%)

**Record output**: Paste actual command output, not "tests passed"

### 2. Code Quality Checks

```bash
# Format
gofmt -w .
git diff --exit-code  # Should be no changes

# Imports
goimports -w .
git diff --exit-code  # Should be no changes

# Linting
golint ./...

# Vet
go vet ./...

# Static analysis
staticcheck ./...

# Security scan
gosec ./...
```

**Requirements**:
- ✅ Code already formatted (gofmt shows no changes)
- ✅ Imports already organized (goimports shows no changes)
- ✅ No lint warnings
- ✅ No vet warnings
- ✅ No staticcheck issues
- ✅ No new gosec findings

**If changes needed**: Apply them, commit, then re-run verification

### 3. Documentation Verification

```bash
# Check CHANGELOG updated
git diff main -- CHANGELOG.md

# Check for godoc on exported types
# (manual inspection of changed files)
```

**Requirements**:
- ✅ CHANGELOG.md has entry under `[Unreleased]`
- ✅ Entry in correct category (Added/Fixed/Changed/Removed)
- ✅ All exported functions have godoc comments
- ✅ Godoc comments start with function/type name
- ✅ Package-level godoc present (if new package)

**For contextd**: Check if spec exists in `docs/specs/<feature>/SPEC.md`

### 4. Standards Compliance Check

**Naming**:
- ✅ No stuttering (e.g., `slack.SlackClient` → `slack.Client`)
- ✅ Package names: lowercase, single word
- ✅ Exported: PascalCase
- ✅ Unexported: camelCase

**Error Handling**:
- ✅ All errors wrapped with context using `%w`
- ✅ No ignored error return values
- ✅ No panic in library code

**Context**:
- ✅ Context passed as first parameter
- ✅ Context propagated through call chain
- ✅ No context stored in structs

**Security (contextd-specific)**:
- ✅ Multi-tenant isolation maintained (project boundaries enforced)
- ✅ Input validation present (file paths, URLs, user data sanitized)
- ✅ No credentials in code (use environment variables)
- ✅ Sensitive data redacted in logs

### 5. Verification Evidence Check

**For major tasks** (features, bugs, refactoring, multi-file):
- ✅ Used `contextd:completing-major-task` skill
- ✅ Provided complete template (build, tests, security, functionality, risk)
- ✅ All evidence fields filled with actual output (not "verified: yes")

**For minor tasks** (typos, comments, single-file cosmetic):
- ✅ Used `contextd:completing-minor-task` skill
- ✅ Answered all 3 questions (what changed, how verified, what breaks)

**If evidence missing**: Go back and use appropriate completion skill

### 6. Git Hygiene

```bash
# Check commit messages
git log origin/main..HEAD --oneline

# Check for WIP commits
git log origin/main..HEAD | grep -i "wip\|fixup\|temp"

# Check branch is up-to-date
git fetch origin
git status

# Check for merge conflicts
git diff --check
```

**Requirements**:
- ✅ All commits follow conventional commits format
- ✅ No WIP, fixup, or temp commits (squash if needed)
- ✅ Branch up-to-date with main (rebase if needed)
- ✅ No merge conflicts
- ✅ No trailing whitespace or conflict markers

## Verification Output Template

Provide results in this format:

```markdown
## Pre-PR Verification: [Branch/Feature Name]

### 0. Pre-commit Hooks
**Status**: ✅ PASS / 🚫 FAIL
**Results**:
- Pre-commit version: [version]
- All hooks passed: [yes/no]
- TruffleHog: [pass/fail]
- gosec: [pass/fail - X findings]
- No --no-verify used: [confirmed]

### 1. Build & Test
**Status**: ✅ PASS / 🚫 FAIL
**Results**:
- Build: [success/failed]
- Tests: [X/Y passed, Z failed]
- Race detector: [pass/fail]
- Coverage: [X%]

### 2. Code Quality
**Status**: ✅ PASS / 🚫 FAIL
**Results**:
- gofmt: [no changes needed / applied changes]
- goimports: [no changes needed / applied changes]
- golint: [X warnings]
- go vet: [pass/fail]
- staticcheck: [pass/fail]
- gosec: [X new findings]

### 3. Documentation
**Status**: ✅ PASS / 🚫 FAIL
**Results**:
- CHANGELOG.md: [updated / missing entry]
- Godoc: [complete / incomplete - missing on X functions]
- Spec reviewed: [yes / no / not applicable]

### 4. Standards Compliance
**Status**: ✅ PASS / 🚫 FAIL
**Results**:
- Naming conventions: [compliant / violations listed below]
- Error handling: [compliant / issues listed below]
- Context propagation: [compliant / issues listed below]
- Security (contextd): [compliant / issues listed below]

### 5. Verification Evidence
**Status**: ✅ PASS / 🚫 FAIL
**Results**:
- Completion skill used: [contextd:completing-major-task / minor / none]
- Template complete: [yes / no - missing fields listed below]

### 6. Git Hygiene
**Status**: ✅ PASS / 🚫 FAIL
**Results**:
- Commit format: [conventional / violations listed]
- No WIP commits: [confirmed / found X commits]
- Branch status: [up-to-date / behind main by X commits]
- Conflicts: [none / X conflicts]

---

### Overall Pre-PR Verdict
**READY FOR REVIEW** ✅ / **NOT READY** 🚫

**Issues Found**: [count]

**Required Actions Before PR**:
1. [Action item 1]
2. [Action item 2]
...

**Next Step**:
- ✅ If READY: Invoke `contextd:code-review` skill
- 🚫 If NOT READY: Fix issues above, re-run verification
```

## Quick Verification Script

Save time with automated checks:

```bash
#!/bin/bash
# .scripts/pre-pr-verify.sh

set -e

echo "=== Pre-PR Verification ==="
echo ""

echo "0. Pre-commit Hooks..."
pre-commit run --all-files || { echo "❌ Pre-commit failed"; exit 1; }
echo "✅ Pre-commit passed"
echo ""

echo "1. Build & Test..."
go build ./... || { echo "❌ Build failed"; exit 1; }
go test ./... || { echo "❌ Tests failed"; exit 1; }
go test -race ./... || { echo "❌ Race detector failed"; exit 1; }
COVERAGE=$(go test -coverprofile=coverage.out ./... | tail -1)
echo "Coverage: $COVERAGE"
echo "✅ Build & Test passed"
echo ""

echo "2. Code Quality..."
gofmt -w .
goimports -w .
go vet ./... || { echo "❌ go vet failed"; exit 1; }
staticcheck ./... || { echo "❌ staticcheck failed"; exit 1; }
gosec ./... || { echo "⚠️  gosec found issues"; }
echo "✅ Code Quality passed"
echo ""

echo "3. Documentation..."
git diff main -- CHANGELOG.md | grep -q "^+" || { echo "⚠️  CHANGELOG.md not updated"; }
echo "✅ Documentation check complete"
echo ""

echo "=== Verification Complete ==="
echo "✅ Ready for code review"
```

Make executable: `chmod +x .scripts/pre-pr-verify.sh`

## Common Rationalizations (Do NOT Fall For These)

| Excuse | Reality |
|--------|---------|
| "CI will catch any issues" | Catch locally, save CI cycles and review time. CI is backup, not primary verification. |
| "I already manually tested it" | Manual testing ≠ comprehensive verification. Need build, tests, coverage, security scan. |
| "It's just a typo/small change" | Even small changes need verification. Docs must render, links must work, examples must run. |
| "Too time-consuming to verify" | Verification takes 5 minutes. Fixing broken PR + re-review takes 2 hours. |
| "Senior dev said skip verification" | Standards apply regardless of authority. Verification policy is mandatory. |
| "Pre-commit hooks are slow" | Hooks are mandatory security layer. NEVER use `--no-verify`. |
| "Will fix issues in follow-up PR" | Fix before THIS PR. No merging with known issues. |
| "Tests are flaky, not my code" | Fix flaky tests before PR. No merging with failing tests. |
| "Coverage will improve later" | Meet ≥80% requirement NOW. No deferring quality standards. |

**All of these mean: Run ALL verification checks NOW.**

## Red Flags - STOP and Verify

If you're thinking any of these, STOP and run full verification:

- "Just need to create the PR quickly"
- "CI will validate everything anyway"
- "Already tested manually, good to go"
- "Too small to need full verification"
- "User/lead approved skipping checks"
- "Using --no-verify to save time"
- "Will address issues in next PR"
- "Tests are flaky, merging anyway"
- "Running some checks but not all" (all sections mandatory)
- "Filling template without running commands" (must run actual commands)

**Every one of these is rationalization. Run verification.**

## Anti-Patterns

### ❌ Creating PR Without Verification

```
Bad: User finished feature, immediately creates PR
Missing: All verification checks
Result: PR has failing tests, missing docs, security issues
```

### ❌ Skipping Pre-commit Hooks

```
Bad: git commit --no-verify -m "Quick fix"
Missing: Secret detection, security scan, format checks
Result: Secrets committed, security vulnerabilities merged
```

### ❌ Deferring Issues to Later

```
Bad: "Tests are failing but will fix in follow-up PR"
Missing: Fix before THIS PR
Result: Main branch broken, team blocked
```

### ❌ Trusting Manual Testing

```
Bad: "I tested it manually, works fine, creating PR"
Missing: Automated tests, coverage, race detector
Result: Untested edge cases, race conditions in production
```

### ❌ Template Without Actual Verification

```
Bad: Fills output template with "✅ PASS" without running commands
Missing: Actual command execution and output pasting
Result: False confidence, issues slip through to PR
```

## Integration with Other Skills

**Workflow sequence**:
1. Complete implementation work
2. Use `contextd:completing-major-task` or `contextd:completing-minor-task`
3. **Use this skill** (`contextd:pre-pr-verification`) ← YOU ARE HERE
4. If READY: Use `contextd:code-review`
5. If APPROVED: Create pull request
6. If CHANGES REQUIRED: Fix and repeat from step 3

**This skill is the gate before code review.** Do not proceed to code review without READY verdict.

## Success Criteria

**Verification is complete when**:
- ✅ ALL 7 sections show PASS status
- ✅ Output template filled with actual results
- ✅ Overall verdict is READY FOR REVIEW
- ✅ No required actions listed

**Verification is incomplete when**:
- 🚫 Any section shows FAIL status
- 🚫 Output is summary without actual command results
- 🚫 Required actions list is not empty
- 🚫 Any check was skipped

**Remember**: 5 minutes of verification saves 2 hours of debugging broken PRs.
