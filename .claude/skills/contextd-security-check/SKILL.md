---
name: contextd-security-check
description: Use when working on auth, session, isolation, RBAC, multi-tenant boundaries, sensitive data handling, input validation, or database queries - enforces comprehensive security validation with explicit evidence before marking security-critical work complete, blocks completion if multi-tenant isolation, input validation, or security testing requirements not met
---

# Security Check

## Overview

**Comprehensive security validation for security-critical code changes.**

This skill enforces defense-in-depth security validation for contextd. All changes affecting authentication, multi-tenant isolation, sensitive data, or input validation MUST pass all 5 security checks before completion.

**Core principle:** Security is non-negotiable. No exceptions, no deferral, no shortcuts.

## When to Use This Skill

**MANDATORY for changes to:**
- Multi-tenant boundaries (project/team/org scoping)
- Database queries or filters (pkg/vectorstore, pkg/adapter)
- Input validation or sanitization (any user input)
- Sensitive data (API keys, embedding service credentials, PII)
- MCP tools accessing protected resources
- HTTP handlers and middleware

**POST-MVP authentication/authorization (pkg/auth, pkg/session, pkg/rbac):**
- When authentication is added, apply all security checks including timing attack tests

**When NOT to use:**
- Pure documentation changes (markdown files ONLY, no .go files)
- Test-only changes (no production code modified)
- Build/CI configuration (no runtime code)

**If change affects ANY production code that processes user input or handles data, use this skill.** If unsure, use this skill. False positive (extra security check) is better than false negative (missed vulnerability).

## The 5 Security Checks (MANDATORY)

### 1. Multi-Tenant Isolation

**CRITICAL for contextd:** Data MUST NOT leak across project/team/org boundaries.

```
☐ Checkpoints use project-specific database ONLY (project_<hash>)
☐ Shared data (remediations/skills) use correct scope (team_<name> or org_<name>)
☐ No cross-project queries possible
☐ Database name derived from project_path hash (SHA256[:8])
☐ NEVER trust user-provided database names
☐ Test: Attempt cross-project query → MUST fail
☐ Test: Attempt cross-team query without permission → MUST fail
```

**Evidence Required:**
- Show database selection code
- Show test proving cross-boundary queries fail
- Confirm no user input controls database name

**Common Failures:**
```go
// ❌ WRONG: User input controls database
teamDB := fmt.Sprintf("team_%s", teamName)  // teamName from user input!

// ✅ RIGHT: Validated and hashed
validatedTeam, err := validateTeamName(teamName)
if err != nil {
    return err
}
teamHash := hashTeamName(validatedTeam)
teamDB := fmt.Sprintf("team_%s", teamHash)
```

### 2. Input Validation

**ALL user inputs MUST be validated at EVERY boundary.** No exceptions.

```
☐ File paths sanitized (no ../../../etc/passwd)
☐ Git URLs validated (no command injection)
☐ Search queries sanitized (no injection)
☐ Filter expressions validated (no injection)
☐ Team/org names validated (no special chars, alphanumeric + hyphen only)
☐ Validation at EVERY entry point (not just service layer)
☐ Test: Malicious input → rejected with clear error
```

**Defense-in-Depth:** Validate at handler, service, AND repository layers.

**Evidence Required:**
- Show validation code at each layer
- Show test with malicious input (path traversal, injection, special chars)
- Confirm validation NOT skipped anywhere

**Common Failures:**
```go
// ❌ WRONG: "Validated elsewhere"
func (r *Repository) Get(ctx context.Context, path string) {
    // No validation - "service layer already validated"
    return r.db.Load(path)
}

// ✅ RIGHT: Validate at EVERY boundary
func (r *Repository) Get(ctx context.Context, path string) {
    if err := validatePath(path); err != nil {  // Validate again!
        return fmt.Errorf("invalid path: %w", err)
    }
    return r.db.Load(path)
}
```

### 3. Sensitive Data Handling

**Credentials, API keys, secrets MUST be protected.** No logging, no leaking.

