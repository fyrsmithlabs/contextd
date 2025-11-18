# Multi-Agent Orchestration Guide

**See [../../CLAUDE.md](../../CLAUDE.md) for project overview.**

## CRITICAL: Superpowers & TaskMaster Integration

**MANDATORY First Steps**:
1. **Check Superpowers**: ALWAYS use `superpowers:using-superpowers` skill before ANY task
2. **Use TaskMaster for Planning**: Leverage TaskMaster agents for task coordination

## TaskMaster Workflow

```bash
# 1. Initialize Task Master (if not already done)
/tm:init

# 2. Parse requirements into tasks
/tm:parse-prd <requirements-file>

# 3. Analyze and expand complex tasks
/tm:analyze-complexity
/tm:expand <task-id>

# 4. Get next recommended task
/tm:next

# 5. Execute with task-executor agent
# Use Task tool with taskmaster:task-executor agent

# 6. Verify with task-checker agent
# Use Task tool with taskmaster:task-checker agent
```

## Superpowers Workflow

- **Before coding**: Use `superpowers:brainstorming` skill (or `/superpowers:brainstorm`)
- **For implementation**: Use `superpowers:writing-plans` (or `/superpowers:write-plan`)
- **For execution**: Use `superpowers:executing-plans` (or `/superpowers:execute-plan`)
- **After completion**: Use `superpowers:requesting-code-review` skill
- **On review feedback**: Use `superpowers:receiving-code-review` skill

## Agent Delegation

| Task Type | Agent/Skill | Command Pattern | Notes |
|-----------|-------------|-----------------|-------|
| **Task Coordination** | **taskmaster:task-orchestrator** | **Deploy at start to analyze dependencies** | Plans parallel/sequential work |
| **Task Execution** | **taskmaster:task-executor** | **Implement specific tasks** | Autonomous implementation |
| **Task Verification** | **taskmaster:task-checker** | **Verify implementation quality** | Quality gates |
| Overall coordination | orchestrator | See `.claude/agents/orchestrator.md` | Session-level orchestration |
| Spec creation | spec-writer | "Have spec-writer agent create..." | Write specifications |
| Architecture | go-architect | "Have go-architect agent design..." | System design |
| **Go Implementation** | **golang-pro** | **"Use golang-pro skill to implement..."** | **TDD, ≥80% coverage, security** |
| Testing strategy | test-strategist | "Use test-strategist to design..." | Test plan design |
| Code review | superpowers:code-reviewer | "Use superpowers:requesting-code-review" | Post-implementation review |
| Error research | research-analyst | "@agent-research-analyst search..." | Root cause research |
| **MCP Protocol Design** | **mcp-developer** | **"Use mcp-developer to research/design..."** | **Protocol spec, gap analysis** |
| QA testing | qa-engineer | "Have qa-engineer execute test skill..." | End-to-end testing |

### Contextd Project Skills (MANDATORY)

| Skill | When to Use | Purpose |
|-------|-------------|---------|
| **contextd:completing-major-task** | Before marking major tasks complete | Enforces comprehensive verification template (build, tests, security, functionality) |
| **contextd:completing-minor-task** | Before marking minor tasks complete | Enforces self-interrogation checklist (what changed, how verified, what breaks) |
| **contextd:code-review** | Before creating PR | Comprehensive code review with structured output (APPROVED/CHANGES REQUIRED/BLOCKED) |
| **contextd:creating-package** | Before creating new package | Package creation workflow + updates pkg/CLAUDE.md + updates category skill |
| **contextd:creating-spec** | Before implementing feature without spec | Creates SPEC.md with approval workflow, blocks implementation until Status: Approved |
| **contextd:pkg-security** | Working on auth, session, isolation, rbac packages | Multi-tenant isolation, input validation, security testing patterns |
| **contextd:pkg-storage** | Working on checkpoint, remediation, cache packages | Qdrant patterns, database-per-project, query security |
| **contextd:pkg-core** | Working on config, telemetry, logging packages | Standard patterns, error handling, initialization |
| **contextd:pkg-api** | Working on mcp, handlers, middleware packages | Request/response, validation, MCP tools |
| **contextd:pkg-ai** | Working on embedding, search, semantic packages | Embeddings, vector operations, AI integrations |
| **contextd:planning-with-verification** | Creating todos for major work | Adds verification subtasks automatically to TodoWrite |
| **contextd:security-check** | For security-critical changes | Deep security validation (multi-tenant, input validation) |
| **contextd:pre-pr-verification** | Before requesting code review | Pre-PR comprehensive check |
| **kinney-documentation** | Writing ANY documentation | Enforces scannable (~150 lines), modular (@imports), noun-heavy approach |

