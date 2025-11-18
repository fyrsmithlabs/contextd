> **⚠️ OUTDATED CHECKPOINT**
>
> This checkpoint documents port 9090 / owner-based authentication architecture.
> Current architecture uses HTTP transport on port 8080 with no authentication.
> See `docs/standards/architecture.md` for current architecture.

---

# Checkpoint: Context-Folding Spec Complete

**Date**: 2025-01-17
**Session**: Brainstorming → Design → Spec Creation
**Status**: Design Phase Complete - Awaiting MCP Fix Before Implementation

---

## Work Completed

### Context-Folding Architecture Spec

Created comprehensive modular specification for context-folding engine based on ByteDance/CMU research.

**Location**: `docs/specs/context-folding/`

**Files Created** (9 files, 1,406 lines):
1. `INDEX.md` - Navigation and executive summary
2. `01-architecture.md` - High-level design, two-tier storage, multi-tenant integration
3. `02-mcp-protocol.md` - Branch/fold MCP tools, NATS state tracking
4. `03-process-rewards.md` - Heuristics without RL training
5. `04-meta-learning.md` - Composite scoring (40/30/30), strategy optimization
6. `05-branch-aware-features.md` - Checkpoints, remediations, skills with branch context
7. `06-secret-scrubbing.md` - Double-scan strategy (execution + fold-time)
8. `07-git-prefetch.md` - LLM-directed pre-fetch
9. `08-implementation-phases.md` - 6-phase rollout (9 weeks)

**Commit**: `4b5b0da` - "docs: add context-folding architecture specification"

---

## Key Design Decisions

**Full ReasoningBank Implementation** (no RL training):
- Strategy memory via remediations (already exists)
- Test-time evolution across search/context/performance
- Meta-learning with composite scoring
- All using heuristics instead of reinforcement learning

**Infrastructure Choices**:
- HTTP transport (not SSE - SSE deprecated for MCP)
- NATS for branch state coordination
- Qdrant two-tier storage (main + archive)
- Gitleaks double-scan (execution + fold)
- Hybrid state: NATS ephemeral + Qdrant persistent

**Scope Clarifications**:
- Skip RL training infrastructure (use heuristics)
- LLM-directed pre-fetch (not automatic)
- Process rewards as soft warnings (configurable hard limits)
- Multi-tenant worktree isolation built-in

---

## Current Implementation Status

**✅ Existing Infrastructure** (~70%):
- NATS event system (perfect for branch state)
- Secret scrubbing with Gitleaks (exact format we designed)
- Multi-tenant architecture (database-per-project)
- Pre-fetch engine (ready for LLM control)
- MCP infrastructure (12 tools, HTTP transport)
- Observability (OTLP, Prometheus)

**❌ To Build** (context-folding features):
- Branch/fold MCP tools
- Process reward heuristics
- Meta-learning layer
- Branch-aware features
- Double-scan integration
- LLM-directed pre-fetch

---

## CRITICAL BLOCKER

**MCP Connection Issue**: Current contextd NOT working before we can implement new features.

**Symptoms**:
- contextd running on port 9090 ✅
- Health check responds ✅
- 12 MCP tools available ✅
- But MCP connection fails ❌

**Configuration Issues Attempted**:
1. Created `.mcp.json` in project (failed - wrong schema)
2. Used `claude mcp add -s user -t sse` (created wrong config structure)
3. Fixed config structure manually (still failing)
4. Removed `.mcp.json`, added to `~/.claude.json` (still failing)

**Next Step**: Debug MCP connection systematically before implementing ANY context-folding features.

---

## Implementation Plan (After MCP Fixed)

**Phase 1** (Weeks 1-2): Core branch/fold mechanism
**Phase 2** (Week 3): Process reward heuristics
**Phase 3** (Week 4): Secret scrubbing integration
**Phase 4** (Weeks 5-6): Meta-learning layer
**Phase 5** (Weeks 7-8): Branch-aware features
**Phase 6** (Week 9): LLM-directed pre-fetch

