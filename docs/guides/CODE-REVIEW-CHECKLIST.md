# Code Review Checklist

**Audience**: Code-reviewer agents (human and AI)
**Purpose**: Comprehensive validation before PR merge
**Status**: MANDATORY | **Last Updated**: 2025-11-18

## Overview

This checklist validates all work before merge. Code-reviewer agents MUST complete all sections and provide structured output.

**Review Scope**:
1. Verification Evidence (from VERIFICATION-POLICY.md)
2. Security Compliance (multi-tenant isolation, input validation)
3. Code Standards (Go idioms, error handling, naming)
4. Test Coverage (≥80%, TDD compliance)
5. Documentation (godoc, CHANGELOG, specs)
6. Architecture Compliance (ADRs, package guidelines)

**Review Verdicts**:
- **APPROVED** - All checks pass, ready to merge
- **CHANGES REQUIRED** - Issues found, must fix before merge
- **BLOCKED** - Critical failures (security, multi-tenancy, data leakage)

---

## 1. Verification Evidence Validation

**Objective**: Ensure completion claims have proof.

### For Major Tasks

**Check that completion includes ALL required fields**:

```markdown
Task: [present and specific]
Type: [present: Feature/Bug Fix/Refactor/Security/Docs]
Changes: [present, file-by-file breakdown]
Verification Evidence:
  ✓ Build: [command + output shown]
  ✓ Tests: [command + output + coverage %]
  ✓ Security: [multi-tenant isolation + input validation + gosec]
  ✓ Functionality: [manual test results or behavior verification]
Risk Assessment: [honest evaluation present]
```

**Validation Criteria**:
- [ ] Template structure complete (all fields present)
- [ ] Build evidence shows command AND output
- [ ] Test evidence shows results AND coverage percentage
- [ ] Coverage meets ≥80% requirement
- [ ] Security validation includes multi-tenant isolation check
- [ ] Security validation confirms no new gosec findings
- [ ] Functionality verification shows actual test results (not "tested and works")
- [ ] Risk assessment is honest and specific (not generic)

**Common Failures**:
- ❌ "Build: Success" (no command or output)
- ❌ "Tests: All passed" (no count, no coverage)
- ❌ "Security: Looks good" (no specific checks)
- ❌ "Verified: yes" (no actual verification)

### For Minor Tasks

**Check that completion includes ALL required questions answered**:

```markdown
Task: [present and specific]
✓ What changed: [specific change, not vague]
✓ How I know it works: [verification performed with evidence]
✓ What breaks if wrong: [honest risk assessment]
```

**Validation Criteria**:
- [ ] All three questions answered
- [ ] "What changed" is specific (file + line number if applicable)
- [ ] "How I know it works" shows actual verification performed
- [ ] "What breaks if wrong" is honest assessment (not "nothing")

**Common Failures**:
- ❌ "What changed: Updated file" (too vague)
- ❌ "How I know it works: Checked it" (no evidence)
- ❌ "What breaks if wrong: N/A" (avoiding the question)

### Evidence Quality Assessment

**For each evidence field, check**:
- [ ] Not empty or placeholder text
- [ ] Shows actual command output (not summarized)
- [ ] Output is complete (not truncated without reason)
- [ ] Output matches claimed changes (consistency check)

**Evidence Cross-Check**:
- If task claims "added tests" → test output must show new tests
- If task claims "fixed bug" → test output must include regression test
- If task claims "improved performance" → evidence must show measurements

---

## 2. Security Review (CRITICAL for contextd)

**Objective**: Prevent data leakage, ensure multi-tenant isolation.

### Multi-Tenant Isolation

**Check ALL changes for isolation compliance**:

- [ ] **Project Boundaries**: Data scoped to `project_<hash>` (SHA256 of project_path)
- [ ] **Team Boundaries**: Team data (`team_<name>`) never leaks to other teams
- [ ] **Database Isolation**: Uses correct database for data type:
  - Checkpoints: project-specific database ONLY
  - Remediations/Skills: shared database within team
  - No cross-team queries without explicit permission
- [ ] **Filter Injection Prevention**: No user-controlled filter parameters (database-per-project eliminates this attack vector)

**Specific Checks**:
```go
// ❌ WRONG: Cross-project query
results := qdrant.Search(collection, query) // searches ALL projects

// ✅ RIGHT: Project-scoped query
projectDB := getProjectDatabase(projectPath) // correct database
results := projectDB.Search(collection, query)
```

### Input Validation

**Check ALL user inputs are validated**:

- [ ] File paths sanitized (no path traversal: `../../../etc/passwd`)
- [ ] Git URLs validated (no command injection)
- [ ] Search queries sanitized (no injection attacks)
- [ ] Filter expressions validated (if any)
- [ ] Team names validated (no special chars, SQL injection)
- [ ] Org names validated (no special chars, SQL injection)

**Validation Pattern**:
```go
// ❌ WRONG: Direct use of user input
filepath := userInput

// ✅ RIGHT: Validated and sanitized
filepath, err := sanitizeFilePath(userInput)
if err != nil {
    return fmt.Errorf("invalid file path: %w", err)
}
```

