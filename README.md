# ContextD

[![Release](https://img.shields.io/github/v/release/fyrsmithlabs/contextd?include_prereleases)](https://github.com/fyrsmithlabs/contextd/releases)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Ffyrsmithlabs%2Fcontextd-blue)](https://ghcr.io/fyrsmithlabs/contextd)
[![Homebrew](https://img.shields.io/badge/homebrew-fyrsmithlabs%2Ftap-orange)](https://github.com/fyrsmithlabs/homebrew-tap)

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

### Using Docker (Recommended)

The Docker image includes everything: ContextD, Qdrant, and FastEmbed embeddings.

```bash
# Pull the image (multi-arch: amd64 and arm64)
docker pull ghcr.io/fyrsmithlabs/contextd:0.1.0-alpha
```

Add to your Claude Code MCP config (`~/.claude.json` or Claude Desktop config):

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "ghcr.io/fyrsmithlabs/contextd:0.1.0-alpha"
      ]
    }
  }
}
```

Restart Claude Code. That's it.

### Using Homebrew (macOS/Linux)

```bash
brew install fyrsmithlabs/tap/contextd
```

Or tap first:

```bash
brew tap fyrsmithlabs/homebrew-tap
brew install contextd
```

**Start Qdrant** (required for vector search):

```bash
docker run -d --name contextd-qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v contextd-qdrant-data:/qdrant/storage \
  --restart always \
  qdrant/qdrant:v1.12.1
```

**Add to Claude Code MCP config** (`~/.claude.json`):

```json
{
  "mcpServers": {
    "contextd": {
      "command": "contextd",
      "args": ["-mcp"],
      "env": {
        "QDRANT_HOST": "localhost",
        "QDRANT_PORT": "6334"
      }
    }
  }
}
```

> **Note:** Homebrew binaries use the TEI embedding provider by default. Set `TEI_URL` if you have a TEI server, or use the Docker image for built-in FastEmbed support.

### Download Binary

Download from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Intel | `contextd_*_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `contextd_*_darwin_arm64.tar.gz` |
| Linux | x64 | `contextd_*_linux_amd64.tar.gz` |
| Linux | ARM64 | `contextd_*_linux_arm64.tar.gz` |
| Windows | x64 | `contextd_*_windows_amd64.zip` |

> **Note:** Pre-built binaries use TEI for embeddings (no CGO). For FastEmbed support, use Docker or build from source with CGO enabled.

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

## Docker Image Tags

| Tag | Description |
|-----|-------------|
| `0.1.0-alpha` | Current alpha release |
| `latest` | Latest stable release (not yet available) |

Multi-arch support: `linux/amd64` and `linux/arm64`

```bash
# Explicit platform
docker pull --platform linux/arm64 ghcr.io/fyrsmithlabs/contextd:0.1.0-alpha
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

MIT License - See [LICENSE](LICENSE) for details.

---

## Links

- [GitHub Repository](https://github.com/fyrsmithlabs/contextd)
- [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases)
- [Docker Image](https://ghcr.io/fyrsmithlabs/contextd)
- [Homebrew Tap](https://github.com/fyrsmithlabs/homebrew-tap)
- [Issue Tracker](https://github.com/fyrsmithlabs/contextd/issues)
