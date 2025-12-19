# Architecture Overview

contextd is a Go-based MCP server providing AI context management with cross-session memory, checkpoints, and error pattern tracking. This document describes the simplified v2 architecture.

---

## High-Level Architecture

```
+-----------------------------------------------------------------------+
|                         Claude Code / AI Agent                          |
|                                   |                                     |
|                           MCP Protocol (stdio)                          |
|                                   |                                     |
|  +----------------------------------------------------------------+    |
|  |                          contextd                               |    |
|  |                                                                 |    |
|  |  +-----------------------------------------------------------+  |    |
|  |  |                      MCP Server Layer                      |  |    |
|  |  |  +----------+ +----------+ +----------+ +---------------+  |  |    |
|  |  |  | Memory   | |Checkpoint| |Remediate | | Repository/   |  |  |    |
|  |  |  | Tools    | | Tools    | | Tools    | | Troubleshoot  |  |  |    |
|  |  |  +----+-----+ +----+-----+ +----+-----+ +-------+-------+  |  |    |
|  |  +-------|------------|------------|---------------|----------+  |    |
|  |          |            |            |               |             |    |
|  |  +-------v------------v------------v---------------v----------+  |    |
|  |  |                    Service Registry                         |  |    |
|  |  |  +-------------+ +-------------+ +-------------+ +--------+ |  |    |
|  |  |  | Reasoning   | | Checkpoint  | | Remediation | | Repo   | |  |    |
|  |  |  | Bank        | | Service     | | Service     | | Service| |  |    |
|  |  |  +------+------+ +------+------+ +------+------+ +----+---+ |  |    |
|  |  +---------|---------------|---------------|--------------|----+  |    |
|  |            |               |               |              |       |    |
|  |  +---------v---------------v---------------v--------------v----+  |    |
|  |  |                  Infrastructure Layer                        |  |    |
|  |  |  +-------------+  +-------------+  +---------------------+   |  |    |
|  |  |  | VectorStore |  | Embeddings  |  |   Secret Scrubber   |   |  |    |
|  |  |  | (chromem)   |  | (FastEmbed) |  |     (gitleaks)      |   |  |    |
|  |  |  +-------------+  +-------------+  +---------------------+   |  |    |
|  |  +--------------------------------------------------------------+  |    |
|  +--------------------------------------------------------------------+    |
|                                   |                                        |
|                                   v                                        |
|  +--------------------------------------------------------------------+    |
|  |                    Local Storage (~/.local/share/contextd)          |    |
|  +--------------------------------------------------------------------+    |
+------------------------------------------------------------------------+
```

---

## Component Overview

### Entry Point (`cmd/contextd/`)

The main entry point initializes all components in order:
1. Logging (Zap)
2. Telemetry (OpenTelemetry, disabled by default)
3. Configuration (Koanf: file + env vars)
4. Secret Scrubber (gitleaks)
5. Embeddings Provider (FastEmbed or TEI)
6. VectorStore (chromem or Qdrant)
7. Core Services (checkpoint, remediation, repository, troubleshoot, reasoningbank)
8. Hooks Manager
9. Service Registry
10. HTTP Server (background)
11. MCP Server (if `--mcp` flag)

### MCP Server (`internal/mcp/`)

Handles Model Context Protocol communication via stdio transport.

**Key Files:**
- `server.go` - Server initialization and lifecycle
- `tools.go` - Tool registration and handlers

**Responsibilities:**
- Parse MCP JSON-RPC messages from stdin
- Route tool calls to appropriate services via registry
- Scrub secrets from all responses
- Return formatted responses to stdout

### Service Registry (`internal/services/`)

Central registry providing dependency injection for all services.

```go
type Registry struct {
    Checkpoint   checkpoint.Service
    Remediation  remediation.Service
    Memory       *reasoningbank.Service
    Repository   *repository.Service
    Troubleshoot *troubleshoot.Service
    Hooks        *hooks.HookManager
    Scrubber     *secrets.Scrubber
}
```

**Benefits:**
- Clean separation of concerns
- Easy testing with mock services
- Graceful degradation when services unavailable

### ReasoningBank (`internal/reasoningbank/`)

Cross-session memory system storing learnings with confidence scores.

**Key Concepts:**
- **Memory**: A recorded strategy, insight, or learning
- **Confidence Score**: Reliability rating (0.0 - 1.0), adjusted by feedback
- **Outcome**: Whether the strategy led to success or failure

