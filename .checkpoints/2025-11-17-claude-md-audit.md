# CLAUDE.md Audit Report - 2025-11-17

## Executive Summary

**For Demo Tomorrow**: 🚨 High-priority issues found that create confusion about project status and version

**Key Issues**:
1. **Version Confusion** - References to v3, v2.0, v2.1, v2.2, and 0.9.0-rc-1 are inconsistent
2. **Broken References** - Dead links to deleted spec files
3. **Outdated Migration Docs** - Confusing for fresh start
4. **Context Bloat** - 519 lines in root + variable package docs (33-274 lines)

## Critical Issues (Fix Before Demo)

### 1. Broken Reference in pkg/prefetch/CLAUDE.md

**Location**: `pkg/prefetch/CLAUDE.md:254`
```markdown
- **V3 Rebuild**: [docs/specs/v3-rebuild/SPEC.md](../../docs/specs/v3-rebuild/SPEC.md)
```

**Problem**: File `docs/specs/v3-rebuild/SPEC.md` does NOT exist
**Impact**: Dead link, suggests v3 rebuild that doesn't exist
**Fix**: Remove this reference entirely

### 2. Missing Product Roadmap File

**Location**: `CLAUDE.md:262`
```markdown
**Product Roadmap**: [docs/PRODUCT-ROADMAP-V3-AGENT-PATTERNS.md](docs/PRODUCT-ROADMAP-V3-AGENT-PATTERNS.md)
```

**Problem**: File `docs/PRODUCT-ROADMAP-V3-AGENT-PATTERNS.md` does NOT exist
**Impact**: Broken link in critical documentation
**Fix**: Either create the file OR remove the reference OR update to correct path

### 3. Version Number Confusion

**Current Reality**:
- Code version: `"dev"` (cmd/contextd/main.go)
- Migration guide says: `0.9.0-rc-1` (docs/guides/MIGRATION-V2-TO-V3.md)
- CLAUDE.md architecture roadmap mentions: v2.1, v2.2, 0.9.0-rc-1
- Multiple files reference: v3, V3, v3-rebuild

**Problem**: Unclear what version this actually is
**Impact**: Confusion about project maturity and status
**User Statement**: "moved to new repo to start fresh (as a completely new project)"

**Recommendations**:
- **Option A**: Remove all version references, treat as fresh `v1.0.0-alpha`
- **Option B**: Clarify 0.9.0-rc-1 is "pre-1.0 prototype"
- **Option C**: Align everything to v2.1 (current target in roadmap)

### 4. Confusing Migration Guide

**File**: `docs/guides/MIGRATION-V2-TO-V3.md`
**Title**: "Migration Guide: v2.0 → 0.9.0-rc-1"

**Problems**:
- Filename says "V2-TO-V3" but content says "v2.0 → 0.9.0-rc-1"
- Implies this is an upgrade path, but user says "fresh start"
- Creates confusion: Is this v3? Is this 0.9.0? Is this v2.1?

**Fix**:
- **For fresh start**: Delete or move to archive
- **If keeping**: Rename to `MIGRATION-V2.0-TO-V0.9.0.md` for accuracy

## Medium Priority Issues

### 5. MCP Tools Count Mismatch

**CLAUDE.md:403** says: "contextd provides 9 MCP tools"
**pkg/CLAUDE.md:446** says: "MCP Tools (9 total)" and lists 9
**Recent implementation** (from session): We actually have 12 MCP tools

**Fix**: Update both files to reflect 12 tools

### 6. MCP Architecture Description Outdated

**CLAUDE.md:405-412** says:
```markdown
**IMPORTANT**: MCP tools connect directly to Qdrant, NOT to the contextd service.
- The contextd service exists ONLY for the `ctxd` CLI client
```

**Problem**: This conflicts with recent MCP work where we:
- Identified missing `/mcp` endpoint
- Found server runs on port 8081 (not 9090)
- Discovered custom REST API instead of proper MCP protocol

**Fix**: Update to reflect actual MCP architecture (HTTP server on 8081, needs `/mcp` endpoint)

### 7. "Never Say Production Ready" Contradiction

**CLAUDE.md:515**: "Never say the project is production ready"
**CLAUDE.md:162**: Mentions "Production-ready Go implementation"

