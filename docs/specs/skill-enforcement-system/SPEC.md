# Skill Enforcement System Specification

**Version**: 1.0.0
**Status**: Design Approved (Not Yet Implemented)
**Date**: 2025-11-18
**Category**: Development Workflow, Quality Assurance

## Overview

### Purpose

The Skill Enforcement System ensures that project-specific skills remain current as the codebase evolves. It provides a structured approach to package development, verification, and code review through mandatory skill invocation and multi-layer enforcement.

### Design Goals

1. **Enforce Evidence-Based Completion**: No task marked complete without verification proof
2. **Maintain Skill Currency**: Skills stay current as packages/features change
3. **Optimize Token Usage**: Hybrid approach (lightweight docs + on-demand skills) minimizes context overhead
4. **Ensure Consistency**: Category skills enforce consistent patterns across similar packages
5. **Enable Scalability**: System scales as codebase grows without documentation bloat

### Key Features

- **Hybrid Documentation**: Lightweight pkg/CLAUDE.md (~80 lines) + category skills loaded on demand
- **5 Category Skills**: Group similar packages by architectural pattern (not per-package)
- **Mandatory Package Creation Workflow**: `contextd:creating-package` skill enforced before creating packages
- **4-Layer Enforcement**: CLAUDE.md rules + creation skill + code review + maintenance checklist
- **73% Token Reduction**: Baseline context drops from 10,500 → 1,000 tokens
- **12 Total Skills**: 3 completion + 1 creation + 5 category + 3 enhancement skills

### Problem Statement

**Without enforcement, skills become stale**:
- New packages created without adding to category skills
- Patterns diverge across similar packages
- Agents work without context
- Documentation drifts from reality
- Token bloat from loading unused context

**Root Causes**:
1. No mandatory workflow for package creation
2. No connection between package changes and skill updates
3. No code review validation of skill maintenance
4. Large pkg/CLAUDE.md files (846 lines) loaded unconditionally

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    CLAUDE.md (Root)                         │
│  - Core Principle: "Skill Maintenance"                     │
│  - Mandatory Rules: MUST invoke completion/creation skills │
│  - Checklist: Before creating package → invoke skill       │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┴──────────────┐
        │                             │
        v                             v
┌─────────────────┐          ┌──────────────────────┐
│ pkg/CLAUDE.md   │          │  Category Skills     │
│  (~80 lines)    │          │  (load on demand)    │
├─────────────────┤          ├──────────────────────┤
│ Package Index   │          │ pkg-security         │
│ Category Map    │───────>  │ pkg-storage          │
│ Quick Patterns  │          │ pkg-core             │
│ Standards @ref  │          │ pkg-api              │
└─────────────────┘          │ pkg-ai               │
                             └──────────────────────┘
                                       │
                                       │
        ┌──────────────────────────────┴───────────────────┐
        │                                                  │
        v                                                  v
┌────────────────────────┐                    ┌───────────────────────┐
│ Creation Workflow      │                    │  Code Review          │
│ (contextd:creating-    │                    │  (contextd:code-      │
│  package skill)        │                    │   review skill)       │
├────────────────────────┤                    ├───────────────────────┤
│ 1. Classify package    │                    │ Validates:            │
│ 2. Create structure    │                    │ - pkg/CLAUDE.md       │
│ 3. Update pkg/CLAUDE   │                    │   updated             │
│ 4. Update category     │                    │ - Category skill      │
│    skill               │                    │   updated             │
│ 5. Create spec (if     │                    │ - Patterns match      │
│    complex)            │                    │   category            │
│ 6. Verify setup        │                    │ - Tests exist         │
└────────────────────────┘                    └───────────────────────┘
```

### Category Skills Design

**5 Category Skills** (not per-package):

| Skill | Packages Covered | Purpose |
|-------|-----------------|---------|
| **contextd:pkg-security** | auth, session, isolation, rbac | Multi-tenant isolation, input validation, security testing |
| **contextd:pkg-storage** | checkpoint, remediation, cache, persistence | Qdrant patterns, database-per-project, query security |
| **contextd:pkg-core** | config, telemetry, logging, health | Standard patterns, error handling, initialization |
| **contextd:pkg-api** | mcp, handlers, middleware, routes | Request/response, validation, MCP tools |
| **contextd:pkg-ai** | embedding, search, semantic, context | Embeddings, vector operations, AI integrations |

**Rationale**: Group by architectural pattern (not name) for reusable guidance across similar packages.

### Skill Content Structure

Each category skill contains:

```markdown
# contextd:pkg-<category>

