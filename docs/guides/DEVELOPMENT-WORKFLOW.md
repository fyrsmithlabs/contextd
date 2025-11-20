# Development Workflow Guide

**See [../../CLAUDE.md](../../CLAUDE.md) for project overview.**

## Quick Reference

**Before ANY task**: Check Superpowers skills and use TaskMaster for planning.

**Workflow**: Superpowers Check → TaskMaster Planning → Research → Review → Refine → Approve → Test (Red) → Implement (Green) → Refactor → Create Test Skill

---

## Mandatory Multi-Agent Orchestration Workflow

**CRITICAL**: ALL tasks MUST follow this multi-agent workflow. Single-agent completion is NOT acceptable.

### Core Principle

**Consensus Requirement**: All specialized agents must review and approve the solution before the task is considered complete. No single agent (including you) can unilaterally declare a task done.

### Agent Assignment by Task Type

| Task Type | Required Agents/Skills | Agreement Criteria |
|-----------|----------------------|-------------------|
| **Go Code Implementation** | 1. `golang-pro` skill (implementation)<br>2. `security-auditor` agent (security review)<br>3. `superpowers:code-reviewer` agent (final review) | All 3 must approve: code quality, security compliance, test coverage ≥80% |
| **MCP Protocol Work** | 1. `mcp-developer` agent (design/research)<br>2. `golang-pro` skill (implementation)<br>3. `superpowers:code-reviewer` agent (validation) | Protocol compliance verified by mcp-developer, implementation approved by golang-pro, final validation by code-reviewer |
| **Security-Critical Changes** | 1. `security-auditor` agent (threat modeling)<br>2. `golang-pro` skill (secure implementation)<br>3. `contextd:security-check` skill (contextd-specific validation)<br>4. `superpowers:code-reviewer` agent (final review) | All 4 must approve: threat model, implementation, contextd multi-tenant compliance, overall quality |
| **Documentation Updates** | 1. `kinney-documentation` skill (structure/style)<br>2. `documentation-engineer` agent (technical accuracy)<br>3. `superpowers:code-reviewer` agent (completeness check) | All 3 must approve: scannable structure, technical accuracy, completeness |
| **Architecture Decisions** | 1. Specialist agent (e.g., `mcp-developer`, `qdrant-specialist`)<br>2. `golang-pro` skill (feasibility check)<br>3. `security-auditor` agent (security implications)<br>4. `superpowers:code-reviewer` agent (ADR compliance) | All 4 must approve: design soundness, implementation feasibility, security, ADR alignment |
| **Test Strategy** | 1. `test-strategist` agent (strategy design)<br>2. `golang-pro` skill (test implementation)<br>3. `superpowers:code-reviewer` agent (coverage validation) | All 3 must approve: strategy completeness, test quality, coverage ≥80% |

### Standard Multi-Agent Workflow (ALL Tasks)

```
Step 1: Planning & Design
├─ TaskMaster: Break down task into subtasks
├─ Specialist Agent: Design solution (architecture, strategy, research)
│  └─ Output: Design document, requirements, approach
└─ Review: Team review of design before implementation

Step 2: Implementation
├─ golang-pro skill: Implement solution following design
│  ├─ TDD (Red → Green → Refactor)
│  ├─ Security-first coding
│  └─ ≥80% test coverage
└─ Output: Working code with comprehensive tests

Step 3: Security Validation (if applicable)
├─ security-auditor agent: Review for vulnerabilities
│  ├─ Multi-tenant isolation
│  ├─ Input validation
│  └─ gosec findings
└─ contextd:security-check skill: Contextd-specific checks

Step 4: Final Code Review (MANDATORY)
├─ superpowers:code-reviewer agent: Comprehensive review
│  ├─ Verification evidence validation
│  ├─ Standards compliance
│  ├─ Documentation completeness
│  └─ Architecture compliance
└─ Verdict: APPROVED / CHANGES REQUIRED / BLOCKED

Step 5: Task Completion (Only if ALL agents approved)
└─ contextd:completing-major-task or contextd:completing-minor-task
```

### Parallel Agent Execution

**For independent subtasks**, deploy agents in parallel using a single message with multiple Task tool calls:

```javascript
// Example: Implementing authentication system
Task("Security threat modeling", "Analyze auth system threats...", "security-auditor")
Task("Go implementation with TDD", "Implement JWT auth with ≥80% coverage...", "golang-pro")
Task("Documentation", "Create auth system docs...", "documentation-engineer")

// After all complete, synchronize for code review
Task("Final code review", "Validate all work...", "superpowers:code-reviewer")
```

### Examples by Task Type

#### Example 1: Go Feature Implementation

