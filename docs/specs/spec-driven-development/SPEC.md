# Spec-Driven Development Specification

**Version**: 1.0.0
**Status**: Design Approved (Not Yet Implemented)
**Date**: 2025-11-18
**Category**: Development Workflow, Documentation, Quality Assurance

## Overview

### Purpose

Spec-Driven Development enforces **NO CODE WITHOUT SPEC** policy. Every feature, package, and significant change MUST have a specification document before implementation begins. This ensures design clarity, reduces rework, and provides authoritative documentation.

### Design Goals

1. **NO CODE WITHOUT SPEC**: Block all implementation until spec exists and is approved
2. **Clear Documentation Structure**: Standardized docs/ folder with defined purposes
3. **Spec-First Workflow**: Design → Spec → Review → Approve → Implement
4. **Enforcement at Multiple Layers**: CLAUDE.md rules + skills + code review
5. **Maintainable Specs**: Living documents that evolve with implementation

### Key Features

- **Mandatory Spec Creation**: Features/packages cannot be implemented without approved spec
- **Standardized docs/ Structure**: Clear separation of specs, guides, standards, plans
- **Spec Templates**: Structured templates for different spec types
- **Approval Workflow**: Specs must be reviewed and approved before implementation
- **Code Review Validation**: PRs rejected if spec missing or not followed
- **Spec Maintenance**: Specs updated when implementation deviates

### Problem Statement

**Without spec-first enforcement**:
- Features implemented without design clarity
- Implementation details hidden in code/commits
- No authoritative reference documentation
- Frequent rework due to unclear requirements
- Context scattered across conversations
- "Just start coding" mentality

**Root Causes**:
1. No mandatory spec requirement
2. No spec approval workflow
3. No code review validation of spec adherence
4. Easy to skip "planning" phase

---

## docs/ Folder Structure

### Standardized Directory Layout

```
docs/
├── specs/                      # Feature/package specifications (AUTHORITATIVE)
│   ├── <feature-name>/
│   │   ├── SPEC.md            # Main specification (REQUIRED)
│   │   ├── research/          # Research documents
│   │   ├── decisions/         # Design decisions
│   │   └── diagrams/          # Architecture diagrams
│   └── README.md              # Spec index and status tracking
│
├── guides/                     # How-to documentation (OPERATIONAL)
│   ├── DEVELOPMENT-WORKFLOW.md
│   ├── VERIFICATION-POLICY.md
│   ├── CODE-REVIEW-CHECKLIST.md
│   └── *.md                   # Operational guides
│
├── standards/                  # Coding standards (FOUNDATIONAL)
│   ├── architecture.md
│   ├── coding-standards.md
│   ├── testing-standards.md
│   └── package-guidelines.md
│
├── architecture/               # Architectural decisions
│   ├── adr/                   # Architecture Decision Records
│   │   ├── 001-*.md
│   │   └── NNN-*.md
│   └── diagrams/              # System architecture diagrams
│
├── plans/                      # Design documents (WORKING DRAFTS)
│   └── YYYY-MM-DD-<topic>.md  # Date-prefixed design docs
│
└── archive/                    # Deprecated/archived documentation
    └── *.md.archived
```

### Directory Purposes

