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

| Task Type | Agent | Command Pattern | Notes |
|-----------|-------|-----------------|-------|
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
