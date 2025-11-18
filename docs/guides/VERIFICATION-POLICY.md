# Verification & Completion Policy

**Status**: MANDATORY | **Last Updated**: 2025-11-18

## Why This Exists

Agents complete tasks without proving their claims. They announce "Done! Updated docs!" without verifying the docs build, render correctly, or contain accurate information. This creates:

- **Broken implementations** marked complete
- **Context waste** re-doing claimed work
- **Demo failures** from unverified features
- **Lost trust** in agent completion claims

**This policy enforces evidence-based completion**: No task can be marked complete without verification proof.

---

## Task Classification

All tasks fall into two categories that determine verification requirements:

### Major Tasks

**Definition**: Tasks that affect functionality, create files, or impact multiple areas.

**Examples**:
- Features, bug fixes, refactoring
- Security changes, performance improvements
- Multi-file changes, new file creation
- Documentation for complex features
- Anything affecting APIs, multi-tenancy, or core functionality

**Verification Requirement**: Comprehensive template via `contextd:completing-major-task` skill

### Minor Tasks

**Definition**: Tasks that make cosmetic or internal-only changes with no functional impact.

**Examples**:
- Typos, comment fixes, formatting
- Single-file cosmetic edits
- Variable renames (internal only)
- Whitespace cleanup

**Verification Requirement**: Self-interrogation checklist via `contextd:completing-minor-task` skill

### When in Doubt

**If unsure whether task is major or minor, treat it as major.** False positive (extra verification) is better than false negative (unverified completion).

---

## Verification Templates

### Major Task Template

**When completing a major task, you MUST provide this evidence:**

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

**Field Explanations**:

- **Task**: Specific description (not vague like "updated code")
- **Type**: Category helps reviewers understand impact
- **Changes**: File-by-file list (e.g., "pkg/auth/jwt.go - added token validation")
- **Build**: Must show command AND output (e.g., "`go build ./...` - Success, no errors")
- **Tests**: Must show passing tests AND coverage (e.g., "47/47 passed, 87% coverage")
- **Security**: Explicit security validation (especially for multi-tenant isolation)
- **Functionality**: Proof feature works as intended (manual test results, example output)
- **Risk Assessment**: Honest evaluation of what could break if verification missed something

**Security Field Requirements** (CRITICAL for contextd):
- **Always check**: Multi-tenant isolation maintained (no cross-project/team data leakage)
- **Always check**: Input validation present (file paths, URLs, user data)
- **Always check**: No new gosec findings (`gosec ./...`)

### Minor Task Template

**When completing a minor task, you MUST answer these questions:**

```markdown
Task: [clear description]
✓ What changed: [specific change made]
✓ How I know it works: [verification performed]
✓ What breaks if wrong: [honest risk assessment]
```

**Field Explanations**:

- **What changed**: Be specific ("Fixed typo 'Installtion' → 'Installation' on line 47")
- **How I know it works**: Show evidence ("Ran `mdl README.md`, no errors; previewed, renders correctly")
- **What breaks if wrong**: Honest assessment ("Typo remains visible, cosmetic only, no functional impact")

---

## Anti-Patterns & Examples

### Major Task Examples

#### ❌ WRONG: No Verification

```
Done! Implemented JWT authentication.
```

**Violations**:
- No template
- No build verification
- No test results
- No security validation
- No risk assessment

#### ✅ RIGHT: Complete Verification

```
Task: Implement JWT authentication for MCP endpoints
Type: Feature
Changes:
  - pkg/mcp/auth.go (new file, JWT validation middleware)
  - pkg/mcp/server.go (added auth middleware to routes)
  - pkg/mcp/auth_test.go (comprehensive tests, 15 test cases)
Verification Evidence:
  ✓ Build: `go build ./...` - Success, no errors
  ✓ Tests: `go test ./pkg/mcp/...` - 47/47 passed, coverage 87%
  ✓ Security:
    - Input validation: JWT signature verification required
    - Multi-tenant isolation: User ID extracted from token, scoped to project
    - gosec: No new findings
  ✓ Functionality:
    - Manual test: Valid JWT → 200 OK, correct user context
    - Manual test: Invalid JWT → 401 Unauthorized
    - Manual test: Expired JWT → 401 Unauthorized
    - Manual test: Malformed JWT → 400 Bad Request
Risk Assessment: If verification insufficient, unauthenticated users could access protected endpoints. Tests cover signature validation, expiration, malformed tokens, and multi-tenant isolation.
```

