# contextd Quick Start

AI agent memory and context management for Claude Code.

## Prerequisites

**You need [Claude Code](https://claude.ai/claude-code) installed first.**

```bash
# macOS/Linux
curl -fsSL https://claude.ai/install.sh | bash

# Verify
claude --version
```

## Installation

### Option 1: Automated Setup (Recommended)

Install the Claude Code plugin and let it handle everything:

```bash
# 1. Add the plugin
claude plugins add fyrsmithlabs/marketplace

# 2. Run auto-setup in Claude Code
/contextd:install
```

This automatically:
- Downloads contextd binary (or uses Docker if binary unavailable)
- Configures MCP settings in `~/.claude/settings.json`
- Validates the setup

**Restart Claude Code and you're done!**

### Option 2: Manual Installation

If you prefer manual control:

**Step 1: Install Binary**

Choose one:

```bash
# Homebrew
brew install fyrsmithlabs/tap/contextd

# Binary Download
# Download from: https://github.com/fyrsmithlabs/contextd/releases/latest
tar -xzf contextd_*.tar.gz
chmod +x contextd ctxd
mv contextd ctxd ~/.local/bin/

# Docker
docker pull ghcr.io/fyrsmithlabs/contextd:latest
```

**Step 2: Configure with CLI**

```bash
ctxd mcp install    # Auto-configure MCP settings
ctxd mcp status     # Verify configuration
```

**Or configure manually** by adding to `~/.claude/settings.json`:

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

**For Docker:**
```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "${HOME}/.config/contextd:/root/.config/contextd", "ghcr.io/fyrsmithlabs/contextd:latest", "--mcp", "--no-http"]
    }
  }
}
```

**Restart Claude Code**

After adding the config, restart Claude Code. These MCP tools become available:

| Category | Tool | Purpose |
|----------|------|---------|
| Memory | `memory_search` | Find relevant past strategies |
| | `memory_record` | Save new memories |
| | `memory_feedback` | Rate memory helpfulness |
| | `memory_outcome` | Report task success/failure |
| | `memory_consolidate` | Merge related memories |
| Checkpoint | `checkpoint_save` | Save context snapshot |
| | `checkpoint_list` | List available checkpoints |
| | `checkpoint_resume` | Resume from checkpoint |
| Remediation | `remediation_search` | Find error fix patterns |
| | `remediation_record` | Record new fixes |
| | `remediation_feedback` | Rate fix helpfulness |
| | `troubleshoot_diagnose` | AI-powered error diagnosis |
| Search | `semantic_search` | Smart search with grep fallback |
| | `repository_index` | Index repo for semantic search |
| | `repository_search` | Search indexed code by meaning |
| Context | `branch_create` | Isolate sub-tasks with token budgets |
| | `branch_return` | Return results from branch |
| | `branch_status` | Check branch progress |
| Conversation | `conversation_index` | Index past conversations |
| | `conversation_search` | Search conversation history |
| Reflection | `reflect_report` | Generate self-reflection report |
| | `reflect_analyze` | Analyze behavioral patterns |

See [MCP Tools API Reference](docs/api/mcp-tools.md) for full documentation.

## Data Location

Data stored in `~/.config/contextd/vectorstore/` by default.

```bash
# Backup
tar czf contextd-backup.tar.gz ~/.config/contextd/

# Restore
tar xzf contextd-backup.tar.gz -C ~/
```

## Architecture

contextd bundles:
- **chromem** - Embedded vector database (zero external dependencies)
- **FastEmbed** - Local ONNX embeddings (no API calls)

Everything runs locally. No external services required.

**Optional:** Set `VECTORSTORE_PROVIDER=qdrant` to use external Qdrant instead.
