---
name: contextd-creating-spec
description: Use when implementing any feature or making significant changes to contextd, before writing any code - enforces mandatory spec-driven development policy where NO CODE can be written without an approved specification in docs/specs/<feature>/SPEC.md
---

# Creating Specification Documents

## Overview

**NO CODE WITHOUT SPEC.** This is not a suggestion - it's a hard requirement for all feature development in contextd.

Before implementing ANY feature, you MUST verify a specification exists at `docs/specs/<feature>/SPEC.md` with `Status: Approved`. If missing or not approved, implementation is BLOCKED.

## When to Use This Skill

**Trigger immediately when:**
- About to implement any new feature
- Asked to add functionality (new MCP tools, API endpoints, packages)
- Asked to make significant behavioral changes
- Adding new parameters, flags, or options
- Implementing "enhancements" or "improvements"

**When NOT to use:**
- Trivial bug fixes (single typo, formatting, no behavior change)
- Pre-existing specs with Status: Approved already exist

## Mandatory Workflow

### Step 1: Check for Existing Spec

**BEFORE writing any implementation code:**

```bash
# Check if spec exists
ls docs/specs/<feature-name>/SPEC.md

# If exists, verify Status: Approved
head -20 docs/specs/<feature-name>/SPEC.md | grep "Status:"
```

**If spec exists with `Status: Approved`**: Proceed to implementation
**If spec missing or status is Draft/Review**: BLOCK implementation, proceed to Step 2

### Step 2: Create Specification (If Missing)

**Use the spec template:**

```markdown
---
title: <Feature Name>
status: Draft
created: <YYYY-MM-DD>
author: <Your Name>
---

# <Feature Name> Specification

## Overview
What is this feature? Why does it exist? (2-3 sentences)

## Requirements

### Functional Requirements
- FR-1: Feature must...
- FR-2: Feature should...

### Non-Functional Requirements
- NFR-1: Performance target...
- NFR-2: Security requirement...

## Architecture

### Design Overview
High-level approach, key components

### Data Model
Structures, database schemas, vector collections

### API Design
Endpoints, MCP tools, function signatures

## Security Considerations

**CRITICAL for contextd**: Multi-tenant isolation analysis

- Does this expose data across project/team boundaries?
- Input validation requirements?
- Authentication/authorization needs?
- Compliance implications (GDPR, SOC 2)?

## Testing Strategy

### Test Coverage
- Unit test requirements (≥80%)
- Integration test scenarios
- Security test cases

### Test Cases
Specific scenarios to verify

## Implementation Plan

### Phases
1. Phase 1: Core implementation
2. Phase 2: Testing
3. Phase 3: Documentation

### Dependencies
Prerequisites, blocked by, blocks

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Test coverage ≥80%
- [ ] Security validation passed
- [ ] Documentation updated

## References
- Related ADRs
- External documentation
- Research documents
```

**Save to**: `docs/specs/<feature-name>/SPEC.md`

### Step 3: Spec Approval Workflow

**Status progression:**
1. **Draft** - Spec written, under development
2. **Review** - Complete, needs approval
3. **Approved** - Ready for implementation

**Change status in frontmatter:**
```yaml
status: Review  # or Approved
```

**Approval criteria:**
- All sections complete (no TBD placeholders)
- Security considerations analyzed
- Testing strategy defined
- Acceptance criteria clear

**Self-approval allowed ONLY when ALL criteria met:**
- Minor features (<100 lines code) AND
- Internal-only tools AND
- Non-security-critical changes AND
- No multi-tenant isolation impact AND
- No API changes

**MUST require review for (NEVER self-approve):**
- Security-sensitive features (ANY security impact)
- Multi-tenant isolation changes
- API changes (MCP tools, endpoints)
- Breaking changes
- Database schema changes
- When in doubt (if uncertain, require review)

### Step 4: Implementation with Spec Reference

**When implementing:**

```go
// Implements list_projects MCP tool
// Spec: docs/specs/list-projects-tool/SPEC.md
func ListProjects(ctx context.Context) ([]Project, error) {
    // Implementation following spec
}
```

**In PR description:**
```markdown
## Specification
Implements: docs/specs/list-projects-tool/SPEC.md
Status: Approved (YYYY-MM-DD)
```

## BLOCKING Behavior

**If spec is missing or not approved, STOP immediately:**

1. **Don't write code** - Not even "draft", "prototype", "POC", or "research code"
2. **Don't "start small"** - Spec comes first, always
3. **Don't continue existing code** - Sunk cost doesn't justify skipping spec
4. **Don't write tests without spec** - TDD still requires spec (spec → tests → code)
5. **Create spec first** - Follow Step 2 template
6. **Get approval** - Change status to Approved (Draft ≠ approved)
7. **Then implement** - Only after Status: Approved

**"Status: Draft" or "Status: Review" = NOT APPROVED = BLOCKED**

## What Requires a Spec

### Requires Spec (No Exceptions)

- New MCP tools
- New API endpoints
- New packages
- New command-line flags/options
- Behavior changes (even "bug fixes" that add functionality)
- Performance optimizations (document approach)
- Security changes
- Multi-file changes
- Database schema changes
- Configuration additions
- Significant refactoring (multi-file, structural changes)

### Does NOT Require Spec

**Only these trivial changes:**
- Single typo fixes (cosmetic only)
- Code formatting (gofmt, whitespace)
- Comment updates (not behavior documentation)
- Variable renames (internal only, no API changes)
- Trivial refactoring (extract single function, same file)

