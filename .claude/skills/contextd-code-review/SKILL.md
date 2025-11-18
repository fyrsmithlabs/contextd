---
name: contextd-code-review
description: Use when completing work and before creating PR, to perform comprehensive code review - validates verification evidence, security compliance, test coverage, and documentation using structured 6-section checklist with explicit APPROVED/CHANGES REQUIRED/BLOCKED verdict
---

# Code Review Skill

## Overview

**This skill performs systematic code review before PR creation using a mandatory 6-section checklist.**

**Core principle**: No approval without verification evidence. Code review validates that work meets all project standards using structured output.

**REQUIRED BACKGROUND**: Read docs/guides/CODE-REVIEW-CHECKLIST.md and docs/guides/VERIFICATION-POLICY.md before using this skill.

---

## When to Use This Skill

**Use when:**
- Before creating pull request (MANDATORY)
- After completing major task (recommended)
- Developer requests code review
- Validating verification evidence

**DO NOT use for:**
- Initial planning or design review
- Mid-implementation feedback
- Minor typo fixes (use contextd:completing-minor-task instead)

---

## Mandatory Review Checklist

**You MUST complete ALL 6 sections. No exceptions.**

### 1. Verification Evidence Validation

**For major tasks, demand completion used THIS template**:
```
Task: [clear description]
Type: [Feature/Bug Fix/Refactor/Security/Docs/Performance]
Changes: [file-by-file breakdown]
Verification Evidence:
  ✓ Build: [command + output]
  ✓ Tests: [command + output + coverage %]
  ✓ Security: [multi-tenant isolation + input validation + gosec]
  ✓ Functionality: [manual test results]
Risk Assessment: [what breaks if verification insufficient]
```

**For minor tasks, demand THIS template**:
```
Task: [clear description]
✓ What changed: [specific change]
✓ How I know it works: [verification performed]
✓ What breaks if wrong: [risk assessment]
```

**Validation criteria**:
- [ ] Template structure complete (all fields present)
- [ ] Build evidence shows command AND output (not "Build: Success")
- [ ] Test evidence shows results AND coverage % (not "Tests: Passed")
- [ ] Coverage meets ≥80% requirement
- [ ] Security validation includes multi-tenant isolation check
- [ ] Security validation confirms no new gosec findings
- [ ] Functionality verification shows actual test results
- [ ] Risk assessment is honest and specific

**If verification missing or insufficient**: Status = CHANGES REQUIRED or BLOCKED

### 2. Security Review (CRITICAL)

**Check multi-tenant isolation**:
- [ ] Data scoped to `project_<hash>` (checkpoints) or `team_<name>` (shared)
- [ ] No cross-project/team queries without explicit permission
- [ ] Database-per-project isolation maintained

**Check input validation**:
- [ ] File paths sanitized (no path traversal)
- [ ] Git URLs validated (no command injection)
- [ ] Search queries sanitized
- [ ] Team/org names validated

**Check sensitive data**:
- [ ] No credentials in code
- [ ] Secrets from environment or secure storage
- [ ] Sensitive data redacted in logs

**Check security tooling**:
- [ ] `gosec ./...` - no new findings
- [ ] Pre-commit hooks passed
- [ ] No `--no-verify` bypass

**If security issues found**: Status = BLOCKED (not negotiable)

### 3. Code Standards (Go-Specific)

**Naming**:
- [ ] Package names lowercase, single word
- [ ] Exported identifiers PascalCase
- [ ] No stuttering (e.g., `http.Server` not `http.HTTPServer`)

**Error handling**:
- [ ] Errors wrapped with context: `fmt.Errorf("...: %w", err)`
- [ ] No ignored errors
- [ ] No panic in library code

**Code quality**:
- [ ] `gofmt` compliant
- [ ] `go vet` passes
- [ ] `golangci-lint` passes
- [ ] No commented-out code
- [ ] No TODO without issue numbers

### 4. Test Coverage & TDD

**Coverage**:
- [ ] Overall ≥80%
- [ ] New code ≥80%
- [ ] Critical paths 100% (auth, security, multi-tenant)