### Sensitive Data Handling

**Check sensitive data is protected**:

- [ ] No credentials in code (API keys, tokens, passwords)
- [ ] Secrets loaded from environment variables or secure storage
- [ ] Sensitive data redacted in logs (use `zap.String("token", "[REDACTED]")`)
- [ ] No PII in error messages or logs

### Security Tooling

**Verify security tools pass**:

- [ ] `gosec ./...` - No new security findings
- [ ] Pre-commit hooks passed (includes TruffleHog secret detection)
- [ ] No `--no-verify` used to bypass security checks

---

## 3. Code Standards (Go-Specific)

**Objective**: Ensure idiomatic Go code.

### Naming Conventions

- [ ] Package names: lowercase, single word, descriptive
- [ ] Exported identifiers: PascalCase, clear purpose
- [ ] Unexported identifiers: camelCase
- [ ] Constants: PascalCase or SCREAMING_SNAKE_CASE (for enums)
- [ ] No stuttering (e.g., `http.HTTPServer` → `http.Server`)

### Error Handling

- [ ] Errors wrapped with context: `fmt.Errorf("operation failed: %w", err)`
- [ ] Errors checked (no ignored `err` return values)
- [ ] Custom errors use `errors.New()` or `fmt.Errorf()`
- [ ] No panic in library code (only in `main` for fatal errors)

**Error Handling Pattern**:
```go
// ❌ WRONG: Error without context
if err != nil {
    return err
}

// ✅ RIGHT: Error with context
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
```

### Code Organization

- [ ] Follows golang-standards/project-layout
- [ ] No `/src/`, `/lib/`, `/models/` directories
- [ ] Package structure logical (by feature, not layer)
- [ ] No circular dependencies

### Code Quality

