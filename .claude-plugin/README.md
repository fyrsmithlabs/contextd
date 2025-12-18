# contextd Plugin for Claude Code

Cross-session memory and context management for AI agents.

## Features

- **Semantic Memory** - Search past learnings and strategies across sessions
- **Checkpoints** - Save and resume session context before hitting limits
- **Error Remediation** - Track and reuse solutions to errors
- **Semantic Search** - Smart code search with auto-fallback to grep
- **Context Folding** - Isolated sub-tasks with token budgets that auto-cleanup
- **Repository Indexing** - Semantic code search over indexed repositories
- **Self-Reflection** - Analyze behavior patterns and improve documentation
- **Secret Scrubbing** - Automatic detection via gitleaks

## Installation

### 1. Install the Plugin

```bash
claude plugins add fyrsmithlabs/contextd
```

### 2. Install the MCP Server

Run the install command after adding the plugin:

```
/contextd:install
```

Or install manually:

**Homebrew (Recommended):**
```bash
brew install fyrsmithlabs/tap/contextd
```

**Binary:**
Download from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases/latest)

### 3. Configure Claude Code

Add to `~/.claude/settings.json`:

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

Restart Claude Code after configuration.

## Commands

| Command | Description |
|---------|-------------|
| `/contextd:install` | Install contextd MCP server (Homebrew, binary, or Docker) |
| `/contextd:init` | Initialize contextd for a new project |
| `/contextd:onboard` | Onboard to existing project with context priming |
| `/contextd:checkpoint` | Save session checkpoint |
| `/contextd:resume` | Resume from checkpoint |
| `/contextd:search` | Search memories and remediations |
| `/contextd:remember` | Record a learning or insight |
| `/contextd:diagnose` | AI-powered error diagnosis |
| `/contextd:reflect` | Analyze behavior patterns and improve docs |
| `/contextd:status` | Show contextd status for session and project |
| `/contextd:help` | Show available commands and skills |
| `/contextd:consensus-review` | Multi-reviewer code review |

## Skills

| Skill | Use When |
|-------|----------|
| `using-contextd` | Starting any session - overview of all tools |
| `session-lifecycle` | Session start/end protocols |
| `cross-session-memory` | Learning loop (search → do → record → feedback) |
| `checkpoint-workflow` | Context approaching 70% capacity |
| `context-folding` | Complex sub-tasks needing isolation |
| `error-remediation` | Resolving errors systematically |
| `repository-search` | Semantic code search |
| `self-reflection` | Reviewing behavior patterns, improving docs |
| `writing-claude-md` | Creating effective CLAUDE.md files |
| `secret-scrubbing` | Understanding secret detection |
| `project-onboarding` | Onboarding to new projects |
| `consensus-review` | Multi-agent code review |

## MCP Tools

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant strategies from past sessions |
| `memory_record` | Save a new learning or strategy |
| `memory_feedback` | Rate whether a memory was helpful |
| `memory_outcome` | Report task success/failure after using a memory |
| `checkpoint_save` | Save current context for later |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from a saved checkpoint |
| `remediation_search` | Find fixes for similar errors |
| `remediation_record` | Record a new error fix |
| `troubleshoot_diagnose` | AI-powered error diagnosis |
| `semantic_search` | Smart code search (auto-fallback to grep) |
| `repository_index` | Index a codebase for semantic search |
| `repository_search` | Semantic search over indexed code |
| `branch_create` | Create isolated sub-task with token budget |
| `branch_return` | Return from branch with scrubbed results |
| `branch_status` | Check branch state and budget usage |

## Quick Start

After installation:

1. **New project**: `/contextd:init` to set up project context
2. **Existing project**: `/contextd:onboard` to prime with existing knowledge
3. **During work**: Memories are automatically searched and recorded
4. **At 70% context**: `/contextd:checkpoint` then `/clear`
5. **Next session**: `/contextd:resume` to continue where you left off

## Context Folding

Context folding creates isolated branches for complex sub-tasks. Each branch has its own token budget and auto-cleans up on return, preventing context bloat.

**When to use:**
- Complex multi-step investigations
- Reading many files for analysis
- Exploratory work that shouldn't pollute main context

**Workflow:**
```
1. branch_create(session_id, description, budget: 4096)
   → Creates isolated branch with token budget

2. Do work in the branch
   → Read files, search, analyze

3. branch_return(branch_id, "Summary of findings")
   → Results scrubbed for secrets
   → Branch cleaned up automatically
```

**Example:**
```json
// Create branch
branch_create({
  "session_id": "main",
  "description": "Analyze auth module",
  "budget": 4096
})
// → branch_id: "br_abc123"

// Do analysis work...

// Return with summary
branch_return({
  "branch_id": "br_abc123",
  "message": "Auth uses JWT with 15min expiry. 3 handlers: login, logout, refresh."
})
```

See the `context-folding` skill for full documentation.

## Hooks

The plugin includes automatic hooks:

- **SessionStart** - Searches memories and lists checkpoints on session start
- **PreCompact** - Auto-saves checkpoint before context compaction

## Links

- [Documentation](https://github.com/fyrsmithlabs/contextd)
- [Issues](https://github.com/fyrsmithlabs/contextd/issues)
- [Releases](https://github.com/fyrsmithlabs/contextd/releases)