```
☐ No credentials in code (use environment variables)
☐ No credentials in logs (use [REDACTED] for sensitive values)
☐ File permissions 0600 for credential files (embedding API keys, etc.)
☐ No secrets in error messages
☐ No secrets in OpenTelemetry trace spans
☐ Test: Verify credentials not in logs/errors
```

**POST-MVP (when authentication added):**
```
☐ Constant-time comparison for tokens (crypto/subtle)
☐ Timing attack tests for authentication
```

**Evidence Required:**
- Show credential loading (environment variables, not hardcoded)
- Show logging code with redaction
- Show file permission check (if credential files)
- POST-MVP: Show constant-time comparison (when auth added)

**Common Failures:**
```go
// ❌ WRONG: API key in logs
log.Printf("Using API key: %s", apiKey)

// ✅ RIGHT: Redacted
log.Printf("Using API key: [REDACTED]")
```

**POST-MVP authentication failures:**
```go
// ❌ WRONG: Timing attack vulnerable (when auth added)
if provided == expected {
    return true
}

// ✅ RIGHT: Constant-time (when auth added)
return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
```

### 4. Security Testing

**Security claims MUST be proven with tests.** No "looks secure" assertions.

```
☐ gosec ./... passes with NO new findings
☐ Multi-tenant isolation test exists and passes
☐ Input validation test with malicious inputs exists and passes
☐ All security tests in CI/CD pipeline
```

**POST-MVP (when authentication/authorization added):**
```
☐ Timing attack test (when auth added)
☐ Privilege escalation test (when RBAC added)
```

**Evidence Required:**
- Paste gosec output showing no new findings
- Show multi-tenant isolation test code and output
- Show input validation test with malicious inputs
- Confirm all tests in CI/CD

**Common Failures:**
- "gosec is too strict" - No. Fix all findings.
- "gosec finding is false positive" - Even if true, fix demonstrates security understanding. Use #nosec with detailed comment.
- "Will add tests in follow-up" - No. Tests required NOW.
- "Manually tested" - Not sufficient. Automated tests required.

### 5. Code Patterns

**Security-first coding patterns MUST be followed.**

```
☐ Wraps all errors with context (fmt.Errorf with %w)
☐ Validates before processing (fail fast)
☐ No panics for runtime errors (return errors)
☐ Context propagated through all calls
☐ No TODO comments for security items
```

**POST-MVP (when authentication added):**
```
☐ Uses crypto/subtle for token comparison (not ==)
```

**Evidence Required:**
- Show error wrapping: `fmt.Errorf("operation failed: %w", err)`
- Show validation before processing (not during/after)
- Confirm no panic() for runtime errors
- POST-MVP: Show crypto/subtle usage (when auth added)

## Output Template (MANDATORY)

**You MUST use this exact structure with ALL 5 sections:**

