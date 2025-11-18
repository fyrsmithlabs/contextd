---
name: completing-major-task
description: Use when completing features, bug fixes, refactoring, security changes, multi-file changes, or any task that affects functionality - enforces comprehensive verification template with build output, test coverage, security validation, and risk assessment before marking work complete
---

# Completing Major Task

## Overview

**No task can be marked complete without verification evidence.**

This skill enforces the comprehensive verification template for major tasks. Major tasks are: features, bug fixes, refactoring, security changes, performance improvements, multi-file changes, new file creation, documentation for complex features, or anything affecting APIs, multi-tenancy, or core functionality.

## When to Use This Skill

**Invoke this skill when you are about to use completion words:**

Completion words: "done", "complete", "updated", "fixed", "ready", "finished", "implemented", "successful"

**Before saying ANY completion word:**
1. PAUSE - Do not continue writing
2. Invoke this skill
3. Follow the template with complete evidence

**If unsure whether task is major or minor**: Treat as major. False positive (extra verification) is better than false negative (unverified completion).

## Major vs Minor Task Classification

### Major Tasks (Use THIS Skill)
- Features, bug fixes, refactoring
- Security changes, performance improvements
- Multi-file changes, new file creation
- Documentation for complex features
- Anything affecting APIs, multi-tenancy, or core functionality

### Minor Tasks (Use completing-minor-task skill instead)
- Typos, comment fixes, formatting
- Single-file cosmetic edits
- Variable renames (internal only)
- Whitespace cleanup

## The Verification Template (MANDATORY)

**You MUST provide ALL fields with actual data (not summaries):**

```markdown
Task: [clear description of what was done]
Type: [Feature/Bug Fix/Refactor/Security/Docs/Performance]
Changes: [specific files modified, what changed in each]
Verification Evidence:
  ✓ Build: [command run + output showing success]
  ✓ Tests: [command run + output + coverage percentage]
  ✓ Security: [no new vulnerabilities, multi-tenant isolation maintained]
  ✓ Functionality: [manual test or specific behavior verified]
Risk Assessment: [what breaks if verification was insufficient]
```

## Field Requirements (Critical)

### Task
- Specific description (not vague like "updated code")
- Example: "Implement JWT authentication for MCP endpoints"
- Not: "Added auth"

### Type
- Choose one: Feature/Bug Fix/Refactor/Security/Docs/Performance
- Helps reviewers understand impact

### Changes
- File-by-file list with what changed
- Example: "pkg/auth/jwt.go (new file, added token validation, 125 lines)"
- Not: "Updated auth files"

### Build Evidence
- MUST show command AND output
- Example: "`go build ./...` - Success, no errors [PASTE OUTPUT]"
- Not: "Build: Success"

### Test Evidence
- MUST show command, output, AND coverage percentage
- Example: "`go test -coverprofile=coverage.out ./...` - 47/47 passed, coverage 87%"
- Not: "Tests: All passed"
- MUST verify ≥80% coverage requirement met

### Security Validation (CRITICAL for contextd)
- MUST check multi-tenant isolation (no cross-project/team data leakage)
- MUST check input validation (file paths, URLs, user data)
- MUST run `gosec ./...` and confirm no new findings
- Example:
  ```
  - Multi-tenant isolation: User ID extracted from token, scoped to project
  - Input validation: JWT signature verified, expiration checked, malformed rejected
  - gosec: No new findings
  ```
- Not: "Security: Looks good"

### Functionality Verification
- MUST show manual test results or specific behavior verified
- Proof feature works as intended (not claims)
- Example:
  ```
  - Manual test: Valid JWT → 200 OK, correct user context
  - Manual test: Invalid JWT → 401 Unauthorized
  - Manual test: Expired JWT → 401 Unauthorized
  ```
- Not: "Functionality: Working correctly"

