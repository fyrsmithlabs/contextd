---
name: writing-claude-md
description: Use when creating or updating CLAUDE.md files for any project - provides structure, best practices, and anti-patterns for effective AI-assisted development documentation
---

# Writing CLAUDE.md

## Overview

CLAUDE.md is the **central source of truth** for AI-assisted development. It provides persistent memory across sessions and helps Claude align with the project vision.

**Core principle:** Be explicit and specific. Vague instructions produce generic responses.

@./kinney-guide.md

## When to Use

- Starting a new project (needs CLAUDE.md from scratch)
- Onboarding existing project (analyzing codebase, creating CLAUDE.md)
- Claude repeatedly asks the same questions
- Architecture, dependencies, or environment variables change
- Adding new ADRs (Architectural Decision Records)

## File Hierarchy

| Location | Purpose | Scope |
|----------|---------|-------|
| `~/.claude/CLAUDE.md` | User preferences | All projects |
| `./CLAUDE.md` | Project rules | Team (Git) |
| Parent directories | Monorepo root | Inherited |
| Subdirectories | Module overrides | On demand |

**All locations load automatically.** Most specific wins for conflicts.

## Critical Rules

**ALWAYS**:
- Start with Status and Last Updated
- Put Critical Rules (NEVER/ALWAYS) at the top
- Use specific versions, paths, exact commands
- Include code examples for key patterns
- Use tables for structured data

**NEVER**:
- Write vague descriptions ("modern tech stack")
- Skip examples for important concepts
- List commands without context
- Duplicate content across sections

## Structure Template

```markdown
# CLAUDE.md - [Project Name]

**Status**: Active Development | Maintenance | Legacy
**Last Updated**: YYYY-MM-DD

---

## Critical Rules

**ALWAYS** [most important constraints]
**NEVER** [dangerous actions to avoid]

---

## Project Overview

[One paragraph: what this is, what problem it solves]

## Architecture

[ASCII diagram or directory structure]
```
src/
├── components/    # [purpose]
├── lib/           # [purpose]
└── utils/         # [purpose]
```

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Runtime   | Node.js    | 20.x    |
| Framework | Next.js    | 14.x    |

## Commands

| Command | Purpose |
|---------|---------|
| `npm run dev` | Start development server |
| `npm run test` | Run unit tests |
| `npm run build` | Production build |

## Code Standards

- [Specific patterns to follow]
- [Import conventions]
- [Error handling approach]

## Known Pitfalls

| Pitfall | Prevention |
|---------|------------|
| [Issue 1] | [How to avoid] |

## ADRs (Architectural Decisions)

### ADR-001: [Decision Title]
**Status**: Accepted | Deprecated | Superseded
**Context**: [Why this decision was needed]
**Decision**: [What was decided]
**Consequences**: [Trade-offs and implications]
```

## Modularization with @imports

For large projects, split documentation:

```markdown
# CLAUDE.md
@docs/architecture.md
@docs/api-conventions.md
@docs/testing-strategy.md
```

**Keep main CLAUDE.md clean.** Only core rules inline. Reference details with `@`.

## Testing Your CLAUDE.md

1. **Ask Claude to summarize**: "Explain this project"
2. **Check specificity**: Responses should reference YOUR details
3. **Generic = needs work**: Add more specific constraints

## Common Anti-Patterns

| Problem | Fix |
|---------|-----|
| "Modern tech stack" | List exact versions: "React 18.2, TypeScript 5.3" |
| Missing examples | Add before/after code for key patterns |
| Commands without context | Add purpose: "`npm run lint` - Check code style" |
| No verification dates | Add Last Updated to header |
| Too long (>500 lines) | Modularize with `@imports` |

## CSO (Claude Search Optimization)

When Claude searches for guidance, it scans:
1. Critical Rules section first
2. Quick reference tables
3. Code examples

**Optimize for this flow:**
- Put most important rules at top
- Use tables for scannable data
- Keywords Claude would search: error names, command names, file paths

## Maintenance Triggers

Update CLAUDE.md when:
- Adding new dependencies
- Changing architecture
- Adding environment variables
- Discovering new gotchas
- Claude asks the same question twice
- ADRs change

## Contextd Integration

After creating or updating CLAUDE.md:

```
# Re-index repository with new documentation
repository_index(path: ".")

# Record the update as a memory
memory_record(
  project_id: "<project>",
  title: "Updated CLAUDE.md with [changes]",
  content: "Added/modified: [sections]. Key additions: [summary]",
  outcome: "success",
  tags: ["claude-md", "documentation"]
)
```

**Before writing:** Search for existing patterns:
```
memory_search(project_id, "CLAUDE.md patterns")
```

## Quick Reference

| Section | Purpose |
|---------|---------|
| Critical Rules | Stop dangerous actions |
| Architecture | Orient Claude to structure |
| Commands | Enable quick execution |
| Code Standards | Ensure consistency |
| Known Pitfalls | Prevent repeated mistakes |
| ADRs | Explain WHY decisions were made |
