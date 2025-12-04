# Architecture Overview

ContextD is a Go-based MCP server that provides AI context management capabilities. This document describes the system architecture, component interactions, and design decisions.

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Claude Code                                 │
│                              │                                       │
│                         MCP Protocol                                 │
│                          (stdio)                                     │
│                              │                                       │
│  ┌───────────────────────────▼───────────────────────────────────┐  │
│  │                       ContextD                                 │  │
│  │                                                                │  │
│  │  ┌──────────────────────────────────────────────────────────┐ │  │
│  │  │                    MCP Server Layer                       │ │  │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────────────┐ │ │  │
│  │  │  │ Memory  │ │Checkpoint│ │Remediate│ │Repository/Diag │ │ │  │
│  │  │  │ Tools   │ │ Tools   │ │ Tools   │ │    Tools        │ │ │  │
│  │  │  └────┬────┘ └────┬────┘ └────┬────┘ └───────┬─────────┘ │ │  │
│  │  └───────┼───────────┼───────────┼──────────────┼───────────┘ │  │
│  │          │           │           │              │             │  │
│  │  ┌───────▼───────────▼───────────▼──────────────▼───────────┐ │  │
│  │  │                   Service Layer                           │ │  │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────┐ │ │  │
│  │  │  │ Reasoning  │ │ Checkpoint │ │Remediation │ │  Repo  │ │ │  │
│  │  │  │   Bank     │ │  Service   │ │  Service   │ │Service │ │ │  │
│  │  │  └─────┬──────┘ └─────┬──────┘ └─────┬──────┘ └───┬────┘ │ │  │
│  │  └────────┼──────────────┼──────────────┼────────────┼──────┘ │  │
│  │           │              │              │            │        │  │
│  │  ┌────────▼──────────────▼──────────────▼────────────▼──────┐ │  │
│  │  │                  Infrastructure Layer                     │ │  │
│  │  │  ┌───────────┐  ┌───────────┐  ┌───────────────────────┐ │ │  │
│  │  │  │  Qdrant   │  │ Embeddings │  │    Secret Scrubber    │ │ │  │
│  │  │  │  Client   │  │  Provider  │  │     (gitleaks)        │ │ │  │
│  │  │  └───────────┘  └───────────┘  └───────────────────────┘ │ │  │
│  │  └──────────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                              │                                       │
│                              ▼                                       │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                        Qdrant                                  │  │
│  │              (Vector Database - Embedded)                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Component Overview

### MCP Server (`internal/mcp/`)

The MCP server handles the Model Context Protocol communication with Claude Code.

**Key Files:**
- `server.go` - Server initialization and lifecycle
- `tools.go` - Tool registration and handlers

**Responsibilities:**
- Parse MCP JSON-RPC messages from stdin
- Route tool calls to appropriate services
- Scrub secrets from all responses
- Return formatted responses to stdout

### ReasoningBank (`internal/reasoningbank/`)

Cross-session memory system for storing and retrieving learnings.

**Key Concepts:**
- **Memory**: A recorded strategy, insight, or learning
- **Confidence Score**: Reliability rating (0.0 - 1.0)
- **Outcome**: Whether the strategy led to success or failure

**How It Works:**
1. Memories are embedded using the configured embedding model
2. Stored in Qdrant with metadata
3. Retrieved via semantic similarity search
4. Confidence adjusted based on feedback

### Checkpoint Service (`internal/checkpoint/`)

Context persistence and recovery system.

**Key Concepts:**
- **Checkpoint**: Snapshot of session state at a point in time
- **Resume Levels**: summary, context, full

**Data Model:**
```go
type Checkpoint struct {
    ID          string
    SessionID   string
    TenantID    string
    ProjectPath string
    Name        string
    Description string
    Summary     string      // Brief summary
    Context     string      // Contextual info
    FullState   string      // Complete state
    TokenCount  int32
    Threshold   float64     // Context % when saved
    AutoCreated bool
    Metadata    map[string]string
    CreatedAt   time.Time
}
```

### Remediation Service (`internal/remediation/`)

Error pattern tracking and fix database.

**Key Concepts:**
- **Remediation**: A recorded fix for an error pattern
- **Scope**: project, team, or org level
- **Category**: Error type classification

**Hierarchical Search:**
When `include_hierarchy` is enabled, search expands:
1. Project scope
2. Team scope (if no project results)
3. Org scope (if no team results)

### Repository Service (`internal/repository/`)

Code indexing for semantic search.

**Indexing Process:**
1. Walk directory tree
2. Apply include/exclude patterns
3. Parse ignore files (.gitignore, etc.)
4. Chunk files for embedding
5. Store in Qdrant with metadata

### Embeddings (`internal/embeddings/`)

Pluggable embedding provider system.

**Providers:**
- `fastembed` - Local ONNX-based embeddings (default)
- `tei` - HuggingFace Text Embeddings Inference

**Provider Interface:**
```go
type Provider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
    Close()
}
```

### Secret Scrubber (`internal/secrets/`)

Automatic secret detection and redaction using gitleaks.

