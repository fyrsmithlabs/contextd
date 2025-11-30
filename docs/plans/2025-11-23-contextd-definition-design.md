# Contextd Definition & Architecture Design

**Date**: 2025-11-23
**Status**: Complete
**Session**: Brainstorming → Architecture → API Design

---

## Executive Summary

Contextd is a **shared knowledge layer** where AI agents and developers draw from the same institutional memory. It enables teams to preserve knowledge that outlasts developer tenure while adapting to existing workflows.

---

## What Contextd Is

| Layer | Purpose |
|-------|---------|
| **ReasoningBank** | Cross-session memory (learns from successes AND failures) |
| **Institutional Knowledge** | Survives developer tenure (Project → Team → Org) |
| **Context Optimization** | Lazy-load everything, never pre-fill |
| **Security** | Secret scrubbing (gitleaks), multi-tenant isolation |

**Core Principle**: Contextd adapts to YOUR workflows. Not the reverse.

---

## What Contextd Is NOT

- NOT a code generator
- NOT a prompt engineering tool
- NOT a workflow replacement
- NOT an orchestration system (Phase 1)

**Future (Phase 2+)**: Model routing, orchestration

---

## Target Users

| Scope | Value |
|-------|-------|
| **Individual** | Fast AI onboarding without learning agent/skill/plugin complexity |
| **Team** | Shared knowledge at project/team levels |
| **Organization** | Institutional memory that outlasts tenure, compliance-ready |

---

## Architecture Decisions

### Tool Binary Architecture

| Decision | Choice |
|----------|--------|
| **Deployment** | Grouped tool binaries (thin clients) + contextd server (heavy logic) |
| **Invocation** | Subcommand: `./safe-exec read '{"path":"...","session_id":"..."}'` |
| **Schema** | Object keyed by tool name in `schema.json` |
| **TOOL.md** | Table with `@schema.json:LINE` refs for lazy loading |

### Tool Binary Groups

| Binary | Tools | Purpose |
|--------|-------|---------|
| `safe-exec` | bash, read, write | Filesystem/shell with scrubbing |
| `memory` | search, store, feedback, get | ReasoningBank |
| `checkpoint` | save, list, resume | Session persistence |
| `policy` | check, list, get | Governance |
| `skill` | list, get, create, update, delete | Skill CRUD |
| `agent` | list, get, create, update, delete | Agent CRUD |
| `remediation` | search, record | Error patterns |

### Server Architecture

| Component | Technology |
|-----------|------------|
| **Server** | Native Go |
| **gRPC** | Per-service typed methods (hashicorp/go-plugin) |
| **HTTP** | REST API for automation |
| **Scrubbing** | gitleaks SDK |
| **Storage** | Qdrant (refs, memories, checkpoints) + cache layer |
| **Isolation** | seccomp + Linux namespaces |

### Response Pattern

**Tiered responses** for token efficiency:
```
{
  "summary": "Read 150 lines from /etc/hosts",
  "content_preview": "...",
  "content_ref": "ref_abc123",
  "tokens_used": 45
}
```

- `summary`: Always returned, concise
- `*_preview`: First N chars/lines
- `*_ref`: ID to resolve full content via RefService

### Session Management

- `session_id` passed from Claude in every JSON input
- Server validates/creates session
- Session scopes: checkpoints, memory queries, audit logs

### Security (Phase 1)

| Aspect | Phase 1 | Future |
|--------|---------|--------|
| **Auth** | None, localhost trust | mTLS or shared secret |
| **Transport** | Localhost gRPC | Unix socket + file perms |
| **Scrubbing** | gitleaks, server-side | Dual: tool + server |

---

## Document Tiers

| Tier | Location | Tokens | Purpose |
|------|----------|--------|---------|
| **0** | Session injection | ~100 | Behavioral directives |
| **1** | CLAUDE.md | ~300 | Project context |
| **2** | docs/CONTEXTD.md | ~1000+ | Full briefing |

