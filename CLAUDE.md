# CLAUDE.md - contextd

**Status**: Active Development (Phase 4 complete, Phase 5-6 pending)
**Last Updated**: 2025-11-30

---

## What This Is

Simplified MCP server for AI agent memory and context management. Calls internal packages directly without gRPC complexity.

**Core Features:**
- ReasoningBank: Cross-session memory with confidence scoring
- Checkpoints: Context persistence and recovery
- Remediation: Error pattern tracking
- Secret scrubbing: gitleaks SDK on all tool responses
- Vectorstore: Qdrant with collection-per-project isolation
- Compression: Extractive, abstractive, and hybrid context compression
- Hooks: Lifecycle hooks for session management and auto-checkpoint

---

## Architecture

```
cmd/contextd/          # Entry point (stdio MCP server)
internal/
├── mcp/               # MCP server + tool handlers
├── reasoningbank/     # Cross-session memory (82% coverage)
├── checkpoint/        # Context snapshots
├── remediation/       # Error patterns
├── vectorstore/       # Qdrant interface
├── secrets/           # gitleaks scrubbing (97% coverage)
├── compression/       # Context compression (extractive, abstractive, hybrid)
├── hooks/             # Lifecycle hooks (session, clear, threshold)
├── config/            # Koanf configuration
├── logging/           # Zap + OTEL bridge
└── telemetry/         # OpenTelemetry
pkg/api/v1/            # Proto definitions (unused - simplified away)
```

---

## Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24+ |
| MCP | github.com/modelcontextprotocol/go-sdk |
| Vector DB | Qdrant (gRPC client) |
| Config | Koanf |
| Logging | Zap |
| Telemetry | OpenTelemetry |

---

## MCP Tools Registered

| Tool | Service | Purpose |
|------|---------|---------|
| `memory_search` | ReasoningBank | Find relevant past strategies |
| `memory_record` | ReasoningBank | Save new memory explicitly |
| `memory_feedback` | ReasoningBank | Rate memory helpfulness |
| `checkpoint_save` | Checkpoint | Save context snapshot |
| `checkpoint_list` | Checkpoint | List available checkpoints |
| `checkpoint_load` | Checkpoint | Resume from checkpoint |
| `remediation_search` | Remediation | Find error fix patterns |
| `remediation_record` | Remediation | Record new fix |

---

## Completed Phases

1. **Foundation** - config, logging, telemetry, entry point
2. **Core Services** - vectorstore, embeddings, checkpoint, remediation, repository, troubleshoot, project, secrets
3. **MCP Integration** - simplified server, tool handlers, scrubbing
4. **ReasoningBank** - memory package, MCP tools, distiller stub

---

## Pending Phases

### Phase 5: HTTP + ctxd CLI
- HTTP server for `/api/scrub` endpoint (Claude Code hooks)
- `ctxd` CLI binary for manual operations
- Hook integration guide

### Phase 6: Documentation
- CONTEXTD.md briefing doc
- Spec updates for new architecture
- Claude Code hook setup guide

---

## Running Tests

```bash
go test ./... -cover
```

All packages have tests. Key coverage:
- secrets: 97%
- project: 97%
- reasoningbank: 82%
- remediation: 82%

---

## Git History

- `main` - Current simplified architecture (this code)
- `old` - Previous v1 architecture (preserved)

Migrated from `contextd-v2` on 2025-11-30.

---

## Key Files

| File | Purpose |
|------|---------|
| `internal/mcp/server.go` | MCP server setup |
| `internal/mcp/tools.go` | Tool registration |
| `internal/reasoningbank/service.go` | Memory operations |
| `internal/secrets/scrubber.go` | gitleaks integration |
| `cmd/contextd/main.go` | Entry point |

---

## Common Commands

```bash
# Run server (stdio transport)
go run ./cmd/contextd

# Run tests
go test ./... -v

# Build binary
go build -o contextd ./cmd/contextd
```
