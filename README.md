# ContextD

[![Release](https://img.shields.io/github/v/release/fyrsmithlabs/contextd?include_prereleases)](https://github.com/fyrsmithlabs/contextd/releases)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Ffyrsmithlabs%2Fcontextd-blue)](https://ghcr.io/fyrsmithlabs/contextd)
[![Homebrew](https://img.shields.io/badge/homebrew-fyrsmithlabs%2Ftap-orange)](https://github.com/fyrsmithlabs/homebrew-tap)

> **ALPHA** - This project is in active development. APIs may change.

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
| **Secret Scrubbing** | Automatic detection via gitleaks (contextd tools only*) | Working |
| **Vector Search** | Semantic search via chromem (embedded) or Qdrant | Working |
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

### Using Claude Code Plugin (Recommended)

Install contextd directly in Claude Code using the plugin system:

```bash
/plugin install contextd@fyrsmithlabs/contextd
```

The plugin automatically:
- Downloads the appropriate binary for your OS/architecture
- Configures MCP settings
- Sets up the contextd server

After installation, restart Claude Code to activate.

> **Prefer Docker?** See [docs/DOCKER.md](docs/DOCKER.md) for manual Docker setup.

---

### Using Homebrew (macOS/Linux)

```bash
brew install fyrsmithlabs/tap/contextd
```

Or tap first:

```bash
brew tap fyrsmithlabs/tap
brew install contextd
```

Add to Claude Code MCP config (`~/.claude.json`):

```json
{
  "mcpServers": {
    "contextd": {
      "command": "contextd",
      "args": ["--mcp", "--no-http"]
    }
  }
}
```

> **Note:** `--no-http` allows multiple Claude Code sessions to run simultaneously.

### Download Binary

Download from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Intel | `contextd_*_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `contextd_*_darwin_arm64.tar.gz` |
| Linux | x64 | `contextd_*_linux_amd64.tar.gz` |

> **Note:** All binaries are built with CGO enabled for FastEmbed/ONNX support.

### Building from Source

```bash
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd

# Without CGO (uses TEI for embeddings)
go build -o contextd ./cmd/contextd
go build -o ctxd ./cmd/ctxd

# With CGO + FastEmbed (requires ONNX runtime)
CGO_ENABLED=1 go build -o contextd ./cmd/contextd
```

---

## Docker

For container isolation, see [docs/DOCKER.md](docs/DOCKER.md).

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `v0.2.0-rcX` | Release candidates |

```bash
docker pull ghcr.io/fyrsmithlabs/contextd:latest
```

---

## Optional: External Qdrant

contextd uses embedded chromem by default. For external Qdrant:

```bash
docker run -d --name contextd-qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v $(pwd)/qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

Configure via environment:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "contextd",
      "args": ["--mcp", "--no-http"],
      "env": {
        "VECTORSTORE_PROVIDER": "qdrant",
        "QDRANT_HOST": "localhost",
        "QDRANT_PORT": "6334"
      }
    }
  }
}
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
| `repository_search` | Semantic search over indexed code |

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
│  │                   │   chromem   │                │    │
│  │                   │  (Vectors)  │  or Qdrant     │    │
│  │                   └─────────────┘                │    │
│  └──────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

**All-in-one binary includes:**
- ContextD MCP server
- chromem embedded vector database (zero external dependencies)
- FastEmbed for local embeddings (ONNX-based, no API calls)
- Optional: Configure `VECTORSTORE_PROVIDER=qdrant` for external Qdrant

---

## Data Persistence

Data is stored in `~/.config/contextd/vectorstore/` by default.

**Backup:**

```bash
tar czf contextd-backup.tar.gz ~/.config/contextd/
```

**Restore:**

```bash
tar xzf contextd-backup.tar.gz -C ~/
```

---

## Documentation

See the [docs/](docs/) directory for detailed documentation:

- [Architecture Overview](docs/architecture.md) - System design and component interactions
- [Configuration Reference](docs/configuration.md) - Environment variables and settings
- [Docker Setup](docs/DOCKER.md) - Running contextd in Docker
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

MIT License - See [LICENSE](LICENSE) for details.

---

## Links

- [GitHub Repository](https://github.com/fyrsmithlabs/contextd)
- [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases)
- [Docker Image](https://ghcr.io/fyrsmithlabs/contextd)
- [Homebrew Tap](https://github.com/fyrsmithlabs/homebrew-tap)
- [Issue Tracker](https://github.com/fyrsmithlabs/contextd/issues)