**Principle**: Never pre-fill context. Always lazy-load.

---

## API Specifications

### gRPC Services

@../spec/mcp-interface/api/contextd.proto

| Service | Methods |
|---------|---------|
| `SafeExecService` | Bash, Read, Write |
| `MemoryService` | Search, Store, Feedback, Get |
| `CheckpointService` | Save, List, Resume |
| `PolicyService` | Check, List, Get |
| `SkillService` | List, Get, Create, Update, Delete |
| `AgentService` | List, Get, Create, Update, Delete |
| `RemediationService` | Search, Record |
| `RefService` | GetContent |
| `SessionService` | Start, End |

### REST API

@../spec/mcp-interface/api/openapi.yaml

| Base | Endpoints |
|------|-----------|
| `/api/v1/safe-exec` | bash, read, write |
| `/api/v1/memory` | search, CRUD |
| `/api/v1/checkpoint` | save, list, resume |
| `/api/v1/policy` | check, list, get |
| `/api/v1/skill` | CRUD |
| `/api/v1/agent` | CRUD |
| `/api/v1/remediation` | search, record |
| `/api/v1/ref` | get content |
| `/api/v1/session` | start, end |

---

## Directory Structure

```
./servers/contextd/
├── safe-exec/
│   ├── safe-exec              # Binary
│   ├── TOOL.md                # Tool list with @schema.json refs
│   └── schema.json            # Input/output specs
├── memory/
├── checkpoint/
├── policy/
├── skill/
├── agent/
└── remediation/
```

---

## Flow Diagram

```
Claude Agent
    │
    │ 1. List ./servers/contextd/
    │ 2. Read TOOL.md (lazy)
    │ 3. Read schema.json:lines (on demand)
    │ 4. Invoke: ./safe-exec bash '{"cmd":"...","session_id":"..."}'
    ▼
Tool Binary (thin client)
    │
    │ gRPC call
    ▼
contextd Server
    ├── Secret Scrubber (gitleaks)
    ├── Process Isolation (seccomp)
    ├── Cache Layer
    ├── Qdrant Client
    └── Audit Logger
    │
    │ Token-optimized response
    ▼
Tool Binary → JSON output → Claude
```

---

## Files Created This Session

| File | Purpose |
|------|---------|
| `docs/CONTEXTD.md` | Tier 2 full briefing |
| `docs/TIER-0-INJECTION.md` | Tier 0 session start template |
| `CLAUDE.md` | Tier 1 project context (updated) |
| `docs/spec/mcp-interface/architecture.md` | Architecture spec |
| `docs/spec/mcp-interface/api/contextd.proto` | gRPC service definitions |
| `docs/spec/mcp-interface/api/openapi.yaml` | REST API spec |

---

## Next Steps

1. [ ] Create example schema.json + TOOL.md for safe-exec
2. [ ] Write Phase 1 implementation plan
3. [ ] Scaffold Go project structure
4. [ ] Implement core gRPC server
5. [ ] Implement safe-exec tool binary
6. [ ] Add gitleaks integration
7. [ ] Add Qdrant client

---

## Research References

| Topic | Source |
|-------|--------|
| ReasoningBank | arXiv:2509.25140 (Google, Sept 2025) |
| Context-Folding | arXiv:2510.11967 (ByteDance, Oct 2025) |
| MCP Spec | modelcontextprotocol.io/specification/2025-03-26 |
| Anthropic Code Execution | anthropic.com/engineering/code-execution-with-mcp |
| Multi-tenant Vector Security | Milvus, Pinecone best practices |

---

## Key Insights from Research

| Insight | Application |
|---------|-------------|
| ReasoningBank: 34% effectiveness gain | Memory from successes AND failures |
| Context-Folding: branch/return | Future Phase 2 feature |
| Anthropic: `./servers/` discovery | Tool binary directory pattern |
| Multi-tenant: Collection-per-tenant | Qdrant isolation strategy |
| Embedding inversion: 92% recovery | Encrypt sensitive data in vectors |
