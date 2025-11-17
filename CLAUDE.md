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

## ⚠️ CRITICAL Rules

### 1. Security First (MANDATORY)

**ALWAYS consider security implications before ANY change**:

**Security-First Checklist** (EVERY code change):
- [ ] Does this expose data across project/owner boundaries?
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
5. **Least Privilege**: Services access only their designated databases/collections per user's team membership
6. **Team Boundary Enforcement**: NEVER leak data across teams without explicit permission (shared projects only)

**Architecture Roadmap**:
- **v2.1** (Current Target): Owner-scoped isolation - [ADR-003](docs/architecture/adr/003-single-developer-multi-repo-isolation.md)
- **v2.2** (Enterprise): Team-aware with org-level sharing - [TEAM-AWARE-ARCHITECTURE-V2.2.md](docs/architecture/TEAM-AWARE-ARCHITECTURE-V2.2.md)
- **0.9.0-rc-1** (Future): Multi-org, OAuth/SSO, fine-grained ACLs

**See**:
- [docs/security/SECURITY-AUDIT-SHARED-DATABASE-2025-01-07.md](docs/security/SECURITY-AUDIT-SHARED-DATABASE-2025-01-07.md) - Known vulnerabilities
- [docs/architecture/adr/003-single-developer-multi-repo-isolation.md](docs/architecture/adr/003-single-developer-multi-repo-isolation.md) - v2.1 owner-scoping
- [docs/architecture/TEAM-AWARE-ARCHITECTURE-V2.2.md](docs/architecture/TEAM-AWARE-ARCHITECTURE-V2.2.md) - v2.2 team/org model

### 2. Superpowers & TaskMaster (MANDATORY)

**ALWAYS check Superpowers skills before ANY task** - See `~/.claude/CLAUDE.md`

**Use TaskMaster for planning** - Coordinate tasks with dependency management

**Full Guide**: [docs/guides/MULTI-AGENT-ORCHESTRATION.md](docs/guides/MULTI-AGENT-ORCHESTRATION.md)

### 3. Concurrent Execution & File Management

**ABSOLUTE RULES**:
1. ALL operations MUST be concurrent/parallel in a single message
2. **NEVER save working files to the root folder**
3. ALWAYS organize files in appropriate subdirectories
4. **USE CLAUDE CODE'S TASK TOOL** for spawning agents concurrently

**Golden Rule**: "1 MESSAGE = ALL RELATED OPERATIONS"

### 4. Task Executor Agents (MANDATORY)

**ALWAYS use task-executor agents for substantial work** - They provide autonomous, focused execution.

**Autonomous Completion Rule:**
- Task-executor agents MUST complete their assigned tasks FULLY and AUTONOMOUSLY
- **NEVER** ask for confirmation mid-execution
- **NEVER** pause to ask "should I continue?"
- **NEVER** present partial results and wait for approval
- Execute the ENTIRE task from start to finish in one go

**When to Use task-executor:**
1. **Multi-file operations** - Creating/editing multiple related files
2. **Research tasks** - Gathering information, analyzing code, writing reports
3. **Documentation work** - Writing specs, guides, or comprehensive documentation
4. **Implementation tasks** - Following implementation plans step-by-step
5. **Analysis work** - Security audits, performance analysis, code reviews
6. **Any substantial work** - If it takes >5 minutes, use task-executor

**Task-Executor Types:**
- `taskmaster:task-executor` - General implementation and execution work
- `taskmaster:task-orchestrator` - For coordinating multiple parallel tasks
- Specialized agents (security-auditor, tooling-engineer, etc.) - For domain-specific work

**Example Usage:**
```
Launch task-executor to create modular spec structure:
- Refactor main SPEC.md into index
- Create 10 feature spec files
- Ensure all cross-references work
Expected: Agent completes ALL work autonomously without mid-execution questions
```

**What NOT to do:**
```
❌ BAD: Launch task-executor, it asks "should I write the full document?"
✅ GOOD: Launch task-executor, it writes complete document start to finish
```