**Test quality**:
- [ ] Table-driven patterns used
- [ ] Clear names (TestFunction_Scenario_Expected)
- [ ] No testing mock behavior
- [ ] No test-only methods in production code
- [ ] Tests deterministic (no race conditions)

**Verify with**:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### 5. Documentation

**Code documentation**:
- [ ] Exported functions have godoc
- [ ] Godoc starts with function name
- [ ] Package has package-level godoc
- [ ] Complex logic has "why" comments

**CHANGELOG.md** (MANDATORY):
- [ ] Entry added under `[Unreleased]`
- [ ] Correct category (Added/Fixed/Changed/Removed)
- [ ] Clear and user-focused
- [ ] **BREAKING** marker if breaking change

**Specs**:
- [ ] Relevant specs updated in `docs/specs/<feature>/`
- [ ] ADRs updated if architectural change
- [ ] Package CLAUDE.md updated if package changes

### 6. Architecture Compliance

**ADR compliance**:
- [ ] Follows existing ADRs in `docs/architecture/adr/`
- [ ] New ADR created if contradicts existing

**YAGNI compliance**:
- [ ] No speculative features
- [ ] No unused code
- [ ] No premature optimization
- [ ] Every feature solves current problem

---

## Structured Output Template (MANDATORY)

**You MUST use this template. Copy it exactly**:

```markdown
## Code Review: [Task/PR Name]

**Reviewer**: [Your name/role]
**Date**: [YYYY-MM-DD]
**Commit**: [commit hash if available]

---

### 1. Verification Evidence
**Status**: ✅ PASS / ⚠️ CHANGES REQUIRED / 🚫 BLOCKED

**Findings**:
- [List findings or "No issues found"]

**Evidence Quality**: [Complete/Incomplete/Missing]

---

### 2. Security Review
**Status**: ✅ PASS / ⚠️ CHANGES REQUIRED / 🚫 BLOCKED

**Multi-Tenant Isolation**: [Pass/Fail - details]
**Input Validation**: [Pass/Fail - details]
**Sensitive Data**: [Pass/Fail - details]
**gosec Results**: [No new findings / Findings listed]

**Findings**:
- [List security issues or "No issues found"]

---

### 3. Code Standards
**Status**: ✅ PASS / ⚠️ CHANGES REQUIRED / 🚫 BLOCKED

**Findings**:
- [List violations or "No issues found"]

---

### 4. Test Coverage
**Status**: ✅ PASS / ⚠️ CHANGES REQUIRED / 🚫 BLOCKED

**Coverage**: [X%] (Requirement: ≥80%)
**TDD Compliance**: [Yes/No - evidence]

**Findings**:
- [List coverage issues or "No issues found"]

---

### 5. Documentation
**Status**: ✅ PASS / ⚠️ CHANGES REQUIRED / 🚫 BLOCKED

**CHANGELOG Updated**: [Yes/No]
**Godoc Complete**: [Yes/No]
**Specs Updated**: [Yes/No/N/A]

**Findings**:
- [List documentation issues or "No issues found"]

---

### 6. Architecture Compliance
**Status**: ✅ PASS / ⚠️ CHANGES REQUIRED / 🚫 BLOCKED

**ADR Compliance**: [Pass/Fail - details]
**YAGNI Compliance**: [Pass/Fail - details]

**Findings**:
- [List architecture issues or "No issues found"]

---

### Overall Verdict
**APPROVED** ✅ / **CHANGES REQUIRED** ⚠️ / **BLOCKED** 🚫

**Summary**: [Brief summary of review outcome]

**Required Actions** (if not APPROVED):
1. [Specific action item]
2. [Specific action item]
...

**Approval Conditions** (if CHANGES REQUIRED):
- [ ] [Condition for approval]
- [ ] [Condition for approval]
```

---

## Verdict Decision Tree