**specs/** - **AUTHORITATIVE SOURCE OF TRUTH**
- **Purpose**: Feature and package specifications
- **When to create**: Before implementing any feature or package
- **Status**: Approved specs required before implementation
- **Content**: Design, API, requirements, testing, security
- **Audience**: Developers, reviewers, future maintainers

**guides/** - **OPERATIONAL GUIDANCE**
- **Purpose**: How-to documentation for workflows and processes
- **When to create**: When establishing repeatable workflows
- **Status**: Living documents, updated as processes evolve
- **Content**: Step-by-step guides, checklists, examples
- **Audience**: Developers, agents, contributors

**standards/** - **FOUNDATIONAL RULES**
- **Purpose**: Project-wide coding standards and patterns
- **When to create**: Rarely (foundational documents)
- **Status**: Stable, infrequently updated
- **Content**: Mandatory rules, patterns, conventions
- **Audience**: All developers, code reviewers

**architecture/** - **DESIGN DECISIONS**
- **Purpose**: Record architectural decisions and their rationale
- **When to create**: When making significant architectural choices
- **Status**: Immutable once decided (append-only)
- **Content**: ADRs with context, decision, consequences
- **Audience**: Architects, technical leads, future decision-makers

**plans/** - **WORKING DRAFTS**
- **Purpose**: Design documents in progress
- **When to create**: During brainstorming and planning phase
- **Status**: Temporary (promoted to spec or archived)
- **Content**: Rough designs, explorations, brainstorming
- **Audience**: Current development team

**archive/** - **HISTORICAL RECORDS**
- **Purpose**: Deprecated documentation for reference
- **When to create**: When docs no longer relevant
- **Status**: Read-only, historical
- **Content**: Old specs, outdated guides
- **Audience**: Historical reference only

---

## Spec Requirements

### What Requires a Spec?

**MANDATORY (Must have spec before implementation)**:
1. **New features** - Any new functionality visible to users
2. **New packages** - Any new Go package in pkg/ or internal/
3. **Major refactors** - Architectural changes affecting multiple packages
4. **API changes** - New endpoints, MCP tools, protocol changes
5. **Security changes** - Anything affecting multi-tenant isolation, auth, permissions
6. **Breaking changes** - Changes that break existing APIs or behavior

**OPTIONAL (Spec recommended but not required)**:
1. **Bug fixes** - Usually documented in regression tests, not specs
2. **Minor refactors** - Internal code improvements without API changes
3. **Documentation updates** - Self-documenting
4. **Test additions** - Test coverage improvements

**NEVER (Do not create spec)**:
1. **Typo fixes** - Trivial corrections
2. **Formatting changes** - gofmt, import ordering
3. **Comment additions** - Code documentation

### When in Doubt

**Default to requiring a spec.** If you're unsure, create a spec. Better to have unnecessary documentation than undocumented complexity.

---

## Spec Template

### Standard SPEC.md Structure

```markdown
# <Feature/Package Name> Specification

**Version**: X.Y.Z
**Status**: Draft | Review | Approved | Implemented
**Date**: YYYY-MM-DD
**Category**: [Feature|Package|API|Security|Refactor]

## Overview

### Purpose
[Why this exists, what problem it solves]

### Design Goals
[What we're optimizing for - ordered by priority]

### Key Features
[Bullet list of main capabilities]

### Problem Statement (for features)
[What problem are we solving, why now]

---

## Architecture (if applicable)

### System Design
[High-level design, components, interactions]

### API Design (if applicable)
[Endpoints, request/response formats, errors]

### Database Schema (if applicable)
[Collections, fields, indexes]

### Security Considerations
[Multi-tenant isolation, input validation, auth]

---

## Requirements

### Functional Requirements
[What the system must do]

### Non-Functional Requirements
[Performance, scalability, security]

### Testing Requirements
[Coverage %, test types, critical paths]

---

## Implementation Plan

### Phase 1: [Name]
[Tasks, deliverables]

### Phase 2: [Name]
[Tasks, deliverables]

---

## Dependencies

### Blocks This Spec
[What must be done before this can be implemented]

### This Spec Blocks
[What can't be done until this is implemented]

### Related Specs
[Links to related specifications]

---

## Design Decisions

### Decision 1: [Topic]
**Options considered:**
- Option A: [pros/cons]
- Option B: [pros/cons]

**Decision**: [Chosen option]
**Rationale**: [Why chosen]

---

## Open Questions
[Questions that need answers before implementation]

---

## References
[Links to research, ADRs, related docs]
```

---

## Enforcement Mechanisms

### Layer 1: CLAUDE.md Mandatory Rules

**Add to "Core Principles"**:

```markdown
10. **Spec-Driven Development** - NO CODE WITHOUT SPEC (non-negotiable)
```

**Add to "Summary" checklist**:

```markdown
**Before writing any code:**

1. Check `superpowers:using-superpowers` skill
2. **MANDATORY: Check for spec** in `docs/specs/<feature>/SPEC.md`
3. **If spec missing**: Invoke `contextd:creating-spec` skill (MANDATORY)
4. **If spec exists**: Read spec, understand requirements
5. **Spec must be Status: Approved** before implementation begins
...
```

### Layer 2: Spec Creation Skill (NEW)

**Skill**: `contextd:creating-spec`

**Purpose**: Guide spec creation AND block implementation until spec approved.

**Mandatory Workflow**:

```markdown
# contextd:creating-spec

## Mandatory Workflow

**This skill is MANDATORY when implementing features or packages without approved specs.**

### Step 1: Determine Spec Requirement
- [ ] Read "What Requires a Spec?" section
- [ ] Classify task: Mandatory | Optional | Never
- [ ] If Mandatory: MUST create spec before coding
- [ ] If Optional: Recommend creating spec
- [ ] If Never: Skip spec creation

### Step 2: Create Spec Directory Structure
- [ ] Create `docs/specs/<feature-name>/`
- [ ] Create `docs/specs/<feature-name>/SPEC.md`
- [ ] Create `docs/specs/<feature-name>/research/` (if needed)
- [ ] Create `docs/specs/<feature-name>/decisions/` (if needed)

### Step 3: Write Spec Using Template
- [ ] Use standard SPEC.md template
- [ ] Fill all required sections
- [ ] Set Status: Draft
- [ ] Include security considerations (MANDATORY for all specs)
- [ ] Define testing requirements (coverage %, test types)

### Step 4: Update Spec Index
- [ ] Update `docs/specs/README.md`
- [ ] Add spec to index table
- [ ] Set status: Draft

### Step 5: Request Spec Review
- [ ] Announce spec creation
- [ ] Request review (user or team)
- [ ] Address review feedback
- [ ] Update spec based on feedback

### Step 6: Approval Gate
- [ ] Spec Status: Approved
- [ ] All open questions resolved or deferred
- [ ] Security requirements defined
- [ ] Testing strategy defined

**CRITICAL**: Implementation CANNOT begin until Status: Approved

### Completion Template
[Use major task verification template for spec creation]
```

### Layer 3: Code Review Validation

**Add to CODE-REVIEW-CHECKLIST.md Section 6 (Architecture Compliance)**:

```markdown
### Spec Adherence

**For ALL PRs that add features or packages**:
- [ ] Spec exists in `docs/specs/<feature>/SPEC.md`
- [ ] Spec Status: Approved (not Draft or Review)
- [ ] Implementation matches spec design
- [ ] If implementation deviates: spec updated with rationale
- [ ] Security requirements from spec implemented
- [ ] Testing requirements from spec met (coverage %, test types)

**If spec missing**:
- 🚫 **BLOCKED** - Cannot proceed without approved spec
- Required action: Create spec, get approval, resubmit PR

**If implementation deviates from spec without update**:
- ⚠️ **CHANGES REQUIRED** - Update spec OR revert to spec design
- Required action: Justify deviation in spec, get re-approval
```

### Layer 4: Spec Index Tracking

**Create `docs/specs/README.md`**:

```markdown
# Specification Index

**Status Legend**:
- **Draft**: In progress, not approved
- **Review**: Ready for review
- **Approved**: Ready for implementation
- **Implemented**: Complete
- **Deprecated**: No longer relevant

| Spec | Status | Version | Date | Category | Implementation PR |
|------|--------|---------|------|----------|-------------------|
| [multi-tenant](multi-tenant/SPEC.md) | Implemented | 2.0.0 | 2025-11-04 | Architecture | #123 |
| [skill-enforcement-system](skill-enforcement-system/SPEC.md) | Approved | 1.0.0 | 2025-11-18 | Workflow | - |
| [spec-driven-development](spec-driven-development/SPEC.md) | Approved | 1.0.0 | 2025-11-18 | Workflow | - |
| ... | ... | ... | ... | ... | ... |

**Metrics**:
- Total specs: 20
- Approved: 18
- In progress (Draft/Review): 2
- Implemented: 16
- Deprecated: 0
```

**Maintained by**: Automated (updated when specs created/approved/implemented)

---

## Workflows

### Feature Development Workflow

```
User: "Implement authentication feature"

Agent:
1. Check for spec: `docs/specs/auth/SPEC.md` exists?

   If NO:
   → "BLOCKED: No spec exists. Invoking contextd:creating-spec skill."
   → Create spec using template
   → Request approval
   → WAIT for approval
   → Once approved: Begin implementation

   If YES:
   → Read spec
   → Check Status field

   If Status: Draft or Review:
   → "BLOCKED: Spec not approved. Cannot implement until Status: Approved."
   → WAIT for approval

   If Status: Approved:
   → "Spec approved. Beginning implementation following spec design."
   → Implement according to spec
   → Invoke contextd:completing-major-task when done
   → Code review validates spec adherence
```

### Package Creation Workflow

```
User: "Create pkg/rbac package"

Agent:
1. Invoke contextd:creating-package skill (MANDATORY)

   Skill checks: Does spec exist for this package?

   If complex package (>500 lines expected):
   → Invoke contextd:creating-spec skill (MANDATORY)
   → Create docs/specs/rbac/SPEC.md
   → Get spec approved
   → THEN create package

   If simple package:
   → Create package directly
   → Document in pkg/CLAUDE.md
```

### Spec Deviation Workflow

```
During Implementation: Discover spec design won't work

Agent:
1. STOP implementation
2. Document why spec design doesn't work
3. Propose alternative in spec
4. Update spec with new design
5. Request re-approval
6. WAIT for approval
7. Resume implementation with new design

Code Review:
- Validates spec was updated
- Validates rationale provided
- Validates new design approved
```

---

## Spec Lifecycle

### States

1. **Draft** - Being written, not ready for review
2. **Review** - Ready for review, awaiting feedback
3. **Approved** - Reviewed and approved, ready for implementation
4. **Implemented** - Code complete, spec reflects reality
5. **Deprecated** - No longer relevant, archived

### State Transitions

```
Draft → Review → Approved → Implemented
                    ↓
                Deprecated
```

**Rules**:
- Implementation CANNOT begin until Status: Approved
- Once Implemented, spec is living document (update if code changes)
- Deprecated specs moved to docs/archive/

---

## Success Metrics

### Quantitative
- 100% of features have approved spec before implementation
- 100% of packages have spec (if complex) or pkg/CLAUDE.md entry
- 0 PRs merged with missing specs (for mandatory features)
- 0 PRs with spec deviations without spec update
- Spec index up to date (all specs listed)

### Qualitative
- Clear design before implementation
- Less rework due to unclear requirements
- Authoritative documentation exists
- Context captured in specs, not just code
- Future developers understand design rationale

---

## Implementation Plan

### Phase 1: Infrastructure
1. Create `docs/specs/spec-driven-development/SPEC.md` (this file)
2. Create `docs/specs/README.md` (spec index)
3. Update root CLAUDE.md (add spec-driven principle + checklist)
4. Update CODE-REVIEW-CHECKLIST.md (add spec adherence section)

### Phase 2: Skills
5. Create `contextd:creating-spec` skill (MANDATORY for features/packages)
6. Update `contextd:creating-package` skill (add spec requirement for complex packages)

### Phase 3: Enforcement
7. Test workflow: Try to implement feature without spec → BLOCKED
8. Test workflow: Create spec, get approval → Implementation allowed
9. Validate code review catches missing specs

### Phase 4: Documentation
10. Update DEVELOPMENT-WORKFLOW.md (add spec-first workflow)
11. Update MULTI-AGENT-ORCHESTRATION.md (add contextd:creating-spec)

---

## Dependencies

### Blocks This Spec
- None (foundational)

### This Spec Blocks
- All future feature development (requires specs)
- All future package creation (requires specs for complex packages)
- Skill enforcement system (requires spec approval before skill implementation)

### Related Specs
- [Skill Enforcement System](../skill-enforcement-system/SPEC.md) - Package creation workflow
- [Verification Policy](../../guides/VERIFICATION-POLICY.md) - Completion verification (needs spec)

---

## Design Decisions

### Why Mandatory Spec Approval Before Implementation?

**Options considered**:
- **A**: Specs optional, create if you want
  - ❌ Doesn't solve "no design" problem
  - ❌ Easy to skip planning
- **B**: Specs required but implementation can start anytime
  - ❌ Specs become afterthought
  - ❌ "Write spec while coding" defeats purpose
- **C**: Specs required and MUST be approved before implementation
  - ✅ Forces design-first thinking
  - ✅ Reduces rework
  - ✅ Creates authoritative docs

**Decision**: Option C - Mandatory approval before implementation

**Rationale**: Only way to enforce spec-driven development is to BLOCK implementation until spec approved. Otherwise specs become documentation of what was already done, not design of what to build.

### Why Separate specs/ and plans/ Directories?

**Options considered**:
- **A**: One docs/ directory for everything
  - ❌ Hard to find approved specs
  - ❌ Mixes drafts with authoritative docs
- **B**: specs/ for approved, plans/ for drafts
  - ✅ Clear separation of authoritative vs working drafts
  - ✅ specs/ are source of truth
  - ✅ plans/ can be messy

**Decision**: Option B - Separate directories

**Rationale**: specs/ contains ONLY approved, authoritative documentation. plans/ contains working drafts that may be promoted to specs or archived.

### Why Status Field in Specs?

**Options considered**:
- **A**: No status tracking
  - ❌ Can't tell if spec is approved
  - ❌ Agents might implement Draft specs
- **B**: Status in filename (SPEC-DRAFT.md, SPEC-APPROVED.md)
  - ❌ Requires file renames
  - ❌ Breaks links
- **C**: Status field in SPEC.md header
  - ✅ Clear approval state
  - ✅ No file renames
  - ✅ Easy to check

**Decision**: Option C - Status field in header

**Rationale**: Status field is easy to check, doesn't require file renames, and clearly indicates approval state.

---

## Technical Debt

**NOTE**: This spec violates our own modularity guidelines (Kinney approach).

**Violation**: Monolithic SPEC.md (~500 lines) instead of scannable main (~150 lines) + @imported details.

**Should be refactored to**:
- Main SPEC.md (~150 lines, scannable overview)
- @./docs-structure.md - Directory purposes and organization
- @./spec-requirements.md - What requires specs, template structure
- @./enforcement.md - 4-layer enforcement mechanisms
- @./workflows.md - Feature/package/deviation workflows

**Status**: Accepted technical debt, will refactor during implementation phase or when updating spec.

**Created**: Before `kinney-documentation` skill existed. Future specs MUST follow modular approach.

---

## Open Questions

1. **Who approves specs?**
   - For solo development: Self-approval after review
   - For teams: Tech lead or designated reviewer
   - Decision: Define approval process per project

2. **How long should spec reviews take?**
   - Target: 24-48 hours for small specs
   - Complex specs: Up to 1 week
   - Decision: No hard deadline, but fast iteration encouraged

3. **Should we require specs for ALL bug fixes?**
   - Current: No (documented in regression tests)
   - Alternative: Yes for complex bugs
   - Decision: No for now, revisit if bug complexity increases

4. **Should specs include implementation timeline estimates?**
   - Current: No (avoid commitment to timelines)
   - Alternative: Yes for planning purposes
   - Decision: No timelines in specs

---

## References

### Design Documents
- [docs/plans/2025-11-18-verification-enforcement-design.md](../../plans/2025-11-18-verification-enforcement-design.md)
- [docs/plans/2025-11-18-skill-enforcement-design.md](../../plans/2025-11-18-skill-enforcement-design.md)

### Standards
- [docs/standards/architecture.md](../../standards/architecture.md)
- [docs/standards/coding-standards.md](../../standards/coding-standards.md)

### Guides
- [docs/guides/DEVELOPMENT-WORKFLOW.md](../../guides/DEVELOPMENT-WORKFLOW.md)