### Risk Assessment
- Honest evaluation of what could break
- Consider consequences if verification missed something
- Example: "If verification insufficient, unauthenticated users could access protected endpoints, violating multi-tenant isolation. Tests cover signature validation, expiration, malformed tokens."
- Not: "Risk: Low"

## Evidence Quality Standards

### What "Sufficient Evidence" Means

**Required for each field:**
- Not empty or placeholder text
- Shows actual command output (not summarized)
- Output is complete (not truncated without reason)
- Output matches claimed changes (consistency check)

**Common failures (all BLOCK completion):**
- ❌ "Build: Success" (no command or output)
- ❌ "Tests: All passed" (no count, no coverage percentage)
- ❌ "Security: Looks good" (no specific checks)
- ❌ "Verified: yes" (no actual verification shown)
- ❌ "Functionality: Working" (no manual test results)
- ❌ "Risk: None" (avoiding honest assessment)

## Anti-Patterns (NEVER Do These)

### ❌ WRONG: No Verification

```
Done! Implemented JWT authentication.
```

**Violations:**
- No template
- No build verification
- No test results
- No security validation
- No risk assessment

### ❌ WRONG: Vague Evidence

```
Task: Add authentication
Type: Feature
Changes: Updated auth files
Verification Evidence:
  ✓ Build: Success
  ✓ Tests: Passed
  ✓ Security: Good
  ✓ Functionality: Working
Risk Assessment: Low risk
```

**Violations:**
- No actual command output
- No coverage percentage
- No specific security checks
- No manual test results
- Generic risk assessment

### ✅ RIGHT: Complete Verification

```
Task: Implement JWT authentication for MCP endpoints
Type: Feature
Changes:
  - pkg/mcp/auth.go (new file, JWT validation middleware, 125 lines)
  - pkg/mcp/server.go (added auth middleware to routes, 8 lines)
  - pkg/mcp/auth_test.go (comprehensive tests, 15 test cases, 247 lines)
Verification Evidence:
  ✓ Build: `go build ./...` - Success, no errors
    [ACTUAL OUTPUT PASTED]
  ✓ Tests: `go test -coverprofile=coverage.out ./pkg/mcp/` - 15/15 passed, coverage 87%
    [ACTUAL OUTPUT PASTED]
    [COVERAGE REPORT PASTED]
  ✓ Security:
    - Multi-tenant isolation: User ID extracted from token, scoped to project
    - Input validation: JWT signature verified, expiration checked, malformed rejected
    - gosec: `gosec ./pkg/mcp/` - No new findings
  ✓ Functionality:
    - Manual test: Valid JWT → 200 OK, correct user context in logs
    - Manual test: Invalid JWT → 401 Unauthorized
    - Manual test: Expired JWT → 401 Unauthorized
    - Manual test: Malformed JWT → 400 Bad Request
Risk Assessment: JWT authentication is CRITICAL security component. If verification insufficient, unauthenticated users could access protected MCP endpoints, violating multi-tenant isolation. Tests cover signature validation, expiration, malformed tokens, and user context extraction. Manual tests confirm behavior matches requirements.
```

## Common Rationalizations (STOP These)

| Rationalization | Reality |
|----------------|---------|
| "This change is straightforward, no need to verify" | Straightforward changes still need evidence. Use template. |
| "I can see from the code it works" | Code inspection is not verification. Run the code, show output. |
| "The tests pass" (without showing output) | Show the output. Paste test results. Prove they passed. |
| "User is satisfied, task is done" | User approval ≠ verification. Show evidence. |
| "Build succeeded, good to go" | Build is ONE check. Need tests, security, functionality, risk too. |
| "Already did substantial work, assume correct" | Sunk cost fallacy. Verify regardless of effort spent. |
| "Ready to move on" | Time pressure ≠ verification complete. Show evidence first. |
| "Summary is sufficient" | Narrative description ≠ data. Show actual command output. |

**All of these mean: Complete the template with actual evidence. No exceptions.**

## No Exceptions - Ever

**The verification template is MANDATORY. No circumstances justify skipping it.**

