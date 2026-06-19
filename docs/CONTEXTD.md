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

### Memory
| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant past strategies/learnings |
| `memory_record` | Save new learning from current session |
| `memory_feedback` | Rate memory helpfulness (adjusts confidence) |
| `memory_outcome` | Report task success after using memory |
| `memory_consolidate` | Merge related memories into refined summaries |
| `memory_consolidate_session` | Consolidate specific memories by ID |

### Checkpoint
| Tool | Purpose |
|------|---------|
| `checkpoint_save` | Save session state for later resumption |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from checkpoint (summary/context/full) |

### Remediation
| Tool | Purpose |
|------|---------|
| `remediation_search` | Find fixes for error patterns |
| `remediation_record` | Record a new error fix |
| `remediation_feedback` | Rate whether a fix was helpful |

### Repository & Search
| Tool | Purpose |
|------|---------|
| `semantic_search` | Smart search with semantic understanding + grep fallback |
| `repository_index` | Index repository for semantic search |
| `repository_search` | Semantic search over indexed code |

### Context Folding
| Tool | Purpose |
|------|---------|
| `branch_create` | Create isolated context branch with token budget |
| `branch_return` | Return from branch with scrubbed results |
| `branch_status` | Check branch status and budget usage |

### Conversation
| Tool | Purpose |
|------|---------|
| `conversation_index` | Index Claude Code conversation files |
| `conversation_search` | Search indexed conversations |

### Utility
| Tool | Purpose |
|------|---------|
| `troubleshoot_diagnose` | AI-powered error diagnosis |
| `reflect_report` | Generate self-reflection report on memories |
| `reflect_analyze` | Analyze behavioral patterns in memories |

---

## MCP Resources

Resources expose contextd state as read-only, JSON MCP resources. All resource
content is secret-scrubbed (gitleaks) before return and tenant-scoped by the
`project_id` embedded in the URI. Resources fail-closed: a missing or invalid
project returns an error rather than another tenant's data or empty results.

| Resource URI | Purpose |
|--------------|---------|
| `contextd://help` | Documents the `contextd://` URI scheme |
| `contextd://{project_id}/memories` | Recent memories (collection) |
| `contextd://{project_id}/memory/{id}` | A single memory |
| `contextd://{project_id}/checkpoints` | Checkpoint list |
| `contextd://{project_id}/checkpoint/{id}` | A single checkpoint |
| `contextd://{project_id}/remediation/{id}` | A single remediation |
| `contextd://{project_id}/remediations{?query}` | Remediation search (`query` required) |

---

## MCP Prompts

Prompts are static workflow templates that mirror the bundled slash commands,
guiding an agent through a common contextd task.

| Prompt | Argument | Purpose |
|--------|----------|---------|
| `contextd_checkpoint` | `summary` (optional) | Save a resumable checkpoint |
| `contextd_remember` | `content` (optional) | Record a learning |
| `contextd_diagnose` | `error` (required) | Diagnose an error and find a known fix |
| `contextd_resume` | `checkpoint_id` (optional) | List/resume a checkpoint |
| `contextd_status` | _(none)_ | Show memories/checkpoints/project status |
| `contextd_search` | `query` (required) | Search memories/remediations/code |

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

The MCP server identifies itself as `contextd` (the `serverInfo` name) over
either of two transports.

### MCP Mode — stdio (Claude Code integration)

```bash
contextd --mcp
```

Runs as a stdio MCP server for local Claude Code integration.

### MCP Mode — Streamable HTTP (remote hosting)

```bash
contextd --mcp-http-port 9095            # optionally --mcp-http-host 0.0.0.0
```

Runs the MCP server over the Streamable HTTP transport for remote hosting. It is
a separate Echo server that exposes the MCP endpoint at `/mcp` plus a `/health`
endpoint. Optional bearer authentication is configured via `--mcp-http-token` or
the `CONTEXTD_MCP_HTTP_TOKEN` environment variable; when unset, the server runs
unauthenticated (intended for localhost/testing) and logs a warning.

This Streamable HTTP MCP transport is separate from the REST `--http-port` API
described below.

### HTTP Mode (standalone REST API)

```bash
contextd --http-port 9090
```

Runs the REST HTTP server with endpoints:
- `GET /api/v1/status` - Health check
- `POST /api/v1/threshold` - Trigger context threshold
- `POST /api/v1/scrub` - Scrub secrets from text

---

## Agent-Swarm Notifications

When multiple agents connect to a single contextd server over the Streamable
HTTP transport (stateful sessions), an agent can `resources/subscribe` to a
collection URI (for example `contextd://{project_id}/memories`). When any agent
records a memory, remediation, or checkpoint, subscribers receive a
`notifications/resources/updated` event and can re-read the shared knowledge,
keeping a swarm in sync over one knowledge layer.

See [Agent-Swarm Notifications spec](./spec/mcp-protocol/notifications-agent-swarm.md).

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
|  | - Branching |  |
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

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "contextd-data:/data",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

---

## Documentation

- [Onboarding Guide](./ONBOARDING.md) - Getting started with contextd
- [Architecture Overview](./architecture.md) - Detailed component descriptions
- [Configuration Reference](./configuration.md) - All configuration options
- [Hook Setup Guide](./HOOKS.md) - Claude Code lifecycle integration
- [MCP Tools API Reference](./api/mcp-tools.md) - Complete tool documentation
- [Docker Guide](./DOCKER.md) - Running contextd in Docker
- [Troubleshooting](./troubleshooting.md) - Common issues and fixes
- [Versioning](./VERSIONING.md) - Version management
- [Releasing](./RELEASING.md) - Creating releases

---

## Links

- [GitHub Repository](https://github.com/fyrsmithlabs/contextd)
- [Docker Images](https://ghcr.io/fyrsmithlabs/contextd)
- [Issue Tracker](https://github.com/fyrsmithlabs/contextd/issues)
