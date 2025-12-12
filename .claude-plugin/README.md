# contextd Plugin for Claude Code

Cross-session memory and context management for AI agents.

## Features

- **Semantic Memory** - Search past learnings and strategies across sessions
- **Checkpoints** - Save and resume session context at 70% capacity
- **Error Remediation** - Track and reuse solutions to errors
- **Repository Search** - Semantic code search over indexed repositories

## Installation

### 1. Install the Plugin (Skills & Commands)

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

**Docker:**
```bash
docker pull ghcr.io/fyrsmithlabs/contextd:latest
```

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

For Docker configuration, see `/contextd:install`.

## Commands

| Command | Description |
|---------|-------------|
| `/contextd:install` | Install contextd MCP server |
| `/contextd:init` | Initialize contextd for a new project |
| `/contextd:onboard` | Onboard to existing project |
| `/contextd:checkpoint` | Save session checkpoint |
| `/contextd:resume` | Resume from checkpoint |
| `/contextd:search` | Search memories and remediations |
| `/contextd:remember` | Record a learning |
| `/contextd:diagnose` | Diagnose an error |
| `/contextd:status` | Show contextd status |

## Skills

| Skill | Use When |
|-------|----------|
| `using-contextd` | Starting any session |
| `session-lifecycle` | Session start/end protocols |
| `cross-session-memory` | Learning loop (search, do, record) |
| `checkpoint-workflow` | Context at 70% capacity |
| `error-remediation` | Resolving errors |
| `repository-search` | Semantic code search |

## Links

- [Documentation](https://github.com/fyrsmithlabs/contextd)
- [Issues](https://github.com/fyrsmithlabs/contextd/issues)
- [Releases](https://github.com/fyrsmithlabs/contextd/releases)
