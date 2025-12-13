# ContextD

[![Release](https://img.shields.io/github/v/release/fyrsmithlabs/contextd?include_prereleases)](https://github.com/fyrsmithlabs/contextd/releases)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Ffyrsmithlabs%2Fcontextd-blue)](https://ghcr.io/fyrsmithlabs/contextd)
[![Homebrew](https://img.shields.io/badge/homebrew-fyrsmithlabs%2Ftap-orange)](https://github.com/fyrsmithlabs/homebrew-tap)

**Cross-session memory and context management for AI agents.**

ContextD helps AI coding assistants remember what works, learn from mistakes, and maintain context across sessions. It's designed for developers who want their AI tools to get smarter over time.

---

## What It Does

| Feature | Description |
|---------|-------------|
| **Cross-session Memory** | Record and retrieve learnings across sessions with semantic search |
| **Checkpoints** | Save and resume context snapshots before hitting limits |
| **Error Remediation** | Track error patterns and fixes - never solve the same bug twice |
| **Repository Search** | Semantic code search over your indexed codebase |
| **Self-Reflection** | Analyze behavior patterns and improve documentation |
| **Secret Scrubbing** | Automatic detection and removal via gitleaks |

---

## Quick Start

### Option 1: Claude Code Plugin (Recommended)

```bash
# Install the plugin (skills, commands, agents)
claude plugins add fyrsmithlabs/contextd

# Run the install command for MCP server setup
/contextd:install
```

The install command will:
- Download the appropriate binary for your OS/architecture
- Configure MCP settings
- Set up the contextd server

### Option 2: Homebrew

```bash
brew install fyrsmithlabs/tap/contextd
```

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

### Option 3: Download Binary

Download from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases/latest):

| Platform | Architecture | File |
|----------|--------------|------|
| macOS | Apple Silicon | `contextd_*_darwin_arm64.tar.gz` |
| macOS | Intel | `contextd_*_darwin_amd64.tar.gz` |
| Linux | x64 | `contextd_*_linux_amd64.tar.gz` |

---

## Daily Workflow

After installation, here's how to use contextd:

```
1. Session Start
   └─→ Memories auto-searched, checkpoints listed
   └─→ Resume from checkpoint if offered

2. During Work
   └─→ /contextd:search <topic>     Find relevant memories
   └─→ /contextd:diagnose <error>   Get help with errors
   └─→ Semantic search with repository_search()

3. Task Complete
   └─→ /contextd:remember           Record what you learned

4. Context High (70%+)
   └─→ /contextd:checkpoint         Save session state
   └─→ /clear                       Reset context
   └─→ /contextd:resume             Continue where you left off

5. New Project
   └─→ /contextd:init               Setup new project
   └─→ /contextd:onboard            Analyze existing codebase
```

---

## Plugin Commands

| Command | Description |
|---------|-------------|
| `/contextd:install` | Install contextd MCP server |
| `/contextd:init` | Initialize contextd for a new project |
| `/contextd:onboard` | Onboard to existing project with context priming |
| `/contextd:checkpoint` | Save session checkpoint |
| `/contextd:resume` | Resume from checkpoint |
| `/contextd:search` | Search memories and remediations |
| `/contextd:remember` | Record a learning or insight |
| `/contextd:diagnose` | AI-powered error diagnosis |
| `/contextd:reflect` | Analyze behavior patterns and improve docs |
| `/contextd:status` | Show contextd status |
| `/contextd:help` | Show available commands and skills |

## Plugin Skills

| Skill | Use When |
|-------|----------|
| `using-contextd` | Starting any session - overview of all tools |
| `session-lifecycle` | Session start/end protocols |
| `cross-session-memory` | Learning loop (search → do → record → feedback) |
| `checkpoint-workflow` | Context approaching 70% capacity |
| `error-remediation` | Resolving errors systematically |
| `repository-search` | Semantic code search |
| `self-reflection` | Reviewing behavior patterns, improving docs |
| `writing-claude-md` | Creating effective CLAUDE.md files |
| `secret-scrubbing` | Understanding secret detection |
| `project-onboarding` | Onboarding to new projects |
| `consensus-review` | Multi-agent code review with specialized reviewers |

---

## MCP Tools

ContextD exposes these tools to Claude Code:

### Memory

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant strategies from past sessions |
| `memory_record` | Save a new learning or strategy |
| `memory_feedback` | Rate whether a memory was helpful |
| `memory_outcome` | Report task success/failure after using a memory |

### Checkpoints

| Tool | Purpose |
|------|---------|
| `checkpoint_save` | Save current context for later |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from a saved checkpoint |

### Remediation

| Tool | Purpose |
|------|---------|
| `remediation_search` | Find fixes for similar errors |
| `remediation_record` | Record a new error fix |
| `troubleshoot_diagnose` | AI-powered error diagnosis |

### Repository

| Tool | Purpose |
|------|---------|
| `repository_index` | Index a codebase for semantic search |
| `repository_search` | Semantic search over indexed code |

---

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                       Claude Code                            │
│                           │                                  │
│                      MCP Protocol                            │
│                           │                                  │
│  ┌────────────────────────▼────────────────────────────┐    │
│  │                     ContextD                         │    │
│  │                                                      │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │    │
│  │  │  Reasoning  │  │ Checkpoint  │  │ Remediation │  │    │
│  │  │    Bank     │  │   Service   │  │   Service   │  │    │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │    │
│  │         │                │                │         │    │
│  │         └────────────────┼────────────────┘         │    │
│  │                          │                          │    │
│  │                   ┌──────▼──────┐                   │    │
│  │                   │   chromem   │  (embedded)       │    │
│  │                   │   Vectors   │  or Qdrant        │    │
│  │                   └─────────────┘                   │    │
│  │                                                      │    │
│  │  + FastEmbed (local ONNX embeddings)                │    │
│  │  + gitleaks (secret scrubbing)                      │    │
│  └──────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

**Key components:**
- **chromem** - Embedded vector database (zero external dependencies)
- **FastEmbed** - Local ONNX embeddings (no API calls required)
- **gitleaks** - Secret detection and scrubbing
- **Optional Qdrant** - External vector database for larger deployments

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VECTORSTORE_PROVIDER` | `chromem` | Vector store (`chromem` or `qdrant`) |
| `VECTORSTORE_PATH` | `~/.config/contextd/vectorstore` | Data storage path |
| `QDRANT_HOST` | `localhost` | Qdrant host (if using qdrant) |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port |
| `EMBEDDING_PROVIDER` | `fastembed` | Embedding provider |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### Using External Qdrant

```bash
docker run -d --name qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v $(pwd)/qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

Configure in `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
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

## Data & Backup

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

## CLI Tools

ContextD includes two binaries:

| Binary | Purpose |
|--------|---------|
| `contextd` | MCP server (run with `--mcp --no-http`) |
| `ctxd` | CLI utility for manual operations |

### ctxd Commands

```bash
ctxd health              # Check server health
ctxd scrub <file>        # Scrub secrets from a file
ctxd init                # Initialize dependencies (ONNX runtime)
ctxd migrate             # Migrate data from Qdrant to chromem
```

---

## Building from Source

```bash
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd

# Build with FastEmbed (requires CGO)
make build

# Or install to $GOPATH/bin
make go-install

# Run tests
make test
```

---

## Documentation

- [Docker Setup](docs/DOCKER.md) - Running contextd in Docker
- [Design Plans](docs/plans/) - Feature design documents
- [Specifications](docs/spec/) - Technical specifications

---

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Submit a pull request

---

## License

MIT License - See [LICENSE](LICENSE) for details.

---

## Links

- [GitHub Repository](https://github.com/fyrsmithlabs/contextd)
- [Releases](https://github.com/fyrsmithlabs/contextd/releases)
- [Docker Image](https://ghcr.io/fyrsmithlabs/contextd)
- [Homebrew Tap](https://github.com/fyrsmithlabs/homebrew-tap)
- [Issues](https://github.com/fyrsmithlabs/contextd/issues)