**Prompt Construction:**
When launching task-executor, provide:
1. Clear goal and deliverables
2. Complete context and requirements
3. Expected file structure
4. Execution steps
5. "Begin execution now" or "Execute completely and autonomously"

**Trust the Agent:**
- Task-executors are designed for autonomous work
- They will complete tasks fully
- Review results after completion, not during execution
- If task is unclear, improve the prompt, don't expect mid-execution clarification

### 5. Go Code Delegation

**ALL Go coding tasks MUST be delegated to the golang-pro skill:**

```
Use the golang-pro skill to [implement/fix/refactor] [description]
```

**Do NOT write Go code directly.** The golang-pro skill enforces TDD, ensures test coverage ≥80%, validates builds, and creates proper commits.

**CRITICAL**: When delegating to golang-pro, ALWAYS include security requirements from Section 1.

### 5a. Multi-Agent Coordination (CRITICAL)

**Specialized agents must work together on complex tasks.** Each agent has domain expertise that must be coordinated.

#### MCP Implementation Pattern (MANDATORY)

**For ANY MCP-related work, use this exact pattern:**

1. **Design Phase**: Use `mcp-developer` agent for protocol research and design
   - Research MCP specification and latest standards
   - Analyze protocol requirements and gaps
   - Design endpoint structure and request/response formats
   - Create implementation plan with protocol details
   - Output: Gap analysis, protocol spec, implementation requirements

2. **Implementation Phase**: Use `golang-pro` skill for Go code
   - Takes mcp-developer's design and requirements
   - Implements Go code following TDD (≥80% coverage)
   - Enforces security requirements from Section 1
   - Creates proper commits with tests
   - Output: Production-ready Go implementation

**Example Coordination Flow**:
```
Step 1: Deploy mcp-developer agent
"Research the MCP Streamable HTTP specification and design
the /mcp endpoint implementation for our contextd server.
Output: Implementation plan with protocol details."

Step 2: Use golang-pro skill (after mcp-developer completes)
"Use golang-pro skill to implement the /mcp endpoint following
the design from mcp-developer. Include:
- Protocol requirements: [from mcp-developer output]
- Security requirements: [from Section 1]
- Endpoint signature: POST/GET/DELETE /mcp
- Session management with Mcp-Session-Id header"
```

#### General Agent Coordination Rules

**Sequential Pattern** (when agents have dependencies):
1. **Specialist agent** (research, design, analysis) completes first
2. **Implementation agent** (golang-pro, task-executor) uses specialist output
3. **Review agent** (code-reviewer, task-checker) validates result

**Parallel Pattern** (when agents work independently):
- Deploy multiple agents in single message with Task tool
- Each agent works on independent subtask
- Coordinate results after all complete

**Key Principles**:
- **Never skip specialist agents** - Their domain expertise prevents costly mistakes
- **Always pass context forward** - Include specialist output when delegating to implementers
- **One agent per domain** - Don't ask golang-pro to research MCP specs or mcp-developer to write Go
- **Maintain security context** - ALWAYS include Section 1 security requirements when delegating

**See**: [docs/guides/MULTI-AGENT-ORCHESTRATION.md](docs/guides/MULTI-AGENT-ORCHESTRATION.md) for complete delegation table

### 6. GitHub Integration

**ALWAYS prefer GitHub MCP tools over gh CLI** when available.

**Tool Selection Priority**:
1. **GitHub MCP tools** (mcp__github__*) - Use FIRST
2. **gh CLI** - Fallback ONLY when MCP doesn't support the feature

---

## Project Overview

`contextd` is a Go-based API service for Claude Code user-level management, built with **security and context optimization as the primary goal**. The system uses Unix domain sockets for security, Qdrant for vector storage, and OpenTelemetry for observability.

**Core Philosophy**: **Minimize context bloat, maximize token efficiency**, local-first operations with background sync.

**PRIMARY GOALS** (in order):
1. **Context Optimization**: Minimize token usage (target: <3K tokens per search vs 12K in v2.0)
2. **Security**: Multi-tenant isolation, no data leakage
3. **Performance**: <100ms search latency

