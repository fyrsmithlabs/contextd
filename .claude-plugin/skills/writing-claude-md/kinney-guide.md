# Kinney CLAUDE.md Guide

Source: https://stevekinney.com/courses/ai-development/claude-dot-md

## Core Concept

CLAUDE.md is the **central source of truth** and a dynamic task board for your AI team. It provides persistent memory across sessions and helps Claude align with the overall project vision.

## File Hierarchy (All Load)

| Location | Purpose | Sharing |
|----------|---------|---------|
| `~/.claude/CLAUDE.md` | User preferences | All projects |
| `./CLAUDE.md` | Project rules | Team (Git) |
| Parent directories | Monorepo root | Inherited |
| Subdirectories | Module overrides | On demand |

## Required Content

### Project Foundation
- **Architecture Overview**: High-level descriptions of structure and components
- **Coding Standards**: Indentation, naming, ES modules, type annotations
- **Common Commands**: Build, test, lint, dev environment setup

### Workflow & Process
- **Git Strategy**: Branching, merge vs rebase, commit format
- **Testing Strategy**: What to test, coverage requirements
- **Known Pitfalls**: Unexpected behaviors, limitations, warnings

### AI Behavior Rules
- Error handling patterns
- API conventions
- Problem-solving approaches
- **ADRs**: Architectural Decision Records with rationale

## Best Practices

### Start with /init
```bash
# Generate foundational CLAUDE.md
/init
```
This analyzes your codebase and creates a starter file. Commit to version control.

### Keep Concise
- Use Markdown headings and bullet points
- Be explicit and detailed, not vague
- Token budget: Every line costs context

### Refine Constantly
- Treat as living document
- Add clarifications when Claude asks repeatedly
- Use `#` key for quick additions during sessions
- Use `/memory` for extensive edits

### Modularize Large Docs
```markdown
@path/to/architecture.md
@path/to/api-conventions.md
```
Keeps main CLAUDE.md clean, saves tokens.

### Nouns vs Verbs
- **CLAUDE.md** = Nouns (what and where)
- **Slash commands** = Verbs (how to do)

## Structure Template

```markdown
# CLAUDE.md - [Project Name]

**Status**: Active Development
**Last Updated**: YYYY-MM-DD

---

## Critical Rules

**ALWAYS** use named exports, NEVER default exports
**ALWAYS** handle errors with Result<T, E> pattern
**NEVER** use try/catch in components

---

## Project Overview

[One paragraph description]

## Architecture

```
src/
├── components/    # React components
├── lib/           # Core business logic
└── utils/         # Pure utilities
```

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Runtime   | Node.js    | 20.x    |

## Commands

- `npm run build`: Build for production
- `npm run test`: Run unit tests
- `npm run typecheck`: Type checking

## Code Style

- Use ES modules (import/export)
- Destructure imports when possible
- Prefer arrow functions

## Do Not

- Edit files in `src/legacy/`
- Commit directly to `main`
- Skip accessibility checks
```

## Testing Your CLAUDE.md

1. **Request summary**: Ask Claude to explain the project
2. **Check specificity**: Responses should reference YOUR details
3. **Generic = needs work**: If Claude gives generic advice, add more specificity

## Maintenance Triggers

Update when:
- Adding new dependencies
- Changing architecture
- Adding environment variables
- Discovering new gotchas
- Claude repeatedly asks for missing info

## Common Pitfalls

| Problem | Solution |
|---------|----------|
| Too generic | Replace "modern tech" with specific versions |
| Outdated | Add verification dates |
| No examples | Always show code patterns |
| Too long | Modularize with `@imports` |
