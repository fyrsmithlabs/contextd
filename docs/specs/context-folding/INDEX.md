# Context-Folding Architecture Specification

**Status**: Design
**Created**: 2025-01-17
**Author**: Architecture Team
**Research**: [Scaling Long-Horizon LLM Agent via Context-Folding](https://context-folding.github.io/) (ByteDance/CMU, Oct 2025)

---

## Executive Summary

Contextd becomes a **context-folding engine** implementing research from ByteDance and Carnegie Mellon University. Agents manage working context through branch/fold operations, achieving 10× context compression while maintaining full trajectory history for meta-learning.

**Core Innovation**: Agent creates sub-trajectory for focused subtask (branch), then collapses intermediate steps into summary (fold). Main thread stays compressed, archive preserves full detail.

**Primary Goal**: Enable long-horizon agent workflows without context overflow.

---

## Specification Structure

This spec uses modular organization. Read sections in sequence:

### Core Mechanism
1. **[Architecture Overview](01-architecture.md)** - High-level design, storage tiers, multi-tenant integration
2. **[MCP Protocol](02-mcp-protocol.md)** - Branch/fold tools, state tracking, session management

### Intelligence Layer
3. **[Process Reward Heuristics](03-process-rewards.md)** - Real-time guidance (token penalties, scope detection, failure tracking)
4. **[Meta-Learning Layer](04-meta-learning.md)** - Strategy optimization, composite scoring, NATS event-driven updates

### Integration
5. **[Branch-Aware Features](05-branch-aware-features.md)** - Checkpoints, remediations, skills with branch context
6. **[Secret Scrubbing](06-secret-scrubbing.md)** - Double-scan strategy, execution + fold-time detection
7. **[Git Pre-Fetch Integration](07-git-prefetch.md)** - LLM-directed pre-fetch, worktree isolation

### Implementation
8. **[Implementation Phases](08-implementation-phases.md)** - 6-phase rollout, success metrics, testing requirements

---

## Research Foundation

**Paper**: "Scaling Long-Horizon LLM Agent via Context-Folding"
**Authors**: Weiwei Sun, Miao Lu, Zhan Ling, et al. (ByteDance/CMU)
**Published**: October 2025
**Results**: 62% pass@1 on BrowseComp-Plus, 58% on SWE-Bench using 32K active context (vs 327K baseline)

**Key Findings**:
- 10× context compression through strategic folding
- Reinforcement learning (FoldGRPO) trains effective branching behavior
- Process rewards guide branch management without RL training
- Adaptive branching scales to harder problems (3-5 branches typical, up to 32 for complex tasks)

---

## Current Implementation Status

Contextd provides strong foundation:

**✅ Implemented** (~70% of infrastructure):
- NATS event system (perfect for branch state coordination)
- Secret scrubbing with Gitleaks (exact redaction format we need)
- Multi-tenant architecture (database-per-project isolation)
- Pre-fetch engine (ready for LLM control)
- MCP infrastructure (12 tools, HTTP transport, SSE streaming)
- Observability (OTLP, Prometheus, distributed tracing)

**❌ To Build** (context-folding features):
- Branch/fold MCP tools and state management
- Process reward heuristics
- Meta-learning layer with composite scoring
- Branch-aware checkpoint/remediation/skill features
- Double-scan secret detection (execution + fold)
- LLM-directed pre-fetch

---

## Design Principles

**YAGNI (You Aren't Gonna Need It)**:
- Build only Phase 1 infrastructure initially
- Validate with real usage before adding intelligence
- Skip RL training scope (use heuristics instead)

**Leverage Existing Infrastructure**:
- NATS for branch state (subjects: `{owner_id}/{project_hash}/{session_id}/branches/{branch_id}`)
- Qdrant two-tier storage (main + archive collections)
- Gitleaks for secret scrubbing (already proven)
- Multi-tenant isolation (worktrees = different projects)

**Incremental Value**:
- Phase 1 delivers basic branch/fold (immediate value)
- Phase 2 adds guidance heuristics (improves quality)
- Phase 4 enables self-improvement (long-term learning)

---

## Success Metrics

**Performance** (from research paper):
- 10× context compression (100K trajectory → 10K main thread)
- <100ms search latency (maintain existing performance)
- >80% test coverage (all new code)

**Effectiveness**:
- 5× token efficiency improvement
- 70%+ strategy recommendation accuracy
- 20-30% token savings on git workflows

**Security**:
- Zero secret leaks (verified via audit)
- 100% scan coverage (every branch operation)
- Double-scan validation (fold-time catches execution misses)

---

## Next Steps

**Before Implementation**:
1. **Fix current MCP connection** - Existing contextd must work before adding features
2. Review and approve spec sections
3. Create implementation worktree

**Phase 1 Implementation** (Weeks 1-2):
- Core branch/fold mechanism
- NATS state tracking
- Two-tier storage
- Basic token counting
- HTTP transport (not SSE for MCP)

---

## Related Documentation

- **Research Paper**: https://context-folding.github.io/
- **Multi-Tenant Spec**: [../multi-tenant/SPEC.md](../multi-tenant/SPEC.md)
- **Pre-Fetch Engine**: [../mcp-prefetch/SPEC.md](../mcp-prefetch/SPEC.md)
- **Secret Scrubbing**: [../logging/SPEC.md](../logging/SPEC.md)
- **Project Guidelines**: [../../CLAUDE.md](../../CLAUDE.md)

---

**CRITICAL**: Current contextd must work before implementing this spec. MCP connection issues block all development.
