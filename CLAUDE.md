# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 📚 Hierarchical Documentation Structure

**CRITICAL**: This repository uses a hierarchical CLAUDE.md system. Always read documentation in this order:

1. **Root** (`CLAUDE.md`) - This file: Project-wide policies, development workflow, critical rules
2. **Package Guidelines** (`pkg/CLAUDE.md`) - Package architecture, design patterns, dependencies
3. **Package-Specific** (`pkg/<package>/CLAUDE.md`) - Individual package documentation with spec references
4. **Specifications** (`docs/specs/<feature>/SPEC.md`) - Detailed feature specifications
5. **Standards** (`docs/standards/*.md`) - Coding standards, testing requirements, architecture patterns

**When to Read What**:
- **Starting work on any task**: Read this file (CLAUDE.md)
- **Working with packages**: Read `pkg/CLAUDE.md` + specific `pkg/<package>/CLAUDE.md`
- **Implementing features**: Read referenced `docs/specs/<feature>/SPEC.md`
- **Code review**: Read standards in `docs/standards/`
- **Architecture decisions**: Read `docs/architecture/adr/`

---

## 🔄 Maintenance Guidelines

**Update this file when:**
- [ ] Adding new major dependencies
- [ ] Changing architectural patterns
- [ ] Modifying directory structure
- [ ] Adding new environment variables
- [ ] Changing API response formats
- [ ] Implementing new testing patterns
- [ ] Discovering performance bottlenecks
- [ ] Making security changes

**Last Updated:** 2025-11-18 | **Version:** 1.0.0-alpha

---

## 🎯 Core Principles (Foundational)

### What This Project IS

`contextd` is a Go-based API service for Claude Code context management, built with **security and context optimization as primary goals**. The system uses HTTP/MCP protocol, Qdrant for vector storage, and OpenTelemetry for observability.

**PRIMARY GOALS** (in order):
1. **Context Optimization**: Minimize token usage (target: <3K tokens per search)
2. **Security**: Multi-tenant isolation, no data leakage
3. **Performance**: <100ms search latency

### Development Philosophy

1. **Security First** - Multi-tenant isolation, input validation, defense in depth (EVERY code change)
2. **Evidence-Based Completion** - No task marked complete without verification proof
3. **YAGNI** - Build only what's needed now; ruthlessly eliminate speculation
4. **TDD** - Mandatory ≥80% coverage (Red → Green → Refactor)
5. **Interface-Driven** - Design minimal interfaces before implementation
6. **Standards-First** - Reference `docs/standards/` before implementation
7. **Spec-Driven** - Check `docs/specs/` for feature specifications
8. **Context Efficiency** - Every feature must reduce token usage or maintain neutrality
9. **Skill Maintenance** - Skills evolve with codebase, must stay current
10. **Spec-Driven Development** - NO CODE WITHOUT SPEC (non-negotiable)
11. **Modular Documentation** - Scannable main files (~150 lines), @imports for details (Kinney approach)

**Version**: v1.0.0-alpha (Pre-release) | **Status**: Actively developed prototype

---

## 🔧 Architecture & Standards (Foundational)

@docs/standards/architecture.md
@docs/standards/coding-standards.md
@docs/standards/testing-standards.md

**Key Technologies:**
- **Language**: Go 1.23+
- **Storage**: Qdrant (vector database)
- **Protocol**: HTTP/MCP (Model Context Protocol)
- **Observability**: OpenTelemetry (Jaeger, Prometheus, Grafana)
- **Embeddings**: TEI (Text Embeddings Inference) or OpenAI API

**Architecture Highlights:**
- HTTP server on port 8080 (remote access supported, configurable host/port)
- Database-per-project physical isolation (`project_<hash>`)
- Multi-tenant mode ONLY (no single-tenant option)
- Multiple concurrent Claude Code sessions supported
- 10-16x faster queries via partition pruning

See: [docs/architecture/adr/](docs/architecture/adr/) for architectural decisions

---

## 🚦 Development Workflow (Operational)

@docs/guides/DEVELOPMENT-WORKFLOW.md

**Quick Workflow**: Superpowers Check → TaskMaster Planning → Research → TDD (Red → Green → Refactor) → Verify → Code Review

