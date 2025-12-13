# contextd Quick Start

AI agent memory and context management for Claude Code.

## Install

### Option 1: Homebrew (Recommended)

```bash
brew install fyrsmithlabs/tap/contextd
```

### Option 2: Binary Download

Download from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases/latest):
- macOS Apple Silicon: `contextd_*_darwin_arm64.tar.gz`
- macOS Intel: `contextd_*_darwin_amd64.tar.gz`
- Linux x64: `contextd_*_linux_amd64.tar.gz`

```bash
tar -xzf contextd_*.tar.gz
chmod +x contextd ctxd
mv contextd ctxd ~/.local/bin/  # or /usr/local/bin/
```

### Option 3: Docker

```bash
docker pull ghcr.io/fyrsmithlabs/contextd:latest
```

## Configure Claude Code

Add to `~/.claude/settings.json`:

### For Homebrew/Binary:

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

### For Docker:

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "${HOME}/.config/contextd:/root/.config/contextd", "ghcr.io/fyrsmithlabs/contextd:latest", "--mcp"]
    }
  }
}
```

## Restart Claude Code

After adding the config, restart Claude Code. These MCP tools become available:

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant past strategies |
| `memory_record` | Save new memories |
| `memory_feedback` | Rate memory helpfulness |
| `memory_outcome` | Report task success/failure |
| `checkpoint_save` | Save context snapshot |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from checkpoint |
| `remediation_search` | Find error fix patterns |
| `remediation_record` | Record new fixes |
| `troubleshoot_diagnose` | Diagnose errors with AI |
| `repository_index` | Index repo for semantic search |
| `repository_search` | Semantic search over indexed code |

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