**Operations:**
- `Search(projectID, query, limit)` - Semantic search for relevant memories
- `Record(memory)` - Store new memory with embedding
- `Feedback(memoryID, helpful)` - Adjust confidence based on usefulness
- `Get(memoryID)` - Retrieve specific memory

### Checkpoint Service (`internal/checkpoint/`)

Context persistence and recovery system.

**Resume Levels:**
| Level | Content | Use Case |
|-------|---------|----------|
| `summary` | Brief summary only | Quick context refresh |
| `context` | Summary + contextual info | Normal resumption |
| `full` | Complete session state | Full restoration |

### Remediation Service (`internal/remediation/`)

Error pattern tracking with hierarchical scope.

**Scopes:**
- `project` - Project-specific fixes
- `team` - Team-shared knowledge
- `org` - Organization-wide patterns

**Hierarchical Search:**
When `include_hierarchy` enabled, search expands: project -> team -> org

### Repository Service (`internal/repository/`)

Code indexing for semantic search.

**Indexing Process:**
1. Walk directory tree
2. Apply include/exclude patterns (respects .gitignore)
3. Chunk files for embedding
4. Store in vectorstore with metadata

### VectorStore (`internal/vectorstore/`)

Pluggable vector storage with provider abstraction.

**Providers:**
| Provider | Type | Use Case |
|----------|------|----------|
| `chromem` | Embedded | Default, no external deps |
| `qdrant` | External | Production, team deployments |

**Interface:**
```go
type Store interface {
    AddDocuments(ctx, collection, docs) error
    Query(ctx, collection, query, limit) ([]Document, error)
    Delete(ctx, collection, ids) error
    Close() error
}
```

### Embeddings (`internal/embeddings/`)

Pluggable embedding provider system.

**Providers:**
| Provider | Type | Model |
|----------|------|-------|
| `fastembed` | Local ONNX | all-MiniLM-L6-v2 (384 dim) |
| `tei` | Remote API | Configurable |

### Secret Scrubber (`internal/secrets/`)

Automatic secret detection and redaction using gitleaks SDK.

**Coverage:**
- API keys (AWS, GCP, Azure, GitHub, etc.)
- Passwords and tokens
- Private keys
- Connection strings
- Custom patterns

**Scrubbing Points:**
1. All MCP tool responses
2. Stored content (memories, checkpoints)
3. HTTP API responses

### Hooks (`internal/hooks/`)

Lifecycle hook management for session events.

**Hook Types:**
| Hook | Trigger |
|------|---------|
| `session_start` | New session begins |
| `session_end` | Session ends |
| `before_clear` | Before `/clear` command |
| `after_clear` | After `/clear` command |
| `context_threshold` | Context usage reaches threshold |

---

## Data Flow

### Memory Search Flow

```
1. Claude Code calls memory_search(project_id, query)
   |
2. MCP Server receives request
   |
3. ReasoningBank.Search() called
   |
4. Query embedded via FastEmbed
   |
5. chromem similarity search on {project}_memories collection
   |
6. Results filtered by confidence threshold
   |
7. Secret scrubber removes sensitive data
   |
8. Response returned to Claude Code
```

### Checkpoint Save Flow

```
1. Claude Code calls checkpoint_save(session_id, tenant_id, ...)
   |
2. MCP Server receives request
   |
3. Checkpoint.Save() validates input
   |
4. Summary embedded for future search
   |
5. Full state stored in org_checkpoints collection
   |
6. Checkpoint ID returned to Claude Code
```

---

## Configuration

### Loading Order

1. Defaults (compiled in)
2. Config file (`~/.config/contextd/config.yaml`)
3. Environment variables (override file)
4. CLI flags (override all)

### Key Configuration

```yaml
vectorstore:
  provider: chromem          # or "qdrant"
  chromem:
    path: ~/.local/share/contextd

embeddings:
  provider: fastembed        # or "tei"
  model: all-MiniLM-L6-v2

server:
  port: 9090
  shutdown_timeout: 5s
```

---

## Key Design Decisions

### Why Simplified from gRPC (v1 -> v2)

| v1 (old branch) | v2 (current) |
|-----------------|--------------|
| gRPC between components | Direct function calls |
| Complex service mesh | Single binary |
| External Qdrant required | Embedded chromem default |
| Multiple processes | Single process |

