# contextd-v2 Consolidation Design

**Date**: 2025-11-29
**Status**: Approved
**Author**: Design session with Claude

---

## Summary

Consolidate three contextd projects into a single, focused MCP server. Cherry-pick working code from contextd (Fyrsmith), apply specs from contextd-reasoning, discard unnecessary complexity.

---

## Problem

Three divergent contextd projects:

| Project | Status | Issue |
|---------|--------|-------|
| contextd (Fyrsmith) | Production RC | Feature sprawl, unclear docs |
| contextd-main (axyzlabs) | Production | Different focus (context folding) |
| contextd-reasoning | Design phase | Over-engineered, mostly specs |

Result: Confusion, duplicated effort, no clear path forward.

---

## Solution

Create `contextd-v2/` - a slim, focused MCP server.

**Core capabilities:**
- MCP tools for Claude Code (stdio transport)
- ReasoningBank (cross-session memory)
- Checkpoints (session persistence)
- Remediation (error pattern storage)
- Project indexing (semantic search over codebase)
- Multi-project isolation (collection-per-project)
- Secret scrubbing (on all tool responses + HTTP endpoint)

**Explicitly excluded:**
- gRPC/HTTP dual-protocol
- Process isolation (seccomp/namespaces)
- safe_* wrapper tools
- Skills system
- Git operations
- Enterprise multi-org/team hierarchy

---

## Architecture

```
Claude Code
    │
    │ MCP (stdio)
    ▼
┌─────────────────────────────────────┐
│         contextd-v2                 │
│  ├── MCP Server (tool handlers)     │
│  ├── HTTP Server (ctxd + hooks)     │
│  ├── Secret Scrubber (gitleaks)     │
│  ├── ReasoningBank                  │
│  ├── Checkpoint/Remediation         │
│  ├── Repository Indexer             │
│  └── Qdrant Client (interface)      │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│         Qdrant                      │
│  └── Collection per project         │
└─────────────────────────────────────┘
```

**Transports:**
- MCP stdio: Claude Code integration
- HTTP: ctxd CLI + hook callbacks

**Database:**
- Qdrant (single provider)
- Interface-based design (swappable later)

---

## Package Structure

```
contextd-v2/
├── cmd/
│   ├── contextd/          # MCP server (stdio)
│   └── ctxd/              # CLI tool
│
├── internal/
│   ├── config/            # Koanf-based configuration
│   ├── logging/           # Zap structured logging
│   ├── telemetry/         # OpenTelemetry (VITAL)
│   │
│   ├── mcp/               # MCP protocol + tool handlers
│   ├── server/            # HTTP server
│   │
│   ├── vectorstore/       # Qdrant client (interface-based)
│   ├── embeddings/        # Embedding generation
│   │
│   ├── checkpoint/        # Session persistence
│   ├── remediation/       # Error pattern storage
│   ├── reasoningbank/     # Cross-session memory (NEW)
│   ├── repository/        # Project indexing
│   │
│   ├── secrets/           # gitleaks integration
│   ├── project/           # Project isolation
│   └── troubleshoot/      # Troubleshooting helpers
│
├── pkg/
│   └── types/             # Shared types (exported)
│
├── docs/
│   ├── CONTEXTD.md
│   └── spec/
│
└── deploy/
    └── docker-compose.yaml
```

14 internal packages. Clean and focused.

---

## MCP Tools

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant past strategies |
| `memory_record` | Save a learning explicitly |
| `memory_feedback` | Rate a memory (helpful/not) |
| `checkpoint_save` | Save session state |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Restore from checkpoint |
| `remediation_search` | Find past error fixes |
| `remediation_record` | Save an error fix |
| `repository_search` | Semantic search over codebase |
| `repository_index` | Index/reindex project |
| `troubleshoot` | Guided troubleshooting |

All MCP tool responses pass through gitleaks scrubbing before return.

---

## HTTP Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /api/scrub` | Scrub secrets from text |
| `GET /api/health` | Health check |
| `GET /api/projects` | List projects |
| `POST /api/projects` | Create/register project |

---

## ReasoningBank

Cross-session memory system.

