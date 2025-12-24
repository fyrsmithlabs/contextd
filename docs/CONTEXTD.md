# contextd

A shared knowledge layer for AI agents providing cross-session memory, context persistence, and error pattern tracking.

contextd is an MCP server that gives AI agents like Claude Code persistent memory across sessions. It learns from successes and failures, saves context for resumption, and tracks error fixes so agents can learn from past solutions. All responses are scrubbed for secrets using gitleaks.

---

## Quick Start

### Docker (Recommended)

```bash
# Run contextd as MCP server
docker run -i --rm \
  -v contextd-data:/data \
  ghcr.io/fyrsmithlabs/contextd:latest --mcp
```

### Homebrew

```bash
brew tap fyrsmithlabs/tap
brew install contextd
contextd --mcp
```

### Build from Source

```bash
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd
go build -o contextd ./cmd/contextd
./contextd --mcp
```

---

## MCP Tools

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant past strategies/learnings |
| `memory_record` | Save new learning from current session |
| `memory_feedback` | Rate memory helpfulness (adjusts confidence) |
| `checkpoint_save` | Save session state for later resumption |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from checkpoint (summary/context/full) |
| `remediation_search` | Find fixes for error patterns |
| `remediation_record` | Record a new error fix |
| `repository_index` | Index repository for semantic search |
| `repository_search` | Semantic search over indexed code |
| `troubleshoot_diagnose` | AI-powered error diagnosis |

---

## Configuration

### Environment Variables

**Core:**
| Variable | Default | Description |
|----------|---------|-------------|
| `CONTEXTD_VECTORSTORE_PROVIDER` | `chromem` | Vector store (`chromem` or `qdrant`) |
| `CONTEXTD_VECTORSTORE_CHROMEM_PATH` | `~/.local/share/contextd` | Data directory |
| `CONTEXTD_EMBEDDINGS_PROVIDER` | `fastembed` | Embeddings (`fastembed` or `tei`) |
| `CONTEXTD_EMBEDDINGS_MODEL` | `all-MiniLM-L6-v2` | Embedding model |

**Hooks:**
| Variable | Default | Description |
|----------|---------|-------------|
| `CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR` | `false` | Auto-save before `/clear` |
| `CONTEXTD_AUTO_RESUME_ON_START` | `true` | Offer resume on start |
| `CONTEXTD_CHECKPOINT_THRESHOLD` | `70` | Context % for threshold hook |

**Telemetry:**
| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_SDK_DISABLED` | `true` | Disable OpenTelemetry |
| `TELEMETRY_ENABLED` | `false` | Enable telemetry |

### Config File

Default location: `~/.config/contextd/config.yaml`

```yaml
vectorstore:
  provider: chromem
  chromem:
    path: ~/.local/share/contextd

embeddings:
  provider: fastembed
  model: all-MiniLM-L6-v2

hooks:
  auto_checkpoint_on_clear: true
  auto_resume_on_start: true
  checkpoint_threshold_percent: 70

server:
  port: 9090
```

---

## Running Modes

### MCP Mode (Claude Code integration)

```bash
contextd --mcp
```

Runs as stdio MCP server for Claude Code integration.

### HTTP Mode (standalone)

```bash
contextd --http-port 9090
```

Runs HTTP server with endpoints:
- `GET /api/v1/status` - Health check
- `POST /api/v1/threshold` - Trigger context threshold
- `POST /api/v1/scrub` - Scrub secrets from text

---

## Architecture

```
Claude Code / AI Agent
        |
        | MCP Protocol (stdio)
        v
+-------------------+
|  contextd Server  |
|  +-------------+  |
|  | MCP Handler |  |
|  +------+------+  |
|         |         |
|  +------v------+  |
|  | Services    |  |
|  | - Memory    |  |
|  | - Checkpoint|  |
|  | - Remediate |  |
|  | - Repository|  |
|  +------+------+  |
|         |         |
|  +------v------+  |
|  | VectorStore |  |
|  | (chromem)   |  |
|  +-------------+  |
+-------------------+
```

**Key design decisions:**
- **chromem default**: Embedded vector store, no external dependencies
- **FastEmbed default**: Local ONNX embeddings, no API calls
- **Direct calls**: Simplified architecture without gRPC complexity
- **Secret scrubbing**: gitleaks SDK on all responses

---

## Claude Code Integration

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp"
      ]
    }
  }
}
```

---

## Documentation

- [Architecture Overview](./architecture.md) - Detailed component descriptions
- [Hook Setup Guide](./HOOKS.md) - Claude Code lifecycle integration
- [Configuration Reference](./configuration.md) - All configuration options
- [Troubleshooting](./troubleshooting.md) - Common issues and fixes

---

## Links

- [GitHub Repository](https://github.com/fyrsmithlabs/contextd)
- [Docker Images](https://ghcr.io/fyrsmithlabs/contextd)
- [Issue Tracker](https://github.com/fyrsmithlabs/contextd/issues)
