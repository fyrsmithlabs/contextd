# Architecture Standards

This document defines the architectural patterns and design principles for contextd.

---

## Core Architecture Principles

### 1. Security-First Design

**Every architectural decision MUST prioritize security:**

- **Dual Transport Support** (HTTP + stdio)
  - **HTTP Transport**: Port 9090 (configurable), remote access, REST API
  - **stdio Transport**: stdin/stdout, Claude Code integration, MCP SDK
  - Session management: `Mcp-Session-Id` header (HTTP), process isolation (stdio)

- **Security Requirements** (per MCP spec 2025-03-26)
  - **REQUIRED**: Origin header validation (HTTP mode)
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

---

## Quick Reference

**Transports**: HTTP Server (port 9090) + stdio (stdin/stdout)
**Protocol**: MCP Streamable HTTP (spec 2025-03-26) + MCP stdio
**Framework**: Echo (HTTP), MCP Go SDK (stdio)
**Vector Store**: Qdrant (local)
**Embeddings**: TEI or OpenAI API
**Observability**: OpenTelemetry (OTLP/HTTP)

**HTTP Endpoints**:
- `/health` - Health check
- `/mcp` - MCP JSON-RPC endpoint (POST/GET)
- `/api/v1/checkpoints` - Checkpoint operations
- `/api/v1/remediations` - Remediation operations

**stdio Tools**: 23 MCP tools via `contextd --mcp`

---

## Component Architecture

**Communication Flow (HTTP)**:
```
Client → HTTP (Port 9090) → Echo Server → Handler → Service → Vector Store
```

**Communication Flow (stdio)**:
```
Claude Code ↔ stdio (contextd --mcp) → HTTP Daemon (localhost:9090) → Service → Vector Store
```

**Components**:
1. **Communication Layer**: HTTP server (Echo) + stdio server (MCP SDK)
2. **Security Layer**: Origin validation, middleware stack (MVP: no auth)
3. **Configuration**: Environment variables → config.yaml → hardcoded defaults
4. **Observability**: OpenTelemetry (traces + metrics)
5. **Vector Store**: Qdrant abstraction layer
6. **Service Layer**: Business logic (checkpoint, remediation, skills, etc.)

**Detailed Component Documentation**:
@./architecture/component-architecture.md

---

## Dual-Mode Operation

### HTTP Mode (Default)

```
./contextd
  → HTTP Server (Port 9090)
  → REST API + MCP JSON-RPC
  → 14 HTTP endpoints
  → No Auth (MVP)
  → For automation hooks, remote access
```

**Use cases**:
- Remote access (distributed teams)
- Automation scripts
- CI/CD integration
- Multiple concurrent sessions

### stdio Mode (Claude Code Integration)

```
./contextd --mcp
  → stdio transport (stdin/stdout)
  → MCP Go SDK
  → 23 MCP tools
  → Real-time progress notifications
  → For Claude Code integration
```

**Use cases**:
- Native Claude Code integration
- Local development
- Interactive sessions
- Better progress visibility

**Both modes share:**
- Same service layer
- Same vector store
- Same configuration
- Same observability

**Detailed stdio Architecture**:
@./architecture/stdio-transport.md

---

## Key Design Decisions

### HTTP + stdio Dual Transport
- **Chosen**: Both HTTP and stdio (not one or the other)
- **Why**: HTTP for remote/automation, stdio for Claude Code native integration
- **Result**: Flexibility for different use cases, shared service layer ensures consistency
- **Trade-off**: Slightly more complex than single transport, but better UX

### HTTP Server vs Unix Socket (HTTP Mode)
- **Chosen**: HTTP server on configurable port
- **Why**: Remote access for distributed teams, standard protocol, multiple sessions
- **Result**: Standard HTTP/1.1 transport, reverse proxy compatible
- **MVP Decision**: No auth (trusted network), add auth post-MVP for production

### MCP SDK vs Custom Protocol (stdio Mode)
- **Chosen**: Official MCP Go SDK
- **Why**: Protocol compliance, progress notifications, future-proof
- **Result**: Native Claude Code integration, real-time progress updates

### stdio Polling vs NATS Subscription
- **Chosen**: HTTP daemon polling (500ms interval)
- **Why**: Simpler implementation, no NATS dependency for stdio mode
- **Result**: Slight latency (500ms updates) but good enough for UX
- **Future**: Could add NATS subscription for real-time events

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

---

## Development & Implementation

**How to extend the architecture**:
@./architecture/development-patterns.md

**Topics covered**:
- Adding new endpoints (HTTP mode)
- Adding new tools (stdio mode)
- Adding new packages
- Configuration changes
- Middleware order
- Error handling patterns
- Testing strategy

---

## Performance & Security

**Performance targets, scalability, and security considerations**:
@./architecture/performance-security.md

**Topics covered**:
- Response time targets (HTTP vs stdio)
- Optimization strategies
- Scalability considerations (current & future)
- Threat model
- Security checklist

---

## Related Standards

- **Coding Standards**: `docs/standards/coding-standards.md`
- **Testing Standards**: `docs/standards/testing-standards.md`
- **Package Guidelines**: `docs/standards/package-guidelines.md`
- **stdio Transport Architecture**: `docs/standards/architecture/stdio-transport.md`
