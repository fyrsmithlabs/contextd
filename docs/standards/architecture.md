# Architecture Standards

This document defines the architectural patterns and design principles for contextd.

## Core Architecture Principles

### 1. Security-First Design

**Every architectural decision MUST prioritize security:**

- **MCP Streamable HTTP Transport** (spec 2025-03-26)
  - Default port: 8080 (configurable via CONTEXTD_HTTP_PORT)
  - Listen address: 0.0.0.0 (accepts remote connections)
  - Endpoint: POST/GET `/mcp` (single endpoint, JSON-RPC routing)
  - Session management: `Mcp-Session-Id` header

- **Security Requirements** (per MCP spec)
  - **REQUIRED**: Origin header validation (prevent DNS rebinding attacks)
  - **RECOMMENDED**: Localhost binding for local servers (127.0.0.1)
  - **STRONGLY RECOMMENDED**: Authentication (Bearer token, JWT, OAuth)

- **MVP Security Posture**
  - No authentication (trusted network assumption)
  - Deploy behind VPN or use SSH tunneling for remote access
  - Production: Add TLS via reverse proxy (nginx/Caddy) + authentication

- **Credential Management**: Never in code or configs
  - API keys in separate files with 0600 permissions
  - Path: `~/.config/contextd/openai_api_key`
  - NEVER cat credentials to context
  - NEVER log token values
  - NEVER commit credentials

### 2. Context Efficiency First

**Primary goal: Minimize context bloat, maximize token efficiency**

- **Local-First Operations**: Instant response, background sync
  - All operations hit local Qdrant
  - Background goroutine for remote sync (future)
  - <50ms response times for MCP tools

- **Checkpoint + Clear at 70%**: NEVER use /compact
  - Checkpointing: <2s (vs /compact 30-60s)
  - Clear context after checkpoint
  - Resume from checkpoint when needed

- **Documentation Structure**: Reference, don't duplicate
  - Small CLAUDE.md files (<1000 lines)
  - Reference detailed docs in separate files
  - Hierarchical: Global → Project → Specialized

### 3. Multi-Tenant Isolation

**Database-per-project physical isolation for security and performance**

```
contextd/
├── shared/                  # Global knowledge
│   ├── remediations         # Error solutions
│   ├── skills               # Reusable templates
│   └── troubleshooting      # Common patterns
│
└── project_abc123de/        # Per-project (isolated)
    ├── checkpoints          # Session checkpoints
    ├── research             # Documentation
    └── notes                # Session notes
```

**Key Properties:**
- **Physical Isolation**: Separate databases/collections per project
- **No Cross-Contamination**: Filter injection attacks eliminated
- **Performance**: 10-16x faster queries (partition pruning)

**Database Naming:**
- Shared: `shared`
- Project: `project_<hash>` where hash = SHA256(project_path)[:8]
- Example: `/home/user/projects/contextd` → `project_abc123de`

**See:** `docs/adr/002-universal-multi-tenant-architecture.md`

## Component Architecture

### 1. Communication Layer

**Transport: HTTP Server**

```
Client → HTTP (Port 8080) → Echo Server → Handler → Service → Vector Store
```

**Why HTTP Server:**
- MCP Streamable HTTP transport (spec 2025-03-26)
- Remote access for distributed teams
- Multiple concurrent connections (multi-session support)
- Easy integration with firewalls/proxies
- Standard JSON-RPC 2.0 protocol

**Protocol: MCP Streamable HTTP (JSON-RPC 2.0)**
- **Version**: 2025-03-26
- **Endpoint**: POST/GET `/mcp` (single endpoint for all methods)
- **Session Management**: `Mcp-Session-Id` header
- **Security**: Origin header validation, localhost binding recommended

**Note:** MCP spec 2025-03-26 requires single `/mcp` endpoint with JSON-RPC method routing. Current implementation uses multiple REST endpoints (`/mcp/checkpoint/save`, etc.); code refactoring to compliant architecture tracked separately.

- Framework: Echo (clean API, OTEL support, middleware ecosystem)
- Middleware stack order matters (see below)

### 2. Security Layer

**MVP Security Model:**

- No authentication required (trusted network assumption)
- Origin header validation (REQUIRED per MCP spec)
- CORS disabled by default
- Deploy behind VPN or SSH tunnel for remote access
- Post-MVP: Add authentication middleware (Bearer token, JWT, OAuth)