**Context Efficiency Mandate**:
- Every feature MUST reduce context usage or maintain neutrality
- Search results MUST be scoped (no irrelevant cross-project pollution)
- Track context metrics: tokens per operation, relevance ratio, deduplication rate
- Target: 5x context reduction (v2.0 → v2.1)

**See**: [docs/architecture/CONTEXT-USAGE-ESTIMATES.md](docs/architecture/CONTEXT-USAGE-ESTIMATES.md) for detailed context optimization strategies

**Development Philosophy**:
- **Superpowers-First**: ALWAYS check superpowers skills before ANY task (mandatory)
- **TaskMaster Planning**: Use TaskMaster for task prioritization and workflow coordination
- **YAGNI (You Aren't Gonna Need It)**: Build only what's needed now, not what might be needed later. Ruthlessly eliminate speculative features, abstractions, and complexity. Every feature must solve a current, concrete problem.
- **Interface Design**: Design minimal, focused interfaces before implementation. Interfaces should represent real abstractions, not premature generalization. Prefer concrete implementations over complex interface hierarchies.
- **Test-Driven Development (TDD)**: Mandatory ≥80% coverage. Write tests first (Red), implement to pass (Green), then refactor. Tests validate behavior, not implementation details.
- **Standards-First Development**: Reference `docs/standards/` before any implementation
- **Spec-Driven Development**: Check `docs/specs/` for feature specifications
- **Research-First Development**: All significant changes must be researched and documented
- **Skill-First Testing**: Every feature requires test skill, every bug requires regression test

**Workflow**: Superpowers Check → TaskMaster Planning → Research → Review → Refine → Approve → Test (Red) → Implement (Green) → Refactor → Create Test Skill

**Documentation Policy**:
- **NEVER create summary, implementation, or completion documentation files unless explicitly requested**
- Only create documentation when: (1) User explicitly asks, (2) Required deliverable (ADRs, research docs), (3) Required by Development Philosophy
- Focus on code changes, not change documentation artifacts

---

## Quick Reference Guides

**Development Workflow**: [docs/guides/DEVELOPMENT-WORKFLOW.md](docs/guides/DEVELOPMENT-WORKFLOW.md)
- Spec-driven development workflow
- Issue selection and PR workflow
- Pre-commit hooks (MANDATORY)
- Code review checklist

**Multi-Agent Orchestration**: [docs/guides/MULTI-AGENT-ORCHESTRATION.md](docs/guides/MULTI-AGENT-ORCHESTRATION.md)
- Superpowers & TaskMaster integration
- Agent delegation table
- Available slash commands

**Product Roadmap**: [docs/PRODUCT-ROADMAP-V3-AGENT-PATTERNS.md](docs/PRODUCT-ROADMAP-V3-AGENT-PATTERNS.md)
- Complete 10-month roadmap
- Current Phase: Phase 1 - Foundation for Pattern Integration (Months 1-2)

**Build & Run**: [docs/guides/GETTING-STARTED.md](docs/guides/GETTING-STARTED.md)
- Installation and setup
- Running contextd
- MCP integration
- Testing

**Monitoring**: [docs/guides/MONITORING-SETUP.md](docs/guides/MONITORING-SETUP.md)
- Grafana dashboards
- Metrics and traces
- Performance monitoring

**Release Process**: [docs/guides/RELEASE-WORKFLOW.md](docs/guides/RELEASE-WORKFLOW.md)
- GoReleaser configuration
- Release types and assets

**Logging and Security**: [docs/specs/logging/SPEC.md](docs/specs/logging/SPEC.md)
- Uber Zap structured logging
- Gitleaks secret scanning (800+ patterns)
- Claude Code hook integration
- 5 layers of defense architecture
- MCP middleware and HTTP interceptor

---

## Documentation Structure

```
docs/
├── standards/              # Template-wide coding standards (REFERENCE THESE)
│   ├── architecture.md     # Architecture patterns
│   ├── coding-standards.md # Go coding standards
│   ├── testing-standards.md # TDD requirements
│   └── package-guidelines.md # Package documentation template
│
├── specs/                  # Project-specific feature specs (CREATE THESE)
│   ├── logging/           # Logging and security spec
│   │   └── SPEC.md         # Comprehensive logging specification
│   └── <feature-or-package>/  # One directory per feature/package
│       ├── SPEC.md         # Main specification document
│       ├── research/       # Research documents
│       ├── decisions/      # Decision documents
│       └── resolutions/    # Error resolution docs
│
├── guides/                 # How-to documentation
│   ├── GETTING-STARTED.md
│   ├── DEVELOPMENT-WORKFLOW.md
│   ├── MULTI-AGENT-ORCHESTRATION.md
│   ├── RELEASE-WORKFLOW.md
│   ├── MONITORING-SETUP.md
│   └── TEI-DEPLOYMENT.md
│
├── architecture/           # Architecture docs
│   ├── adr/               # Architecture Decision Records
│   └── research/          # Research documents
│
└── testing/               # Testing documentation
    ├── RESEARCH-FIRST-TDD-WORKFLOW.md
    └── regression/        # Bug tracking
```

---

## Project Context Management (CRITICAL)

**When working on this project (contextd), use contextd's own tools for context management.**

### Maintaining Project Context

**Priority Order** (for contextd project specifically):

1. **Use Contextd MCP Tools** (FIRST CHOICE - dogfooding our own product):
   ```bash
   # Index the repository once (or after major changes)
   mcp__contextd__index_repository path="$(pwd)"

   # Search for relevant context
   mcp__contextd__checkpoint_search query="configuration management" top_k=5

   # Save checkpoints during work
   mcp__contextd__checkpoint_save
     summary="Working on YAML config improvements"
     project_path="$(pwd)"
     tags=["config", "yaml"]
   ```

2. **Direct File Reading** (when you know specific files):
   ```bash
   Read(pkg/config/config.go)
   Read(CLAUDE.md)
   Grep(pattern="LoadConfig", path="pkg/")
   Glob(pattern="**/*config*.go")
   ```

3. **Auto-Checkpoint System** (session management):
   ```bash
   /auto-checkpoint          # Manual checkpoint save
   /context-check            # Check context usage
   ```

**DO NOT use Task tool with Explore agent** for this project:
- ❌ Higher context cost (defeats our PRIMARY goal)
- ❌ Slower than contextd queries
- ❌ Doesn't demonstrate our product
- ✅ Use contextd tools instead (we're the context optimization product!)

**Why This Matters**:
- **Dogfooding**: We use our own product to validate it works
- **Context Efficiency**: Aligns with PRIMARY goal (minimize token usage)
- **Performance**: <100ms search vs agent exploration overhead
- **Validation**: Real-world testing of contextd features

---

## Contextd-Specific Quick Reference

### Build & Run Commands

```bash
# Build binaries
go build -o contextd ./cmd/contextd/
go build -o ctxd ./cmd/ctxd/

# Install and setup
./ctxd install
./ctxd setup-claude

# Check status
systemctl --user status contextd  # Linux
sudo launchctl list com.axyzlabs.contextd  # macOS

# View logs
journalctl --user -u contextd -f  # Linux
tail -f /tmp/contextd.log  # macOS
```

### MCP Integration

contextd provides 9 MCP tools for Claude Code. See [docs/guides/GETTING-STARTED.md](docs/guides/GETTING-STARTED.md) for setup.

**IMPORTANT**: MCP tools connect directly to Qdrant, NOT to the contextd service.
- **DO NOT troubleshoot contextd systemd service** when MCP tools fail
- The contextd service exists ONLY for the `ctxd` CLI client
- MCP tool failures indicate issues with:
  - Qdrant connectivity (check `docker ps` for Qdrant container)
  - MCP server configuration (check `.mcp.json` and `~/.claude.json`)
  - Environment variables (API keys, Qdrant URL)
  - NOT the contextd systemd service

### Multi-Tenant Architecture

**Status**: Complete - Multi-tenant mode is now the ONLY supported mode (v2.0.0+)

- Database-per-project physical isolation
- Project databases: `project_<hash>` (SHA256 of project_path)
- Shared database for global knowledge (remediations, skills)
- Filter injection attacks eliminated
- 10-16x faster queries (partition pruning)

**Documentation**: [docs/adr/002-universal-multi-tenant-architecture.md](docs/adr/002-universal-multi-tenant-architecture.md)

### Embedding Options

- **TEI (Text Embeddings Inference)** - Recommended (no API costs, local)
- **OpenAI API** - Alternative ($0.02 per 1M tokens)

See [docs/guides/TEI-DEPLOYMENT.md](docs/guides/TEI-DEPLOYMENT.md)

---

## Changelog Maintenance

**CRITICAL**: CHANGELOG.md MUST be updated for every feature, bug fix, and change.

See [CHANGELOG.md](CHANGELOG.md) for format and examples.

**When to Update**:
- **For Every Feature**: Add under `### Added` in `[Unreleased]`
- **For Every Bug Fix**: Add under `### Fixed` in `[Unreleased]`
- **For Breaking Changes**: Add under `### Changed` with **BREAKING** marker

---

## Development Checklist

### For Every Feature
- [ ] Research completed and documented
- [ ] Spec exists or created in `docs/specs/<feature>/SPEC.md`
- [ ] Related research/decisions documented in same directory
- [ ] Design reviewed and approved
- [ ] Tests written (TDD red phase)
- [ ] Implementation complete (TDD green)
- [ ] Code refactored (TDD refactor)
- [ ] **Test skill created**
- [ ] **Persona agents executed tests**
- [ ] Code review passed
- [ ] Documentation updated
- [ ] **CHANGELOG.md updated** (Added section under [Unreleased])
- [ ] Committed to repository

### For Every Bug Fix
- [ ] Bug reproduced and documented
- [ ] **Bug record created** (tests/regression/bugs/)
- [ ] **Regression test created**
- [ ] Root cause identified
- [ ] Fix implemented
- [ ] Regression test passes
- [ ] No new bugs introduced
- [ ] Code review passed
- [ ] Bug record updated (commit hash, status)
- [ ] **CHANGELOG.md updated** (Fixed section under [Unreleased])
- [ ] Committed with regression test

---

## Summary

**Before writing any code:**

1. **MANDATORY**: Check `superpowers:using-superpowers` skill first
2. **Use TaskMaster for planning**:
   - `/tm:init` (if not initialized)
   - `/tm:parse-prd` or `/tm:next` for task selection
   - Deploy `taskmaster:task-orchestrator` for dependency analysis
3. **Use Superpowers workflows**:
   - Use `superpowers:brainstorming` before coding (or `/superpowers:brainstorm`)
   - Use `superpowers:writing-plans` for implementation plans
   - Use `superpowers:executing-plans` for execution
4. Read applicable standards from `docs/standards/`
5. Read applicable specs from `docs/specs/<feature>/SPEC.md`
6. Review related research/decisions in the spec directory
7. Check architecture docs in `docs/architecture/`
8. Check package CLAUDE.md (if exists)
9. Write tests first (TDD using `superpowers:test-driven-development`)
10. Delegate Go code to golang-pro skill
11. Run tests and verify coverage ≥80%
12. Run linters and fix issues
13. Use `superpowers:requesting-code-review` when done
14. Follow Pull Request code review loop until approved

**Remember**:
- **ALWAYS check Superpowers skills before ANY task** (mandatory)
- **Use TaskMaster for task prioritization and coordination**
- **Use task-executor for substantial work** (multi-file ops, docs, implementation)
- **Task-executors complete work autonomously** - no mid-execution confirmations
- Standards and specs are the source of truth
- Always delegate Go code to golang-pro skill
- Never skip the code review loop
- Keep issue updated throughout workflow
- Security and context efficiency are PRIMARY goals
- Never say the project is production ready

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