```
Has verification evidence?
├─ No → CHANGES REQUIRED (demand evidence first)
└─ Yes → Check security
    ├─ Security issues?
    │  ├─ Critical (data leakage, multi-tenant violation) → BLOCKED
    │  └─ Non-critical → CHANGES REQUIRED
    └─ No security issues → Check other sections
        ├─ Test coverage <80%? → CHANGES REQUIRED
        ├─ CHANGELOG missing? → CHANGES REQUIRED
        ├─ Code standard violations? → CHANGES REQUIRED
        ├─ YAGNI violations? → CHANGES REQUIRED
        └─ All pass → APPROVED
```

---

## Anti-Patterns (What NOT to Do)

### ❌ Anti-Pattern 1: Narrative Feedback Without Checklist

**Bad**:
```
This looks great! The code is well-structured and the tests are comprehensive.
I like how you handled the error cases. Ready to merge!
```

**Why bad**: No verification of evidence, no systematic checklist, no structured output

**Fix**: Use mandatory template, complete all 6 sections

### ❌ Anti-Pattern 2: Accepting Claims Without Evidence

**Bad**:
```
Developer: "I tested it thoroughly"
Reviewer: "Since you tested it, looks good!"
```

**Why bad**: No evidence validation, defers to authority

**Fix**: Demand verification evidence template, validate actual output

### ❌ Anti-Pattern 3: Generic Approval

**Bad**:
```
✅ Verification: Looks good
✅ Security: No issues
✅ Tests: Pass
Overall: APPROVED
```

**Why bad**: No specifics, didn't actually check (shows command output, coverage %, etc.)

**Fix**: Validate actual evidence (command output, coverage percentage, gosec results)

### ❌ Anti-Pattern 4: Skipping Sections

**Bad**:
```
Checked code quality and tests. Everything looks good. APPROVED.
```

**Why bad**: Missing security review, documentation check, verification evidence

**Fix**: Complete ALL 6 sections, no skipping

---

## Common Rationalizations (Don't Fall For These)

| Excuse | Reality |
|--------|---------|
| "Code looks well-tested" | Show me coverage %. Demand evidence. |
| "Developer is senior" | Authority doesn't bypass verification. Demand evidence. |
| "Time pressure to merge" | Broken code merged costs more time. Demand evidence. |
| "Tests exist" | Passing tests ≠ adequate coverage. Check %. |
| "Security looks fine" | Multi-tenant isolation must be explicitly validated. |
| "Just a small change" | Small changes break systems. Full checklist required. |
| "I trust the developer" | Trust + Verify. Demand evidence template. |
| "CHANGELOG can be updated later" | No. MANDATORY before merge. |

**All of these mean: Complete the full checklist. Demand verification evidence. Use structured output.**

---

## Red Flags - STOP and Demand Evidence

**If you're about to say any of these, STOP**:
- "Looks good!"
- "Seems ready to merge"
- "I trust your testing"
- "Code quality is high"
- "Since you already tested..."

**Instead, say**:
- "Please provide verification evidence using the major task template"
- "I need to see test coverage percentage and output"
- "Show me gosec results and multi-tenant isolation validation"
- "CHANGELOG.md must be updated before approval"

---

## Integration with Other Skills

**This skill requires**:
- Developer MUST have used `contextd:completing-major-task` or `contextd:completing-minor-task`
- Verification evidence MUST be present before review

**This skill validates**:
- Verification templates are complete
- Standards are followed
- Security is maintained
- Documentation is updated

**After APPROVED**:
- Developer can create PR
- GitHub Actions runs automated checks (secondary validation)

---

## Quick Reference

**Review workflow**:
1. Demand verification evidence first (major or minor template)
2. Complete ALL 6 checklist sections
3. Provide structured output (copy template)
4. Give explicit verdict (APPROVED/CHANGES REQUIRED/BLOCKED)
5. If not approved: List specific actions required

**Blocking verdicts**:
- Missing verification evidence → CHANGES REQUIRED
- Security violations (data leakage, multi-tenant) → BLOCKED
- Coverage <80% → CHANGES REQUIRED
- CHANGELOG missing → CHANGES REQUIRED
- No gosec clean scan → BLOCKED

**Remember**: You are the last line of defense before merge. Be thorough, systematic, and demand evidence.