**Coverage:**
- API keys (AWS, GCP, Azure, GitHub, etc.)
- Passwords and tokens
- Private keys
- Connection strings
- Custom patterns

---

## Data Flow

### Memory Search Flow

```
1. Claude Code calls memory_search
   ↓
2. MCP Server receives request
   ↓
3. Query is embedded via Embeddings Provider
   ↓
4. Qdrant similarity search
   ↓
5. Results filtered by confidence
   ↓
6. Secret scrubber removes sensitive data
   ↓
7. Response returned to Claude Code
```

### Checkpoint Save Flow

```
1. Claude Code calls checkpoint_save
   ↓
2. MCP Server receives request
   ↓
3. Checkpoint Service validates input
   ↓
4. Summary is embedded for future search
   ↓
5. Full state stored in Qdrant
   ↓
6. ID returned to Claude Code
```

---

## Container Architecture

The Docker image bundles all components:

```
┌─────────────────────────────────────────┐
│           Docker Container               │
│                                          │
│  ┌──────────────────────────────────┐   │
│  │        Entrypoint Script          │   │
│  │  1. Start Qdrant (background)     │   │
│  │  2. Wait for Qdrant ready         │   │
│  │  3. exec contextd                 │   │
│  └──────────────────────────────────┘   │
│                                          │
│  ┌────────────┐    ┌────────────────┐   │
│  │   Qdrant   │◄───│   ContextD     │   │
│  │  (gRPC)    │    │  (MCP Server)  │   │
│  └────────────┘    └────────────────┘   │
│                                          │
│  ┌──────────────────────────────────┐   │
│  │         ONNX Runtime              │   │
│  │    (for FastEmbed embeddings)     │   │
│  └──────────────────────────────────┘   │
│                                          │
│  Volume: /data                           │
│  └── qdrant/storage/                     │
└─────────────────────────────────────────┘
```

---

## Security Model

### Multi-Tenancy

All data is isolated by `tenant_id`:
- Each tenant has separate vector collections
- No cross-tenant queries possible
- Tenant ID derived from authenticated context

### Secret Protection

Three layers of protection:

1. **Input Validation**: Reject requests containing obvious secrets
2. **Storage Scrubbing**: Scrub before storing
3. **Output Scrubbing**: Scrub all responses

### Transport Security

- MCP uses stdio (no network exposure)
- Qdrant internal (localhost only by default)
- HTTP server for health checks only

---

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.25+ | Core application |
| MCP | github.com/modelcontextprotocol/go-sdk | Protocol implementation |
| Vector DB | Qdrant | Semantic storage and search |
| Embeddings | FastEmbed (ONNX) | Local text embeddings |
| Logging | Zap | Structured logging |
| Telemetry | OpenTelemetry | Observability |
| Secrets | gitleaks | Secret detection |
| Config | Koanf | Configuration loading |

---

## Directory Structure

```
contextd/
├── cmd/
│   └── contextd/          # Main entry point
├── internal/
│   ├── checkpoint/        # Checkpoint service
│   ├── compression/       # Context compression
│   ├── config/            # Configuration
│   ├── embeddings/        # Embedding providers
│   ├── hooks/             # Lifecycle hooks
│   ├── logging/           # Zap logging
│   ├── mcp/               # MCP server
│   │   └── handlers/      # Tool handlers
│   ├── project/           # Project management
│   ├── qdrant/            # Qdrant client
│   ├── reasoningbank/     # Memory system
│   ├── remediation/       # Error tracking
│   ├── repository/        # Code indexing
│   ├── secrets/           # Secret scrubbing
│   ├── telemetry/         # OpenTelemetry
│   ├── tenant/            # Multi-tenancy
│   ├── troubleshoot/      # Diagnostics
│   └── vectorstore/       # Vector abstraction
├── deploy/
│   ├── entrypoint.sh      # Container entrypoint
│   └── supervisord.conf   # Process management
├── docs/                  # Documentation
└── Dockerfile             # Container build
```

---

## Extension Points

### Custom Embedding Provider

Implement the `Provider` interface:

```go
type CustomProvider struct {
    // your fields
}

func (p *CustomProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // your embedding logic
}

func (p *CustomProvider) Dimension() int {
    return 384 // your model dimension
}

func (p *CustomProvider) Close() {
    // cleanup
}
```

### Custom Ignore Patterns

Create `.contextdignore` in your project:

```gitignore
# Additional patterns for ContextD indexing
*.test.ts
*.spec.js
coverage/
dist/
```

---

## Performance Considerations

### Memory Usage

- Qdrant: ~100MB base + data
- FastEmbed: ~200MB for model
- ContextD: ~50MB

### Latency

| Operation | Typical Latency |
|-----------|-----------------|
| Memory search | 50-100ms |
| Checkpoint save | 100-200ms |
| Repository index (1000 files) | 30-60s |
| Embedding (single text) | 10-20ms |

### Scaling

ContextD is designed for single-user operation. For team deployments:

1. Run separate instances per user, or
2. Use external Qdrant with tenant isolation
