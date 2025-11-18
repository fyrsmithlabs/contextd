---
name: contextd-pkg-security
description: Use when working on auth, session, isolation, or RBAC packages in contextd - enforces multi-tenant isolation, input validation, constant-time comparison, and security-first development with mandatory gosec validation and defense-in-depth patterns
---

# Security-First Package Development

## Overview

**Security is PRIMARY, not negotiable.** When working on security packages (auth, session, isolation, rbac), every line of code MUST prioritize security over performance, convenience, or complexity.

**Core Principle:** Defense in depth. Validate at EVERY layer. Trust nothing. Isolate everything.

## When to Use This Skill

Use when working on:
- `pkg/auth` - Authentication (bearer tokens, JWT)
- `pkg/session` - Session management
- `pkg/isolation` - Multi-tenant isolation
- `pkg/rbac` - Role-based access control
- Any code handling credentials, tokens, or access control

**Trigger symptoms:**
- Code accepts user input (file paths, team names, search queries)
- Code validates credentials or tokens
- Code enforces team/project/org boundaries
- Code handles secrets (API keys, passwords)

## The Iron Law of Security-First Development

```
SECURITY ALWAYS COMES FIRST
```

**No exceptions for:**
- Performance optimization
- Code simplicity
- Tight deadlines
- "Trusted" sources
- "Unlikely" attacks
- Localhost-only deployment
- Expert opinions
- Mathematical proofs
- Environment differences (dev vs prod)
- Phased rollouts

**Violating the letter of security rules is violating the spirit of security.**

**No conditional security:** Security requirements apply EVERYWHERE (dev, staging, prod). No toggles, no environment-based skipping, no temporary disabling.

## Critical Security Requirements

### 1. Multi-Tenant Isolation (MANDATORY)

**Database-per-project physical isolation** prevents filter injection attacks.

**Boundaries:**
- **Project**: `project_<hash>` - Private to single project ONLY
- **Team**: `team_<name>` - Shared within team
- **Org**: `org_<name>` - Shared within organization

