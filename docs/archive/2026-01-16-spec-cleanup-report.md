# Specification Cleanup Report

**Date**: 2026-01-16
**Task**: Clean up documentation to match MCP architecture (migrated from gRPC)

## Summary

Reviewed all high-priority spec directories and classified each as CURRENT, OUTDATED, or OBSOLETE.
Made updates to align documentation with the actual codebase.

## Classifications and Actions

### CURRENT (No changes needed)

| Spec | Status | Notes |
|------|--------|-------|
| `docs/spec/context-folding/SPEC.md` | Current | Already references `internal/folding/` correctly |
| `docs/spec/context-folding/CONSENSUS-REVIEW.md` | Current | Documents architectural decisions accurately |
| `docs/spec/reasoning-bank/SPEC.md` | Current | Updated with MCP tools and implementation notes |
| `docs/spec/policy/SPEC.md` | Current | Already marked as "FUTURE - Not Implemented" |
| `docs/spec/skills-system/SPEC.md` | Current | Already marked as "FUTURE - Not Implemented" |
| `docs/spec/interface/SPEC.md` | Current | Already marked as DEPRECATED with proper notice |

### UPDATED (Refreshed to match implementation)

| Spec | Changes Made |
|------|--------------|
| `docs/spec/reasoning-bank/ARCH.md` | Updated status to "Implemented"; Changed diagram to show `internal/mcp/`, `internal/reasoningbank/`, `internal/vectorstore/`; Updated collection schema to 384-dim vectors with chromem; Replaced Qdrant-specific code with generic vectorstore interface; Added notes about unimplemented team/org scoping |
| `docs/spec/reasoning-bank/DESIGN.md` | Updated status to "Implemented"; Added note about pending consolidation features; Added reference to `internal/reasoningbank/` |
| `docs/spec/context-folding/ARCH.md` | Updated status from "Draft" to "Implemented"; Added implementation path reference |
| `docs/spec/context-folding/DESIGN.md` | Updated status to "Implemented"; Added note about tool name changes (`branch` -> `branch_create`, `return` -> `branch_return`) |

### ARCHIVED (Moved to docs/archive/obsolete-specs/)

| Original Path | Reason |
|---------------|--------|
| `docs/spec/interface/architecture.md` | References gRPC, cmux, PolicyService, SkillService, AgentService - none exist in current codebase |
| `docs/spec/interface/REQUIREMENTS.md` | References gRPC proxy, hashicorp/go-plugin, mTLS, seccomp - not used in MCP architecture |

## Package Status Reference

The following packages mentioned in old specs **DO NOT EXIST**:

- `internal/grpc/` - Replaced by `internal/mcp/`
- `internal/audit/` - Never implemented
- `internal/policy/` - Planned but not implemented
- `internal/skill/` - Never implemented
- `internal/agent/` - Never implemented
- `internal/isolation/` - Never implemented (seccomp/namespaces)

The following packages **EXIST** and are correctly referenced:

- `internal/mcp/` - MCP server and tool handlers
- `internal/vectorstore/` - VectorStore abstraction (chromem default, Qdrant optional)
- `internal/embeddings/` - FastEmbed local ONNX embeddings
- `internal/reasoningbank/` - Memory operations with Bayesian confidence
- `internal/folding/` - Context-folding implementation
- `internal/secrets/` - gitleaks scrubbing (97% coverage)
- `internal/checkpoint/` - Context snapshots
- `internal/remediation/` - Error pattern tracking
- `internal/repository/` - Repository indexing and semantic search

## Files Created

| File | Purpose |
|------|---------|
| `docs/archive/obsolete-specs/README.md` | Explains why files were archived and references current architecture |

## Recommendations for Future Work

1. **Policy System**: If implemented, update `docs/spec/policy/SPEC.md` status and create `internal/policy/` package
2. **Skills System**: If implemented, update `docs/spec/skills-system/SPEC.md` and create `internal/skill/` package
3. **Team/Org Scoping**: The reasoning-bank ARCH.md mentions team/org memory scoping that isn't implemented - implement or remove from documentation
4. **Memory Consolidation**: The DESIGN.md describes consolidation features not yet implemented - track as future enhancement

## Verification

All spec files now either:
- Accurately describe implemented features with correct package references
- Are clearly marked as DEPRECATED/FUTURE with explanatory notes
- Have been moved to archive with explanation