**Mandatory Usage Rules**:
- ❌ **NO task completion without verification skill** - Must use completing-major-task or completing-minor-task
- ❌ **NO code without spec** - Must use creating-spec if spec missing, ensure Status: Approved before coding
- ❌ **NO PR without code review** - Must use code-review before creating PR
- ❌ **NO package creation without workflow** - Must use creating-package before creating packages
- ❌ **NO documentation without kinney-documentation** - Must use for all CLAUDE.md, specs, guides, READMEs

**See**: [VERIFICATION-POLICY.md](VERIFICATION-POLICY.md) for complete verification requirements

### Multi-Agent Coordination Patterns

#### MCP Implementation Pattern (MANDATORY)

**For ANY MCP-related work:**

```
Phase 1: Design (mcp-developer agent)
└─> Research MCP spec, analyze gaps, design solution
    Output: Gap analysis + implementation requirements

Phase 2: Implementation (golang-pro skill)
└─> Implement Go code following mcp-developer's design
    Output: Production-ready code with tests (≥80% coverage)

Phase 3: Verification (code-reviewer)
└─> Validate against requirements and protocol compliance
    Output: Approval or revision requests
```

**Example**:
```
Step 1: "Use mcp-developer agent to research MCP Streamable HTTP
        specification and design /mcp endpoint implementation"

Step 2: "Use golang-pro skill to implement /mcp endpoint with:
        - Protocol requirements from mcp-developer
        - Security requirements from CLAUDE.md Section 1
        - TDD with ≥80% test coverage"

Step 3: "Use superpowers:requesting-code-review to verify implementation"
```

#### Other Common Patterns

**Security-Critical Feature**:
```
security-auditor → golang-pro → code-reviewer
(audit) → (implement with mitigations) → (verify)
```

**Performance Optimization**:
```
performance-engineer → golang-pro → test-strategist
(profile/analyze) → (implement fixes) → (benchmark tests)
```

**Documentation Generation**:
```
documentation-engineer → task-executor → code-reviewer
(design docs structure) → (generate content) → (review accuracy)
```

## Available Slash Commands

### Superpowers Commands (MANDATORY workflows)
- `/superpowers:brainstorm` - Interactive design refinement before coding
- `/superpowers:write-plan` - Create detailed implementation plan
- `/superpowers:execute-plan` - Execute plan in batches with review checkpoints

### TaskMaster Commands (Task Prioritization & Planning)
- `/tm:init` - Initialize Task Master project
- `/tm:parse-prd <file>` - Generate tasks from requirements
- `/tm:next` - Get next recommended task based on dependencies
- `/tm:list [filters]` - List tasks with natural language filters
- `/tm:expand <id>` - Break down complex task into subtasks
- `/tm:analyze-complexity` - AI complexity analysis for all tasks
- `/tm:set-status/to-in-progress <id>` - Mark task as in progress
- `/tm:set-status/to-done <id>` - Mark task as complete
- `/tm:set-status/to-review <id>` - Mark task for review
- See full list: `~/.claude/plugins/marketplaces/taskmaster/.claude/TM_COMMANDS_GUIDE.md`

### Repository Management
- `/create-repo <name> <description>` - Create private repository
- `/configure-repo [phase]` - Configure repository

### Specification & Planning
- `/create-spec-issue <name>` - Create issue for missing spec
- `/spec-to-issue <name>` - Convert spec to GitHub issues
- `/review-priorities [timeframe]` - Review and update priorities

### Development Workflow
- `/start-task <issue-number>` - Initialize development environment
- `/run-quality-gates [scope]` - Execute quality checks
- `/check-dependencies [--update]` - Vulnerability scanning
- `/debug-issue "<description>"` - AI-assisted debugging

### Contextd-Specific
- `/checkpoint save "summary"` - Quick checkpoint
- `/checkpoint search "query"` - Find past work
- `/remediation search "error"` - Find error solutions
- `/troubleshoot "error message"` - AI diagnosis
- `/index repository path=/repo` - Index repository
- `/status` - Check contextd service status

See `.claude/commands/` for all available commands.