## When to Use This Skill
[Trigger conditions]

## Packages in This Category
[List with brief purpose]

## Common Patterns
[Reusable code patterns with examples]

## Testing Requirements
- Coverage requirements
- Security test checklist (if applicable)
- Integration test patterns

## Security Checklist (if applicable)
[Multi-tenant isolation, input validation, etc.]

## Common Pitfalls
[What NOT to do, with examples]

## Verification Template
[What to verify before completion]

## Code Review Checklist
[What reviewer should check]
```

**Target Size**: ~150 lines per category skill

---

## Lightweight pkg/CLAUDE.md Design

**Target**: 80-100 lines total

### Structure

```markdown
# Package Guidelines

See [root CLAUDE.md](../CLAUDE.md) for project-wide policies.

## Package Philosophy
[Public vs Internal distinction - 4 lines]

## Package-Skill Mapping
[Table with 10-15 packages, categories, skills, security levels - 15 lines]

## When Working in a Package
[Mandatory workflow: read → invoke skill → follow patterns - 8 lines]

## Adding New Package
[MANDATORY: Invoke contextd:creating-package skill - 10 lines]

## Standards Reference
[@imports to standards docs - 4 lines]

## Quick Pattern Reference
[3 minimal examples: Service pattern, Interface design, Error handling - 30 lines]
```

**Example Mapping Table**:

| Package | Category | Skill to Invoke | Security Level |
|---------|----------|-----------------|----------------|
| pkg/auth | Security | contextd:pkg-security | Critical |
| pkg/session | Security | contextd:pkg-security | Critical |
| pkg/checkpoint | Storage | contextd:pkg-storage | High |
| pkg/remediation | Storage | contextd:pkg-storage | Medium |
| pkg/config | Core | contextd:pkg-core | Medium |
| pkg/telemetry | Core | contextd:pkg-core | Low |
| pkg/logging | Core | contextd:pkg-core | High (secret redaction) |
| pkg/mcp | API | contextd:pkg-api | High |
| pkg/embedding | AI | contextd:pkg-ai | Medium |
| pkg/search | AI | contextd:pkg-ai | Medium |

---

## Enforcement Mechanisms

### Layer 1: CLAUDE.md Mandatory Rules

**Add to Core Principles**:

```markdown
9. **Skill Maintenance** - Skills evolve with codebase, must stay current
```

**Add to Summary checklist**:

```markdown
**Before creating/modifying packages or features:**

1. Check `superpowers:using-superpowers` skill
2. **Creating new package**: Invoke `contextd:creating-package` skill (MANDATORY)
3. **Modifying existing package**: Invoke relevant category skill (see pkg/CLAUDE.md)
4. **New feature**: Check if category skill needs update
...
```

### Layer 2: Package Creation Skill (NEW)

**Skill**: `contextd:creating-package`

**Purpose**: Guide package creation AND enforce skill maintenance.

**Mandatory Workflow**:

```markdown
### Step 1: Classify Package
- [ ] Identify category (Security/Storage/Core/API/AI)
- [ ] Determine security level (Critical/High/Medium/Low)

### Step 2: Create Package Structure
- [ ] Create pkg/<name>/ directory
- [ ] Create pkg/<name>/<name>.go (main file)
- [ ] Create pkg/<name>/<name>_test.go (tests)
- [ ] Add package-level godoc

### Step 3: Update pkg/CLAUDE.md
- [ ] Add package to mapping table
- [ ] Specify category and skill
- [ ] Set security level