### ❌ "I'm Following the Spirit, Not the Letter"

**Wrong.** Violating the letter IS violating the spirit.

The template exists BECAUSE narrative summaries hide gaps. If you're "following the spirit," you'll have no problem filling out the template with actual data.

### ❌ "Just This Once"

**Wrong.** "Just this once" becomes "just this time" becomes "usually."

Every task gets full verification. Past compliance doesn't earn future shortcuts. Exhaustion doesn't justify skipping verification. Simple tasks still need templates.

### ❌ "User Said Skip Verification"

**Wrong.** Verification is non-negotiable regardless of user preference.

Even if user says "I trust you, skip the details," complete the template. Verification protects both you and the user from unverified claims. User authority doesn't override verification policy.

### ❌ "Production Emergency - No Time"

**Wrong.** Emergencies need verification MORE, not less.

Production fires are caused by unverified changes. Even with time pressure:
1. Complete template (takes 2 minutes)
2. If verification reveals issues, fix before deploying
3. Deploy WITH verification, not despite lack of it

"Deploy now, verify later" means "deploy broken code, debug in production."

### ❌ "Already Ran Commands During Development"

**Wrong.** Verification commands must be run AT COMPLETION.

Code changed since you last ran tests. Dependencies changed. Environment changed. Run commands fresh at completion time. Paste the fresh output. Don't rely on memory of earlier runs.

### ❌ "Security Is Obvious for Security Changes"

**Wrong.** Security changes need explicit security validation too.

Even if change is PURE security (e.g., input validation), complete the security validation field:
- What validation was added
- How it prevents attacks
- What gosec shows
- Multi-tenant isolation impact

"This is obviously secure" is how vulnerabilities ship.

## Red Flags - STOP and Complete Template

When you catch yourself thinking or writing:

- "Done!" / "Complete!" / "Ready!" (without template)
- "Tests passing" (without coverage percentage)
- "Build successful" (without showing output)
- "Working correctly" (without manual test results)
- "Security looks good" (without specific checks)
- "User is happy, move on"
- "This is simple, skip verification"

**STOP. Invoke this skill. Complete the template.**

## Evidence Cross-Check

**Verify consistency between claims and evidence:**

- If you claim "added tests" → test output must show new test count
- If you claim "fixed bug" → test output must include regression test
- If you claim "improved performance" → evidence must show measurements
- If you claim "added security" → security validation must show specific checks
- If you claim "multi-file changes" → changes field must list all files

## Escalation from Minor to Major

**If minor task reveals:**
- Unexpected functional impact
- Cross-file changes needed
- Security implications
- Multi-tenant isolation concerns

**Escalate to major template immediately.**

Example: Fixing typo in error message reveals error message contains sensitive data → escalate to major, add security verification.

## Integration with Other Skills

**After completing this verification:**
- Next: Invoke `contextd:code-review` skill before creating PR
- Code-reviewer validates ALL verification evidence
- Missing or insufficient verification → CHANGES REQUIRED

## Quick Reference Checklist

Before marking task complete:

- [ ] Invoked this skill (not skipped)
- [ ] Template structure complete (all fields present)
- [ ] Build evidence shows command + output
- [ ] Test evidence shows results + coverage %
- [ ] Coverage ≥80%
- [ ] Security validation includes multi-tenant isolation
- [ ] Security validation confirms no new gosec findings
- [ ] Functionality shows actual manual test results
- [ ] Risk assessment is honest and specific
- [ ] No hand-wavy fields ("yes", "good", "success")
- [ ] Output pasted (not summarized)

## The Bottom Line

**Token cost of comprehensive verification is FAR less than token cost of debugging "completed" work that doesn't actually work.**

Verification is not optional. Verification is not negotiable. Verification prevents:
- Broken implementations marked complete
- Context waste re-doing claimed work
- Demo failures from unverified features
- Lost trust in completion claims

Complete the template. Show the evidence. Every time.
