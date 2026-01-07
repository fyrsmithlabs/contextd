# ContextD

[![Release](https://img.shields.io/github/v/release/fyrsmithlabs/contextd?include_prereleases)](https://github.com/fyrsmithlabs/contextd/releases)
[![Test](https://github.com/fyrsmithlabs/contextd/actions/workflows/test.yml/badge.svg)](https://github.com/fyrsmithlabs/contextd/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/fyrsmithlabs/contextd/branch/main/graph/badge.svg)](https://codecov.io/gh/fyrsmithlabs/contextd)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Ffyrsmithlabs%2Fcontextd-blue)](https://ghcr.io/fyrsmithlabs/contextd)
[![Homebrew](https://img.shields.io/badge/homebrew-fyrsmithlabs%2Ftap-orange)](https://github.com/fyrsmithlabs/homebrew-tap)

**Cross-session memory and context management for AI agents.**

ContextD helps AI coding assistants remember what works, learn from mistakes, and maintain context across sessions. It's designed for developers who want their AI tools to get smarter over time.

---

## ⚠️ Alpha Status

**This project is in active alpha development.** Features and APIs change frequently as we refine the product based on user feedback.

- ✅ Core functionality is stable and tested
- ⚠️ Breaking changes are generally avoided but can still occur
- 📝 We document all changes in release notes
- 🚀 Expect rapid iteration and improvements

If you encounter issues, please [report them on GitHub](https://github.com/fyrsmithlabs/contextd/issues).

---

## Prerequisites

**You need [Claude Code](https://claude.ai/claude-code) installed first.**

Claude Code is Anthropic's AI coding assistant. ContextD extends Claude Code with persistent memory via the MCP (Model Context Protocol) server integration.

Install Claude Code:
```bash
# macOS/Linux
curl -fsSL https://claude.ai/install.sh | bash

# Or visit: https://claude.ai/download
```

Verify installation:
```bash
claude --version
```

---

## What It Does

| Feature | Description |
|---------|-------------|
| **Cross-session Memory** | Record and retrieve learnings across sessions with semantic search |
| **Checkpoints** | Save and resume context snapshots before hitting limits |
| **Context-Folding** | Isolate complex sub-tasks with dedicated token budgets |
| **Error Remediation** | Track error patterns and fixes - never solve the same bug twice |
| **Repository Search** | Semantic code search over your indexed codebase |
| **Self-Reflection** | Analyze behavior patterns and improve documentation |
| **Secret Scrubbing** | Automatic detection and removal via gitleaks |

---

## Data Privacy & Security

**All data stays local on your machine.**

- No data is sent to external servers
- Memories and checkpoints stored in `~/.config/contextd/`
- Embeddings generated locally using ONNX (no API calls)
- Secrets automatically scrubbed from all tool responses using gitleaks
- Git integration uses local repository info only (remote URL for project identification)

---

## Quick Start

Choose **one** of the following installation methods:

### Option 1: Automated Plugin Setup (Easiest)

If you already have Claude Code installed:

```bash
# 1. Install the plugin (adds skills, commands, agents)
claude plugins add fyrsmithlabs/contextd

# 2. Run auto-setup in Claude Code
/contextd:install
```

This automatically:
- ✅ Downloads contextd binary (or uses Docker if unavailable)
- ✅ Configures MCP settings in `~/.claude/settings.json`
- ✅ Validates the connection

**Restart Claude Code and verify:**
```bash
# In Claude Code, type:
/mcp
# Should show "✓ contextd - connected"
```

**That's it!** See [ONBOARDING.md](docs/ONBOARDING.md) for a guided tutorial.

### Option 2: Homebrew (macOS/Linux)

```bash
# Add the tap
brew tap fyrsmithlabs/tap

# Install contextd
brew install contextd
```

Then add the MCP configuration (see [Configuration](#configuration) below).

### Option 3: Download Binary

Download from [GitHub Releases](https://github.com/fyrsmithlabs/contextd/releases/latest):

| Platform | Architecture | File |
|----------|--------------|------|
| macOS | Apple Silicon | `contextd_*_darwin_arm64.tar.gz` |
| macOS | Intel | `contextd_*_darwin_amd64.tar.gz` |
| Linux | x64 | `contextd_*_linux_amd64.tar.gz` |

Extract and install:
```bash
# Extract
tar xzf contextd_*.tar.gz

# Move to PATH (choose one)
mv contextd ~/.local/bin/       # User install
# OR
sudo mv contextd /usr/local/bin/  # System install
```

Then add the MCP configuration (see [Configuration](#configuration) below).

---

## Configuration

### Automated (Recommended)

Use the CLI tool for automatic configuration:

```bash
ctxd mcp install    # Auto-configure MCP settings
ctxd mcp status     # Verify configuration
ctxd mcp uninstall  # Remove configuration
```

Or use the plugin install command in Claude Code: `/contextd:install`

### Manual Configuration

If you prefer manual setup, add to `~/.claude/settings.json`:

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

**Note:** If the file doesn't exist, create it with just this content.

### Claude Desktop App (Alternative)

If using the Claude Desktop app instead of Claude Code CLI:

**macOS/Linux:** `~/.claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

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

### Verify Setup

After adding configuration, restart Claude Code and verify:

```bash
# Check MCP connection (in Claude Code)
/mcp
# Should show: contextd - connected

# Or test a tool
# In conversation: "Use memory_search to check for existing memories"
```

---

## First Run Behavior

On first run, contextd automatically downloads required dependencies:

```
ONNX runtime not found. Downloading v1.23.0...
Downloaded to ~/.config/contextd/lib/libonnxruntime.so
Downloading fast-bge-small-en-v1.5...
```

This one-time download (~100MB) happens automatically. Subsequent runs start instantly.

---

## Project Identification

ContextD automatically identifies your project using git:

1. **Tenant ID** - Derived from git remote URL (e.g., `github.com/username`)
2. **Project ID** - Derived from repository name

This means:
- Different repositories have isolated memories
- Forked repos share tenant but have separate project memories
- Non-git directories use a fallback identifier

No configuration needed - it works automatically based on your current directory.

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

### Context-Folding

| Tool | Purpose |
|------|---------|
| `branch_create` | Create isolated context branch with token budget |
| `branch_return` | Return from branch with scrubbed results |
| `branch_status` | Get branch status and budget usage |

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

## Multi-Tenancy

ContextD uses **payload-based tenant isolation** to ensure data separation between organizations, teams, and projects.

### How It Works

- All documents stored in shared collections with tenant metadata
- Queries automatically filtered by tenant context
- Missing tenant context returns an error (fail-closed security)

### Tenant Hierarchy

| Scope | Description | Example |
|-------|-------------|---------|
| TenantID | Organization/user identifier | `github.com/acme-corp` |
| TeamID | Team within organization | `platform` |
| ProjectID | Project within team | `contextd` |

### Security Guarantees

| Behavior | Description |
|----------|-------------|
| Fail-closed | Operations without tenant context return errors |
| Filter injection blocked | Users cannot override tenant filters |
| Metadata enforced | Tenant fields always set from authenticated context |

### Automatic Detection

ContextD automatically detects tenant context from git:

1. **TenantID** - Derived from git remote URL (e.g., `github.com/username`)
2. **ProjectID** - Derived from repository name

No manual configuration needed for single-user deployments.

---

## Advanced Configuration

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

For larger deployments or team use:

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

Data is stored in `~/.config/contextd/` by default:

```
~/.config/contextd/
├── vectorstore/          # Memories, checkpoints, remediations
├── lib/                  # ONNX runtime (auto-downloaded)
└── config.yaml           # Optional config file
```

**Backup:**
```bash
tar czf contextd-backup.tar.gz ~/.config/contextd/
```

**Restore:**
```bash
tar xzf contextd-backup.tar.gz -C ~/
```

---

## Troubleshooting

### "contextd not found" after installation

Ensure the binary is in your PATH:
```bash
# Check if contextd is found
which contextd

# If not, add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"
```

### MCP server not connecting

1. Check settings.json syntax (valid JSON?)
2. Verify the path to contextd is correct
3. Restart Claude Code after config changes

```bash
# Test manually
contextd --version
contextd --mcp --no-http  # Should start without errors
```

### First run is slow

Expected behavior - contextd downloads ONNX runtime (~50MB) and embedding model (~50MB) on first run. This only happens once.

### "permission denied" errors

```bash
chmod +x ~/.local/bin/contextd
```

### Still stuck?

See [docs/troubleshooting.md](docs/troubleshooting.md) or [open an issue](https://github.com/fyrsmithlabs/contextd/issues).

---

## CLI Tools

ContextD includes two binaries:

| Binary | Purpose |
|--------|---------|
| `contextd` | MCP server (run with `--mcp --no-http`) |
| `ctxd` | CLI utility for manual operations |

### ctxd Commands

```bash
# MCP Configuration (NEW)
ctxd mcp install         # Auto-configure MCP server settings
ctxd mcp status          # Verify MCP configuration
ctxd mcp uninstall       # Remove MCP configuration

# Statusline
ctxd statusline install  # Configure Claude Code statusline
ctxd statusline run      # Run statusline (used by Claude Code)

# Utilities
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
- [Configuration](docs/configuration.md) - Full configuration reference
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions
- [Architecture](docs/architecture.md) - Technical architecture
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