**Rules:**
- Checkpoints: Project database ONLY, NEVER cross-project
- Remediations/Skills: Team database, NEVER cross-team without permission
- Search: project → team → org → public (explicit permission required)
- ALL operations need isolation (reads AND writes, no exceptions)
- No phased migration (don't defer reads, fix everything)

**Code Pattern:**

```go
// ❌ WRONG: Could leak across projects
func SearchCheckpoints(query string) ([]Result, error) {
    return store.Search("checkpoints", query) // Which project?
}

// ✅ RIGHT: Project-scoped with validation
func SearchCheckpoints(projectPath string, query string) ([]Result, error) {
    // ALWAYS validate input first
    if err := validateProjectPath(projectPath); err != nil {
        return nil, fmt.Errorf("invalid project path: %w", err)
    }

    // Use project-specific database
    projectHash := hashProjectPath(projectPath)
    db := fmt.Sprintf("project_%s", projectHash)

    return store.Search(db, "checkpoints", query)
}
```

### 2. Input Validation (ALWAYS)

**EVERY user input MUST be validated.** No exceptions for "trusted" sources.

**Validate:**
- File paths (no `../../etc/passwd`)
- Git URLs (no command injection)
- Search queries (no injection)
- Team/org names (no special chars, SQL injection)
- Filter expressions (if any)

**Code Pattern:**

```go
// ❌ WRONG: Direct use of user input
func GetTeamProjects(teamName string) ([]Project, error) {
    query := fmt.Sprintf("SELECT * FROM projects WHERE team='%s'", teamName)
    // SQL injection: teamName = "' OR '1'='1"
    return db.Query(query)
}

// ✅ RIGHT: Validated and sanitized
func GetTeamProjects(teamName string) ([]Project, error) {
    // Validate team name format
    if !isValidTeamName(teamName) {
        return nil, fmt.Errorf("invalid team name: %s", teamName)
    }

    // Use parameterized query
    query := "SELECT * FROM projects WHERE team = ?"
    return db.Query(query, teamName)
}

func isValidTeamName(name string) bool {
    // Only alphanumeric, hyphens, underscores
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
    return matched && len(name) > 0 && len(name) <= 64
}
```

### 3. Constant-Time Comparison (REQUIRED)

**ALWAYS use `crypto/subtle` for token/password comparison.** String comparison leaks timing information.

**Code Pattern:**

```go
import "crypto/subtle"

// ❌ WRONG: Timing attack vulnerable
func ValidateToken(provided, expected string) bool {
    return provided == expected // Leaks timing information!
}

// ✅ RIGHT: Constant-time comparison
func ValidateToken(provided, expected string) bool {
    return subtle.ConstantTimeCompare(
        []byte(provided),
        []byte(expected),
    ) == 1
}
```

### 4. Sensitive Data Handling

**Never log, expose, or store credentials insecurely.**

**Rules:**
- Credentials from environment variables ONLY
- File permissions 0600 for token/key files
- Redact in logs: `zap.String("token", "[REDACTED]")`
- Never in error messages or stack traces

**Code Pattern:**

```go
// ❌ WRONG: Logs sensitive data
log.Printf("Validating token: %s", token) // Token in logs!

// ✅ RIGHT: Redacted logging
log.Printf("Validating token for user: %s", username)
if !validateToken(token) {
    log.Printf("Invalid token for user: %s", username)
    // Token value never logged
}
```

## Security Testing Requirements

**Tests MUST validate security properties:**

### Multi-Tenant Isolation Tests

```go
func TestSearchCheckpoints_CrossProjectIsolation(t *testing.T) {
    // Setup two projects
    project1 := "/home/user/project1"
    project2 := "/home/user/project2"

    // Save checkpoint to project1
    err := service.SaveCheckpoint(project1, checkpoint1)
    assertNoError(t, err)

    // Try to search from project2
    results, err := service.SearchCheckpoints(project2, checkpoint1.Summary)
    assertNoError(t, err)

    // MUST NOT return checkpoint from project1
    if len(results) > 0 {
        t.Error("Cross-project data leakage detected!")
    }
}
```

### Input Validation Tests

```go
func TestValidateProjectPath_RejectsTraversal(t *testing.T) {
    malicious := []string{
        "../../etc/passwd",
        "/etc/passwd",
        "../../../root/.ssh/id_rsa",
        "~/.aws/credentials",
    }

    for _, path := range malicious {
        err := validateProjectPath(path)
        if err == nil {
            t.Errorf("Accepted malicious path: %s", path)
        }
    }
}
```

### Timing Attack Tests

```go
func TestValidateToken_ConstantTime(t *testing.T) {
    expected := "correct-token-12345678"

    // Time correct token
    start := time.Now()
    for i := 0; i < 10000; i++ {
        ValidateToken(expected, expected)
    }
    correctDuration := time.Since(start)

    // Time incorrect token (different early)
    start = time.Now()
    for i := 0; i < 10000; i++ {
        ValidateToken("wrong", expected)
    }
    wrongDuration := time.Since(start)

    // Should be within 5% (constant-time)
    ratio := float64(correctDuration) / float64(wrongDuration)
    if ratio < 0.95 || ratio > 1.05 {
        t.Errorf("Timing difference detected: ratio=%f", ratio)
    }
}
```

### gosec Validation

```bash
# MUST pass with no new findings
gosec ./pkg/auth/...
gosec ./pkg/session/...
gosec ./pkg/isolation/...
gosec ./pkg/rbac/...
```

### Security Tests are BLOCKERS

**Security tests MUST be completed BEFORE merge. No exceptions.**

- No "add in follow-up PR"
- No "defer to next sprint"
- Security tests = merge requirement (same as build passing)
- Missing security tests = PR blocked

**Required security tests:**
- Multi-tenant isolation tests
- Input validation tests (malicious inputs)
- Timing attack tests (constant-time verification)
- gosec passing

## Common Rationalizations (STOP AND REJECT THESE)

| Excuse | Reality |
|--------|---------|
| "Input is from our own code, it's safe" | WRONG. Validate at EVERY entry point. Defense in depth. |
| "Validation hurts performance" | WRONG. Validation is microseconds. Security is non-negotiable. |
| "Localhost-only = no timing attacks" | WRONG. Local attackers exist. Always use constant-time. |
| "Timing attacks are theoretical" | WRONG. They're practical and well-documented. Use `crypto/subtle`. |
| "Multi-tenant checks are too complex" | WRONG. Complexity is necessary. Use database-per-project pattern. |
| "This is an internal API" | WRONG. Internal APIs need validation. Never trust input. |
| "Authentication means trusted" | WRONG. Authenticated ≠ validated. Check EVERY input. |
| "Security slows down development" | WRONG. Security bugs slow down EVERYTHING. Do it right first. |
| "Disable validation in prod for performance" | WRONG. Production needs MORE security, not less. |
| "Fix writes first, reads later" | WRONG. Reads leak data too. Fix ALL operations at once. |
| "Add security tests in follow-up PR" | WRONG. Security tests REQUIRED before merge. Blocker. |
| "Security expert approved this exception" | WRONG. Rules apply to everyone, including experts. |
| "Math proves attack impractical" | WRONG. Use constant-time anyway. No exceptions. |
| "Document exception in ADR" | WRONG. No exceptions to document. Follow the rules. |

**All of these mean: Stop. Follow security requirements. No shortcuts.**

## Red Flags - STOP and Fix Immediately

**If you catch yourself saying or thinking:**
- "We can skip validation here"
- "This input is trusted"
- "Constant-time is overkill"
- "Performance matters more"
- "Attack is unlikely"
- "Localhost is safe"
- "Too complicated for this use case"
- "We can disable this in production"
- "Let's fix the critical path first, then come back"
- "Reads can't leak data"
- "Security tests in a follow-up PR"
- "The security expert approved this exception"
- "Mathematical analysis shows low risk"
- "We'll document the exception in an ADR"

**STOP. You are about to introduce a security vulnerability.**

## Integration with Verification Skills

**Before completing security package work:**

1. **Use completing-major-task skill** with FULL security validation:
   - Build passes
   - Tests pass with >80% coverage
   - Security tests included (isolation + validation + timing)
   - `gosec ./...` passes with no new findings
   - Manual security verification performed

2. **Before PR: Use code-review skill**
   - Code reviewer validates security checklist
   - Multi-tenant isolation verified
   - Input validation verified
   - Constant-time comparison verified
   - gosec findings addressed

## Quick Reference: Security Checklist

Before marking security package work complete:

- [ ] ALL user inputs validated (file paths, URLs, team names, queries)
- [ ] Multi-tenant isolation enforced (database-per-project pattern)
- [ ] Constant-time comparison for ALL credential checks
- [ ] No credentials in code (environment variables only)
- [ ] No sensitive data in logs (use `[REDACTED]`)
- [ ] File permissions 0600 for credential files
- [ ] Security tests written (isolation + validation + timing)
- [ ] `gosec ./...` passes with no new findings
- [ ] Code review completed with security focus

## The Bottom Line

**Security is not optional.** Every security package change MUST:

1. Validate ALL inputs (no "trusted source" exception)
2. Enforce multi-tenant isolation (database-per-project)
3. Use constant-time comparison (always `crypto/subtle`)
4. Pass gosec with no new findings
5. Include security-specific tests

**No performance, complexity, or deadline justification overrides these requirements.**

If you violate these rules, you WILL introduce security vulnerabilities. Follow the checklist. Every time.
