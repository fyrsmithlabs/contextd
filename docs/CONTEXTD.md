# Contextd: Shared Knowledge Layer for AI Agents

**Version**: 1.0.0-draft
**Status**: Design Phase
**Protocol**: MCP 2025-03-26

---

## Maintenance Instructions (For Agents)

**Update this document when:**
- New tool categories added to contextd
- Architecture changes (new components)
- Security model changes
- New agent compatibility confirmed
- Core principles change (requires human approval)

**Do NOT update when:**
- Implementation details change (update @imported specs instead)
- Adding examples or tutorials (create separate guides)
- Temporary or experimental features

**How to update:**
1. Update the `Version` field
2. Keep main file ≤150 lines (use @imports for details)
3. Maintain noun-heavy style (what/where, not how/when)
4. Run through Lyra optimization if changing Tier 0

---

## What Contextd Is

A shared knowledge layer where **AI agents and developers draw from the same institutional memory**.

| Layer | Purpose |
|-------|---------|
| **ReasoningBank** | Cross-session memory (learns from successes AND failures) |
| **Institutional Knowledge** | Survives developer tenure (Project → Team → Org) |
| **Context Optimization** | Lazy-load everything, never pre-fill |
| **Security** | Secret scrubbing (gitleaks), multi-tenant isolation |

**Core Principle**: Contextd adapts to YOUR workflows. Not the reverse.

---

## What Contextd Is NOT

- **NOT a code generator** - provides context, doesn't write code
- **NOT a prompt engineering tool** - memory/knowledge focus
- **NOT a workflow replacement** - enhances existing workflows
- **Future (Phase 2+)**: Model routing, orchestration

---

## Who Uses Contextd

| Scope | Value |
|-------|-------|
| **Individual** | Fast AI onboarding without learning agent/skill/plugin complexity |
| **Team** | Shared knowledge at project/team levels |
| **Organization** | Institutional memory that outlasts tenure, compliance-ready |

**Hierarchy**: Project → Team → Org (knowledge cascades upward)

---

## Architecture

```
AI Agents (Claude, Grok, Codex, OpenCode)
         │
         │ MCP Protocol (stdio / Streamable HTTP)
         ▼
┌─────────────────────────────────────────┐
│         contextd MCP Server             │
│  ├── Memory Manager (ReasoningBank)     │
│  ├── Session Manager (checkpoints)      │
│  ├── Standards Engine (policies)        │
│  ├── Secret Scrubber (gitleaks)         │
│  └── Tool Executors (seccomp/namespace) │
└─────────────────────────────────────────┘
         │
         │ gRPC / HTTP API
         ▼
┌─────────────────────────────────────────┐
│         Qdrant Vector Store             │
│  memories, remediations, policies,      │
│  skills, agents, checkpoints            │
└─────────────────────────────────────────┘
```

**Details**: @./spec/interface/SPEC.md

---

## Tool Discovery

**Default**: List `./servers/contextd/` per [Anthropic MCP pattern](https://www.anthropic.com/engineering/code-execution-with-mcp)

**Fallback**: MCP `tools/list` (for systems without filesystem)

**If unavailable**: Warn user, ask how to proceed

---

## Tool Categories

| Category | Tools | Purpose |
|----------|-------|---------|
| **Memory** | `memory_search`, `memory_record`, `memory_feedback` | ReasoningBank core |
| **Checkpoint** | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` | Session persistence |
| **Policy** | `policy_list`, `policy_check`, `policy_*` (CRUD) | Org/team/project governance |
| **Skills** | `skill_list`, `skill_get`, `skill_*` (CRUD) | Shareable workflows |
| **Agents** | `agent_list`, `agent_get`, `agent_*` (CRUD) | Agent configurations |
| **Remediation** | `remediation_search`, `remediation_record` | Error fixes learned |
| **Safe Exec** | `safe_bash`, `safe_read`, `safe_write` | Secret-scrubbed wrappers |
| **Knowledge** | `briefing_get`, `knowledge_promote` | Onboarding, scope escalation |

**Full Reference**: @./spec/interface/architecture.md

---

## Security Model

| Layer | Protection |
|-------|------------|
| **D1: Process Isolation** | seccomp + Linux namespaces |
| **D2: Capability Model** | Deny-by-default tool access |
| **D3: Secret Scrubbing** | gitleaks (tool-level + server-level) |
| **D4: Multi-Tenant** | Collection-per-tenant in Qdrant |
| **D5: Audit Logging** | All operations logged |

**Details**: @./spec/multi-tenancy/SPEC.md

---

## Agent Compatibility

| Agent | Compatible | Notes |
|-------|------------|-------|
| Claude Code | Yes | stdio transport |
| OpenCode | Yes | stdio transport |
| Grok | Yes | Streamable HTTP |
| Codex | Yes | Streamable HTTP |
| Any MCP client | Yes | Protocol 2025-03-26 |

**Requirement**: MCP server support OR MCP code execution tools

---

## Session Behavior

| Event | Contextd Action |
|-------|-----------------|
| **session_start** | Inject Tier 0 (~100 tokens), lazy-load via tools |
| **Auto-checkpoint** | Silent save at configurable context % thresholds |
| **`/clear`** | Prompt: "Resume from checkpoint?" |
| **`/resume`** | Load optimized summary only (not full context) |
| **session_end** | Async distillation → ReasoningBank |

**Principle**: Never pre-fill context. Always lazy-load.

---

## Tier 0 Injection (session_start)

@./TIER-0-INJECTION.md

---

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Qdrant over Pinecone | Local-first, open source, native multi-tenancy |
| Collection-per-tenant | Physical isolation for SaaS compliance |
| Go + seccomp/namespace | Native performance, proven isolation (Phase 1) |
| gitleaks SDK | Industry-standard secret detection |
| Async distillation | Don't block agent sessions |

---

## References

| Resource | Link |
|----------|------|
| ReasoningBank Paper | arXiv:2509.25140 |
| Context-Folding Paper | arXiv:2510.11967 |
| MCP Specification | modelcontextprotocol.io/specification/2025-03-26 |
| Anthropic Code Execution | anthropic.com/engineering/code-execution-with-mcp |
| Qdrant Docs | qdrant.tech/documentation |

---

## Detailed Specifications

- @./spec/reasoning-bank/SPEC.md
- @./spec/context-folding/SPEC.md
- @./spec/interface/SPEC.md
- @./spec/collection-architecture/SPEC.md
- @./spec/multi-tenancy/SPEC.md
- @./spec/consolidation/SPEC.md