**Template Rules:**
- ALL 5 sections REQUIRED (no skipping)
- If section truly N/A, explain WHY (don't just skip or mark N/A without justification)
- Evidence MUST include actual code/output, not summaries ("Checked: Yes" is insufficient)
- "Show code" means paste minimum 3 lines of actual code
- "Show test" means paste test function and output, not "tested and passed"

```markdown
## Security Check: [Task Name]

### 1. Multi-Tenant Isolation
**Status**: ✅ PASS / 🚫 FAIL

**Database Selection:**
[Show code for database selection]

**Cross-Boundary Test:**
[Show test code and output proving cross-boundary queries fail]

**Findings:**
- [Specific findings or "No issues found"]

---

### 2. Input Validation
**Status**: ✅ PASS / 🚫 FAIL

**Validation Layers:**
- Handler: [Show validation code]
- Service: [Show validation code]
- Repository: [Show validation code]

**Malicious Input Test:**
[Show test with path traversal, injection, special chars]

**Findings:**
- [Specific findings or "No issues found"]

---

### 3. Sensitive Data Handling
**Status**: ✅ PASS / 🚫 FAIL

**Credential Management:**
[Show how credentials loaded - env vars, not hardcoded]

**Logging Redaction:**
[Show logging code with [REDACTED]]

**Constant-Time Comparison:**
[POST-MVP: Show crypto/subtle usage when authentication added]

**Findings:**
- [Specific findings or "No issues found"]

---

### 4. Security Testing
**Status**: ✅ PASS / 🚫 FAIL

**gosec scan:**
```
[Paste full gosec output showing findings count]
[If findings exist: ALL addressed - either fixed OR #nosec with detailed justification]
```

**Multi-Tenant Test:**
```
[Show test code and passing output]
```

**Input Validation Test:**
```
[Show test code and passing output]
```

**Findings:**
- [Specific findings or "No issues found"]

---

### 5. Code Patterns
**Status**: ✅ PASS / 🚫 FAIL

**Checks:**
- Error wrapping with %w: [Yes/No]
- Validation before processing: [Yes/No]
- No panics: [Yes/No]
- Context propagation: [Yes/No]

**POST-MVP Checks (when authentication added):**
- crypto/subtle for tokens: [Yes/No/N/A]

**Findings:**
- [Specific findings or "No issues found"]

---

## Overall Security Verdict

**APPROVED** ✅ / **BLOCKED** 🚫

**Summary:** [1-2 sentence security assessment]

**Required Actions** (if BLOCKED):
1. [Specific action required]
2. [Specific action required]
...

**Approval Conditions** (if BLOCKED):
- [ ] [Condition that must be met]
- [ ] [Condition that must be met]
...
```

## Common Security Testing Commands

**Run gosec:**
```bash
# Install if needed
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Scan all packages
gosec ./...

# Scan specific package
gosec ./pkg/auth/...
```

**Multi-Tenant Isolation Test Example:**
```go
func TestMultiTenantIsolation_CrossProjectQueryFails(t *testing.T) {
    // Setup: Create two projects with separate databases
    project1 := "/home/user/project1"
    project2 := "/home/user/project2"

    service1 := NewService(project1)
    service2 := NewService(project2)

    // Add checkpoint to project1
    checkpoint := &Checkpoint{Summary: "Project 1 data"}
    service1.Save(ctx, checkpoint)

    // Attempt to search from project2 - MUST NOT find project1 data
    results, err := service2.Search(ctx, "Project 1 data", 10)

    if err != nil {
        t.Fatalf("Search failed: %v", err)
    }

    if len(results) > 0 {
        t.Errorf("SECURITY VIOLATION: Found %d results from different project", len(results))
    }
}
```

**Input Validation Test Example:**
```go
func TestInputValidation_MaliciousInput_Rejected(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"path traversal", "../../../etc/passwd"},
        {"null byte", "file\x00.txt"},
        {"command injection", "file; rm -rf /"},
        {"SQL injection", "'; DROP TABLE users--"},
        {"special chars", "file<>:\"|?*"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePath(tt.input)
            if err == nil {
                t.Errorf("SECURITY VIOLATION: Malicious input accepted: %s", tt.input)
            }
        })
    }
}
```

## Anti-Patterns & Rationalizations

### "This is internal code, security doesn't apply"

**WRONG.** ALL code needs security validation.

Internal code becomes external. APIs get exposed. Code gets reused. Defense-in-depth means EVERY layer is secure, not just the edge.

**Correct approach:** Apply full security validation regardless of "internal" label.

### "Service layer already validated, repository doesn't need to"

**WRONG.** Validate at EVERY boundary.

What if repository is called directly? What if service validation has bug? What if future refactoring removes service validation?

**Correct approach:** Validate at handler, service, AND repository layers.

### "Tests pass, so security must be fine"

**WRONG.** Functional tests ≠ security tests.

Functional tests verify behavior. Security tests verify malicious inputs are rejected, isolation is enforced, and vulnerabilities don't exist.

**Correct approach:** Run gosec, write isolation tests, write malicious input tests.

### "This is a small change, full security check is overkill"

**WRONG.** Change size ≠ security impact.

One-line changes can introduce vulnerabilities. Adding a header can enable injection. Cosmetic changes can leak data.

**Correct approach:** Apply full security check regardless of change size.

### "Will add security tests in follow-up PR"

**WRONG.** Security validation required NOW, not later.

Later never comes. Follow-up PRs get deprioritized. Security debt accumulates. Vulnerabilities ship to production.

**Correct approach:** Complete all security testing before marking task complete.

### "gosec is too strict, some findings are false positives"

**WRONG.** Fix all gosec findings, no exceptions.

False positives are rare. Even if false positive, fixing demonstrates security understanding. Allowing bypass creates security culture problems.

**Correct approach:** Fix all gosec findings. Use `#nosec` ONLY with detailed justification comment.

### "This is an emergency, security checks after incident"

**WRONG.** Security NEVER bypassed, even for emergencies.

Emergency hotfixes that bypass security create vulnerabilities. Those vulnerabilities become new emergencies. Security shortcuts compound problems.

**Correct approach:** Run security checks even for hotfixes. Security testing takes <5 minutes.

### "Validating twice is redundant and wasteful"

**WRONG.** Defense-in-depth REQUIRES multiple layers.

Single validation point creates single point of failure. Redundant validation catches bugs, prevents future issues, and demonstrates security rigor.

**Correct approach:** Validate at every boundary, even if "redundant".

### "This package isn't listed in When to Use triggers"

**WRONG.** Triggers are examples, not exhaustive list.

If change affects ANY production code processing user input or handling data, security check applies. Triggers guide but don't limit scope.

**Correct approach:** Apply security check if change affects ANY production code with user input or data handling.

### "Change has code but it's mostly documentation"

**WRONG.** ANY .go file modification needs security check.

"Mostly documentation" still contains code changes. Code changes can introduce vulnerabilities, even in comments (e.g., commented-out credentials).

**Correct approach:** If ANY .go file modified, run security check.

### "Security validated in previous/related PR"

**WRONG.** Each change needs independent security validation.

Previous PR validated different code. This PR introduces new changes. Security validation is per-change, not per-feature.

**Correct approach:** Run full security check for THIS change, regardless of previous reviews.

### "Providing minimal evidence to save time"

**WRONG.** Evidence MUST be complete and verifiable.

"Validated: Yes" or "Tested and passed" provides no verification. Reviewers need actual code and output to validate security claims.

**Correct approach:** Paste actual code (minimum 3 lines), paste actual test output, show specific evidence.

## Red Flags - STOP and Run Security Check

**If you're thinking any of these, STOP and run full security check:**

- "Internal code, security not critical"
- "Tests pass, must be secure"
- "Will add security tests later"
- "This is too small for full security review"
- "Validated elsewhere, don't need to validate again"
- "gosec is being too strict"
- "gosec finding is false positive"
- "Emergency, security checks after"
- "Perfect is enemy of good"
- "Security checks slow me down"
- "This package isn't listed in When to Use"
- "Change is mostly documentation"
- "Previous PR already had security review"
- "Just providing summary to save time"

**All of these mean: STOP. Run full security check NOW.**

## Integration with Other Skills

**Completion workflow:**
1. Complete implementation
2. Run `contextd:security-check` (this skill)
3. If APPROVED → Run `contextd:completing-major-task`
4. If BLOCKED → Fix issues, return to step 2

**Code review workflow:**
1. Security check APPROVED
2. Completion verification complete
3. Run `contextd:code-review`

**Security check is gate:** MUST pass before completion or code review.

## Summary

**Security is non-negotiable:**
- Run ALL 5 checks (no skipping)
- Use structured output template (no shortcuts)
- Provide specific evidence (no "looks good")
- Block if ANY check fails (no partial approval)
- Require gosec passing (no exceptions)

**Remember:** Token cost of comprehensive security validation is FAR less than token cost (and business cost) of security breach.
