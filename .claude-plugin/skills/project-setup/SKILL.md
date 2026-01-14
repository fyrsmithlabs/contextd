---
name: project-setup
description: Use when onboarding to a project or creating/updating CLAUDE.md - covers codebase analysis, documentation generation, and policy management
---

# Project Setup

## When to Use

- Joining existing project without CLAUDE.md
- Project has outdated/incomplete CLAUDE.md
- Taking over maintenance of unfamiliar codebase
- Claude repeatedly asks the same questions

## Onboarding Workflow

### Phase 1: Discovery

```
1. Repository structure scan (Glob)
2. Package/dependency analysis (package.json, go.mod, requirements.txt)
3. Configuration files (tsconfig, .eslintrc, Makefile)
4. Existing documentation (README, docs/, CONTRIBUTING)
5. CI/CD configuration (.github/workflows)
```

### Phase 2: Pattern Extraction

| Target | Search Strategy |
|--------|-----------------|
| Entry points | main.*, index.*, cmd/, src/ |
| Architecture | Directory structure, imports |
| Testing | *_test.*, *.spec.*, jest.config |
| Build commands | Makefile, package.json scripts |

### Phase 3: CLAUDE.md Generation

Generate with this structure:

```markdown
# CLAUDE.md - [Project Name]

**Status**: Active Development | Maintenance | Legacy
**Last Updated**: YYYY-MM-DD

---

## Critical Rules

**ALWAYS** [most important constraints]
**NEVER** [dangerous actions to avoid]

---

## Architecture

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

## Commands

| Command | Purpose |
|---------|---------|
| `npm run dev` | Start development server |
| `npm run test` | Run unit tests |

## Known Pitfalls

| Pitfall | Prevention |
|---------|------------|
| [Issue 1] | [How to avoid] |
```

## CLAUDE.md Best Practices

**ALWAYS:**
- Start with Status and Last Updated
- Put Critical Rules (NEVER/ALWAYS) at top
- Use specific versions, paths, exact commands
- Include code examples for key patterns

**NEVER:**
- Write vague descriptions ("modern tech stack")
- Skip examples for important concepts
- List commands without context

## File Hierarchy

| Location | Purpose | Scope |
|----------|---------|-------|
| `~/.claude/CLAUDE.md` | User preferences | All projects |
| `./CLAUDE.md` | Project rules | Team (Git) |
| Parent directories | Monorepo root | Inherited |
| Subdirectories | Module overrides | On demand |

All locations load automatically. Most specific wins.

## Modularization with @imports

For large projects, split documentation:

```markdown
# CLAUDE.md
@docs/architecture.md
@docs/api-conventions.md
@docs/testing-strategy.md
```

Keep main CLAUDE.md clean. Only core rules inline.

## Policies (Strict Guidelines)

Policies are STRICT constraints that MUST be followed:

```json
{
  "project_id": "global",
  "title": "POLICY: test-before-fix",
  "content": "RULE: Always run tests before claiming a fix is complete.\nCATEGORY: verification\nSEVERITY: high",
  "outcome": "success",
  "tags": ["type:policy", "category:verification", "severity:high"]
}
```

**Categories:** verification, process, security, quality, communication

**Severity:** critical, high, medium

## After Setup

```
# Re-index repository with new documentation
repository_index(path: ".")

# Record the setup as a memory
memory_record(
  project_id: "<project>",
  title: "Project onboarded with CLAUDE.md",
  content: "Created CLAUDE.md with architecture, commands, pitfalls...",
  outcome: "success",
  tags: ["onboarding", "claude-md"]
)
```

## Common Mistakes

| Mistake | Prevention |
|---------|------------|
| Asking user "what framework?" | Check package.json, go.mod first |
| Generic architecture description | Run actual discovery commands |
| Missing version numbers | Extract from lockfiles |
| No examples for key patterns | Always show code patterns |
| Too long CLAUDE.md (>500 lines) | Modularize with @imports |

## Quick Reference

| Step | Action |
|------|--------|
| 1 | Scan repo structure |
| 2 | Read package/config files |
| 3 | Extract patterns |
| 4 | Generate CLAUDE.md |
| 5 | Verify commands work |
| 6 | Index repository |
