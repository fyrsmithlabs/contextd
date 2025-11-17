# Contextd 0.9.0-rc-1 Rebuild Specification

**Version**: 0.9.0-rc-1
**Status**: Design
**Created**: 2025-01-14
**Author**: Architecture Team

---

## Executive Summary

This specification defines a complete rebuild of contextd from the ground up, applying all lessons learned from v2.0. The primary driver is a **hard switch from stdio to HTTP/SSE transport**, enabling better multi-tenancy, security, and shared instances across development teams.

**Core Philosophy**: Build only what's needed now (YAGNI), with clean interfaces and comprehensive tests (TDD).

**Primary Goals** (in order):
1. **Context Optimization**: 5× reduction in token usage (12K → <3K per search)
2. **Security**: Multi-tenant isolation, server-side secret scrubbing
3. **Performance**: <100ms search latency, <1GB memory footprint

---

## Table of Contents

1. [High-Level Architecture](#1-high-level-architecture)
2. [Multi-Tenant Collection Model](#2-multi-tenant-collection-model)
3. [Secret Scrubbing Architecture](#3-secret-scrubbing-architecture)
4. [HTTP/SSE Transport](#4-httpsse-transport)
5. [Pluggable Embeddings & Vector Stores](#5-pluggable-embeddings--vector-stores)
6. [Advanced Features Roadmap](#6-advanced-features-roadmap)
7. [Implementation Priorities](#7-implementation-priorities)
8. [Success Metrics](#8-success-metrics)
9. [Migration Path](#9-migration-path)

---

## 1. High-Level Architecture

### 1.1 System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Claude Code (Client)                    │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTP/SSE (MCP Protocol)
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   Contextd v3 Server                         │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ MCP Handler  │  │ Pre-Fetcher  │  │ Auth Layer   │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                  │                  │             │
│  ┌──────▼──────────────────▼──────────────────▼───────┐   │
│  │           Collection Manager                        │   │
│  │  (Branch-aware, Owner-scoped isolation)            │   │
│  └──────┬──────────────────────────────────────┬──────┘   │
│         │                                       │           │
│  ┌──────▼───────┐                      ┌───────▼──────┐   │
│  │ Secret       │                      │ Embedding    │   │
│  │ Scrubber     │◄─────Gitleaks───────│ Pipeline     │   │
│  └──────┬───────┘                      └───────┬──────┘   │
└─────────┼──────────────────────────────────────┼──────────┘
          │                                       │
          └───────────────┬───────────────────────┘
                          │
          ┌───────────────▼───────────────┐
          │   langchaingo Abstraction     │
          └───────────────┬───────────────┘
                          │
          ┌───────────────▼───────────────┐
          │    Vector Store (Qdrant)      │
          │    Embeddings (TEI)           │
          └───────────────────────────────┘
```

### 1.2 Core Principles

**YAGNI (You Aren't Gonna Need It)**:
- Build only what's needed for MVP
- No speculative features
- Benchmark before optimizing

**Interface Design**:
- Minimal, focused interfaces
- Use langchaingo abstractions (no custom interfaces)
- Prefer concrete implementations over hierarchies

**Test-Driven Development (TDD)**:
- Write tests first (Red)
- Implement to pass (Green)
- Refactor (Clean)
- ≥80% coverage mandatory

### 1.3 Technology Stack

**Language**: Go 1.21+
**HTTP Framework**: Chi router (stdlib-compatible)
**Vector Abstraction**: langchaingo
**Secret Scanning**: Gitleaks SDK
**Observability**: OpenTelemetry (traces only)
**Config**: Viper + YAML
**Defaults**: TEI embeddings + Qdrant vector store (Docker Compose)

---

## 2. Multi-Tenant Collection Model

### 2.1 Owner-Scoped Isolation (v2.1 Level)

**Design Decision**: Collections scoped by `owner_id` (SHA256 of system username)

**Benefits**:
- ✅ No cross-owner data leakage
- ✅ Simple auth model (username → hash)
- ✅ Foundation for future team/org features (v3.2+)

**Collection Hierarchy**:

```
owner_<hash>/
├── project_<hash>/          # Project-scoped collections
│   ├── main                 # Base collection (stable)
│   ├── branch_feature-auth  # Delta collection (changes only)
│   └── branch_fix-bug-123   # Delta collection (changes only)
├── project_<other>/
│   └── main
└── shared/                  # Owner-level shared knowledge
    ├── remediations
    ├── skills
    └── troubleshooting
```

### 2.2 Branch-Aware Layered Collections

**Problem**: Feature branches pollute context with unchanged files from main.

**Solution**: Delta model where branches only store **changes** (new/modified files).

**Search Strategy**:
```
Query on feature branch:
1. Search branch_feature-auth (deltas)
2. Search main (base)
3. Merge results with deduplication
4. Rank by relevance
```

**Benefits**:
- ✅ Prevents context pollution from unchanged files
- ✅ Reduces storage (no duplication)
- ✅ Fast branch switching (query different collection)
- ✅ Main branch remains clean canonical source

**Collection Naming Convention**:
- Main: `owner_<hash>/project_<hash>/main`
- Branch: `owner_<hash>/project_<hash>/branch_<sanitized-name>`
- Shared: `owner_<hash>/shared/<type>`

### 2.3 Worktree Detection

**Auto-detect two scenarios**:
1. **Git worktree**: Different directory, same repo
2. **Branch switch**: Same directory, different branch

Both use the same delta+base search strategy.

**Detection Logic**:
```go
// Pseudo-code
if gitDir := detectGitDir(projectPath); gitDir != "" {
    branch := getCurrentBranch(gitDir)
    if branch != "main" {
        return []string{
            collectionName(owner, project, branch), // Delta
            collectionName(owner, project, "main"),  // Base
        }
    }
}
```

---

## 3. Secret Scrubbing Architecture

### 3.1 Prevention at Ingestion (Not Scrubbing on Read)

**Design Decision**: Redact secrets **before** indexing to vector store.

**Benefits**:
- ✅ Cleaner: Secrets never enter the vector database
- ✅ More secure: Can't accidentally leak via query results
- ✅ Performance: No runtime scrubbing overhead
- ✅ Simpler: Single point of enforcement

**Implementation**: Gitleaks SDK integration at ingestion pipeline.

### 3.2 Ingestion Pipeline

```
File → Read Content → Gitleaks Scan → Redact Matches → Embed → Store
                            ↓
                    Check Allowlist
```

**Redaction Strategy**:
```
Original: github_pat_abc123def456...
Redacted: [REDACTED:github-pat:abc1]
          └─────┬─────┴──────┬────┴─┬─┘
                │           │      └─ First 4 chars (context)
                │           └─ Rule ID (what type)
                └─ Marker (indicates redaction)
```

**Preserves context** (what type of secret) without exposing value.

### 3.3 Hierarchical Allowlist System

**Two-Level Allowlist**:

**1. Project-level** (`.gitleaks.toml` in repo):
- Committed to version control
- Team-shared exceptions
- Example: Mock credentials in test fixtures

**2. User-level** (`~/.config/contextd/allowlist.toml`):
- Personal exceptions
- Not committed
- Example: User's demo API keys they want indexed

**Merge Strategy**: Union of both allowlists (project OR user).

**Configuration Format** (extends Gitleaks):
```toml
[allowlist]
description = "Contextd allowlist"
paths = [
  '''test/fixtures/.*\.env''',
  '''docs/examples/.*'''
]
regexes = [
  '''DEMO_API_KEY''',
  '''EXAMPLE_SECRET'''
]
```

### 3.4 Benefits

- **Security by Default**: All secrets caught unless explicitly allowed
- **Team Collaboration**: Project allowlist shared via git
- **Personal Flexibility**: User allowlist for individual needs
- **Audit Trail**: Redaction logs show what was caught
- **Context Preserved**: Redaction markers maintain semantic meaning

---

## 4. HTTP/SSE Transport

### 4.1 Clean Break from Stdio

**Why HTTP/SSE**:
- ✅ Better multi-tenancy (proper auth/authz)
- ✅ Network-ready (shared instances across teams)
- ✅ Standard observability (OpenTelemetry, metrics, traces)
- ✅ Security (TLS, JWT, rate limiting)
- ✅ Simpler deployment (systemd, Docker, cloud-native)

**No Stdio Support**: Hard switch, no backward compatibility.

### 4.2 MCP over HTTP/SSE

**Protocol**:
- MCP messages over HTTP POST endpoints
- SSE for streaming responses (long-running operations)
- JSON-RPC 2.0 message format (MCP standard)

**Endpoints**:
```
POST /mcp/tools/call          # Execute tool
POST /mcp/resources/read      # Read resource
GET  /mcp/resources/list      # List resources
GET  /mcp/tools/list          # List tools
GET  /mcp/sse                 # SSE event stream
GET  /health                  # Health check
```

### 4.3 Authentication & Authorization

**Owner-Based Auth (v2.1 MVP)**:
- System username → owner_id hash
- No passwords/tokens for single-user mode
- Unix socket for local security (optional)

**Future (v3.1+)**:
- JWT tokens for team/org mode
- OAuth/SSO integration
- Fine-grained ACLs

### 4.4 SSE Streaming

**Use Cases**:
- Progress updates for long operations (indexing, searching)
- Real-time notifications (index complete, errors)
- Connection management (heartbeat, reconnect)

**Example SSE Stream**:
```
event: progress
data: {"operation":"index","percent":45,"message":"Processing pkg/vectorstore"}

event: progress
data: {"operation":"index","percent":100,"message":"Indexing complete"}

event: complete
data: {"indexed_files":1234,"duration_ms":5678}
```

### 4.5 Benefits

- **Standard Tooling**: curl, Postman, OpenAPI docs
- **Observability**: Logs, metrics, traces built-in
- **Scalability**: Can add load balancers, replicas
- **Security**: Industry-standard auth patterns
- **Developer Experience**: Familiar HTTP debugging tools

---

## 5. Pluggable Embeddings & Vector Stores

### 5.1 langchaingo Abstraction Layer

**Design Decision**: Use langchaingo SDK instead of custom interfaces.

**Benefits**:
- ✅ Avoids over-engineering (YAGNI principle)
- ✅ Battle-tested abstractions
- ✅ Community support and updates
- ✅ 12+ vector databases supported out-of-box
- ✅ Multiple embedding providers

**Why NOT custom interfaces**: We don't need flexibility beyond what langchaingo provides.

### 5.2 Supported Vector Databases

**Via langchaingo**:
- Qdrant (default)
- Milvus
- Chroma
- Pinecone
- Weaviate
- PGVector (Postgres)
- Redis
- And 6+ more

**Configuration**:
```yaml
vectorstore:
  provider: "qdrant"  # or "milvus", "chroma", etc.
  url: "http://localhost:6333"
  collection_prefix: "contextd"
  timeout: 30s
```

### 5.3 Supported Embedding Providers

**Via langchaingo**:
- TEI (Text Embeddings Inference) - **Default**
- OpenAI
- Ollama
- Cohere
- Hugging Face
- Vertex AI

**Configuration**:
```yaml
embeddings:
  provider: "tei"  # Default: local TEI
  model: "BAAI/bge-small-en-v1.5"
  url: "http://localhost:8080"
  dimensions: 384

  # Alternative: OpenAI
  # provider: "openai"
  # model: "text-embedding-3-small"
  # api_key: "${OPENAI_API_KEY}"
```

### 5.4 Default Setup: TEI + Qdrant via Docker Compose

**Opinionated Local-First Setup**:

```yaml
# docker-compose.yml (shipped with contextd)
version: '3.8'

services:
  tei:
    image: ghcr.io/huggingface/text-embeddings-inference:latest
    ports:
      - "8080:80"
    volumes:
      - tei-data:/data
    environment:
      - MODEL_ID=BAAI/bge-small-en-v1.5
    command: --model-id ${MODEL_ID}

  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant-data:/qdrant/storage

volumes:
  tei-data:
  qdrant-data:
```

**Getting Started**:
```bash
# One command to start
docker-compose up -d

# contextd auto-detects and uses local services
contextd start
```

### 5.5 Future: hashicorp/go-plugin (v3.3+)

**Extensibility** (post-MVP):
- Plugin system for custom vector stores
- Plugin system for custom embeddings
- Users can provide their own implementations
- Hot-reload plugins without recompiling

**Not part of MVP** (YAGNI - build if proven necessary).

---

## 6. Advanced Features Roadmap

All three features align with contextd's **PRIMARY goal: minimize context bloat, maximize token efficiency**.

### 6.1 Pre-Fetching (Phase 1 - MVP)

**Concept**: "If you already know what tools you'll want the model to call, just call them DETERMINISTICALLY and let the model do the hard part of figuring out how to use their outputs." — 12-Factor Agents

**Implementation**:

**Scenario 1: User opens file in editor**
```
Traditional:
User opens file → Claude requests context → Search → Return results
(2 round trips, wasted tokens)

Pre-Fetch:
User opens file → Contextd detects → Auto-search related files → Include in context
(0 round trips, immediate context)
```

**Scenario 2: Git branch switch**
```
Traditional:
User switches branch → Claude asks "what changed?" → Diff → Return
(2 round trips)

Pre-Fetch:
User switches branch → Contextd detects → Auto-diff → Include in context
(0 round trips)
```

**Deterministic Pre-Fetch Rules**:
- File opened → Search for related files (same package, imports, tests)
- Branch switched → Include git diff summary
- New repo cloned → Include README, CLAUDE.md, package structure
- Error encountered → Include relevant docs, past solutions

**Configuration**:
```yaml
prefetch:
  enabled: true
  triggers:
    - file_open
    - branch_switch
    - error_detect
  max_context_kb: 50  # Limit pre-fetch size
```

**Metrics**:
- Pre-fetch hit rate (% of time model uses pre-fetched data)
- Round trips saved
- Context tokens reduced

### 6.2 Context Folding (Phase 2 - v3.4)

**From Research**: Context Folding Paper - 10× context reduction via branch/fold mechanism.

**Concept**: Agent manages working context by creating sub-trajectories for subtasks and folding them upon completion.

**Implementation**:

```
Active Context Window:
├── Current task context (always visible)
├── Folded summaries (compressed sub-tasks)
└── Unfolded detail (on-demand expansion)
```

**Features**:
- **Branch**: Create focused sub-context for specific task
- **Fold**: Compress completed sub-task to summary
- **Unfold**: Expand summary back to full detail when needed
- **Auto-fold**: Automatically fold old/irrelevant context

**API**:
```
POST /mcp/context/branch    # Create sub-context
POST /mcp/context/fold      # Compress to summary
POST /mcp/context/unfold    # Expand summary
GET  /mcp/context/status    # View fold hierarchy
```

**Benefits**:
- 10× smaller active context (proven in research)
- Maintains performance (no accuracy loss)
- User controls fold/unfold granularity

### 6.3 ReasoningBank (Phase 3 - v3.5)

**From Research**: ReasoningBank Paper - Distill reasoning strategies from successes AND failures.

**Concept**: Memory framework that captures generalizable patterns from agent experiences, enabling test-time scaling.

**Memory Item Structure**:
```json
{
  "title": "Search by interface for Go abstractions",
  "description": "When looking for implementations, search interface name first",
  "content": "In Go codebases, searching 'type Vectorstore interface' finds all implementations via grep. More efficient than description-based semantic search.",
  "trigger": "user asks about implementations/adapters",
  "success_count": 15,
  "failure_count": 2,
  "confidence": 0.88,
  "created_at": "2025-01-14T10:30:00Z",
  "last_used": "2025-01-14T15:45:00Z"
}
```

**Pattern Types**:
- **Success Patterns**: What worked well
- **Failure Patterns**: What didn't work (prevents repeat mistakes)
- **Contextual Triggers**: When to apply each strategy

**API**:
```
POST /mcp/reasoning/record      # Capture success/failure
GET  /mcp/reasoning/suggest     # Get relevant strategies
GET  /mcp/reasoning/patterns    # View all patterns
```

**Benefits**:
- Learn from past mistakes (failure patterns prevent repeat errors)
- Bootstrap new contexts faster (apply known strategies)
- Test-time scaling (better decisions over time)
- Owner-scoped learning (personal reasoning bank per user)

### 6.4 Phased Rollout

**Phase 1 (0.9.0-rc-1 MVP)**: Pre-Fetching
- Immediate value, low complexity
- Measurable ROI (tokens saved, latency reduced)
- Foundation for context management

**Phase 2 (v3.4)**: Context Folding
- Build on pre-fetch infrastructure
- Requires fold/unfold UI/UX design
- High impact (10× context reduction)

**Phase 3 (v3.5)**: ReasoningBank
- Requires successful folding (compressed memories)
- Needs time to collect success/failure patterns
- Long-term learning system

---

## 7. Implementation Priorities

### 7.1 MVP Scope (0.9.0-rc-1)

**Must Have**:

1. **HTTP/SSE Transport**
   - Basic HTTP server with MCP endpoints
   - SSE streaming for long operations
   - Owner-based auth (system username hash)
   - Health check endpoint

2. **Multi-Tenant Collections**
   - Owner-scoped isolation
   - Project-level collections
   - Branch-aware delta collections
   - Collection lifecycle (create, delete, list)

3. **Secret Scrubbing**
   - Gitleaks SDK integration
   - Redaction at ingestion
   - Hierarchical allowlist (project + user)
   - Basic redaction markers

4. **Pluggable Architecture**
   - langchaingo for embeddings
   - langchaingo for vector stores
   - Config-based provider switching
   - YAML configuration

5. **Default Stack**
   - TEI for embeddings (Docker Compose)
   - Qdrant for vector store (Docker Compose)
   - One-command setup

6. **Pre-Fetching (Phase 1)**
   - File open detection → Related file search
   - Branch switch detection → Git diff
   - Basic trigger rules
   - Metrics collection (hit rate, tokens saved)

**Nice to Have** (if time permits):
- Basic observability (OpenTelemetry traces)
- Graceful shutdown
- Config validation

### 7.2 Post-MVP Features

**v3.1 (Security & Auth)**:
- JWT authentication
- Rate limiting
- API key management
- Audit logging

**v3.2 (Team Features)**:
- Team-scoped collections
- Shared knowledge bases
- Organization hierarchy
- RBAC

**v3.3 (Extensibility)**:
- hashicorp/go-plugin system
- Custom vector store plugins
- Custom embedding plugins

**v3.4 (Context Folding)**:
- Branch/fold mechanism
- Fold/unfold API
- Auto-fold policies
- Fold hierarchy visualization

**v3.5 (ReasoningBank)**:
- Success/failure pattern capture
- Reasoning strategy storage
- Contextual suggestion engine
- Test-time scaling

**v3.6 (Advanced Features)**:
- Multi-modal embeddings (code + images + docs)
- Incremental indexing (delta updates)
- Distributed vector search

### 7.3 Implementation Timeline (12 weeks)

**Week 1-2: Foundation**
1. Project structure (golang-standards/project-layout)
2. HTTP server with Chi router
3. Config system (Viper + YAML)
4. Owner-scoped auth (username → hash)

**Week 3-4: Vector Core**
5. langchaingo integration (embeddings)
6. langchaingo integration (vector stores)
7. Collection management (create, delete, list)
8. Delta collection model (branch-aware)

**Week 5-6: Secret Scrubbing**
9. Gitleaks SDK integration
10. Allowlist system (project + user)
11. Redaction pipeline
12. Redaction marker format

**Week 7-8: MCP Protocol**
13. MCP tool endpoints
14. MCP resource endpoints
15. SSE streaming
16. Error handling

**Week 9-10: Pre-Fetching**
17. File watch detection
18. Git event detection
19. Pre-fetch rules engine
20. Metrics collection

**Week 11-12: Polish & Release**
21. Docker Compose defaults (TEI + Qdrant)
22. Documentation
23. Migration guide from v2.0
24. Release 0.9.0-rc-1

### 7.4 What We're NOT Building (YAGNI)

**Hard Constraints (Never)**:
- ❌ **GraphQL API** - Adds complexity without benefit. HTTP/SSE is the protocol, period.
- ❌ **Stdio transport** - Clean break, no backward compatibility

**Explicitly Deferred (Evaluate Later)**:
- ❌ NATS cache (benchmark first, add if proven needed)
- ❌ Web UI (CLI + MCP tools sufficient for MVP)
- ❌ Multi-region replication (single-user first)
- ❌ Distributed tracing (basic traces sufficient)
- ❌ Custom query language (vector search sufficient)
- ❌ Real-time collaboration (async sufficient)

**Rationale**: Build ONLY when proven necessary by real usage data.

---

## 8. Success Metrics

### 8.1 Context Efficiency (Primary Goal)

- [ ] **5× reduction in tokens per search** (12K → <3K)
- [ ] **<100ms search latency** (p95)
- [ ] **>90% relevance ratio** (useful results / total results)
- [ ] **Pre-fetch hit rate >70%**

### 8.2 Security

- [ ] **Zero cross-owner data leakage**
- [ ] **100% secret redaction** (no leaks in vector DB)
- [ ] **Allowlist override rate <5%** (most secrets caught)

### 8.3 Developer Experience

- [ ] **One command setup** (`docker-compose up -d && contextd start`)
- [ ] **Zero config** for default stack
- [ ] **Migration from v2.0 in <1 hour**

### 8.4 Performance

- [ ] **Index 10K files in <5 minutes**
- [ ] **Support 100+ concurrent connections**
- [ ] **<1GB memory footprint** for typical project

---

## 9. Migration Path

### 9.1 Breaking Changes

**No Backward Compatibility**:
- ❌ No stdio support (HTTP/SSE only)
- ❌ New collection model (requires re-indexing)
- ❌ New config format (YAML vs flags)

**This is a clean break.** Users must migrate intentionally.

### 9.2 Migration Steps

```bash
# 1. Export v2.0 data
contextd-v2 export --output=/tmp/contextd-export.json

# 2. Install 0.9.0-rc-1
go install github.com/contextd/contextd/v3@latest

# 3. Start default stack
docker-compose up -d

# 4. Import data
contextd import --input=/tmp/contextd-export.json

# 5. Update Claude Code config
contextd setup-claude

# 6. Verify migration
contextd status
```

### 9.3 What Migrates

**Preserved**:
- ✅ Checkpoints (converted to new format)
- ✅ Skills (owner-scoped)
- ✅ Remediations (owner-scoped)
- ✅ Troubleshooting entries (owner-scoped)

**NOT Preserved**:
- ❌ Search history (fresh start)
- ❌ Metrics (new metrics system)
- ❌ Old collection metadata (re-index required)

### 9.4 Migration Tool

**Provided**: `contextd migrate` command

```bash
# Dry run (no changes)
contextd migrate --from=v2 --to=v3 --dry-run

# Actual migration
contextd migrate --from=v2 --to=v3

# Rollback (if needed within 24h)
contextd migrate --rollback
```

---

## Appendix A: Package Preservation

### A.1 Packages to Preserve (from v2.0)

**Keep (with refactoring)**:
- `pkg/logging` - Uber Zap integration (well-tested)
- `pkg/security` - Gitleaks wrapper (core to v3)
- `pkg/telemetry` - OpenTelemetry basics (for v3 observability)
- `pkg/config` - Viper integration (refactor for YAML-first)

**Keep (minimal changes)**:
- `pkg/checkpoint` - Core checkpoint logic (change storage layer only)
- `pkg/skills` - Skills management (owner-scoping updates)
- `pkg/remediation` - Remediation storage (owner-scoping updates)

### A.2 Packages to Remove

**Complete Removal**:
- `pkg/mcp` - Old stdio-based MCP (replaced with HTTP/SSE)
- `pkg/vectorstore` - Custom abstractions (replaced with langchaingo)
- `pkg/embedding` - Custom interfaces (replaced with langchaingo)
- `pkg/milvus` - Direct integration (replaced with langchaingo)
- `pkg/qdrant` - Direct integration (replaced with langchaingo)
- `pkg/session` - Stdio session management (not needed for HTTP)
- `pkg/hooks` - Stdio-specific hooks (replaced with HTTP middleware)
- `pkg/migration` - Old migration tools (v3 has new migration)
- `pkg/analytics` - Old analytics (new metrics system)
- `pkg/metrics` - Old metrics (new system)
- `pkg/testmetrics` - Old test metrics (new system)
- `pkg/compression` - Not used in MVP (YAGNI)
- `pkg/ratelimit` - Defer to v3.1 (not in MVP)
- `pkg/auth` - Old auth system (replaced with owner-scoped)
- `pkg/backup` - Not in MVP (YAGNI)
- `pkg/composition` - Not in MVP (YAGNI)
- `pkg/tool-composition` - Not in MVP (YAGNI)
- `pkg/detector` - Environment detection (not needed for HTTP)
- `pkg/docscraper` - Not in MVP (YAGNI)
- `pkg/installer` - Old installer (new setup process)
- `pkg/research` - Not in MVP (defer to v3.5)
- `pkg/troubleshooting` - Not in MVP (defer to v3.5)
- `pkg/validation` - Replace with new validation

---

## Appendix B: References

**Research Papers**:
- Context Folding: Agent-based context management via branch/fold
- ReasoningBank: Reasoning strategy distillation from successes/failures
- 12-Factor Agents (Appendix 13): Pre-fetching deterministic tool calls

**Architecture Documents**:
- ADR-003: Single-developer multi-repo isolation (v2.1)
- Team-Aware Architecture v2.2: Future team/org model

**External Documentation**:
- langchaingo: https://github.com/tmc/langchaingo
- Gitleaks: https://github.com/gitleaks/gitleaks
- MCP Specification: https://modelcontextprotocol.io
- Text Embeddings Inference (TEI): https://github.com/huggingface/text-embeddings-inference

---

**End of Specification**
