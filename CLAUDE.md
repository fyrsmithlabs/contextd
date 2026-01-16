# CLAUDE.md - contextd

**Status**: Active Development (Phase 6 complete)
**Last Updated**: 2026-01-06

---

## ⚠️ CRITICAL: Contextd-First Search (Priority #1)

**ALWAYS use contextd MCP tools before filesystem search. ALWAYS.**

### MANDATORY Pre-Flight Check

Before ANY codebase exploration or task work:

```
1. mcp__contextd__semantic_search(query, project_path: ".")
   → Semantic search with automatic grep fallback
   → NEVER skip this - it's your first tool for code lookup

2. mcp__contextd__memory_search(project_id, query)
   → Check past learnings and solutions
```

**DO NOT use Read, Grep, or Glob until AFTER semantic_search.**

---

## ⚠️ CRITICAL: GitHub MCP over gh CLI (Priority #2)

**ALWAYS use GitHub MCP tools (`mcp__github__*`) instead of `gh` CLI.**

| Task | Use This | NOT This |
|------|----------|----------|
| Create PR | `mcp__github__create_pull_request` | `gh pr create` |
| List issues | `mcp__github__list_issues` | `gh issue list` |
| Get PR details | `mcp__github__get_pull_request` | `gh pr view` |
| Create branch | `mcp__github__create_branch` | `gh repo clone && git checkout -b` |
| Search code | `mcp__github__search_code` | `gh search code` |

**Why:** MCP tools are structured, faster, and don't require shell escaping.

---

## ⚠️ CRITICAL: Update Claude Plugin on Changes (Priority #3)

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
- Vectorstore: chromem (embedded, default) or Qdrant (external) with payload-based tenant isolation
- Compression: Extractive, abstractive, and hybrid context compression
- Hooks: Lifecycle hooks for session management and auto-checkpoint
- Context-Folding: Isolate complex sub-tasks with dedicated token budgets

---

## Context-Folding

Context-Folding provides **active context management** within a single agent session using `branch_create`, `branch_return`, and `branch_status` MCP tools. This enables **90%+ context compression** by isolating subtask reasoning from the main context.

### How It Works

When you need to perform a complex sub-task (file exploration, research, trial-and-error debugging), create a **branch** with its own token budget. The branch executes in isolation, and only a scrubbed summary returns to the main context.

```
Main Context (16K tokens used)
    │
    ├─► Branch: "Search 10 files for function definition" (4K budget)
    │       • Reads 10 files (3.5K tokens consumed)
    │       • Finds target function
    │       • Returns: "Function found in src/auth.go:42"
    │       ✓ Main context grows by ~50 tokens (not 3.5K!)
    │
    └─► Main context continues with clean, focused state
```

### Use Cases

| Scenario | Without Context-Folding | With Context-Folding |
|----------|------------------------|---------------------|
| **File Exploration** | Agent reads 10 files into main context (10K+ tokens) | Branch reads files, returns summary only (~200 tokens) |
| **API Research** | Agent fetches 5 docs, clutters context (8K+ tokens) | Branch fetches docs, returns relevant excerpts (~500 tokens) |
| **Trial-and-Error Debugging** | Agent tries 3 fixes, all attempts stay in context | Branch tries fixes in isolation, returns only successful approach |

### When to Use

✅ **Use context-folding when:**
- Exploring multiple files to find a specific function/class
- Researching API documentation or web sources
- Trying multiple debugging approaches
- Running experiments that might fail
- Any task where the **process is verbose** but the **result is concise**

❌ **Don't use context-folding when:**
- Implementing a single file change (no benefit)
- User needs to see the full reasoning (defeats the purpose)
- Task is already simple and focused

### Tools

| Tool | Purpose | Example |
|------|---------|---------|
| `branch_create` | Create isolated branch with token budget | `branch_create("Find auth function", "Search src/ for authenticate()", 4000)` |
| `branch_return` | Return from branch with scrubbed results | `branch_return("Found in src/auth.go:42")` |
| `branch_status` | Check branch budget and status | `branch_status()` → `{used: 3500, budget: 4000, depth: 1}` |

### Security & Features

- **Secret Scrubbing**: All `branch_return()` content is automatically scrubbed for secrets using gitleaks
- **Budget Enforcement**: Branches are force-terminated when budget is exhausted
- **Nested Limits**: Max 3 levels of nesting (configurable)
- **Rate Limiting**: Max 10 concurrent branches per session
- **Memory Injection**: Branches can optionally inject relevant ReasoningBank memories to provide context (20% of branch budget)