**Problem**: Contradicts itself
**Fix**: Clarify distinction (production-ready code ≠ production-ready project)

## Low Priority (But Should Fix)

### 8. Context Bloat

**Root CLAUDE.md**: 519 lines
**pkg/CLAUDE.md**: 829 lines (!)
**Package-specific**: Range from 33-274 lines

**Issues**:
- pkg/CLAUDE.md duplicates a lot from root (testing patterns, service patterns)
- Some packages have extensive implementation details better suited for godoc
- Multiple references to same concepts across files

**Recommendation**:
- Extract common patterns to `docs/standards/go-patterns.md`
- Keep CLAUDE.md files focused on "when to use this" not "how it works internally"
- Link to specs/standards instead of duplicating

### 9. Inconsistent File Structure References

**CLAUDE.md shows**:
```
docs/
├── standards/
├── specs/
├── guides/
├── architecture/
└── testing/
```

**But references dead paths**:
- `docs/adr/002-...` (should be `docs/architecture/adr/002-...`)
- Mixed use of `/docs/specs/` vs `/docs/architecture/`

**Fix**: Audit all file paths, ensure consistent structure

## Version Confusion Detailed Breakdown

**Found References**:
- v2.0 (mentioned in migration context)
- v2.1 (current target per CLAUDE.md:50)
- v2.2 (enterprise future per CLAUDE.md:51)
- 0.9.0-rc-1 (future per CLAUDE.md:52, but migration guide says current!)
- v3 (broken references in prefetch, product roadmap)
- "dev" (actual code version)

**Recommended Resolution** (pick one):

**Option A - Fresh Start (Recommended for "new repo")**:
```markdown
Current Version: v1.0.0-alpha
Status: Pre-release prototype
Roadmap:
- v1.0.0-beta (multi-tenant complete)
- v1.0.0 (production-ready)
- v1.1.0 (context-folding)
- v2.0.0 (enterprise team features)
```

**Option B - Continuation**:
```markdown
Current Version: 0.9.0-rc-1 (pre-1.0)
Previous: v2.0 (deprecated, old architecture)
Roadmap:
- 1.0.0 (stable multi-tenant)
- 1.1.0 (context-folding)
- 2.0.0 (enterprise features)
```

## Demo-Critical Checklist

For tomorrow's demo, minimum fixes:

- [ ] Fix broken reference in pkg/prefetch/CLAUDE.md (delete line 254)
- [ ] Fix or remove Product Roadmap reference in CLAUDE.md
- [ ] Clarify version in CLAUDE.md Project Overview section
- [ ] Update MCP tools count (9 → 12)
- [ ] Consider hiding/archiving MIGRATION-V2-TO-V3.md to avoid confusion

## Recommendations by Priority

### Must Fix (Before Demo)
1. Remove broken v3-rebuild reference
2. Fix/remove product roadmap dead link
3. Add version clarity section to CLAUDE.md
4. Update MCP tool count

### Should Fix (This Week)
5. Rename or archive migration guide
6. Update MCP architecture description
7. Fix "production ready" contradiction
8. Audit all file path references

### Nice to Have (Future)
9. Reduce CLAUDE.md bloat (extract to standards)
10. Standardize package CLAUDE.md template
11. Create automated link checker

## Files Needing Updates

**High Priority**:
- `CLAUDE.md` (lines 262, 403, 50-52, 515)
- `pkg/prefetch/CLAUDE.md` (line 254)
- `pkg/CLAUDE.md` (line 446)

**Medium Priority**:
- `docs/guides/MIGRATION-V2-TO-V3.md` (rename or delete)
- All files with version references (32 files found)

## Estimated Fix Time

- **Demo-critical fixes**: 15-30 minutes
- **All high/medium fixes**: 1-2 hours
- **Complete cleanup**: 4-6 hours

## Questions to Answer

1. **What version IS this actually?** (fresh v1.0.0-alpha OR continuation 0.9.0-rc-1?)
2. **Keep or delete migration guide?** (if fresh start, delete it)
3. **Product roadmap - create or remove link?**
4. **MCP architecture - when will proper `/mcp` endpoint be implemented?**