### Minor Task Examples

#### ❌ WRONG: No Verification

```
Updated the README with installation instructions.
```

**Violations**:
- No checklist
- No proof docs render
- No evidence examples work

#### ✅ RIGHT: Self-Interrogation Checklist

```
Task: Fix typo in README.md installation section
✓ What changed: Changed "Installtion" to "Installation" on line 47
✓ How I know it works: Ran `mdl README.md` (no errors), previewed in VS Code (renders correctly), spell-check passes
✓ What breaks if wrong: Typo remains visible in rendered docs (cosmetic only, no functional impact)
```

---

## Enforcement & Review

### During Work

**When you write completion words** ("done", "complete", "updated", "fixed", "ready"):
1. **PAUSE** - Do not continue writing
2. **Classify** - Is this task major or minor?
3. **Invoke** - Use appropriate completion skill:
   - Major: `contextd:completing-major-task`
   - Minor: `contextd:completing-minor-task`
4. **Provide** - Follow skill's template with complete evidence

### Before PR Creation

**Before creating pull request**:
- **MUST** invoke: `contextd:code-review` skill
- Code reviewer validates ALL verification evidence
- Missing or insufficient verification → CHANGES REQUIRED

### Code Review Validation

Code reviewer checks:
1. **Template presence** - Required template used
2. **Evidence quality** - Fields not empty or hand-wavy ("Verified: yes" fails)
3. **Consistency** - Evidence matches claimed changes
4. **Task-appropriateness** - Security changes show security verification, etc.

**Review Verdicts**:
- **APPROVED** - Verification complete and sufficient
- **CHANGES REQUIRED** - Verification missing or insufficient
- **BLOCKED** - Critical verification failures (security, multi-tenancy)

---

## Edge Cases

### Escalation from Minor to Major

Escalate to major template if minor task reveals:
- Unexpected functional impact
- Cross-file changes needed
- Security implications
- Multi-tenant isolation concerns

**Example**: Fixing typo in error message reveals error message contains sensitive data → escalate to major, add security verification.

### Multiple Related Tasks

When completing multiple related tasks in one session:
- Each major task gets separate verification template
- Minor tasks can be grouped if related (but still need checklist)

**Example**: "Fixed 5 typos in README" → one minor template listing all 5 changes.

### Partial Completion

**Never mark task complete if partially done.** If blocked:
1. Keep task status as `in_progress`
2. Create new task for blocker
3. Provide status update (what's done, what's blocked, next steps)

---

## Common Rationalizations (Don't Fall For These)

### "This change is straightforward, no need to verify"

**Wrong.** Straightforward changes still need evidence. The simpler the change, the faster verification should be. Use minor template for quick verification.

### "I can see from the code it works"

**Wrong.** Code inspection is not verification. Run the code, show output.

### "The tests pass" (without showing test output)

**Wrong.** Show the output. Paste the test results. Prove they passed.

### "This is just docs, verification doesn't apply"

**Wrong.** Docs need verification: Do they render? Do examples run? Are links valid?

### "I already verified it in my head"

**Wrong.** Verification means evidence. If it's in your head, it didn't happen.

---

## Summary

**Completion Rule**: No completion claim without verification evidence.

**Major Tasks**:
- Features, bugs, refactoring, security, multi-file → `contextd:completing-major-task`
- Comprehensive template: build + tests + security + functionality + risk

**Minor Tasks**:
- Typos, comments, formatting, single-file cosmetic → `contextd:completing-minor-task`
- Self-interrogation: what changed, how verified, what breaks if wrong

**Enforcement**: Code review validates all evidence. Missing verification → CHANGES REQUIRED.

**Remember**: Token cost of comprehensive verification is FAR less than token cost of debugging "completed" work that doesn't actually work.