### See Also

- Specification: `docs/spec/context-folding/SPEC.md`
- Architecture: `docs/spec/context-folding/ARCH.md`
- Research Paper: [arXiv:2510.11967](https://arxiv.org/abs/2510.11967) (ByteDance, Oct 2025)

---

## Multi-Tenancy Architecture

contextd uses **payload-based tenant isolation** as the default multi-tenant strategy. All documents are stored in shared collections with automatic tenant filtering.

### Tenant Context

All operations require tenant context via Go's `context.Context`:

```go
import "github.com/fyrsmithlabs/contextd/internal/vectorstore"

// Create tenant-scoped context (REQUIRED for all operations)
ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
    TenantID:  "org-123",      // Required: Organization/user identifier
    TeamID:    "platform",     // Optional: Team scope
    ProjectID: "contextd",     // Optional: Project scope
})

// All operations automatically filtered by tenant
results, err := store.Search(ctx, "query", 10)
```

### Isolation Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `PayloadIsolation` | Single collection with metadata filtering | **Default, recommended** |
| `FilesystemIsolation` | Separate database per tenant/project | Legacy, migration path available |
| `NoIsolation` | No tenant filtering | **Testing only** |

### Security Guarantees

| Behavior | Description |
|----------|-------------|
| **Fail-closed** | Missing tenant context returns `ErrMissingTenant`, not empty results |
| **Filter injection blocked** | User-provided `tenant_id`/`team_id`/`project_id` filters are rejected with `ErrTenantFilterInUserFilters` |
| **Metadata enforced** | Tenant fields always set from authenticated context, never from user input |

### Configuration

```go
// PayloadIsolation is the default - no explicit config needed
config := vectorstore.ChromemConfig{
    Path:              "/data/vectorstore",
    DefaultCollection: "memories",
    VectorSize:        384,
    // Isolation: vectorstore.NewPayloadIsolation(), // Default if not specified
}

// Or explicitly set isolation mode
config.Isolation = vectorstore.NewPayloadIsolation()    // Recommended
config.Isolation = vectorstore.NewFilesystemIsolation() // Legacy
config.Isolation = vectorstore.NewNoIsolation()         // Testing only!
```

### Key Types

| Type | Purpose |
|------|---------|
| `TenantInfo` | Holds TenantID, TeamID, ProjectID |
| `ContextWithTenant()` | Adds tenant info to context |
| `TenantFromContext()` | Extracts tenant info (returns `ErrMissingTenant` if absent) |
| `ApplyTenantFilters()` | Merges user filters with tenant filters, rejects injection attempts |
| `IsolationMode` | Interface for isolation strategies |

### See Also

- Security spec: `docs/spec/vector-storage/security.md`
- Migration guide: `docs/migration/payload-filtering.md`
- Package docs: `internal/vectorstore/README.md`

---

## Architecture

### Visual Overview

```
+-----------------------------------------------------------------------+
|                      Claude Code / AI Agent                           |
|                               |                                       |
|                       MCP Protocol (stdio)                            |
|                               |                                       |
|  +----------------------------------------------------------------+   |
|  |                         contextd                                |   |
|  |                                                                 |   |
|  |  +-----------------------------------------------------------+  |   |
|  |  |                    MCP Server Layer                        |  |   |
|  |  |  +----------+ +----------+ +----------+ +---------------+  |  |   |
|  |  |  | Memory   | |Checkpoint| |Remediate | | Repository/   |  |  |   |
|  |  |  | Tools    | | Tools    | | Tools    | | Troubleshoot  |  |  |   |
|  |  |  +----+-----+ +----+-----+ +----+-----+ +-------+-------+  |  |   |
|  |  |  | Context- |                                             |  |  |   |
|  |  |  | Folding  |  branch_create, branch_return, branch_status |  |  |   |
|  |  |  +----------+                                             |  |  |   |
|  |  +-------|------------|------------|---------------|----------+  |   |
|  |          |            |            |               |             |   |
|  |  +-------v------------v------------v---------------v----------+  |   |
|  |  |                  Service Registry                          |  |   |
|  |  |  +-------------+ +-------------+ +-------------+ +--------+ |  |   |
|  |  |  | Reasoning   | | Checkpoint  | | Remediation | | Repo   | |  |   |
|  |  |  | Bank        | | Service     | | Service     | | Service| |  |   |
|  |  |  +------+------+ +------+------+ +------+------+ +----+---+ |  |   |
|  |  +---------|---------------|---------------|--------------|----+  |   |
|  |            |               |               |              |       |   |
|  |  +---------v---------------v---------------v--------------v----+  |   |
|  |  |                Infrastructure Layer                          |  |   |
|  |  |  +-------------+  +-------------+  +---------------------+   |  |   |
|  |  |  | VectorStore |  | Embeddings  |  |   Secret Scrubber   |   |  |   |
|  |  |  |  (chromem   |  | (FastEmbed  |  |     (gitleaks)      |   |  |   |
|  |  |  |   default)  |  | local ONNX) |  +---------------------+   |  |   |
|  |  |  +-------------+  +-------------+                            |  |   |
|  |  |  | Qdrant opt. |                                             |  |   |
|  |  |  +-------------+                                             |  |   |
|  |  +--------------------------------------------------------------+  |   |
|  |                                                                    |   |
|  |  +--------------------------------------------------------------+  |   |
|  |  |                      Hooks Manager                           |  |   |
|  |  |  session_start, session_end, before_clear, after_clear,      |  |   |
|  |  |  context_threshold (auto-checkpoint, auto-resume)             |  |   |
|  |  +--------------------------------------------------------------+  |   |
|  +--------------------------------------------------------------------+   |
|                               |                                          |
|                               v                                          |
|  +--------------------------------------------------------------------+   |
|  |              Local Storage (~/.local/share/contextd)                |   |
|  +--------------------------------------------------------------------+   |
+------------------------------------------------------------------------+
```

### Directory Structure

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
| `memory_outcome` | ReasoningBank | Report task success/failure after using memory |
| `checkpoint_save` | Checkpoint | Save context snapshot |
| `checkpoint_list` | Checkpoint | List available checkpoints |
| `checkpoint_resume` | Checkpoint | Resume from checkpoint |
| `remediation_search` | Remediation | Find error fix patterns |
| `remediation_record` | Remediation | Record new fix |
| `semantic_search` | Repository | Smart search with semantic understanding + grep fallback |
| `repository_index` | Repository | Index repo for semantic search |
| `repository_search` | Repository | Semantic search over indexed code |
| `troubleshoot_diagnose` | Troubleshoot | AI-powered error diagnosis |
| `branch_create` | Context-Folding | Create isolated context branch with token budget |
| `branch_return` | Context-Folding | Return from branch with scrubbed results |
| `branch_status` | Context-Folding | Get branch status and budget usage |
| `conversation_index` | Conversation | Index Claude Code conversation files |
| `conversation_search` | Conversation | Search indexed conversations |
| `reflect_report` | Reflection | Generate self-reflection report on memories and patterns |
| `reflect_analyze` | Reflection | Analyze behavioral patterns in memories |

---

## Completed Phases

1. **Foundation** - config, logging, telemetry, entry point
2. **Core Services** - vectorstore, embeddings, checkpoint, remediation, repository, troubleshoot, project, secrets
3. **MCP Integration** - simplified server, tool handlers, scrubbing
4. **ReasoningBank** - memory package, MCP tools, distiller stub
5. **HTTP + ctxd CLI** - HTTP server with `/api/v1/scrub`, `/api/v1/threshold`, `/api/v1/status` endpoints; `ctxd` CLI binary
6. **Documentation** - CONTEXTD.md briefing doc, spec updates for new architecture, Claude Code hook setup guide

---

## Temporal Workflows (Automation)

The repository includes Temporal-based workflows for internal automation tasks.

### Plugin Update Validation

**Location:** `internal/workflows/`

Automatically validates that Claude plugin files are updated when code changes require it.

**Triggers:**
- PR opened, synchronized, or reopened

**Process:**
1. Fetch PR file changes via GitHub API
2. Categorize files (code vs plugin vs other)
3. Validate plugin schemas if modified
4. Post reminder or success comment to PR

**Components:**
- **Workflow:** `PluginUpdateValidationWorkflow` (orchestration)
- **Activities:** GitHub API calls, file categorization, schema validation
- **Worker:** `cmd/plugin-validator/main.go` (executes workflows)
- **Webhook:** `cmd/github-webhook/main.go` (receives GitHub events)

**Running:**
```bash
# Set environment variables
export GITHUB_TOKEN=ghp_xxx
export GITHUB_WEBHOOK_SECRET=your_secret

# Start full stack
docker-compose -f deploy/docker-compose.temporal.yml up

# Access Temporal UI
open http://localhost:8080
```

**See:** `internal/workflows/README.md` for complete documentation

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