**If it changes behavior, adds functionality, or touches multiple files: IT REQUIRES A SPEC.**

**Refactoring clarification:**
- Significant refactoring (multi-function/file, structural) → Spec required
- Trivial refactoring (rename, extract single function) → No spec
- When in doubt → Create spec

## Common Rationalizations (Don't Fall For These)

| Excuse | Reality |
|--------|---------|
| "Feature is simple/obvious" | Simple features still need specs. Clarity now prevents confusion later. |
| "No time for spec, urgent deadline" | Urgency makes specs MORE critical. Spec takes 15 min, debugging unclear requirements takes hours. |
| "Spec can be added after implementation" | Specs-after = documentation. Specs-first = design. You need design. |
| "Code already exists, too late for spec" | Delete the code. Start with spec. Sunk cost doesn't justify broken process. |
| "Just adding parameters, not a feature" | Parameters ARE features. They add functionality. Spec required. |
| "It's a bug fix, not a feature" | Does it add functionality? New code paths? New parameters? Then it's a feature. Spec required. |
| "Spec would just repeat the description" | If spec = description, writing it takes 5 minutes. Do it. |
| "This is an enhancement, not a feature" | No distinction. Enhancement = feature = requires spec. |
| "Requirements are clear, everyone knows" | "Everyone knows" is not documentation. Write the spec. |
| "Already approved verbally" | Verbal approval is not `Status: Approved` in spec file. Write it down. |
| "I'll write a prototype to explore" | Prototypes ARE code. No code without spec. Explore in spec document. |
| "Tests are the spec (TDD)" | Tests verify behavior. Specs document design. Both required. Spec → Tests → Code. |
| "Just refactoring, not new features" | Significant refactoring needs spec. If multi-file or structural → spec required. |
| "Spec is in Draft, I can start" | Draft ≠ Approved. Status must be "Approved" to start implementation. BLOCKED. |

**All of these mean: Stop coding. Write spec first. No exceptions.**

## Red Flags - STOP and Create Spec

**If you're thinking ANY of these, you're about to violate spec-driven development:**

- "I'll just implement this quickly"
- "Spec can come later"
- "This is too simple for a spec"
- "Already started coding"
- "Just need to finish this part"
- "It's obvious what this should do"
- "Bug fix" (for new functionality)
- "Minor change" (for behavior additions)
- "Let me write a prototype/POC"
- "Just researching implementation approaches"
- "Tests are my spec (TDD)"
- "Spec is in Draft, good enough"
- "Just refactoring" (for structural changes)
- "Self-approving this spec" (for security/API changes)

**All of these mean: STOP. Check for spec. Create if missing. Get approval. THEN code.**

## Relationship with TDD

**Both specs AND TDD are mandatory. They serve different purposes:**

| Aspect | Specification | Tests (TDD) |
|--------|---------------|-------------|
| Purpose | Document design and requirements | Verify behavior |
| When | Before ANY code | Before implementation code |
| What | Architecture, API, security, acceptance criteria | Executable verification of behavior |
| Format | Markdown document | Go test code |

**Correct workflow:**
1. **Spec** - Design the feature (architecture, requirements, security)
2. **Tests** - Write tests based on spec acceptance criteria (TDD red phase)
3. **Code** - Implement to pass tests (TDD green phase)
4. **Refactor** - Improve while maintaining passing tests

**Specs and tests are complementary, not substitutes:**
- Spec answers "What should we build and why?"
- Tests answer "Does it work as specified?"

**TDD doesn't eliminate need for specs. Specs guide what tests to write.**

## Integration with golang-pro

**When delegating to golang-pro skill:**

```markdown
BEFORE delegating to golang-pro:
1. Verify spec exists: docs/specs/<feature>/SPEC.md
2. Verify spec status: Status: Approved
3. Provide spec path to golang-pro

"Use golang-pro skill to implement <feature> following specification at docs/specs/<feature>/SPEC.md"
```

**golang-pro will:**
- Reference spec during implementation
- Write tests first (TDD) based on spec acceptance criteria
- Implement according to spec requirements
- Validate against spec acceptance criteria

## Spec Template Location

**Full template**: `docs/specs/spec-driven-development/SPEC.md`

**Quick template** (use for simple features):
```markdown
---
title: Feature Name
status: Draft
created: YYYY-MM-DD
---

# Feature Name

## Overview
[What + Why]

## Requirements
- Functional: [What it must do]
- Security: [Multi-tenant isolation, input validation]
- Performance: [Targets if applicable]

## Implementation
- Approach: [High-level design]
- Files: [What needs to change]
- Tests: [Test strategy]

## Acceptance Criteria
- [ ] Core functionality works
- [ ] Tests pass (≥80% coverage)
- [ ] Security validated
- [ ] Documentation updated
```

## Summary

**Spec-driven development is mandatory:**

1. **Check** for spec before ANY implementation
2. **Create** spec if missing (use template)
3. **Approve** spec (change status to Approved)
4. **Implement** following spec
5. **Reference** spec in code and PR

**No exceptions for:**
- Time pressure
- Simple features
- Existing code
- "Minor" changes
- "Bug fixes" that add functionality

**Violating spec-driven development = violating core project policy.**

If you're writing code without an approved spec, you're doing it wrong. Stop and create the spec first.
