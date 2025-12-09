# CLAUDE.md - contextd

**Status**: Active Development (Phase 5 complete, Phase 6 pending)
**Last Updated**: 2025-12-05

---

## ⚠️ CRITICAL: Contextd-First Search (Priority #1)

**ALWAYS use contextd MCP tools before filesystem search. ALWAYS.**

---

## ⚠️ CRITICAL: Update Claude Plugin on Changes (Priority #2)

**After ANY feature/fix/release, update the contextd-marketplace claude-plugin.**

When these occur:
- New feature added
- Release candidate (rc) tagged
- Bug fix affecting user behavior
- New skills/commands/agents
- MCP tool changes

**Action:** Update `contextd-marketplace` repo's claude-plugin to expose new capabilities to users. DO NOT skip this step.

**Required Order:**
1. ✅ `mcp__contextd__memory_search` - Search past learnings/strategies first
2. ✅ `mcp__contextd__remediation_search` - Check for known error fixes
3. ✅ Semantic search via indexed repository
4. ⏳ Only THEN use Grep/Glob/Read if contextd doesn't have what you need

**Why:** This project IS contextd. Use your own tools. Semantic search is faster, more accurate, and builds the knowledge base.

---

## What This Is

Simplified MCP server for AI agent memory and context management. Calls internal packages directly without gRPC complexity.

**Core Features:**
- ReasoningBank: Cross-session memory with confidence scoring
- Checkpoints: Context persistence and recovery
- Remediation: Error pattern tracking
- Secret scrubbing: gitleaks SDK on all tool responses
- Vectorstore: chromem (embedded, default) or Qdrant (external) with collection-per-project isolation
- Compression: Extractive, abstractive, and hybrid context compression
- Hooks: Lifecycle hooks for session management and auto-checkpoint

---

## Architecture

```
cmd/contextd/          # Entry point (stdio MCP server + HTTP server)
cmd/ctxd/              # CLI binary for manual operations
internal/
├── mcp/               # MCP server + tool handlers
├── http/              # HTTP API server (scrub, threshold, status endpoints)
├── reasoningbank/     # Cross-session memory (82% coverage)
├── checkpoint/        # Context snapshots
├── remediation/       # Error patterns
├── repository/        # Repository indexing + semantic search
├── vectorstore/       # Store interface (chromem default, Qdrant optional)
├── secrets/           # gitleaks scrubbing (97% coverage)
├── compression/       # Context compression (extractive, abstractive, hybrid)
├── hooks/             # Lifecycle hooks (session, clear, threshold)
├── services/          # Service registry pattern
├── config/            # Koanf configuration
├── logging/           # Zap + OTEL bridge
└── telemetry/         # OpenTelemetry
pkg/api/v1/            # Proto definitions (unused - simplified away)
```

---

## Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.25+ |
| MCP | github.com/modelcontextprotocol/go-sdk |
| Vector DB | chromem (default, embedded) or Qdrant (external) |
| Embeddings | FastEmbed (local ONNX) or TEI |
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
| `checkpoint_resume` | Checkpoint | Resume from checkpoint |
| `remediation_search` | Remediation | Find error fix patterns |
| `remediation_record` | Remediation | Record new fix |
| `repository_index` | Repository | Index repo for semantic search |
| `repository_search` | Repository | Semantic search over indexed code |
| `troubleshoot_diagnose` | Troubleshoot | AI-powered error diagnosis |

---

## Completed Phases

1. **Foundation** - config, logging, telemetry, entry point
2. **Core Services** - vectorstore, embeddings, checkpoint, remediation, repository, troubleshoot, project, secrets
3. **MCP Integration** - simplified server, tool handlers, scrubbing
4. **ReasoningBank** - memory package, MCP tools, distiller stub
5. **HTTP + ctxd CLI** - HTTP server with `/api/v1/scrub`, `/api/v1/threshold`, `/api/v1/status` endpoints; `ctxd` CLI binary

---

## Pending Phases

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
| `internal/vectorstore/interface.go` | Store interface definition |
| `internal/vectorstore/chromem.go` | chromem (embedded) implementation |
| `internal/vectorstore/factory.go` | Provider factory |
| `internal/embeddings/provider.go` | Embedding provider factory |
| `internal/embeddings/fastembed.go` | FastEmbed local ONNX embeddings |
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
