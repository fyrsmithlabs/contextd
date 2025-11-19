# Component Architecture (Detailed)

**Parent**: [Architecture Standards](../architecture.md)

This document provides detailed descriptions of contextd's component architecture.

---

## 1. Communication Layer

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
- Middleware stack order matters (see parent doc)

---

## 2. Security Layer

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

---

## 3. Configuration System

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

---

## 4. Observability Stack

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

---

## 5. Vector Store Abstraction

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

---

## 6. Service Layer

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

---

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

---

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

---

## Route Structure

```
Public (no auth for MVP):
  GET  /health          - Health check with version
  GET  /ready           - Readiness probe

  POST /mcp             - MCP JSON-RPC endpoint (single endpoint per spec 2025-03-26)

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
