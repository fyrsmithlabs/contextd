---
name: kinney-documentation
description: Use when writing any documentation (CLAUDE.md, specs, guides) - enforces Steve Kinney's modular, scannable approach with nouns vs verbs, ~150 line max, @imports for details
---

# Kinney Documentation Approach

## Overview

This skill enforces Steve Kinney's documentation principles for ALL documentation: CLAUDE.md, specs, guides, and README files.

**Core Principle**: Documentation should be **scannable** (quick orientation) with **modular depth** (@imports for details).

**MANDATORY**: Use this skill whenever writing or editing documentation files.

---

## When to Use This Skill

Use this skill when:
- ✅ Writing or editing CLAUDE.md files
- ✅ Writing or editing SPEC.md files
- ✅ Writing or editing guides (docs/guides/*.md)
- ✅ Writing or editing README files
- ✅ Creating any documentation >50 lines

**DO NOT** skip this skill. Documentation is critical infrastructure.

---

## Kinney's Core Principles

### 1. Nouns vs Verbs

**Documentation should focus on NOUNS (what/where things are), not VERBS (how/when to do things).**

**Nouns** (Foundational - what/where):
- What the project is
- Where components live
- What technologies are used
- What the architecture looks like
- Where to find information

**Verbs** (Operational - how/when):
- How to set up the project
- When to use certain patterns
- How to implement features
- When to invoke skills

**Rule**: Main documentation files (CLAUDE.md, SPEC.md) should be noun-heavy. Use @imports to separate out verb-heavy content.

**Example**:
```markdown
## ❌ BAD: Mixed nouns and verbs in main file (hard to scan)

# CLAUDE.md

## Architecture
We use Go 1.23 with Qdrant for vector storage. To set up Qdrant, first install Docker, then run `docker-compose up -d`. The system uses three-tier architecture with project isolation. When creating new packages, follow these steps: 1) Create pkg/<name> directory, 2) Add package-level godoc...

## ✅ GOOD: Nouns in main, verbs in @imports (scannable)

# CLAUDE.md

## Architecture (Nouns)
**Language**: Go 1.23+
**Storage**: Qdrant (vector database)
**Architecture**: Three-tier with project isolation

**Setup**: @docs/guides/GETTING-STARTED.md
**Package Creation**: @docs/guides/CREATING-PACKAGES.md
```

### 2. Modularity (~150 Line Maximum)

**Main documentation files should be scannable in one screen (~150 lines max).**

**If file exceeds 150 lines**:
1. Identify sections that can be extracted
2. Create separate detail files
3. Use @imports to reference them
4. Keep main file as scannable index

**Example Structure**:
```markdown
# Main SPEC.md (~150 lines)

## Overview
[Brief description]

## Quick Reference
[Essential info only]

## Detailed Documentation
@./enforcement.md - Enforcement mechanisms
@./workflows.md - Package creation workflows
@./implementation.md - Implementation phases

## Summary
[Wrap-up]
```

### 3. Hierarchical Organization

**Use hierarchical structure: User-level → Project-level → Directory-specific**

**User-level** (`~/.claude/CLAUDE.md`): Global settings, personal preferences
**Project-level** (`./<project>/CLAUDE.md`): Project-wide policies
**Directory-specific** (`./pkg/CLAUDE.md`): Package-specific rules

**Each level can @import more specific guidance.**

### 4. Token Efficiency

**Goal**: Minimize tokens loaded into context while maintaining usefulness.

**Strategies**:
- Scannable main files (agents can navigate to what they need)
- @imports load only when needed
- Concise, declarative statements (not prose)
- Code examples over explanations
- Cross-references instead of duplication

**Bad** (high token, hard to scan):
```markdown
The authentication system works by using Bearer tokens that are
generated using crypto/rand with 32 bytes that are then converted
to hexadecimal format. The tokens are stored in the user's home
directory under .config/contextd/token with file permissions that
must be set to 0600 to ensure security. When comparing tokens, we
use constant-time comparison to prevent timing attacks which could
leak information about the token contents...
```

**Good** (low token, scannable):
```markdown
## Authentication
- **Method**: Bearer tokens
- **Generation**: crypto/rand (32 bytes → hex)
- **Storage**: ~/.config/contextd/token (0600 permissions)
- **Comparison**: constant-time (prevents timing attacks)

**Details**: @./auth-implementation.md
```

---

## Mandatory Documentation Checklist

**Before completing ANY documentation, verify**:

### ☐ Noun-Heavy Main File
- [ ] Main file focuses on WHAT and WHERE (nouns)
- [ ] HOW and WHEN extracted to @imported files (verbs)
- [ ] Easy to scan for specific information

### ☐ Length Check
- [ ] Main file ≤150 lines (preferred) or ≤200 lines (maximum)
- [ ] If longer: Extract sections to separate files
- [ ] Each @imported file also ≤200 lines

### ☐ Modular Structure
- [ ] Main file is scannable index/overview
- [ ] Detailed content in @imported files
- [ ] Clear navigation between files
- [ ] No duplication across files

### ☐ Token Efficiency
- [ ] Concise, declarative statements
- [ ] Code examples instead of prose
- [ ] Cross-references instead of duplication
- [ ] Clear section headers (agents can navigate)

### ☐ Hierarchical Placement
- [ ] File at correct level (user/project/directory)
- [ ] Doesn't duplicate content from higher levels
- [ ] References higher-level docs with @imports

---

## Common Patterns

### Pattern 1: Scannable CLAUDE.md

```markdown
# CLAUDE.md (~150 lines)

## Core Principles
[3-5 numbered principles, concise]

## Architecture & Standards
@docs/standards/architecture.md
@docs/standards/coding-standards.md

**Key Technologies**:
- Language: Go 1.23+
- Storage: Qdrant
- Protocol: HTTP/MCP

## Development Workflow
@docs/guides/DEVELOPMENT-WORKFLOW.md

**Quick Workflow**: Design → Spec → Test → Implement → Review

## Critical Rules
[5-7 absolute rules, numbered]

## Quick References
[Commands, build, test - essential only]
```

### Pattern 2: Modular Spec

```markdown
# Main SPEC.md (~150 lines)

**Version**: 1.0.0
**Status**: Approved
**Date**: YYYY-MM-DD

## Overview
[Purpose, goals, features - 30 lines]

## Quick Reference
[Essential facts - 20 lines]

## Detailed Documentation
@./architecture.md - System design
@./requirements.md - Functional/non-functional requirements
@./implementation.md - Phase-by-phase plan
@./workflows.md - Example workflows

## Summary
[Wrap-up - 10 lines]
```

### Pattern 3: Guide with TOC

```markdown
# Guide Name

**See**: [CLAUDE.md](../../CLAUDE.md) for project overview

## Table of Contents
- [Quick Start](#quick-start)
- [Detailed Steps](#detailed-steps)

## Quick Start
[Essential steps only - 20 lines]

**Full Details**: @./detailed-setup.md

## Detailed Steps
@./step-by-step.md

## Common Issues
@./troubleshooting.md
```

---

## Anti-Patterns (What NOT to Do)

### ❌ Anti-Pattern 1: Monolithic Files

**Bad**:
```markdown
# SPEC.md (500 lines of everything)

## Overview
...150 lines...

## Architecture
...100 lines with diagrams...

## Requirements
...80 lines...

## Implementation
...170 lines...
```

**Why bad**: Impossible to scan, high token cost, hard to maintain

**Fix**: Break into main + @imported files

### ❌ Anti-Pattern 2: Verbose Prose

**Bad**:
```markdown
The system is designed to handle multiple projects by creating a
separate database for each project. This approach was chosen because
it provides better isolation between projects and improves query
performance by allowing the database to only scan the relevant
database for a given project rather than filtering through all
projects in a shared database. Each database is named using a hash...
```

**Why bad**: High token cost, hard to scan, buries key facts

**Fix**: Use concise, declarative format with bullets

### ❌ Anti-Pattern 3: Duplication

**Bad**: Same information in CLAUDE.md AND spec AND guide

**Why bad**: Inconsistency when one updates but not others, token waste

**Fix**: One authoritative source, others @import it

### ❌ Anti-Pattern 4: Mixed Concerns

**Bad**: Setup instructions in architecture doc, architecture diagrams in setup guide

**Why bad**: Hard to find information, unclear organization

**Fix**: Separate nouns (architecture) from verbs (setup)

---

## Verification Template

**Before marking documentation complete, use this template**:

```markdown
Task: [Document name written/edited]

**Kinney Principles Checklist**:
✓ Noun-heavy main file: [What/Where focus confirmed]
✓ Length: [X lines, under 150/200 limit]
✓ Modular: [Main file + N @imported files listed]
✓ Token efficient: [Concise statements, code examples used]
✓ Hierarchical: [Correct level: user/project/directory]

**Files Created/Modified**:
- [Main file]: X lines
- [@imported file 1]: Y lines
- [@imported file 2]: Z lines

**Verification**:
- [ ] Main file is scannable (quick read confirms structure)
- [ ] @imports work (no broken references)
- [ ] No duplication across files
- [ ] Clear navigation between files
- [ ] Token-efficient (concise, declarative)

**Evidence**: [Preview of main file structure or line counts]
```

---

## Integration with Other Skills

**This skill works WITH**:

- **`elements-of-style:writing-clearly-and-concisely`** - Use for prose quality
  - Kinney skill: Structure and modularity
  - Elements of style: Sentence clarity and conciseness
  - Use both together for documentation

- **`superpowers:writing-skills`** - Use when creating skill files
  - Kinney skill: Ensure skill is modular and scannable
  - Writing-skills: Test skill with subagents, iterate

**Workflow**:
1. Plan documentation structure (this skill)
2. Write prose (elements-of-style)
3. If creating skill: Test with subagents (writing-skills)

---

## Examples

### Example 1: Refactor Monolithic CLAUDE.md

**Before** (518 lines, monolithic):
```markdown
# CLAUDE.md

[518 lines of mixed content: principles, architecture,
workflow, commands, troubleshooting, examples, all inline]
```

**After** (292 lines, modular):
```markdown
# CLAUDE.md

## Core Principles
[Concise list]

## Architecture & Standards
@docs/standards/architecture.md
@docs/standards/coding-standards.md

## Development Workflow
@docs/guides/DEVELOPMENT-WORKFLOW.md

## Quick References
[Essential commands only]
```

**Result**: 44% reduction (518 → 292 lines), scannable, modular

### Example 2: Modular Spec

**Before** (400 lines in one file):
```markdown
# skill-enforcement-system/SPEC.md (400 lines)
[Everything inline]
```

**After** (modular):
```markdown
# skill-enforcement-system/SPEC.md (150 lines)

## Overview
...

## Detailed Documentation
@./category-skills.md
@./enforcement.md
@./workflows.md
@./implementation-plan.md
```

**Result**: Scannable overview, details on demand

---

## Summary

**Core Rules**:
1. **Nouns in main file** (what/where), verbs in @imports (how/when)
2. **~150 line maximum** for main files
3. **Modular structure** (main + @imported details)
4. **Token efficient** (concise, declarative, code examples)
5. **Hierarchical** (user → project → directory)

**Checklist**:
- [ ] Noun-heavy main file
- [ ] Length ≤150 lines (main file)
- [ ] Modular (@imported details)
- [ ] Token efficient (concise)
- [ ] Correct hierarchical level

**Integration**:
- Use WITH `elements-of-style:writing-clearly-and-concisely`
- Use WITH `superpowers:writing-skills` for skill files

**Remember**: Documentation is infrastructure. Make it scannable, modular, and token-efficient.