### Step 4: Update Category Skill
- [ ] Read category skill (e.g., contextd:pkg-security)
- [ ] Add package to "Packages in This Category" section
- [ ] Add package-specific patterns if unique
- [ ] Update testing requirements if needed

### Step 5: Create Package Spec (if complex)
- [ ] Create docs/specs/<package>/SPEC.md (if package is complex)
- [ ] Document API design
- [ ] Security requirements
- [ ] Testing strategy

### Step 6: Verify Setup
- [ ] Package builds: `go build ./pkg/<name>/`
- [ ] Tests exist and pass: `go test ./pkg/<name>/`
- [ ] Coverage ≥80%
- [ ] pkg/CLAUDE.md updated
- [ ] Category skill updated

### Completion Template
[Use major task verification template]
```

**Enforcement**: CLAUDE.md rule says "MUST invoke `contextd:creating-package`" → agents can't skip.

### Layer 3: Code Review Validation

**Add to CODE-REVIEW-CHECKLIST.md Section 6 (Architecture Compliance)**:

```markdown
### Package Changes

**If PR creates new package**:
- [ ] `contextd:creating-package` skill was invoked
- [ ] pkg/CLAUDE.md mapping table updated
- [ ] Relevant category skill updated (package listed)
- [ ] Package follows category patterns

**If PR modifies existing package**:
- [ ] Relevant category skill was invoked (see pkg/CLAUDE.md)
- [ ] Patterns match category skill
- [ ] If new pattern added: category skill updated

**If PR adds feature that affects multiple packages**:
- [ ] Feature documented in relevant specs
- [ ] Category skills updated if patterns changed
- [ ] Cross-package consistency validated
```

### Layer 4: Maintenance Checklist

**Add to CLAUDE.md "Maintenance Guidelines"**:

```markdown
**Update this file AND category skills when:**
- [ ] Adding new packages (update pkg/CLAUDE.md mapping + category skill)
- [ ] Changing package patterns (update category skill with new pattern)
- [ ] Discovering security vulnerabilities (update contextd:pkg-security)
- [ ] Adding new testing patterns (update relevant category skill)
- [ ] Refactoring multi-package features (update affected category skills)
```

---

## Complete Skill List

**Total Skills: 12**

### Priority 1: Essential (Blocks Core Workflows)
1. **contextd:completing-major-task** - Major task verification template
2. **contextd:completing-minor-task** - Minor task self-interrogation
3. **contextd:code-review** - Comprehensive code review

### Priority 2: Package Infrastructure
4. **contextd:creating-package** - Package creation workflow (MANDATORY)
5. **contextd:pkg-security** - Security-critical packages

### Priority 3: Core Development
6. **contextd:pkg-storage** - Storage patterns
7. **contextd:planning-with-verification** - TodoWrite with verification subtasks

### Priority 4: Complete Coverage
8. **contextd:pkg-core** - Core utilities
9. **contextd:pkg-api** - API patterns
10. **contextd:pkg-ai** - AI/ML patterns
11. **contextd:security-check** - Deep security validation (evaluate redundancy)
12. **contextd:pre-pr-verification** - Pre-PR check (evaluate redundancy)

**Note**: Skills 11-12 may be redundant with other skills; evaluate after implementing 1-10.

---

## Token Budget Analysis

### Before (Nested CLAUDE.md Files)

- **pkg/CLAUDE.md**: 846 lines (~10,500 tokens)
- **Loaded**: Every time working in pkg/ directory
- **Cost**: High baseline overhead

### After (Hybrid Approach)

**Baseline (Always Loaded)**:
- **pkg/CLAUDE.md**: 80 lines (~1,000 tokens)

**On-Demand (Loaded When Invoked)**:
- **Category skill**: ~150 lines (~1,800 tokens)

**Total Cost**:
- **Best case** (just orientation): 1,000 tokens
- **Worst case** (with skill): 2,800 tokens

**Savings**: 73% baseline reduction (10,500 → 1,000 tokens)

---

## Implementation Workflow

### Example: Creating pkg/cache

```
User: "Create pkg/cache for in-memory caching"

