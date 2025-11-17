# 8. Implementation Phases

[← Back to Git Pre-Fetch](07-git-prefetch.md) | [Back to Index](INDEX.md)

---

## Overview

Context-folding rolls out in 6 phases over 9 weeks. Each phase delivers value independently, validates before proceeding.

**CRITICAL**: Fix current MCP connection before starting Phase 1.

---

## Phase 1: Core Branch/Fold Mechanism (Weeks 1-2)

**Deliverables**:
- MCP tools: `context_branch`, `context_return`, `context_branch_status`
- NATS state tracking: `{owner}/{project}/{session}/branches/{branch_id}`
- Two-tier storage (main + archive collections)
- Basic token counting
- HTTP transport validation

**Success Criteria**:
- ✅ Create branch, operate, fold successfully
- ✅ Multi-tenant isolation verified (worktrees independent)
- ✅ Token counts accurate

**Test Cases**:
- Single branch workflow
- Nested branches (main → br1 → br2)
- Multi-project isolation

---

## Phase 2: Process Reward Heuristics (Week 3)

**Deliverables**:
- Unfolded token penalty
- Out-of-scope detection (semantic similarity)
- Failure penalty tracking
- Warning system in MCP responses

**Success Criteria**:
- ✅ Warnings appear when main thread >50% full
- ✅ Out-of-scope operations detected with >80% accuracy
- ✅ Failed tool calls penalize branch score

---

## Phase 3: Secret Scrubbing Integration (Week 4)

**Deliverables**:
- Execution-time Gitleaks scanning
- Fold-time summary scanning
- Archive encryption (AES-256-GCM)
- NATS secret detection events

**Success Criteria**:
- ✅ Secrets redacted before storage
- ✅ LLM summaries scrubbed
- ✅ Zero secrets in main collections
- ✅ Allowlists respected

---

## Phase 4: Meta-Learning Layer (Weeks 5-6)

**Deliverables**:
- Strategy collection in Qdrant
- Real-time NATS subscribers
- Scheduled analysis job
- `context_recommend_strategy` tool
- Composite scoring

**Success Criteria**:
- ✅ Strategies created from patterns
- ✅ Scores calculated (40/30/30)
- ✅ Recommendations improve over time

---

## Phase 5: Branch-Aware Features (Weeks 7-8)

**Deliverables**:
- Checkpoint: save/restore branch state
- Remediation: track branch patterns
- Skills: branch templates
- New tools: `checkpoint_restore_with_branches`, `skill_apply_with_branching`

**Success Criteria**:
- ✅ Checkpoint restores branch state
- ✅ Remediations track "works in branch" patterns
- ✅ Skills auto-create branches

---

## Phase 6: LLM-Directed Pre-Fetch (Week 9)

**Deliverables**:
- Git hook integration
- Preview generation
- `prefetch_handle` tool
- Worktree isolation verification

**Success Criteria**:
- ✅ Large diffs recommend auto-branch
- ✅ Small diffs suggest main thread
- ✅ 20-30% token savings maintained

---

## Success Metrics (Overall)

**Performance**:
- 10× context compression (100K → 10K)
- <100ms search latency
- >80% test coverage

**Effectiveness**:
- 5× token efficiency improvement
- >70% strategy recommendation accuracy
- 20-30% token savings on git workflows

**Security**:
- Zero secret leaks
- 100% scan coverage
- Double-scan validation

---

[← Back to Git Pre-Fetch](07-git-prefetch.md) | [Back to Index](INDEX.md)