- [ ] `gofmt` compliant (formatting)
- [ ] `goimports` compliant (import ordering)
- [ ] `go vet` passes (suspicious constructs)
- [ ] `golangci-lint` passes (comprehensive linting)
- [ ] No commented-out code (remove, don't comment)
- [ ] No TODO comments without issue numbers

---

## 4. Test Coverage & TDD Compliance

**Objective**: Ensure tests validate behavior, meet coverage requirements.

### Coverage Requirements

- [ ] Overall coverage ≥80%
- [ ] New code coverage ≥80% (check diff)
- [ ] Critical paths have 100% coverage (auth, security, multi-tenant isolation)

**Measure Coverage**:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### TDD Compliance

**Check that tests were written FIRST**:

- [ ] Git history shows test commit before implementation
- [ ] OR: Tests exist for all new functionality
- [ ] Tests validate behavior, not implementation

**TDD Red-Green-Refactor Evidence**:
- Red: Test exists that would fail before implementation
- Green: Implementation makes test pass
- Refactor: Code improved without breaking tests

### Test Quality

- [ ] Tests use table-driven patterns for multiple cases
- [ ] Tests have clear names (TestFunctionName_Scenario_ExpectedBehavior)
- [ ] No testing mock behavior (mock setup ≠ verification)
- [ ] No test-only methods in production code
- [ ] No mocks without understanding dependencies
- [ ] Tests are deterministic (no race conditions, timing dependencies)

**Table-Driven Test Pattern**:
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "test", false},
        {"empty input", "", true},
        {"invalid chars", "../etc", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## 5. Documentation Review

**Objective**: Ensure code is documented and CHANGELOG updated.

### Code Documentation

- [ ] All exported functions have godoc comments
- [ ] Godoc comments start with function name
- [ ] Package has package-level godoc
- [ ] Complex logic has inline comments explaining "why" (not "what")

**Godoc Pattern**:
```go
// ❌ WRONG: Missing godoc
func ProcessData(data []byte) error {

// ✅ RIGHT: Proper godoc
// ProcessData validates and processes the input data.
// Returns error if data is invalid or processing fails.
func ProcessData(data []byte) error {
```

### CHANGELOG Update

**CRITICAL**: CHANGELOG.md MUST be updated for every change.

- [ ] Entry added under `[Unreleased]` section
- [ ] Entry in correct category:
  - `### Added` for features
  - `### Fixed` for bug fixes
  - `### Changed` for modifications (use **BREAKING** marker if breaking)
  - `### Removed` for deletions
- [ ] Entry is clear and user-focused
- [ ] Entry follows conventional commits style

### Specification Updates

- [ ] Relevant specs updated in `docs/specs/<feature>/SPEC.md`
- [ ] Architecture decisions documented in `docs/architecture/adr/` (if applicable)
- [ ] Package CLAUDE.md updated (if package-level changes)

---

## 6. Architecture Compliance

**Objective**: Ensure changes follow architectural decisions.

### ADR Compliance

- [ ] Change follows existing ADRs in `docs/architecture/adr/`
- [ ] If contradicts ADR: new ADR created or existing ADR updated
- [ ] No architectural decisions made without documentation

### Package Guidelines

- [ ] New packages follow `docs/standards/package-guidelines.md`
- [ ] Package structure documented in `pkg/CLAUDE.md`
- [ ] Dependencies justified and minimal

### Interface Design

- [ ] Interfaces are minimal and focused
- [ ] Interfaces represent real abstractions (not premature generalization)
- [ ] Concrete implementations preferred over complex hierarchies
- [ ] No empty interfaces (`interface{}` → `any`, used sparingly)

### YAGNI Compliance

- [ ] No speculative features ("we might need this later")
- [ ] No unused code or dead code paths
- [ ] No premature optimization
- [ ] Every feature solves a current, concrete problem

---

## Pre-Commit & Pre-PR Verification

**Objective**: Ensure automated checks passed.

### Pre-Commit Hooks

- [ ] Pre-commit hooks installed (`pre-commit --version` succeeds)
- [ ] All pre-commit checks passed (no `--no-verify` used)
- [ ] Secret detection passed (TruffleHog)
- [ ] Security scanning passed (gosec)
- [ ] Formatting passed (gofmt, goimports)
- [ ] Linting passed (go vet, golangci-lint)
- [ ] Commit message format valid (conventional commits)

### Build & Test Verification

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `go test -race ./...` passes (no race conditions)
- [ ] `go test -coverprofile=coverage.out ./...` meets coverage requirement

---

## Structured Review Output Template

**Code-reviewer agents MUST use this output format**:

```markdown
## Code Review: [Task/PR Name]

**Reviewer**: [Agent name or human reviewer]
**Date**: [YYYY-MM-DD]
**Commit**: [commit hash]

---

### 1. Verification Evidence
**Status**: ✅ PASS / ⚠️  CHANGES REQUIRED / 🚫 BLOCKED

**Findings**:
- [List findings, if any]

**Evidence Quality**: [Assessment of evidence completeness]

---

### 2. Security Review
**Status**: ✅ PASS / ⚠️  CHANGES REQUIRED / 🚫 BLOCKED

**Multi-Tenant Isolation**: [Assessment]
**Input Validation**: [Assessment]
**Sensitive Data**: [Assessment]
**gosec Results**: [No new findings / Findings listed below]

**Findings**:
- [List security issues, if any]

---

### 3. Code Standards
**Status**: ✅ PASS / ⚠️  CHANGES REQUIRED / 🚫 BLOCKED

**Findings**:
- [List code standard violations, if any]

---

### 4. Test Coverage
**Status**: ✅ PASS / ⚠️  CHANGES REQUIRED / 🚫 BLOCKED

**Coverage**: [X%] (Requirement: ≥80%)
**TDD Compliance**: [Assessment]

**Findings**:
- [List test coverage issues, if any]

---

### 5. Documentation
**Status**: ✅ PASS / ⚠️  CHANGES REQUIRED / 🚫 BLOCKED

**CHANGELOG Updated**: [Yes/No]
**Godoc Complete**: [Yes/No]
**Specs Updated**: [Yes/No/N/A]

**Findings**:
- [List documentation issues, if any]

---

### 6. Architecture Compliance
**Status**: ✅ PASS / ⚠️  CHANGES REQUIRED / 🚫 BLOCKED

**ADR Compliance**: [Assessment]
**YAGNI Compliance**: [Assessment]

**Findings**:
- [List architecture issues, if any]

---

### Overall Verdict
**APPROVED** ✅ / **CHANGES REQUIRED** ⚠️ / **BLOCKED** 🚫

**Summary**: [Brief summary of review outcome]

**Required Actions** (if CHANGES REQUIRED or BLOCKED):
1. [Action item 1]
2. [Action item 2]
...

**Approval Conditions** (if CHANGES REQUIRED):
- [ ] [Condition 1]
- [ ] [Condition 2]
...
```

---

## Review Workflow Integration

### When Code Review is Invoked

Code review happens at these points:

1. **After Major Task Completion** - Optional, recommended for complex tasks
2. **Before PR Creation** - MANDATORY via `contextd:code-review` skill
3. **After PR Creation** - Automated GitHub Actions code review workflow

### Review Loop

1. Developer completes work
2. Developer invokes `contextd:code-review` skill
3. Code-reviewer agent executes this checklist
4. Code-reviewer provides structured output (using template above)
5. If **CHANGES REQUIRED** or **BLOCKED**: Developer fixes issues, returns to step 2
6. If **APPROVED**: Developer creates PR
7. GitHub Actions runs automated review (secondary validation)
8. PR merged after all approvals

---

## Summary for Code-Reviewer Agents

**Your Job**:
1. Execute ALL sections of this checklist
2. Provide structured output using template
3. Be thorough but fair
4. Block only for critical issues (security, data leakage)
5. Require changes for violations of standards
6. Approve when all criteria met

**Remember**:
- Verification evidence is MANDATORY (no exceptions)
- Security issues BLOCK merge (not negotiable)
- Test coverage ≥80% REQUIRED
- CHANGELOG update MANDATORY
- YAGNI violations require justification

**You are the last line of defense before merge. Take this responsibility seriously.**