Agent:
1. "Invoking contextd:creating-package skill"

Skill loads → guides through 6-step checklist:

Step 1: Classify
- Category: Storage (caching is storage pattern)
- Security: Medium

Step 2: Create structure
- pkg/cache/cache.go (LRU implementation)
- pkg/cache/cache_test.go (85% coverage)
- Package godoc

Step 3: Update pkg/CLAUDE.md
- Add row: | pkg/cache | Storage | contextd:pkg-storage | Medium |

Step 4: Update contextd:pkg-storage skill
- Add to package list: "pkg/cache (in-memory LRU caching)"
- Add LRU pattern if not present

Step 5: Create spec (skip - not complex)

Step 6: Verify
✓ Package builds
✓ Tests pass (85% coverage)
✓ pkg/CLAUDE.md updated
✓ Category skill updated

Agent: "Invoking contextd:completing-major-task for verification"
[Provides complete verification template]

Agent: "Ready for code review"

Code Review (contextd:code-review):
✓ Validates pkg/CLAUDE.md updated
✓ Validates contextd:pkg-storage updated
✓ Validates tests exist
→ APPROVED
```

---

## Success Metrics

### Quantitative
- 100% of new packages have pkg/CLAUDE.md entry
- 100% of new packages listed in category skill
- 0 packages created without `contextd:creating-package` invocation
- Category skills updated in same PR as package changes
- 73% token reduction (10,500 → 1,000 baseline)

### Qualitative
- Consistent patterns across packages in same category
- Agents know which skill to invoke for any package
- Skills contain current, accurate patterns
- No orphaned packages (unlisted in pkg/CLAUDE.md)

---

## Implementation Plan

### Phase 1: Infrastructure (Before Skill Creation)
1. Create lightweight pkg/CLAUDE.md (~80 lines)
2. Update root CLAUDE.md (add skill maintenance principle + checklist)
3. Update CODE-REVIEW-CHECKLIST.md (add package changes section)
4. Delete outdated guides (CLAUDE-MD-STRUCTURE.md, CLAUDE-MD-NAVIGATION.md)

### Phase 2: Core Skills (Implement First)
5. `contextd:creating-package` - **HIGHEST PRIORITY**
6. `contextd:completing-major-task`
7. `contextd:completing-minor-task`
8. `contextd:code-review`

### Phase 3: Category Skills
9. `contextd:pkg-security` - Security-critical packages
10. `contextd:pkg-storage` - Qdrant patterns
11. `contextd:pkg-core` - Config, logging, telemetry
12. `contextd:pkg-api` - MCP, handlers, middleware
13. `contextd:pkg-ai` - Embeddings, search

### Phase 4: Enhancement Skills (Evaluate Need)
14. `contextd:planning-with-verification` - TodoWrite integration
15. `contextd:security-check` - Deep security (may merge with pkg-security)
16. `contextd:pre-pr-verification` - Pre-PR check (may merge with code-review)

### Phase 5: Documentation Cleanup
17. Update MULTI-AGENT-ORCHESTRATION.md (add all new skills)
18. Update DEVELOPMENT-WORKFLOW.md (reference new skills)

---

## Dependencies

### Blocks This Spec
- None (foundational specification)

### This Spec Blocks
- All future package development (requires `contextd:creating-package`)
- Verification system implementation (requires completion skills)
- Code review enforcement (requires `contextd:code-review`)

### Related Specs
- [Verification Policy](../verification-policy/SPEC.md) - Completion verification templates (NOT YET CREATED - needs spec)
- [Code Review Checklist](../code-review/SPEC.md) - Review validation process (NOT YET CREATED - needs spec)
- [Multi-Tenant Architecture](../multi-tenant/SPEC.md) - Security requirements for pkg-security skill

---

## Design Decisions

### Why Category Skills Instead of Per-Package Skills?

**Category approach**:
- ✅ Reusable patterns across similar packages
- ✅ Lower maintenance burden (1 skill for multiple packages)
- ✅ Consistent patterns across category
- ✅ Token efficient (load only relevant skill)

**Per-package approach**:
- ❌ ~30 skills for 30 packages
- ❌ High duplication across similar packages
- ❌ Maintenance nightmare as codebase grows
- ❌ Token inefficient (many small skills)

**Decision**: Category skills with 5 categories.

### Why Hybrid (Lightweight Docs + Skills)?

**Pure docs approach**:
- ❌ 846-line pkg/CLAUDE.md loaded unconditionally
- ❌ High token overhead
- ❌ Hard to maintain

**Pure skills approach**:
- ❌ No automatic orientation when entering package
- ❌ Agents must remember to invoke
- ❌ No quick reference

**Hybrid approach**:
- ✅ Lightweight pkg/CLAUDE.md (automatic orientation)
- ✅ Skills load on demand (deep focus when needed)
- ✅ 73% token reduction
- ✅ Scalable

**Decision**: Hybrid with 80-line pkg/CLAUDE.md + category skills.

### Why 4 Enforcement Layers?

**Rationale**: Defense in depth prevents skill drift through multiple checkpoints:

1. **CLAUDE.md rules** - First line (agents must invoke skills)
2. **Creation skill** - Second line (structured workflow)
3. **Code review** - Third line (validates updates)
4. **Maintenance checklist** - Fourth line (reminder system)

**Decision**: All 4 layers required for robust enforcement.

---

## Technical Debt

**NOTE**: This spec violates our own modularity guidelines (Kinney approach).

**Violation**: Monolithic SPEC.md (~450 lines) instead of scannable main (~150 lines) + @imported details.

**Should be refactored to**:
- Main SPEC.md (~150 lines, scannable overview)
- @./category-skills.md - Category skill design details
- @./enforcement.md - 4-layer enforcement details
- @./workflows.md - Package creation workflow examples
- @./implementation-plan.md - Phase-by-phase implementation

**Status**: Accepted technical debt, will refactor during implementation phase or when updating spec.

**Created**: Before `kinney-documentation` skill existed. Future specs MUST follow modular approach.

---

## Open Questions

1. **Should contextd:security-check be merged with contextd:pkg-security?**
   - Evaluate after implementing both
   - May be redundant
   - Decision: Defer to Phase 4

2. **Should contextd:pre-pr-verification be merged with contextd:code-review?**
   - Pre-PR is developer self-check
   - Code-review is reviewer validation
   - May be same checklist
   - Decision: Defer to Phase 4

3. **How to handle packages that span multiple categories?**
   - Example: pkg/mcp (both API and Storage patterns)
   - Current: Assign primary category (API)
   - Alternative: Multi-category packages invoke multiple skills
   - Decision: Primary category for now, revisit if problematic

4. **Should we create pkg/<name>/CLAUDE.md for very complex packages?**
   - Current: No per-package CLAUDE.md files
   - Alternative: Optional for extremely complex packages (>5K lines)
   - Decision: Evaluate on case-by-case basis

---

## References

### Design Documents
- [docs/plans/2025-11-18-skill-enforcement-design.md](../../plans/2025-11-18-skill-enforcement-design.md) - Detailed design
- [docs/plans/2025-11-18-verification-enforcement-design.md](../../plans/2025-11-18-verification-enforcement-design.md) - Verification system
- [docs/plans/2025-11-18-guides-audit-report.md](../../plans/2025-11-18-guides-audit-report.md) - Guides audit

### Standards
- [docs/standards/coding-standards.md](../../standards/coding-standards.md)
- [docs/standards/testing-standards.md](../../standards/testing-standards.md)
- [docs/standards/package-guidelines.md](../../standards/package-guidelines.md)

### Related Guides
- [docs/guides/DEVELOPMENT-WORKFLOW.md](../../guides/DEVELOPMENT-WORKFLOW.md)
- [docs/guides/MULTI-AGENT-ORCHESTRATION.md](../../guides/MULTI-AGENT-ORCHESTRATION.md)
- [docs/guides/VERIFICATION-POLICY.md](../../guides/VERIFICATION-POLICY.md)
