# ContextD

> **⚠️ ALPHA** - This project is in active development. APIs may change.

**A developer-first AI context and reasoning engine.**

> **Note:** This README describes our vision and roadmap. See [Current Status](#current-status) for what's implemented today.

---

## Vision

ContextD is designed to eliminate the complexity of building with AI. It gives teams a smart baseline for context management, organizational memory, and workflow-aware reasoning—without forcing them into rigid pipelines or spending weeks learning prompts, agents, and skills.

**This is not RAG.** It's a self-improving system that learns how your team actually works, helping developers get real value from AI faster, safer, and with far less friction.

### Core Principles

- **Developer-first**: Works with your existing tools, not against them
- **Zero lock-in**: Runs locally, on-prem, or in the cloud
- **Self-improving**: Learns from successes and failures to get better over time
- **Privacy-conscious**: Your data stays yours; secrets are automatically scrubbed

---

## Current Status

ContextD is in active development. Here's what works today:

### Implemented

| Feature | Description | Status |
|---------|-------------|--------|
| **Cross-session Memory** | Record and retrieve learnings across sessions | Working |
| **Checkpoints** | Save and resume context snapshots | Working |
| **Remediation Tracking** | Store error patterns and fixes | Working |
| **Secret Scrubbing** | Automatic detection via gitleaks | Working |
| **Vector Search** | Semantic search via Qdrant | Working |
| **Local Embeddings** | FastEmbed with ONNX (no API calls) | Working |
| **MCP Integration** | Works with Claude Code out of the box | Working |

### Roadmap

| Feature | Description | Status |
|---------|-------------|--------|
| Workflow-aware reasoning | Learn team patterns automatically | Planned |
| Multi-tenant organization memory | Share learnings across teams | Planned |
| Self-improving confidence scoring | Adjust based on feedback loops | In Progress |
| Cloud deployment options | Managed hosting | Planned |

---

## Quick Start

### Using Homebrew (macOS/Linux)

```bash
brew tap fyrsmithlabs/tap
brew install contextd
```

You'll also need Qdrant running:

```bash
brew install qdrant
qdrant &
```

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "contextd",
      "args": ["-mcp"]
    }
  }
}
```

### Using Docker (Recommended for All-in-One)

```bash
docker pull ghcr.io/fyrsmithlabs/contextd:latest
```

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "ghcr.io/fyrsmithlabs/contextd:latest"
      ]
    }
  }
}
```

Restart Claude Code. That's it.

### Download Binary

Download from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Intel | `contextd_*_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `contextd_*_darwin_arm64.tar.gz` |
| Linux | x64 | `contextd_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `contextd_*_linux_arm64.tar.gz` |
| Windows | x64 | `contextd_*_windows_amd64.zip` |

### Building from Source

```bash
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd
go build -o contextd ./cmd/contextd
go build -o ctxd ./cmd/ctxd
```

---

## MCP Tools

ContextD exposes these tools to Claude Code:

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

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Claude Code                          │
│                         │                                │
│                    MCP Protocol                          │
│                         │                                │
│  ┌──────────────────────▼──────────────────────────┐    │
│  │                  ContextD                        │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │    │
│  │  │ Reasoning   │  │ Checkpoint  │  │ Remediate│ │    │
│  │  │ Bank        │  │ Service     │  │ Service  │ │    │
│  │  └──────┬──────┘  └──────┬──────┘  └────┬─────┘ │    │
│  │         │                │               │       │    │
│  │         └────────────────┼───────────────┘       │    │
│  │                          │                       │    │
│  │                   ┌──────▼──────┐                │    │
│  │                   │   Qdrant    │                │    │
│  │                   │  (Vectors)  │                │    │
│  │                   └─────────────┘                │    │
│  └──────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

**All-in-one container includes:**
- ContextD MCP server
- Qdrant vector database
- FastEmbed for local embeddings (ONNX-based, no API calls)

---

## Data Persistence

All data persists in a Docker volume:

```bash
# Backup
docker run --rm -v contextd-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/contextd-backup.tar.gz /data

# Restore
docker run --rm -v contextd-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/contextd-backup.tar.gz -C /
```

---

## Documentation

See the [docs/](docs/) directory for detailed documentation:

- [Architecture Overview](docs/architecture.md) - System design and component interactions
- [Configuration Reference](docs/configuration.md) - Environment variables and settings
- [MCP Tools API](docs/api/mcp-tools.md) - Complete tool reference with examples
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions
- [Contributing Guide](CONTRIBUTING.md) - How to contribute to ContextD

---

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Development setup
- Coding standards
- Testing requirements
- Pull request process

---

## License

[License TBD]

---

## Links

- [GitHub Repository](https://github.com/fyrsmithlabs/contextd)
- [Docker Image](https://ghcr.io/fyrsmithlabs/contextd)
- [Issue Tracker](https://github.com/fyrsmithlabs/contextd/issues)