```
User: "Implement JWT authentication for MCP endpoints"

1. Security threat modeling (security-auditor agent)
   → Identifies: token storage, replay attacks, timing attacks

2. Implementation (golang-pro skill)
   → Implements: constant-time comparison, secure storage, tests
   → Result: 87% coverage, all tests pass

3. Security validation (security-auditor agent + contextd:security-check)
   → Validates: Multi-tenant isolation maintained, no new gosec findings

4. Code review (superpowers:code-reviewer agent)
   → Validates: All standards met, documentation complete
   → Verdict: APPROVED

5. Task completion (contextd:completing-major-task)
   → Evidence: Build output, test results, security validation, functionality proof
```

#### Example 2: MCP Protocol Design

```
User: "Design MCP batch operations endpoint"

1. Protocol research (mcp-developer agent)
   → Researches: MCP spec 2025-06-18, batch patterns, error handling
   → Output: Design document with spec compliance

2. Feasibility check (golang-pro skill)
   → Validates: Implementation feasible in Go, identifies edge cases

3. Security review (security-auditor agent)
   → Validates: No SSRF, rate limiting, multi-tenant safe

4. Implementation (golang-pro skill)
   → Implements: Following mcp-developer's design
   → Result: Protocol-compliant, tested

5. Protocol validation (mcp-developer agent)
   → Validates: Spec compliance, proper JSON-RPC

6. Final review (superpowers:code-reviewer agent)
   → Verdict: APPROVED
```

#### Example 3: Documentation Update

```
User: "Update architecture documentation for stdio transport"

1. Structure design (kinney-documentation skill)
   → Designs: Scannable structure, ~150 lines, @imports for details

2. Technical content (documentation-engineer agent)
   → Writes: Accurate technical descriptions, diagrams, examples

3. Accuracy validation (mcp-developer agent, if MCP-related)
   → Validates: Protocol accuracy, terminology correctness

4. Completeness check (superpowers:code-reviewer agent)
   → Validates: All sections present, CHANGELOG updated

5. Task completion (contextd:completing-minor-task or major)
```

### Anti-Patterns (What NOT to Do)

❌ **Single-Agent Completion**
```
Bad: "I implemented and reviewed the code myself. Task complete."
Why: No independent validation, bias in self-review
Fix: Deploy multiple agents, get consensus
```

❌ **Skipping Security Review**
```
Bad: "Code looks secure, skipping security-auditor agent"
Why: Security issues missed, multi-tenant isolation not validated
Fix: ALWAYS run security-auditor for ANY code change
```

❌ **Sequential When Parallel Possible**
```
Bad: Design → wait → Implement → wait → Test → wait → Review
Why: Wastes time when subtasks are independent
Fix: Deploy agents in parallel for independent work
```

❌ **Accepting First Approval**
```
Bad: "golang-pro approved, shipping it!"
Why: Missing security review, code review, verification
Fix: ALL required agents must approve before completion
```

### Consensus Decision Making

**When agents disagree:**

1. **Document disagreement**: Capture each agent's concerns
2. **Identify root cause**: Technical limitation? Security risk? Standard violation?
3. **Escalate if needed**: Ask user for guidance on tradeoffs
4. **Iterate solution**: Address all concerns until consensus reached
5. **No forced consensus**: If fundamental disagreement exists, do NOT proceed

**Example Disagreement Resolution**:
```
security-auditor: "Timing attack possible in token comparison"
golang-pro: "Using strings.Compare for efficiency"

Resolution:
1. Acknowledge security concern (valid threat)
2. Implement constant-time comparison (crypto/subtle)
3. Re-review by both agents
4. Consensus: APPROVED (security + efficiency via proper library)
```

### Verification at Each Stage

**Each agent must provide structured output:**

```markdown
Agent: [Agent name]
Task: [What was reviewed]
Status: APPROVED / CHANGES REQUIRED / BLOCKED
Findings:
  - [Specific finding with file:line reference]
  - [Specific finding with file:line reference]
Recommendations:
  - [Actionable recommendation]
```

**Final completion requires:**
- [ ] All required agents for task type have reviewed
- [ ] All agents status: APPROVED
- [ ] Verification evidence template complete
- [ ] CHANGELOG.md updated
- [ ] All tests passing with ≥80% coverage

---

## Critical: Always Check Standards & Specs First

Before starting any task:

1. **Read relevant standards** from `docs/standards/` based on your task
2. **Read feature specs** from `docs/specs/` if implementing a feature
3. **Check architecture docs** in `docs/architecture/` for design decisions
4. **Check package CLAUDE.md** if working in a specific package
5. **DELEGATE TO golang-pro** for all Go code implementation

## Standard Selection Guide

