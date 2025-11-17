# contextd by Fyrsmith Labs

[![TDD Enforcement](https://github.com/fyrsmithlabs/contextd/workflows/TDD%20Enforcement/badge.svg)](https://github.com/fyrsmithlabs/contextd/actions/workflows/tdd-enforcement.yml)
[![Release](https://img.shields.io/badge/release-0.9.0-rc-1-blue)](https://github.com/fyrsmithlabs/contextd/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/fyrsmithlabs/contextd)](https://goreportcard.com/report/github.com/fyrsmithlabs/contextd)
[![Go Version](https://img.shields.io/github/go-mod/go-version/fyrsmithlabs/contextd)](go.mod)
[![codecov](https://codecov.io/gh/fyrsmithlabs/contextd/branch/main/graph/badge.svg)](https://codecov.io/gh/fyrsmithlabs/contextd)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![GitHub Issues](https://img.shields.io/github/issues/fyrsmithlabs/contextd)](https://github.com/fyrsmithlabs/contextd/issues)
[![GitHub Stars](https://img.shields.io/github/stars/fyrsmithlabs/contextd?style=social)](https://github.com/fyrsmithlabs/contextd/stargazers)

> Context Daemon for Claude Code - Semantic search, session management, and AI-powered troubleshooting

**contextd** is a production-grade service that supercharges Claude Code with persistent memory, semantic search across past work, and intelligent error remediation. Built with security and performance as primary goals, it uses Unix domain sockets for local-only access and vector databases for blazing-fast semantic search.

## Features

### Core Features
- **Semantic Search** - Find past work, solutions, and context across all your Claude Code sessions
- **Session Management** - Save and restore checkpoints with full context preservation
- **AI-Powered Troubleshooting** - Get intelligent error diagnosis and similar solution recommendations
- **Model Context Protocol (MCP)** - First-class integration with Claude Code via 9 MCP tools
- **Local Embeddings** - Use TEI (Text Embeddings Inference) for zero-cost, quota-free embeddings

### 0.9.0-rc-1 New Features
- **Pre-Fetch Engine** - Automatic context loading on git events (20-30% token savings)
- **Secret Scrubbing** - 5-layer defense with Gitleaks integration (800+ patterns)
- **HTTP/SSE Protocol** - JSON-RPC 2.0 with Server-Sent Events streaming
- **OpenTelemetry Observability** - Distributed tracing and Prometheus metrics
- **Modular Services** - Checkpoint, remediation, and prefetch services
- **YAML Configuration** - Clean, hierarchical configuration with environment overrides

### Production Features
- **HTTP Server** - Echo router with graceful shutdown and health checks
- **systemd/launchd Integration** - Native service management on Linux and macOS
- **Comprehensive Observability** - Traces, metrics, and structured logging
- **Developer Experience** - Beautiful TUI monitor, comprehensive CLI, extensive documentation

## Quick Start

### Installation

**Option 1: Download Binary Release (Recommended)**

```bash
# Linux (amd64)
curl -L https://github.com/fyrsmithlabs/contextd/releases/latest/download/contextd_linux_amd64.tar.gz -o contextd.tar.gz
tar -xzf contextd.tar.gz
sudo mv contextd /usr/local/bin/
sudo mv ctxd /usr/local/bin/

# macOS (amd64)
curl -L https://github.com/fyrsmithlabs/contextd/releases/latest/download/contextd_darwin_amd64.tar.gz -o contextd.tar.gz
tar -xzf contextd.tar.gz
sudo mv contextd /usr/local/bin/
sudo mv ctxd /usr/local/bin/
```

**Option 2: Install from Source**

```bash
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd
make build-all
sudo mv contextd /usr/local/bin/
sudo mv ctxd /usr/local/bin/
```

**Option 3: Homebrew (Coming Soon)**

```bash
brew tap fyrsmithlabs/tap
brew install contextd
```

### Setup

**1. Choose Embedding Backend**

**TEI (Recommended - No Quotas)**
```bash
# Start embedding and infrastructure services
docker-compose up -d

# Verify services are running
docker-compose ps

# Expected: qdrant, tei, nats (all running)
```

**OpenAI API (Alternative)**
```bash
export OPENAI_API_KEY=sk-your-key-here
```

**2. Create Configuration**

```bash
# Create config directory
mkdir -p ~/.config/contextd

# Create config file
cat > ~/.config/contextd/config.yaml <<'EOF'
server:
  port: 8080

observability:
  enable_telemetry: true

prefetch:
  enabled: true
  cache_ttl: 5m
EOF
```

**3. Build and Run**

```bash
# Build contextd
go build -o contextd ./cmd/contextd/

# Run with config
./contextd --config ~/.config/contextd/config.yaml

# Or install as system service
# Linux: systemctl --user enable --now contextd
# macOS: launchctl load ~/Library/LaunchAgents/com.axyzlabs.contextd.plist
```

**4. Configure Claude Code MCP**

Update your Claude Code configuration (`~/.claude.json`):

```json
{
  "mcpServers": {
    "contextd": {
      "url": "http://localhost:8080/mcp",
      "transport": {
        "type": "http"
      }
    }
  }
}
```

**5. Restart Claude Code**

Completely quit and restart Claude Code to load the MCP server.

### Usage

**Slash Commands in Claude Code**

```bash
/checkpoint save "completed user authentication"
/checkpoint search "database migration"
/checkpoint list

/remediation search "connection timeout"
/troubleshoot "panic: runtime error"

/index repository path=/home/user/myproject
/status
```

**Health Check**

```bash
# Check service health
curl http://localhost:8080/health

# Expected: {"status":"ok","version":"0.9.0-rc-1"}
```

**Service Management**

```bash
# Linux (systemd)
systemctl --user status contextd
systemctl --user restart contextd

# macOS (launchd)
launchctl list | grep contextd
launchctl kickstart -k gui/$(id -u)/com.fyrsmithlabs.contextd

# View logs
journalctl --user -u contextd -f  # Linux
tail -f ~/.config/contextd/logs/app.log  # macOS
```

## What's New in 0.9.0-rc-1

contextd 0.9.0-rc-1 is a complete architectural rebuild focused on context optimization, security, and observability.

### Pre-Fetch Engine
Automatically loads context when you switch git branches or make commits, reducing round trips by 20-30%.

**How it works**:
- Detects git events (branch switch, new commit)
- Executes 3 deterministic rules in parallel (<2s)
  - `branch_diff`: Git diff summary between branches
  - `recent_commit`: Latest commit message and context
  - `common_files`: Pre-fetches frequently changed files
- Caches results for 5 minutes (configurable)
- Injects cached data into next MCP response (instant)

**User guide**: [docs/guides/PREFETCH-USER-GUIDE.md](docs/guides/PREFETCH-USER-GUIDE.md)

### Secret Scrubbing
5-layer defense preventing credential leakage with Gitleaks integration.

**Protection layers**:
1. Gitleaks pre-commit hook (800+ patterns)
2. Ingestion filtering (API keys, tokens)
3. Storage redaction (logs, database)
4. Retrieval scrubbing (responses)
5. Claude Code hook integration

### HTTP/SSE Protocol
Modern JSON-RPC 2.0 protocol with real-time streaming.

**Features**:
- Server-Sent Events (SSE) for long-running operations
- NATS JetStream for async operation tracking
- Better error handling and debugging
- Health check endpoints

### OpenTelemetry Observability
Complete distributed tracing and metrics.

**Metrics**:
- 8 pre-fetch metrics (cache hit rate, rule timeouts, token savings)
- MCP operation tracking (duration, status codes)
- Service health gauges
- Prometheus + Grafana dashboards

### Migration Guide
Upgrading from v2.0? See [docs/guides/MIGRATION-V2-TO-V3.md](docs/guides/MIGRATION-V2-TO-V3.md)

---

## MCP Tools

contextd provides 9 Model Context Protocol tools for Claude Code:

| Tool | Description |
|------|-------------|
| `checkpoint_save` | Save session checkpoints with semantic indexing |
| `checkpoint_search` | Semantic search across all past sessions |
| `checkpoint_list` | Browse recent checkpoints |
| `remediation_save` | Store error solutions for future reference |
| `remediation_search` | Find similar error fixes (70% semantic + 30% string match) |
| `troubleshoot` | AI-powered error diagnosis with recommendations |
| `list_patterns` | Browse troubleshooting knowledge base |
| `index_repository` | Index repositories for semantic code search |
| `status` | Service health and status information |

## Architecture

contextd 0.9.0-rc-1 uses a modular, layered architecture optimized for context efficiency and security.

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Claude Code (MCP Client)             │
└────────────────────┬────────────────────────────────────┘
                     │ HTTP + SSE
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  HTTP Server (Echo Router)              │
│  ┌──────────────┬──────────────┬────────────────────┐  │
│  │ Health Check │  MCP Handler │  Metrics (/metrics)│  │
│  └──────────────┴──────────────┴────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┼────────────┬──────────────┐
        ▼            ▼            ▼              ▼
┌──────────────┐ ┌─────────┐ ┌───────────┐ ┌─────────┐
│  Checkpoint  │ │Remedia- │ │ Pre-Fetch │ │ Secret  │
│   Service    │ │  tion   │ │  Engine   │ │Scrubbing│
└──────┬───────┘ └────┬────┘ └─────┬─────┘ └────┬────┘
       │              │            │            │
       └──────────────┴────────────┴────────────┘
                      ▼
         ┌────────────────────────────┐
         │   Vector Core (langchaingo)│
         │  ┌──────────┬────────────┐ │
         │  │ Embedding│   Qdrant   │ │
         │  │ Provider │  VectorDB  │ │
         │  └──────────┴────────────┘ │
         └────────────────────────────┘
```

### Security Architecture

**Multi-Tenant Isolation**:
- Database-per-project physical isolation
- Owner-scoped collections (SHA256 project hash)
- No cross-project data leakage
- Secret scrubbing at 5 layers

**Authentication**:
- Bearer token authentication
- Constant-time comparison
- Token stored with 0600 permissions

**Network Security**:
- HTTP server (local-only by default)
- Health checks without authentication
- MCP endpoints require bearer token (future)

### Performance Characteristics

- **Search Latency**: <100ms (semantic + hybrid)
- **Pre-Fetch Execution**: <2s (3 rules parallel)
- **Cache Hit Latency**: <10ms (in-memory)
- **Token Savings**: 20-30% (with pre-fetch enabled)
- **Memory Usage**: ~500MB baseline + ~100KB per cached project

### Observability

- **Traces**: OpenTelemetry distributed tracing
- **Metrics**: Prometheus (pre-fetch, MCP, services)
- **Logs**: Structured logging with zap (INFO/DEBUG/WARN/ERROR)
- **Health**: `/health` endpoint (status, version, uptime)

## Documentation

### User Guides
- **[Migration Guide (v2→v3)](docs/guides/MIGRATION-V2-TO-V3.md)** - Step-by-step upgrade instructions
- **[Pre-Fetch User Guide](docs/guides/PREFETCH-USER-GUIDE.md)** - Configuration and troubleshooting
- **[TEI Deployment](docs/guides/TEI-DEPLOYMENT.md)** - Local embeddings setup (no quotas!)
- **[Development Workflow](docs/guides/DEVELOPMENT-WORKFLOW.md)** - Development setup and workflow

### Developer Documentation
- **[CLAUDE.md](CLAUDE.md)** - Development guide and project overview
- **[v3 Specification](docs/specs/v3-rebuild/SPEC.md)** - Complete 0.9.0-rc-1 architecture
- **[Pre-Fetch Engine Design](docs/plans/2025-01-15-prefetch-engine-design.md)** - Pre-fetch technical design
- **[Package Guidelines](pkg/CLAUDE.md)** - Package-level documentation

## Development

**Prerequisites**
- Go 1.24+
- make

**Setup Pre-commit Hooks**

```bash
make pre-commit-install  # Install pre-commit hooks (recommended)
```

**Build**

```bash
make build-all        # Build both contextd and ctxd
make test             # Run tests
make test-coverage    # Generate coverage report
make lint             # Run linters
```

**Development Workflow**
1. Research first (see [RESEARCH-FIRST-POLICY.md](docs/RESEARCH-FIRST-POLICY.md))
2. Test-driven development (TDD required, ≥80% coverage)
3. Code review and quality gates
4. Documentation updates
5. Release automation via GitHub Actions

**Testing Philosophy**
- Unit tests: ≥80% coverage required
- Integration tests: All MCP tools and API endpoints
- Regression tests: Every bug gets a test
- Performance benchmarks: Track improvements

## Security

All releases include cryptographic verification:

- **SHA256 Checksums** - Verify download integrity
- **Cosign Signatures** - Coming in Issue #48
- **SBOM (SPDX format)** - Software bill of materials
- **Vulnerability Scanning** - Automated with Grype

```bash
# Verify checksums
sha256sum -c checksums.txt

# Verify signatures (coming soon)
cosign verify --key cosign.pub contextd
```

## Contributing

We welcome contributions! Please see our development guidelines:

1. **Research First** - Document SDK research before implementation
2. **TDD Required** - Write tests before code (≥80% coverage)
3. **Code Review** - All PRs require review
4. **Issue Tracking** - Create issues for bugs and features
5. **Conventional Commits** - Use semantic commit messages

See [CLAUDE.md](CLAUDE.md) for detailed development workflow.

## Monitoring

contextd includes production-grade observability:

**Grafana Dashboards**
- contextd Overview (MCP tools, API, database, embeddings)
- Testing & Quality (coverage, bugs, regressions)
- Agents & Skills (performance, usage, time saved)

**Metrics**
- Request duration, count, status codes
- Vector database performance
- Embedding generation time
- Cache hit rates

**Start Monitoring Stack**

```bash
docker-compose up -d

# Access dashboards
# Grafana: http://localhost:3001 (admin/admin)
# Jaeger: http://localhost:16686
# VictoriaMetrics: http://localhost:8428
```

See [MONITORING-SETUP.md](docs/MONITORING-SETUP.md) for complete setup.

## Roadmap

### 0.9.0-rc-1 (Current - January 2025)
- ✅ HTTP server with Echo router
- ✅ Pre-fetch engine (git-centric, deterministic)
- ✅ Secret scrubbing (Gitleaks integration)
- ✅ OpenTelemetry observability (traces + metrics)
- ✅ Modular service architecture
- ✅ YAML configuration system
- ✅ SSE streaming for long operations
- ✅ Comprehensive test coverage (≥80%)

### v3.1 (Q1 2025)
- Skills management system (full implementation)
- Index repository handler (semantic code search)
- Grafana dashboards for pre-fetch metrics
- Enhanced error remediation patterns
- Performance optimizations

### v3.2 (Q2 2025)
- Team-aware architecture (multi-user support)
- Org-level knowledge sharing
- Advanced RBAC (role-based access control)
- Enhanced pre-fetch rules (ML-based)

### Future
- OAuth/SSO integration
- Multi-org support
- Advanced analytics dashboard
- Homebrew distribution
- Plugin system for custom rules

See [GitHub Issues](https://github.com/fyrsmithlabs/contextd/issues) for detailed backlog.

## Credits and Attribution

This project incorporates patterns and methodologies from the following excellent projects:

### Superpowers Plugin

Our skill authoring methodology is based on patterns from the **[superpowers plugin](https://github.com/superpowers-labs/superpowers)** by @dmarx and contributors.

- **Skill Creation**: TDD approach to documentation (RED-GREEN-REFACTOR)
- **Testing Methodology**: Pressure scenarios and rationalization tables
- **Claude Search Optimization (CSO)**: Discovery and keyword patterns
- **Documentation**: See [docs/specs/skills/SKILL-AUTHORING.md](docs/specs/skills/SKILL-AUTHORING.md)

Thank you to the superpowers community for developing and sharing these proven patterns!

## License

MIT License - see [LICENSE.md](LICENSE.md) for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/fyrsmithlabs/contextd/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fyrsmithlabs/contextd/discussions)
- **Documentation**: [docs/](docs/)

---

Built with ❤️ by Fyrsmith Labs for Claude Code users who want persistent context and intelligent assistance.
