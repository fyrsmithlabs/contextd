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

| Task Type | Agent | Command Pattern |
|-----------|-------|-----------------|
| **Task Coordination** | **taskmaster:task-orchestrator** | **Deploy at start to analyze dependencies** |
| **Task Execution** | **taskmaster:task-executor** | **Implement specific tasks** |
| **Task Verification** | **taskmaster:task-checker** | **Verify implementation quality** |
| Overall coordination | orchestrator | See `.claude/agents/orchestrator.md` |
| Spec creation | spec-writer | "Have spec-writer agent create..." |
| Architecture | go-architect | "Have go-architect agent design..." |
| **Go Implementation** | **golang-pro** | **"Use golang-pro skill to implement..."** |
| Testing | test-engineer | "Have test-engineer agent verify..." |
| Code review | superpowers:code-reviewer | "Use superpowers:requesting-code-review" |
| Error research | research-analyst | "@agent-research-analyst search..." |
| MCP features | mcp-developer | "Use mcp-developer for..." |
| QA testing | qa-engineer | "Have qa-engineer execute test skill..." |

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