**Rationale:** AI agent use case doesn't need distributed architecture. Simplicity reduces failure modes and deployment complexity.

### Why chromem as Default

- **Zero dependencies**: No external database to run
- **Embedded**: Data persists in local files
- **Migration path**: Can switch to Qdrant for team deployments
- **Performance**: Sub-100ms for typical queries

### Why Service Registry

- **Testability**: Easy to mock individual services
- **Graceful degradation**: Server starts even if some services fail
- **Clean DI**: No global state, explicit dependencies

### Why Separate HTTP Server

- **Health checks**: Kubernetes/Docker health probes
- **Threshold triggers**: External context monitoring
- **Debugging**: Status endpoint for troubleshooting
- **Future API**: Foundation for REST API if needed

---

## Directory Structure

```
contextd/
+-- cmd/
|   +-- contextd/           # Main entry point
|   +-- ctxd/               # CLI tool
+-- internal/
|   +-- checkpoint/         # Context persistence
|   +-- compression/        # Context compression
|   +-- config/             # Koanf configuration
|   +-- embeddings/         # Embedding providers
|   +-- hooks/              # Lifecycle hooks
|   +-- http/               # HTTP API server
|   +-- logging/            # Zap logging
|   +-- mcp/                # MCP server + handlers
|   +-- project/            # Project management
|   +-- reasoningbank/      # Cross-session memory
|   +-- remediation/        # Error pattern tracking
|   +-- repository/         # Code indexing
|   +-- secrets/            # gitleaks scrubbing
|   +-- services/           # Service registry
|   +-- telemetry/          # OpenTelemetry
|   +-- troubleshoot/       # Error diagnosis
|   +-- vectorstore/        # Vector storage abstraction
+-- docs/                   # Documentation
+-- deploy/                 # Docker/deployment files
```

---

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.25+ | Core application |
| MCP | github.com/modelcontextprotocol/go-sdk | Protocol implementation |
| Vector DB | chromem (default) / Qdrant | Semantic storage |
| Embeddings | FastEmbed (ONNX) | Local embeddings |
| Config | Koanf | Configuration loading |
| Logging | Zap | Structured logging |
| Telemetry | OpenTelemetry | Observability |
| Secrets | gitleaks | Secret detection |

---

## Performance Characteristics

### Latency

| Operation | Typical |
|-----------|---------|
| Memory search | 50-100ms |
| Checkpoint save | 100-200ms |
| Repository index (1000 files) | 30-60s |
| Embedding (single text) | 10-20ms |

### Resource Usage

| Component | Memory |
|-----------|--------|
| contextd base | ~50MB |
| FastEmbed model | ~200MB |
| chromem per 10K docs | ~100MB |

---

## Security Model

### Multi-Tenancy

contextd uses **payload-based tenant isolation** as the default strategy:

**Tenant Context Flow:**
```
1. Tenant info set in context via ContextWithTenant()
   |
2. All vectorstore operations extract tenant from context
   |
3. Queries: TenantFilter() injected into all searches
   |
4. Writes: TenantMetadata() injected into all documents
   |
5. Missing context: ErrMissingTenant returned (fail-closed)
```

**Isolation Modes:**

| Mode | Isolation | Use Case |
|------|-----------|----------|
| `PayloadIsolation` | Metadata filtering in shared collection | **Default** |
| `FilesystemIsolation` | Separate database per tenant | Legacy |
| `NoIsolation` | None | Testing only |

**Defense-in-Depth:**

| Threat | Defense |
|--------|---------|
| Cross-tenant query | Tenant filters injected on all queries |
| Filter injection | `ApplyTenantFilters()` rejects user tenant fields |
| Metadata poisoning | Tenant fields overwritten from context |
| Context bypass | Fail-closed returns error, not empty results |

### Secret Protection

| Layer | Protection |
|-------|------------|
| Input | Reject obvious secrets |
| Storage | Scrub before storing |
| Output | Scrub all responses |

### Transport Security

- MCP: stdio (no network exposure)
- HTTP: localhost only by default
- Qdrant: localhost only by default

---

## Extension Points

### Custom Embedding Provider

Implement the `Provider` interface:

```go
type Provider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
    Close()
}
```

### Custom VectorStore

Implement the `Store` interface for alternative backends.

### Custom Ignore Patterns

Create `.contextdignore` in project root for repository indexing.
