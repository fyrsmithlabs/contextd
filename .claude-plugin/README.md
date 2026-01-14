# contextd Plugin for Claude Code

Cross-session memory and context management for AI agents.

## Features

- **Semantic Memory** - Search past learnings and strategies across sessions
- **Checkpoints** - Save and resume session context before hitting limits
- **Error Remediation** - Track and reuse solutions to errors
- **Semantic Search** - Smart code search with auto-fallback to grep
- **Context Folding** - Isolated sub-tasks with token budgets that auto-cleanup
- **Self-Reflection** - Analyze behavior patterns and improve documentation
- **Secret Scrubbing** - Automatic detection via gitleaks

## Installation

### Automated Setup (Recommended)

```bash
# 1. Install the plugin
claude plugins add fyrsmithlabs/contextd

# 2. Run setup in Claude Code
/contextd:init
```

This command automatically:
- Downloads contextd binary (or uses Docker)
- Configures MCP settings in `~/.claude/settings.json`
- Validates the setup

**Restart Claude Code and you're ready!**

### Manual Setup (Alternative)

**1. Install Binary:**
```bash
# Homebrew
brew install fyrsmithlabs/tap/contextd

# Or download from releases
# https://github.com/fyrsmithlabs/contextd/releases/latest
```

**2. Configure MCP:**
```bash
ctxd mcp install    # Auto-configure
ctxd mcp status     # Verify setup
```

Or manually edit `~/.claude/settings.json`:
```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "contextd",
      "args": ["--mcp", "--no-http"]
    }
  }
}
```

**3. Restart Claude Code**

## Commands

| Command | Description |
|---------|-------------|
| `/contextd:search` | Semantic search across codebase and memories |
| `/contextd:remember` | Quick memory record |
| `/contextd:checkpoint` | Save, list, or resume checkpoints |
| `/contextd:diagnose` | AI-powered error diagnosis |
| `/contextd:status` | Check contextd health and session state |
| `/contextd:init` | Initialize new project (use `--full` for existing projects) |
| `/contextd:reflect` | Generate self-reflection report on behavior patterns |
| `/contextd:consensus-review` | Multi-agent code/PR validation |
| `/contextd:help` | List available commands |

## Agents

| Agent | Purpose |
|-------|---------|
| `contextd-task-agent` | Unified agent for debugging, refactoring, architecture analysis, and general tasks with automatic mode detection |
| `contextd-orchestrator` | Multi-agent workflow orchestration with context-folding for parallel sub-task execution |

## Skills

| Skill | Use When |
|-------|----------|
| `using-contextd` | Canonical reference for all MCP tools and usage patterns |
| `contextd-workflow` | Pre-flight, work, and post-flight workflow protocols |
| `context-folding` | Isolated sub-task execution with token budgets |
| `project-setup` | Onboarding projects and generating CLAUDE.md files |
| `consensus-review` | Multi-agent code/PR validation workflows |
| `self-reflection` | Behavior pattern analysis and documentation improvement |

## Hooks

| Hook | Trigger | Action |
|------|---------|--------|
| SessionStart | New session begins | Check for checkpoints, offer resume |
| UserPromptSubmit | User sends prompt | Pre-flight reminder + context monitoring |
| PreCompact | Before compaction | Force checkpoint save |
| PostToolUse (Bash fail) | Bash command fails | Trigger SRE debug flow |
| Stop | Task completion | Post-flight reminder to record learnings |

## Quick Start

1. **New project**: `/contextd:init` to set up project context
2. **Existing project**: `/contextd:init --full` to prime with existing knowledge
3. **During work**: Use contextd-first search, memories auto-recorded
4. **At 70% context**: `/contextd:checkpoint` then `/clear`
5. **Next session**: Resume offered automatically via SessionStart hook

## Context Folding

Context folding creates isolated branches for complex sub-tasks. Each branch has its own token budget and auto-cleans up on return, preventing context bloat.

**When to use:**
- Complex multi-step investigations
- Reading many files for analysis
- Exploratory work that shouldn't pollute main context

**Workflow:**
```
1. branch_create(session_id, description, budget: 4096)
   -> Creates isolated branch with token budget

2. Do work in the branch
   -> Read files, search, analyze

3. branch_return(branch_id, "Summary of findings")
   -> Results scrubbed for secrets
   -> Branch cleaned up automatically
```

See the `context-folding` skill for full documentation.

## Links

- [Documentation](https://github.com/fyrsmithlabs/contextd)
- [Issues](https://github.com/fyrsmithlabs/contextd/issues)
- [Releases](https://github.com/fyrsmithlabs/contextd/releases)