**Schema:**
```go
type Memory struct {
    ID          string
    ProjectID   string
    Title       string
    Content     string
    Outcome     string    // "success" | "failure"
    Confidence  float64   // 0.0-1.0
    UsageCount  int
    Tags        []string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Flow:**
1. Session start → `memory_search` (top 3-5, confidence > 0.7)
2. During session → `memory_record` explicit learnings
3. Session end → Async distillation extracts learnings
4. Feedback → `memory_feedback` adjusts confidence

**Initial confidence:**
- Explicit `memory_record`: 0.8
- Distillation-extracted: 0.6

---

## Secret Scrubbing

Two integration points:

### 1. MCP Tool Responses
All tool output passes through gitleaks before returning to Claude. Built into contextd-v2.

### 2. Claude Code Hooks
For native tools (Bash, Read), hooks call contextd:

```json
{
  "hooks": {
    "postToolExecution": {
      "command": "curl -s -X POST http://localhost:9090/api/scrub -d @-",
      "tools": ["Bash", "Read"],
      "replaceOutput": true
    }
  }
}
```

Hook fires after tool execution, before output enters context. Scrubbed output replaces original.

---

## Multi-Project Isolation

Collection-per-project in Qdrant:

```
{project_id}_memories
{project_id}_checkpoints
{project_id}_remediations
{project_id}_codebase
```

No org/team hierarchy. Single user, multiple projects.

---

## Implementation Phases

### Phase 1: Foundation
- Create `contextd-v2/` directory structure
- Copy from contextd (Fyrsmith): `config`, `logging`, `telemetry`
- Set up `go.mod`
- Basic `cmd/contextd` entry point

### Phase 2: Core Services
- Port `vectorstore` with interface from contextd-reasoning specs
- Port `embeddings` from contextd (Fyrsmith)
- Port `checkpoint`, `remediation`, `repository`, `troubleshoot`
- Add `project` package for isolation

### Phase 3: MCP Integration
- Port `mcp` package from contextd (Fyrsmith)
- Wire tools to services
- Add gitleaks scrubbing to all responses

### Phase 4: ReasoningBank
- Implement `reasoningbank` package (new)
- Add `memory_*` MCP tools
- Basic async distillation

### Phase 5: HTTP + ctxd
- Port `server` package
- Add `/api/scrub` endpoint
- Port/simplify `ctxd` CLI

### Phase 6: Documentation
- Clean CONTEXTD.md
- Port relevant specs
- Hook setup guide

---

## Source Material

**Cherry-pick from contextd (Fyrsmith):**
- `pkg/config` → `internal/config`
- `pkg/logging` → `internal/logging`
- `pkg/telemetry` → `internal/telemetry`
- `pkg/mcp` → `internal/mcp`
- `pkg/checkpoint` → `internal/checkpoint`
- `pkg/remediation` → `internal/remediation`
- `pkg/repository` → `internal/repository`
- `pkg/troubleshoot` → `internal/troubleshoot`
- `pkg/embeddings` → `internal/embeddings`
- `pkg/server` → `internal/server`
- `pkg/secrets` → `internal/secrets`
- `cmd/contextd`, `cmd/ctxd`

**Use specs from contextd-reasoning:**
- `docs/spec/interface/` → vectorstore interface design
- `docs/spec/reasoning-bank/` → ReasoningBank design
- `docs/spec/collection-architecture/` → project isolation

**Discard:**
- contextd (Fyrsmith): `pkg/skills`, `pkg/git`, `pkg/auth`, `pkg/security`, `pkg/collections`
- contextd-reasoning: gRPC services, process isolation, safe_* tools
- contextd-main: context folding (maybe later), Milvus support

---

## Success Criteria

- [ ] MCP tools work in Claude Code
- [ ] ReasoningBank stores/retrieves memories
- [ ] Checkpoints save/resume sessions
- [ ] Secret scrubbing on all tool responses
- [ ] `/api/scrub` endpoint works with hooks
- [ ] Multi-project isolation functional
- [ ] Telemetry operational
- [ ] Clean documentation

---

## Next Steps

1. Create `contextd-v2/` directory
2. Begin Phase 1 (Foundation)
3. Iterate through phases
4. When stable, replace original `contextd/` with v2