**Key Rules**:
1. **Superpowers-First**: ALWAYS check `superpowers:using-superpowers` before ANY task
2. **TaskMaster Planning**: Use TaskMaster for task prioritization and coordination
3. **Pre-commit hooks**: MANDATORY - NEVER use `git commit --no-verify`
4. **Spec-Driven**: Check for specs in `docs/specs/<feature>/SPEC.md` before implementing
5. **Research-First**: SDK research mandatory before custom implementations

**Workflow Guides**:
- **Multi-Agent Orchestration**: [docs/guides/MULTI-AGENT-ORCHESTRATION.md](docs/guides/MULTI-AGENT-ORCHESTRATION.md)
- **Build & Run**: [docs/guides/GETTING-STARTED.md](docs/guides/GETTING-STARTED.md)
- **Monitoring**: [docs/guides/MONITORING-SETUP.md](docs/guides/MONITORING-SETUP.md)
- **Release Process**: [docs/guides/RELEASE-WORKFLOW.md](docs/guides/RELEASE-WORKFLOW.md)

---

## ⚠️ CRITICAL: Security First (MANDATORY)

**ALWAYS consider security implications before ANY change.**

**Security-First Checklist** (EVERY code change):
- [ ] Does this expose data across project/owner/team boundaries?
- [ ] Are all user inputs validated and sanitized?
- [ ] Is sensitive data encrypted/redacted?
- [ ] Are there access control checks?
- [ ] Does this maintain multi-tenant isolation?
- [ ] Could this cause compliance violations (GDPR, HIPAA, SOC 2)?

**Critical Security Principles**:
1. **Multi-Tenant Isolation**: Project (`project_<hash>`) vs Team (`team_<name>`) vs Org (`org_<name>`) scopes MUST remain orthogonal
2. **Data Segregation Hierarchy**:
   - Checkpoints: Private to project ONLY
   - Remediations/Skills/Troubleshooting: Shared within team, optionally org-wide
   - Search: project → team → org → public (never cross-team without permission)
3. **Input Validation**: ALL user inputs sanitized (file paths, git URLs, search queries, filter expressions, team names, org names)
4. **Defense in Depth**: Database boundaries + application-layer checks + type safety + RBAC
5. **Team Boundary Enforcement**: NEVER leak data across teams without explicit permission

**See**:
- [docs/security/SECURITY-AUDIT-SHARED-DATABASE-2025-01-07.md](docs/security/SECURITY-AUDIT-SHARED-DATABASE-2025-01-07.md)
- [docs/architecture/adr/003-single-developer-multi-repo-isolation.md](docs/architecture/adr/003-single-developer-multi-repo-isolation.md)

---

## ✅ Verification & Completion Policy (MANDATORY)

@docs/guides/VERIFICATION-POLICY.md

**No task can be marked complete without verification evidence.**

### Mandatory Completion Skills

**For Major Tasks** (features, bug fixes, refactoring, security, multi-file changes):
- **MUST** invoke: `contextd:completing-major-task`
- Requires: Build output, test results + coverage, security validation, functionality verification

**For Minor Tasks** (typos, comments, formatting, single-file edits):
- **MUST** invoke: `contextd:completing-minor-task`
- Requires: Self-interrogation checklist (what changed, how verified, what breaks if wrong)

### Code Review

**Before PR creation**:
- **MUST** invoke: `contextd:code-review`
- Validates: Verification evidence, security compliance, test coverage ≥80%, standards adherence

**Code Review Checklist**: @docs/guides/CODE-REVIEW-CHECKLIST.md

**Enforcement**: Code review BLOCKS merge if verification evidence missing or insufficient.

---

## 🤖 Agent Delegation (Operational)

@docs/guides/MULTI-AGENT-ORCHESTRATION.md

### Agent Coordination Rules

**Go Code (MANDATORY)**:
- **ALL Go coding tasks** → `golang-pro` skill
- Enforces: TDD, ≥80% coverage, security requirements, proper commits

**Substantial Work**:
- Multi-file operations → `taskmaster:task-executor`
- Task coordination → `taskmaster:task-orchestrator`
- Autonomous completion (no mid-execution questions)