| **When working on...** | **Read these standards (in order)** |
|---|---|
| Architecture decisions | standards/architecture.md → coding-standards.md |
| Any code changes | standards/coding-standards.md → testing-standards.md |
| New packages | standards/package-guidelines.md → architecture.md |
| Writing tests | standards/testing-standards.md → [relevant feature spec] |
| GitHub Actions workflows | .github/workflows/CLAUDE.md |

---

## Spec-Driven Development Workflow

**CRITICAL**: Follow this workflow for ALL new features and significant changes.

### 1. Issue Selection and Setup

```javascript
// Get issue details using GitHub MCP
mcp__github__get_issue(owner: "dahendel", repo: "contextd", issue_number: <issue-number>)

// Assign to yourself using GitHub MCP
mcp__github__update_issue(owner: "dahendel", repo: "contextd", issue_number: <issue-number>, assignees: ["@me"])

// Create feature branch
git checkout -b feature/<issue-number>-<description>
```

**IMPORTANT: Keep Feature Branches Updated**
- Feature branches can become stale and have outdated GitHub Actions workflows
- **ALWAYS rebase against main** before starting work and regularly during development
- Workflow failures with YAML syntax errors often indicate outdated workflow files
- To update: `git checkout feature-branch && git rebase origin/main`

### 2. Check for Specifications

**Before implementing any feature:**

1. **Check if spec exists** in `docs/specs/<feature-or-package>/SPEC.md`
2. **If spec is missing**:
   - Run `/create-spec-issue <feature-name>`
   - Have spec-writer agent create specification
   - Save to `docs/specs/<feature-or-package>/SPEC.md`
   - Research/decisions go in same directory
3. **If spec exists**:
   - Read `SPEC.md` and understand requirements
   - Review related research/decision documents
   - Follow architectural decisions

### 3. Open Draft PR Immediately

```javascript
// Create draft PR using GitHub MCP (before implementation!)
mcp__github__create_pull_request(
  owner: "dahendel",
  repo: "contextd",
  title: "feat: <description>",
  head: "feature/<issue-number>-<description>",
  base: "main",
  draft: true,
  body: `## WIP - Research and Implementation

**Issue**: Closes #<issue-number>

## Research Phase
- [ ] SDK/library research complete
- [ ] Implementation strategy documented
- [ ] Architectural decisions documented

## Implementation Phase
- [ ] Core implementation complete
- [ ] Tests written (>80% coverage)
- [ ] Documentation updated
- [ ] Code review passed

## Status
Currently in research phase

---
Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>`
)
```

**Why Draft PR First?**
- Early visibility into work in progress
- CI/CD checks start running
- Enables early feedback and discussion
- Tracks all research and decisions

### 4. Research Phase (SDK-First, MANDATORY)

**SDK Research is MANDATORY before custom code**

See [RESEARCH-FIRST-POLICY.md](RESEARCH-FIRST-POLICY.md) for full policy.

**Quick Process**:
1. Search SDKs: GitHub, pkg.go.dev, awesome-go
2. Evaluate top 3-5: stars, docs, maintenance, license
3. Document: Use `docs/research/TEMPLATE-SDK-RESEARCH.md`
4. Decide: SDK vs custom (justify custom in ADR)

### 5. Implementation Phase

**Delegate to golang-pro for Go implementation:**

```
Use the golang-pro skill to implement [feature based on research findings]
```

**Implementation Checklist**:
- [ ] Follow researched architecture
- [ ] Use recommended SDKs/libraries
- [ ] Implement core functionality
- [ ] Add comprehensive tests (>80% coverage)
- [ ] Add error handling and validation
- [ ] Add OpenTelemetry instrumentation
- [ ] Update documentation
- [ ] Add examples if applicable

### 6. Pull Request Code Review Loop

**CRITICAL: ALWAYS follow this workflow**:

1. **Start Work - Update Issue Status**
   - Comment on issue: "Started working on this issue"
   - Update issue status to "In Progress"

2. **Create Pull Request**
   - Link to issue (use "Closes #123")
   - Include comprehensive description and test coverage
   - **Update issue**: Comment with PR link

3. **Wait for Code Review Workflow**
   - CI/CD runs automated code review
   - **Update issue**: Comment with code review status

4. **Remediate Findings**
   - Address all issues identified
   - Push changes to PR branch
   - **Update issue**: Comment on each remediation cycle

5. **Repeat Until Approved**
   - Continue remediating findings
   - **Update issue**: Keep updated with progress
   - **ONLY proceed when Status: APPROVED**

6. **Merge and Close**
   - Merge PR (squash recommended)
   - **Update issue**: Add final comment
   - Issue auto-closes

---

## Pre-commit Hooks (MANDATORY)

**CRITICAL: Pre-commit hooks MUST be installed and ALL errors MUST be resolved before pushing.**

### Installation (Required for All Development)