**Success Metrics**:
- 10× context compression (100K → 10K tokens)
- <100ms search latency
- >80% test coverage
- Zero secret leaks

---

## Context Review Findings

**Code Review Results**:
- `pkg/mcp/server.go` - 12 MCP tools via HTTP, SSE streaming for progress
- `pkg/mcp/operations.go` - NATS operation tracking with subjects `operations.{owner}.{op_id}.{event}`
- `pkg/secrets/` - Gitleaks SDK with `[REDACTED:rule-id:preview]` format
- `pkg/prefetch/` - Git event detection, worktree support, deterministic rules
- `pkg/checkpoint/`, `pkg/remediation/`, `pkg/skills/` - Core services ready for branch awareness
- Multi-tenant: `project_<hash>` databases, owner-based auth

**Gap Analysis**:
- No branch/fold primitives (need Phase 1)
- No process reward calculation (need Phase 2)
- No meta-learning strategy storage (need Phase 4)
- No branch context in existing services (need Phase 5)

---

## Version Clarification

**NOT v3.0** - User corrected this assumption.
**Version TBD** - Will decide later, focus on features not version numbers.

---

## Research Foundation

**Paper**: "Scaling Long-Horizon LLM Agent via Context-Folding"
**Authors**: Weiwei Sun, Miao Lu, Zhan Ling, et al. (ByteDance/CMU)
**Published**: October 2025
**URL**: https://context-folding.github.io/

**Results**:
- 62% pass@1 on BrowseComp-Plus
- 58% pass@1 on SWE-Bench Verified
- Using 32K active context vs 327K baseline
- 10× context compression

---

## Next Session Actions

1. **Fix MCP connection** (CRITICAL - blocks everything)
   - Debug why connection fails despite correct config
   - Test with minimal configuration
   - Verify HTTP transport working
   - Ensure contextd MCP tools discoverable

2. **Validate existing tools work**
   - Test checkpoint_save/search
   - Test remediation_save/search
   - Test collection management
   - Verify multi-tenant isolation

3. **Then begin Phase 1** (only after MCP working)
   - Create implementation worktree
   - Write detailed task breakdown
   - Implement core branch/fold mechanism

---

## Files Modified (Not Committed)

Unstaged changes exist:
- `CHANGELOG.md`
- `README.md`
- `cmd/contextd/main.go`
- Deleted old specs (backup, embedding-migration, skills, tui-dashboard, v3-rebuild)
- Modified MCP helpers
- New auth middleware
- New collection management

**Decision**: Don't commit these until MCP connection works.

---

## Key Quotes from Session

**User**: "I also just learned The SSE (Server-Sent Events) transport is deprecated. Use HTTP servers instead"

**User**: "It's very important that this MUST work across projects/git worktrees/branches"

**User**: "Write the design in as a spec chunking the files. Before we implement I NEED the current version working"

**User**: "write a checkpoint file. context 1%"

---

## Resume Instructions

**Immediate Priority**: Fix MCP connection before implementing context-folding.

**Steps**:
1. Verify contextd service status
2. Check actual MCP configuration in `~/.claude.json`
3. Test HTTP endpoint manually: `curl http://localhost:9090/mcp/tools/list`
4. Review contextd logs for errors
5. Test minimal MCP config
6. Restart Claude Code with working config

**Once MCP Works**:
1. Test all 12 existing MCP tools
2. Create implementation worktree for Phase 1
3. Follow spec: `docs/specs/context-folding/08-implementation-phases.md`

---

## References

- **Spec**: `docs/specs/context-folding/INDEX.md`
- **Research**: https://context-folding.github.io/
- **Multi-Tenant**: `docs/specs/multi-tenant/SPEC.md`
- **Pre-Fetch**: `docs/specs/mcp-prefetch/SPEC.md`
- **Project Guidelines**: `CLAUDE.md`