**Specialized Work**:
- MCP protocol design → `mcp-developer` agent
- Security analysis → `security-auditor` agent
- Documentation → `documentation-engineer` agent
- Testing strategy → `test-strategist` agent

**Coordination Patterns**:
- **Sequential**: Specialist (research/design) → Implementer (golang-pro/task-executor) → Reviewer (code-review)
- **Parallel**: Deploy multiple agents in single message for independent subtasks

**Key Principle**: Never skip specialist agents - their domain expertise prevents costly mistakes.

---

## 📚 Quick References

### Build & Run Commands

```bash
# Build binaries
go build -o contextd ./cmd/contextd/
go build -o ctxd ./cmd/ctxd/

# Install and setup
./ctxd install
./ctxd setup-claude

# Test
go test ./...                    # All tests
go test -race ./...              # Race detection
go test -coverprofile=coverage.out ./...  # Coverage
```

### Context Management (Dogfooding)

**Use contextd's own MCP tools for context management** (validates our product works):

```bash
# Index repository
mcp__contextd__index_repository path="$(pwd)"

# Search for context
mcp__contextd__checkpoint_search query="auth middleware" top_k=5

# Save checkpoint
mcp__contextd__checkpoint_save summary="Verification system" project_path="$(pwd)"
```

**DO NOT use Task tool with Explore agent** - defeats PRIMARY goal (context optimization).

### Changelog Maintenance

**CRITICAL**: CHANGELOG.md MUST be updated for every feature, bug fix, and change.

- Feature → `### Added` under `[Unreleased]`
- Bug fix → `### Fixed` under `[Unreleased]`
- Breaking change → `### Changed` with **BREAKING** marker

---

## 📂 Documentation Structure

```
docs/
├── standards/              # Coding standards (REFERENCE THESE)
│   ├── architecture.md
│   ├── coding-standards.md
│   ├── testing-standards.md
│   └── package-guidelines.md
├── specs/                  # Feature specifications (CREATE THESE)
│   └── <feature>/SPEC.md
├── guides/                 # How-to documentation
│   ├── GETTING-STARTED.md
│   ├── DEVELOPMENT-WORKFLOW.md
│   ├── VERIFICATION-POLICY.md
│   ├── CODE-REVIEW-CHECKLIST.md
│   └── MULTI-AGENT-ORCHESTRATION.md
├── architecture/           # Architecture decisions
│   └── adr/
└── testing/               # Testing documentation
    └── regression/
```

---

## Summary

**Before writing any code:**

1. Check `superpowers:using-superpowers` skill
2. **MANDATORY: Check for spec** in `docs/specs/<feature>/SPEC.md`
3. **If spec missing**: Invoke `contextd:creating-spec` skill (MANDATORY)
4. **If spec exists**: Read spec, ensure Status: Approved before coding
5. Use TaskMaster for planning (`/tm:init`, `/tm:next`)
6. Read applicable standards from `docs/standards/`
7. Write tests first (TDD red phase)
8. Delegate Go code to `golang-pro` skill
9. **Before completion**: Invoke appropriate completion skill (`contextd:completing-major-task` or `contextd:completing-minor-task`)
10. **Before PR**: Invoke `contextd:code-review`
11. Follow Pull Request code review loop until APPROVED

**Before writing any documentation:**

1. **MANDATORY**: Invoke `kinney-documentation` skill (scannable, modular approach)
2. **For prose quality**: Use `elements-of-style:writing-clearly-and-concisely` skill
3. **For skill files**: ALSO use `superpowers:writing-skills` (test with subagents)
4. **Length check**: Main file ≤150 lines (preferred) or ≤200 lines (maximum)
5. **Modularity**: Use @imports for detailed content
6. **Verification**: Use kinney-documentation skill's verification template

**Before creating/modifying packages:**

1. **Creating new package**: Invoke `contextd:creating-package` skill (MANDATORY)
2. **Modifying existing package**: Invoke relevant category skill (see pkg/CLAUDE.md)
3. **If complex package**: Create spec in `docs/specs/<package>/SPEC.md`

**Remember**:
- Security and context efficiency are PRIMARY goals
- No completion without verification evidence
- Never skip code review loop
- Standards and specs are source of truth
- Never say the project is production ready

---

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
