# Multi-Agent Code Review Workflow

**Purpose**: Consensus-driven code review using parallel expert agents (Security, QA, Go) with systematic remediation.

**When to Use**: After completing significant features, before merging, or when quality gates needed.

---

## Overview

Three expert agents review code in parallel from different perspectives:
- **Security Expert**: Vulnerabilities, secret handling, input validation, multi-tenancy
- **QA Expert**: Test coverage, error paths, boundary conditions, test quality
- **Go Expert**: Idioms, performance, API design, concurrency safety

Reviews synthesized into consensus findings, then remediated in priority order.

---

## Workflow Steps

### Phase 1: Parallel Expert Reviews

**Dispatch 3 agents simultaneously:**

```
Task tool (general-purpose):
  - Security Expert: Review from security perspective
  - QA Expert: Review from testing perspective
  - Go Expert: Review from Go best practices perspective
```

**Each expert produces:**
- Critical Findings (MUST fix)
- Important Findings (SHOULD fix)
- Minor Findings (nice-to-have)
- Strengths (what's done well)
- Recommendations (specific, actionable)

**Review Scope:**
- All files in target package
- Tests and implementation
- Documentation
- Error handling
- Edge cases

### Phase 2: Synthesize Consensus

**Analyze findings across experts:**

| Issue | Security | QA | Go | Consensus |
|-------|----------|----|----|-----------|
| Secret YAML leakage | CRITICAL | - | Important | **CRITICAL** |
| Missing tests | - | HIGH | - | **IMPORTANT** |
| Package docs | - | - | CRITICAL | **CRITICAL** |

**Build priority matrix:**
1. **Critical**: All experts agree OR any expert says CRITICAL
2. **Important**: 2+ experts flag OR significant risk
3. **Minor**: 1 expert mentions, low impact

**Create remediation plan:**
- Group by priority (Critical → Important → Minor)
- Identify parallelizable fixes
- Estimate effort

### Phase 3: Remediate in Priority Order

**For each consensus issue:**

1. **Dispatch remediation agent** (taskmaster:task-executor)
   - Clear task description
   - Security requirements if applicable
   - Test requirements
   - CHANGELOG update

2. **Agent implements fix:**
   - Follow TDD if adding features
   - Write tests first for bugs
   - Update docs
   - Commit with conventional message

3. **Verify fix:**
   - Tests pass
   - Coverage maintained/improved
   - No regressions

**Parallel execution:**
- Independent fixes can run in parallel
- Dependent fixes run sequentially

### Phase 4: Final Verification

**Run comprehensive checks:**

```bash
go test ./... -v -cover
go vet ./...
go mod tidy && go mod verify
```

**Verify metrics:**
- Test coverage (target: >80%)
- All tests passing
- No lint issues
- CHANGELOG updated

**Compare before/after:**
- Coverage delta
- Security posture
- Code quality grade

---

## Expert Agent Prompts

### Security Expert Template

```markdown
You are a **Senior Security Engineer** conducting a security review.

## Your Expertise
- Secret management and credential handling
- Input validation and injection attacks
- Configuration security
- Multi-tenant isolation
- Defense in depth

## Code Location
[path to package]

## Your Task
Review ALL files from security perspective:

1. **Secret Handling**: Can secrets leak?
2. **Input Validation**: Can malicious input cause issues?
3. **Configuration Security**: File permissions, path traversal?
4. **Multi-tenant Isolation**: Can tenant A access tenant B data?
5. **Validation Bypass**: Can validation be circumvented?

## Report Format
- Critical Findings (MUST fix - vulnerabilities)
- Important Findings (SHOULD fix - hardening)
- Minor Findings (nice-to-have)
- Strengths (what's good)
- Recommendations (specific fixes with code)

Include file paths and line numbers.
```

### QA Expert Template

```markdown
You are a **Senior QA Engineer** conducting quality review.

## Your Expertise
- Test coverage and quality
- Edge cases and boundary conditions
- Error handling
- Integration testing
- Test maintainability

## Code Location
[path to package]

## Your Task
Review ALL files from QA perspective:

1. **Test Coverage**: Which paths lack tests?
2. **Test Quality**: Deterministic? Representative data?
3. **Error Handling**: All error paths tested?
4. **Testability**: Easy to test? Mocks appropriate?
5. **Boundary Conditions**: Max/min/empty/nil tested?

## Report Format
- Critical Findings (test gaps exposing risk)
- Important Findings (test quality issues)
- Minor Findings (improvements)
- Coverage Analysis (untested paths with line numbers)
- Recommendations (specific test cases with examples)

Run tests, check coverage, analyze quality.
```

### Go Expert Template

```markdown
You are a **Senior Go Engineer** conducting best practices review.

## Your Expertise
- Go idioms and conventions
- Performance and efficiency
- API design and ergonomics
- Error handling patterns
- Concurrency safety
- Memory management

## Code Location
[path to package]

## Your Task
Review ALL files from Go perspective:

1. **API Design**: Idiomatic? Clean? Minimal?
2. **Error Handling**: Wrapped with %w? Actionable messages?
3. **Performance**: Unnecessary allocations? Efficient?
4. **Concurrency Safety**: Race conditions? Atomic needed?
5. **Go Conventions**: Package docs? Naming? Organization?
6. **Memory**: Leaks? Resource cleanup? Struct layout?

## Report Format
- Critical Findings (breaks Go conventions or has bugs)
- Important Findings (not idiomatic or suboptimal)
- Minor Findings (polish)
- Performance Notes (efficiency observations)
- Recommendations (specific improvements with examples)

Check Go 1.23+ compatibility and stdlib usage.
```

---

## Remediation Agent Template

```markdown
You are implementing [ISSUE DESCRIPTION] from multi-agent code review.

## Issue
[Detailed description of the problem]

## Expert Consensus
- Security: [finding]
- QA: [finding]
- Go: [finding]

## Your Task
1. [Specific fix step 1]
2. [Specific fix step 2]
3. Write/update tests
4. Update CHANGELOG.md under ### Security or ### Fixed
5. Commit: "[type](scope): [description]"

Work from: [directory]

IMPORTANT:
- [Any special requirements]
- [Security considerations]
- [Test requirements]

Report back:
- What you implemented
- Test results
- Files changed
- Verification
```

---

## Success Criteria

### Metrics Before/After

| Metric | Target | Track |
|--------|--------|-------|
| Test Coverage | >80% | Δ% |
| Security Issues | 0 critical | Count by severity |
| Lint Issues | 0 | Count |
| Tests Passing | 100% | Pass/fail |
| Documentation | Complete | Package docs, examples |

### Expert Consensus Matrix

**Critical Issue Definition:**
- ANY expert marks as CRITICAL, OR
- 2+ experts mark as Important with security/data-loss risk, OR
- Violates project security policies

**Important Issue Definition:**
- 2+ experts flag as Important, OR
- 1 expert flags with significant production risk, OR
- Required for compliance/standards

**Minor Issue Definition:**
- 1 expert mentions
- Low/no production impact
- Polish or nice-to-have

### Production Readiness Checklist

- [ ] All Critical issues fixed
- [ ] All Important issues fixed or documented as accepted risk
- [ ] Test coverage ≥80%
- [ ] No security vulnerabilities
- [ ] Documentation complete (package docs, examples, CHANGELOG)
- [ ] Lint/vet passing
- [ ] Go version current (security patches)

---

## When to Use This Workflow

### Required (Mandatory)

- **Before production deployment** - Any code going to prod
- **After major features** - Significant functionality added
- **Security-sensitive changes** - Auth, secrets, multi-tenant, validation
- **Before v1.0 or major releases** - Quality gate

### Recommended (Strongly Suggested)

- **After refactoring** - Verify no regressions
- **When test coverage drops** - Quality slipping
- **After dependency updates** - Verify no breaking changes
- **Before merging long-lived branches** - Integration check

### Optional (Beneficial)

- **Learning/training** - See how experts think
- **Architecture review** - Get multiple perspectives
- **Performance optimization** - Expert analysis
- **Before adding to library/framework** - Extra scrutiny for reusable code

---

## Execution Example

```markdown
User: "conduct code-review and remediate following the multi-agent workflow"