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

### Using Claude Code Plugin (Easiest)

Install contextd directly in Claude Code using the plugin system:

```bash
# Install binary (downloads to ~/.local/bin)
/plugin install contextd@fyrsmithlabs/contextd

# OR install Docker variant (uses container)
/plugin install contextd:docker@fyrsmithlabs/contextd
```

The plugin automatically:
- Downloads the appropriate binary for your OS/architecture (or pulls Docker image)
- Configures MCP settings
- Sets up the contextd server

After installation, restart Claude Code to activate.

> **Note:** The default install downloads a native binary (~15MB). Use the `:docker` variant if you prefer container isolation or have issues with the binary.

---

### Using Docker (Recommended for Manual Setup)

The Docker image includes everything: ContextD with embedded chromem vectorstore and FastEmbed embeddings (zero external dependencies).

**Quick Setup (Persistent Container):**

```bash
# Pull image
docker pull ghcr.io/fyrsmithlabs/contextd:latest

# Create persistent container (shared across all sessions)
docker run -d \
  --name contextd-server \
  --restart unless-stopped \
  --memory=2g \
  --cpus=2 \
  --user "$(id -u):$(id -g)" \
  -v contextd-data:/data \
  -v "${HOME}:${HOME}:ro" \
  -w "${HOME}" \
  ghcr.io/fyrsmithlabs/contextd:latest \
  tail -f /dev/null
```

Add to your Claude Code MCP config (`~/.claude.json`):

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "docker",
      "args": [
        "exec", "-i", "-w", "${PWD}",
        "contextd-server",
        "contextd", "-mcp"
      ],
      "env": {}
    }
  }
}
```

> **Why persistent?** contextd is designed for cross-session memory. A persistent container avoids 500-2000ms startup overhead per tool call and keeps the embedding model loaded.

**Container Management:**
```bash
docker stop contextd-server   # Stop
docker start contextd-server  # Start
docker logs contextd-server   # View logs
docker rm -f contextd-server  # Remove (data preserved in volume)
```

Restart Claude Code to activate.

### Using Homebrew (macOS/Linux)

```bash
brew install fyrsmithlabs/tap/contextd
```

Or tap first:

```bash
brew tap fyrsmithlabs/tap
brew install contextd
```

**Optional: External Qdrant** (contextd uses embedded chromem by default, but you can use external Qdrant if preferred):

**Option 1: Docker** (recommended for development):

```bash
# Pull the latest Qdrant image
docker pull qdrant/qdrant

# Run Qdrant with persistent storage
docker run -d --name contextd-qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v $(pwd)/qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

**Option 2: Docker Compose** (for more control):

Create `docker-compose.qdrant.yml`:

```yaml
services:
  qdrant:
    image: qdrant/qdrant:latest
    restart: always
    ports:
      - 6333:6333
      - 6334:6334
    volumes:
      - ./qdrant_data:/qdrant/storage
```

Then run:

```bash
docker-compose -f docker-compose.qdrant.yml up -d
```

**Option 3: Build from source** (advanced users):

```bash
# Install Rust toolchain
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Clone and build Qdrant
git clone https://github.com/qdrant/qdrant.git
cd qdrant
cargo build --release --bin qdrant

# Run Qdrant
mkdir -p ~/qdrant/storage
./target/release/qdrant --uri 0.0.0.0:6334 &
```

**Verify Qdrant is running:**

```bash
curl http://localhost:6333/
# Should return: "Welcome to Qdrant!"
```

**Add to Claude Code MCP config** (`~/.claude.json`):

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

To use Qdrant instead of the embedded chromem:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "contextd",
      "args": ["-mcp"],
      "env": {
        "VECTORSTORE_PROVIDER": "qdrant",
        "QDRANT_HOST": "localhost",
        "QDRANT_PORT": "6334"
      }
    }
  }
}
```

> **Note:** Homebrew binaries now include ONNX runtime for FastEmbed support. If ONNX is unavailable, set `EMBEDDINGS_PROVIDER=tei` and configure `TEI_URL`.

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

## Docker Image Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release (always updated) |
| `v0.2.0-rc5` | Current release candidate |
| `v0.2.0-rc4` | Previous release candidate |

Platform: `linux/amd64`, `linux/arm64`

```bash
# Pull latest (recommended)
docker pull ghcr.io/fyrsmithlabs/contextd:latest

# Pull specific version
docker pull ghcr.io/fyrsmithlabs/contextd:v0.2.0-rc5
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

**All-in-one container includes:**
- ContextD MCP server
- chromem embedded vector database (zero external dependencies)
- FastEmbed for local embeddings (ONNX-based, no API calls)
- Optional: Configure `VECTORSTORE_PROVIDER=qdrant` for external Qdrant

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
