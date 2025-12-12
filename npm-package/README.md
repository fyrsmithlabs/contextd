# @fyrsmithlabs/contextd-mcp

MCP server for cross-session memory, checkpoints, and error remediation for AI agents.

## Installation

```bash
npm install -g @fyrsmithlabs/contextd-mcp
```

Or use with npx (downloads on first use):

```bash
npx @fyrsmithlabs/contextd-mcp
```

## Usage with Claude Code

Add to your `~/.claude.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "npx",
      "args": ["@fyrsmithlabs/contextd-mcp"]
    }
  }
}
```

Or install the Claude Code plugin which configures this automatically:

```bash
/plugin install contextd@fyrsmithlabs/contextd
```

## What it provides

- **Cross-session memory**: Record and retrieve learnings across sessions
- **Checkpoints**: Save and resume context snapshots
- **Remediation tracking**: Store error patterns and fixes
- **Semantic search**: Find relevant past strategies using vector search

## MCP Tools

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant strategies from past sessions |
| `memory_record` | Save a new learning or strategy |
| `memory_feedback` | Rate whether a memory was helpful |
| `checkpoint_save` | Save current context for later |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from a saved checkpoint |
| `remediation_search` | Find fixes for similar errors |
| `remediation_record` | Record a new error fix |
| `troubleshoot_diagnose` | AI-powered error diagnosis |
| `repository_index` | Index a codebase for semantic search |
| `repository_search` | Semantic search over indexed code |

## Environment Variables

- `CONTEXTD_VERSION`: Specific version to download (default: `latest`)
- `CONTEXTD_FORCE_DOWNLOAD`: Force re-download of binary

## Supported Platforms

- macOS (Apple Silicon): `darwin-arm64`
- macOS (Intel): `darwin-x64`
- Linux (x64): `linux-x64`
- Linux (ARM64): `linux-arm64`

## Links

- [GitHub Repository](https://github.com/fyrsmithlabs/contextd)
- [Documentation](https://github.com/fyrsmithlabs/contextd#readme)
- [Issues](https://github.com/fyrsmithlabs/contextd/issues)

## License

MIT