**Middleware Order (DO NOT CHANGE):**
1. Logger - Must be first to log everything
2. Recover - Catch panics early
3. RequestID - Generate ID for correlation
4. otelecho - OTEL instrumentation
5. Route-specific (e.g., auth for /api/v1/*)

### 3. Configuration System

**Priority: Environment → Hardcoded Defaults**

```go
pkg/config:
  Load() → getEnv() for each config value
```

**Key Environment Variables:**
```bash
# MCP Streamable HTTP Transport
CONTEXTD_HTTP_PORT=8080
CONTEXTD_HTTP_HOST=0.0.0.0
CONTEXTD_BASE_URL=http://localhost:8080
MCP_PROTOCOL_VERSION=2025-03-26

# MCP Security (optional for MVP, recommended for production)
MCP_ORIGIN_VALIDATION=true
MCP_ALLOWED_ORIGINS=https://claude.ai,https://app.anthropic.com

# Embeddings (TEI or OpenAI)
EMBEDDING_BASE_URL=http://localhost:8080/v1  # TEI
EMBEDDING_MODEL=BAAI/bge-small-en-v1.5
OPENAI_API_KEY=sk-xxx  # Alternative

# Vector Store
QDRANT_URI=localhost:6334

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=https://otel.dhendel.dev
OTEL_SERVICE_NAME=contextd
OTEL_ENVIRONMENT=production
```

### 4. Observability Stack

**OpenTelemetry Integration (pkg/telemetry):**

```
Server Startup → Initialize OTEL → Setup Traces + Metrics → Export to OTLP/HTTP
```

**Instrumentation:**
- **Traces**: OTLP/HTTP exporter, 5s batch timeout, 512 spans per batch
- **Metrics**: 60s export interval
- **Middleware**: `otelecho` auto-instruments all HTTP requests
- **Resource**: service.name, service.version, deployment.environment
- **Propagation**: W3C Trace Context + Baggage

**Metrics Collected:**
- HTTP request duration (histogram)
- HTTP request count (counter)
- Active connections (gauge)
- HTTP status codes (labels)
- MCP tool call performance
- Vector store operation timing
- Embedding generation duration

### 5. Vector Store Abstraction

**Universal Interface (pkg/vectorstore):**

```go
type VectorStore interface {
    CreateDatabase(ctx, database) error
    UpsertPoints(ctx, database, collection, points) error
    Search(ctx, database, collection, vector, limit) ([]SearchResult, error)
    DeletePoints(ctx, database, collection, filter) error
}
```

**Adapters:**
- **Qdrant** (`pkg/adapter/qdrant`): Native database support
- **Future**: Weaviate, Pinecone, Chroma, Redis

**Adapter Selection:**
- Configuration-driven (no code changes)
- Adapter implements VectorStore interface

### 6. Service Layer

**Service Architecture:**

```
Handler → Service (business logic) → VectorStore → Embedding Service
```

**Services:**
- **checkpoint** (`pkg/checkpoint`): Session checkpoint management
- **remediation** (`pkg/remediation`): Error solution storage/search
- **troubleshoot** (`pkg/troubleshoot`): AI-powered error diagnosis
- **skills** (`pkg/skills`): Reusable template management
- **repository** (`pkg/repository`): Code indexing and search

**Common Patterns:**
- Context propagation for tracing
- Error wrapping with context
- Instrumentation spans for operations
- Input validation at service boundary
- Transaction-like operations (all-or-nothing)

## Dual-Mode Operation

### API Mode (Default)

```
./contextd
  → HTTP Server (Port 8080)
  → REST API
  → No Auth (MVP)
  → For automation hooks
```

### MCP Mode

```
./contextd --mcp
  → stdio transport
  → JSON-RPC protocol
  → 9 MCP tools
  → For Claude Code integration
```

**Both modes share:**
- Same service layer
- Same vector store
- Same configuration
- Same observability

## Embedding Service Architecture

### TEI (Recommended)

```
contextd → HTTP → TEI (localhost:8080) → Model (BAAI/bge-small-en-v1.5)
```

**Benefits:**
- No API quotas or costs
- Runs locally via Docker
- Fast (<100ms per embedding)
- No rate limits

**Setup:** `docker-compose up -d tei`

### OpenAI API (Alternative)

```
contextd → HTTPS → OpenAI API → text-embedding-3-small
```

**Tradeoffs:**
- Costs $0.02 per 1M tokens
- Subject to rate limits
- Requires API key management
- Network dependency

## Server Lifecycle

```
1. Load config from environment
2. Initialize OpenTelemetry (traces + metrics)
3. Create Echo server with middleware stack
4. Setup routes (public endpoints, no auth for MVP)
5. Start HTTP server on configured port (default: 8080)
6. Start server in goroutine
7. Wait for SIGINT/SIGTERM
8. Graceful shutdown (10s timeout)
9. Exit cleanly
```

**Graceful Shutdown:**
- Wait up to 10 seconds for in-flight requests
- OTEL gets 5 seconds to flush data
- Exit cleanly

## Route Structure

```
Public (no auth for MVP):
  GET  /health          - Health check with version
  GET  /ready           - Readiness probe

  POST /mcp             - MCP JSON-RPC endpoint (single endpoint per spec 2025-03-26)
  GET  /mcp             - MCP SSE streaming endpoint

  GET    /api/v1/checkpoints        - List checkpoints
  POST   /api/v1/checkpoints        - Create checkpoint
  GET    /api/v1/checkpoints/:id    - Get checkpoint
  DELETE /api/v1/checkpoints/:id    - Delete checkpoint

  POST   /api/v1/checkpoints/search - Search checkpoints

  GET    /api/v1/remediations       - List remediations
  POST   /api/v1/remediations       - Create remediation
  POST   /api/v1/remediations/search - Search remediations

  POST   /api/v1/troubleshoot       - AI diagnosis
  GET    /api/v1/patterns           - List patterns

  POST   /api/v1/index              - Index repository

Note: MCP spec 2025-03-26 requires single /mcp endpoint. Current multiple REST endpoints
(/mcp/checkpoint/save, etc.) will be refactored to compliant architecture.
```

## Key Design Decisions

### HTTP Server vs Unix Socket
- **Chosen**: HTTP server on configurable port
- **Why**: Remote access for distributed teams, standard protocol, multiple sessions
- **Result**: Standard HTTP/1.1 transport, SSE streaming, reverse proxy compatible
- **MVP Decision**: No auth (trusted network), add auth post-MVP for production

### Authentication Strategy
- **Chosen**: No authentication for MVP
- **Why**: Trusted network assumption, faster development
- **Result**: Deploy behind VPN/SSH tunnel for security
- **Post-MVP**: Add Bearer token, JWT, or OAuth for production

### Echo vs chi/gorilla
- **Chosen**: Echo framework
- **Why**: Clean API, excellent middleware, built-in OTEL support
- **Result**: Less boilerplate, better observability

### Local-First Qdrant
- **Chosen**: Local Qdrant for all operations
- **Why**: Instant response, no network dependency
- **Result**: <50ms response times, offline capable

### Universal Multi-Tenancy
- **Chosen**: Database-per-project isolation
- **Why**: Portability, security, performance
- **Result**: Works with multiple vector databases, no filter injection

### Context Optimization
- **Chosen**: Checkpoint+clear at 70%
- **Why**: Primary goal is token efficiency
- **Result**: All architectural decisions driven by context efficiency

## Development Patterns

### Adding New Endpoints

1. Define handler function in appropriate package
2. Add route in `cmd/contextd/main.go` setupRoutes()
3. Use `api.Group()` for authenticated endpoints
4. Public endpoints go directly on `e` (Echo instance)

### Adding New Package

Follow standard Go layout:
```
pkg/
  newpackage/
    newpackage.go      # Public API
    internal.go        # Internal helpers (optional)
    newpackage_test.go # Tests
```

### Configuration Changes

1. Add to `pkg/config/config.go` Config struct
2. Add to Load() function with getEnv() helper
3. Document in this file and README

### Middleware Order

**Current order (DO NOT CHANGE without reason):**
1. Logger - Must be first to log everything
2. Recover - Catch panics early
3. RequestID - Generate ID for correlation
4. otelecho - OTEL instrumentation
5. Route-specific (e.g., auth for /api/v1/*)

## Error Handling Patterns

### Service Layer

```go
func (s *Service) Operation(ctx context.Context, input Input) (Output, error) {
    // Validate input
    if err := input.Validate(); err != nil {
        return Output{}, fmt.Errorf("invalid input: %w", err)
    }

    // Create span for tracing
    ctx, span := tracer.Start(ctx, "operation")
    defer span.End()

    // Perform operation
    result, err := s.store.DoSomething(ctx, input)
    if err != nil {
        span.RecordError(err)
        return Output{}, fmt.Errorf("failed to do something: %w", err)
    }

    // Return success
    return result, nil
}
```

### Handler Layer

```go
func (h *Handler) HandleRequest(c echo.Context) error {
    // Parse input
    var input Input
    if err := c.Bind(&input); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
    }

    // Call service
    output, err := h.service.Operation(c.Request().Context(), input)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    // Return response
    return c.JSON(http.StatusOK, output)
}
```

## Performance Considerations

### Response Time Targets

- **Health checks**: <10ms
- **Checkpoint save**: <100ms
- **Checkpoint search**: <200ms
- **Remediation search**: <300ms (hybrid matching)
- **AI troubleshoot**: <2s (OpenAI API dependency)
- **Repository indexing**: Variable (depends on size)

### Optimization Strategies

1. **Local-First**: All operations hit local Qdrant
2. **Batch Operations**: Upsert multiple points at once
3. **Concurrent Processing**: Use goroutines for independent tasks
4. **Connection Pooling**: Reuse HTTP clients and connections
5. **Caching**: Cache embeddings for repeated content (future)

## Scalability Considerations

### Current Architecture (Single-User)

- **Design**: Single-user localhost service
- **Concurrency**: Handles concurrent requests via goroutines
- **Storage**: Local Qdrant (grows with usage)
- **Memory**: Bounded by Qdrant configuration

### Future Multi-User (If Needed)

- **Auth**: Move to JWT with user claims
- **Database**: User-specific databases (extend multi-tenant)
- **Transport**: Add TLS via reverse proxy (nginx/Caddy)
- **Rate Limiting**: Per-user rate limits

## Security Considerations

### Threat Model

**In Scope:**
- Local privilege escalation
- File permission issues
- Timing attacks on auth
- Log injection

**Out of Scope (MVP only - add post-MVP):**
- Authentication/authorization (MVP uses trusted network)
- Rate limiting (add for production)
- DDoS protection (use reverse proxy for production)

**Out of Scope (by design):**
- SQL injection (no SQL)
- XSS (no web UI)
- CSRF (no web sessions)

### Security Checklist

- ✅ HTTP server with configurable port and host
- ✅ CORS disabled by default (same-origin only)
- ✅ Rate limiting recommended for production
- ⚠️  MVP: No authentication (use VPN/SSH tunnel for security)
- ⚠️  Production: Add auth layer (Bearer token, JWT, OAuth)
- ✅ No credentials in code/config
- ✅ No credential logging
- ✅ Graceful error handling (no stack traces in responses)
- ✅ Input validation at service boundary
- ✅ Context propagation for tracing
- ✅ OTEL for security event monitoring

## Testing Strategy

### Unit Tests

- **Coverage Target**: ≥80% overall
- **Critical Paths**: 100% coverage
- **Location**: `*_test.go` files alongside implementation

### Integration Tests

- **Scope**: Service layer + vector store
- **Setup**: In-memory or test Qdrant instance
- **Teardown**: Cleanup test data

### End-to-End Tests

- **Scope**: Full request/response cycle
- **Setup**: Test server with test socket
- **Cleanup**: Remove test socket and data

**See:** `docs/standards/testing-standards.md`

## Documentation References

- **ADRs**: `docs/adr/` - Architectural decision records
- **Research**: `docs/research/` - Investigation and analysis
- **Migration**: `docs/MIGRATION-FROM-LEGACY.md`
- **Multi-Tenant**: `docs/MULTI-TENANT-COMPLETION-STATUS.md`
- **TEI Deployment**: `docs/TEI-DEPLOYMENT.md`

## Related Standards

- **Coding Standards**: `docs/standards/coding-standards.md`
- **Testing Standards**: `docs/standards/testing-standards.md`
- **Package Guidelines**: `docs/standards/package-guidelines.md`