```bash
# Install pre-commit hooks (run once per machine)
./scripts/setup-pre-commit.sh
```

### Mandatory Pre-commit Policy

1. **NEVER use `git commit --no-verify`**
   - This bypasses security scans (gosec) and quality checks
   - Violations may be caught in CI and block PR merging
   - Only exception: Emergency hotfixes with explicit approval

2. **If pre-commit is not installed: Install it immediately**
   ```bash
   # Check if installed
   pre-commit --version

   # If not installed, run setup script
   ./scripts/setup-pre-commit.sh
   ```

3. **All pre-commit errors MUST be resolved before pushing**
   - Fix formatting issues: Run `gofmt -w .` and `goimports -w .`
   - Fix linting issues: Address `golangci-lint` warnings
   - Fix security issues: Address `gosec` findings
   - Fix commit message: Follow conventional commits format

4. **Pre-commit checks include:**
   - Secret detection (TruffleHog) - detects hardcoded credentials
   - Security scanning (gosec) - Go security vulnerabilities
   - Go formatting (gofmt, goimports)
   - Go linting (go vet, golangci-lint)
   - YAML/Markdown linting
   - Commit message validation
   - File quality checks

### What to Do When Pre-commit Fails

```bash
# 1. Read the error message carefully
git commit -m "feat: Your change"
# [ERROR] gosec found security issue in file.go:123

# 2. Fix the issue
# Edit the file and address the security/quality issue

# 3. Stage the fix
git add file.go

# 4. Commit again (hooks run automatically)
git commit -m "feat: Your change"
# [PASSED] All checks passed!

# 5. Push
git push
```

### Emergency Bypass (Requires Justification)

Only use `--no-verify` in true emergencies with explicit justification:

```bash
# NOT RECOMMENDED - Only for critical hotfixes
git commit --no-verify -m "hotfix: Critical security patch

Emergency bypass reason: Production outage, security patch needed immediately.
Will create follow-up PR to address pre-commit findings.

Approved-by: [Name]"
```

See [DEVELOPMENT-SETUP.md](DEVELOPMENT-SETUP.md) for detailed pre-commit documentation.

---

## Verification & Completion Policy

**See**: [VERIFICATION-POLICY.md](VERIFICATION-POLICY.md) for complete policy.

**Quick Reference**:
- **Major tasks** (features, bugs, refactoring, multi-file) → Use `contextd:completing-major-task` skill
- **Minor tasks** (typos, comments, single-file cosmetic) → Use `contextd:completing-minor-task` skill
- **Before PR** → Use `contextd:code-review` skill

**Key Rule**: No task marked complete without verification evidence.

---

## Pre-PR Checklist (For Developers)

**CRITICAL: Complete these checks BEFORE requesting code review.**

### 0. Pre-commit Verification (FIRST!)
- [ ] Pre-commit hooks installed: `pre-commit --version`
- [ ] All pre-commit checks pass: `pre-commit run --all-files`
- [ ] No `--no-verify` used in commit history

### 1. Build & Test Verification
- [ ] Code builds: `go build ./...`
- [ ] All tests pass: `go test ./...`
- [ ] No race conditions: `go test -race ./...`
- [ ] Test coverage ≥ 80%: `go test -coverprofile=coverage.out ./...`

### 2. Code Quality Verification
- [ ] Code formatted: `gofmt -w .`
- [ ] Linting passes: `golint ./...`
- [ ] Vet passes: `go vet ./...`
- [ ] Static analysis: `staticcheck ./...`

### 3. Documentation & Standards
- [ ] CHANGELOG.md updated (Added/Fixed/Changed section)
- [ ] Relevant specs read from `docs/specs/<feature>/SPEC.md`
- [ ] Code follows naming conventions
- [ ] Tests written first (TDD)
- [ ] Errors properly handled and wrapped
- [ ] No credentials in code

### 4. Completion Verification
- [ ] **Major tasks**: Invoked `contextd:completing-major-task` with complete template
- [ ] **Minor tasks**: Invoked `contextd:completing-minor-task` with checklist
- [ ] All verification evidence provided

### Quick Pre-PR Verification Script

```bash
go build ./... && \
go test ./... -coverprofile=coverage.out && \
go test -race ./... && \
gofmt -w . && \
golint ./... && \
go vet ./... && \
staticcheck ./... && \
echo "All checks passed! Ready for code review."
```

Or use: `./scripts/pre-pr.sh`

---

## Code Review

**See**: [CODE-REVIEW-CHECKLIST.md](CODE-REVIEW-CHECKLIST.md) for complete reviewer checklist.

**When to Request Code Review**:
1. After completing pre-PR checklist above
2. Invoke `contextd:code-review` skill
3. Code-reviewer validates all work using comprehensive checklist
4. Address findings and repeat until APPROVED
