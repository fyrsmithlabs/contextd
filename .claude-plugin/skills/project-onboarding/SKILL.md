---
name: project-onboarding
description: Use when onboarding to an existing project without CLAUDE.md - analyzes codebase structure, identifies patterns, tech stack, and generates comprehensive CLAUDE.md following Kinney best practices
---

# Project Onboarding

## Overview

Systematic analysis of existing codebases to generate comprehensive CLAUDE.md documentation. Extracts architecture, patterns, commands, and pitfalls from code rather than asking the user.

**Core principle:** Investigate first, document findings. Don't ask what you can discover.

## When to Use

- Joining existing project without CLAUDE.md
- Project has outdated/incomplete CLAUDE.md
- Taking over maintenance of unfamiliar codebase
- Preparing codebase for AI-assisted development

**Use `/init` instead for:** Brand new projects starting from scratch.

## Onboarding Workflow

### Phase 1: Discovery

```
1. Repository structure scan (Glob)
2. Package/dependency analysis (package.json, go.mod, requirements.txt, etc.)
3. Configuration files (tsconfig, .eslintrc, Makefile, etc.)
4. Existing documentation (README, docs/, CONTRIBUTING)
5. CI/CD configuration (.github/workflows, Jenkinsfile, etc.)
6. Environment setup (.env.example, docker-compose)
```

### Phase 2: Pattern Extraction

| Target | Search Strategy |
|--------|-----------------|
| Entry points | main.*, index.*, cmd/, src/ |
| Architecture | Directory structure, imports graph |
| Error handling | grep: try/catch, Result<, errors.New |
| Testing | *_test.*, *.spec.*, jest.config |
| Build commands | Makefile, package.json scripts |
| Code style | .editorconfig, prettier, eslint |

### Phase 3: CLAUDE.md Generation

Apply **writing-claude-md** skill structure:

1. **Status header** - Active/Maintenance based on commit frequency
2. **Critical Rules** - Extracted from linters, pre-commit hooks
3. **Architecture** - Directory tree with purposes
4. **Tech Stack** - Exact versions from lockfiles
5. **Commands** - From Makefile/package.json with purposes
6. **Code Standards** - From config files
7. **Known Pitfalls** - From TODOs, FIXMEs, issue tracker patterns

## Discovery Commands

### Universal
```bash
# Directory structure
find . -type d -name node_modules -prune -o -type d -print | head -50

# Configuration files
find . -maxdepth 2 -name "*.json" -o -name "*.yaml" -o -name "*.toml" -o -name "Makefile"

# Entry points
ls -la cmd/ src/ main.* index.* 2>/dev/null
```

### Language-Specific

| Language | Key Files |
|----------|-----------|
| Go | go.mod, go.sum, cmd/, internal/, Makefile |
| Node.js | package.json, tsconfig.json, .nvmrc |
| Python | pyproject.toml, requirements.txt, setup.py |
| Rust | Cargo.toml, Cargo.lock, src/main.rs |

## Pattern Detection

### Architecture Patterns

| Pattern | Indicators |
|---------|------------|
| Monorepo | packages/, apps/, lerna.json, nx.json |
| Microservices | services/, docker-compose.yml, k8s/ |
| Layered | controllers/, services/, repositories/ |
| Clean/Hex | domain/, adapters/, ports/ |
| MVC | models/, views/, controllers/ |

### Testing Patterns

| Pattern | Indicators |
|---------|------------|
| Unit tests | *_test.*, *.spec.*, __tests__/ |
| Integration | test/integration/, e2e/ |
| Coverage | .nycrc, jest --coverage, go test -cover |

## Critical Rules Extraction

```bash
# Pre-commit hooks
cat .pre-commit-config.yaml 2>/dev/null

# Linter configs
cat .eslintrc* .golangci.yml rustfmt.toml 2>/dev/null

# CI checks
cat .github/workflows/*.yml | grep -A5 "run:"
```

Transform linter rules into ALWAYS/NEVER constraints.

## Common Discoveries to Document

| Discovery | CLAUDE.md Section |
|-----------|-------------------|
| Database schemas | Architecture |
| API endpoints | Architecture or separate @api.md |
| Environment vars | Commands (setup) |
| Build artifacts | Commands (build) |
| Test fixtures | Code Standards |
| Legacy code warnings | Known Pitfalls |

## Anti-Patterns

| Mistake | Prevention |
|---------|------------|
| Asking user "what framework?" | Check package.json, go.mod first |
| Generic architecture description | Run actual discovery commands |
| Missing version numbers | Extract from lockfiles |
| Skipping CI/CD analysis | Always check .github/workflows |

## Onboarding Checklist

- [ ] Clone/access repository
- [ ] Run Phase 1 discovery
- [ ] Run Phase 2 pattern extraction
- [ ] Generate CLAUDE.md draft
- [ ] Verify commands work (`npm run build`, `make test`, etc.)
- [ ] Present draft to user for validation
- [ ] Record memory with project context

## Quick Reference

| Step | Action |
|------|--------|
| 1 | Scan repo structure |
| 2 | Read package/config files |
| 3 | Extract patterns |
| 4 | Generate CLAUDE.md |
| 5 | Verify commands |
| 6 | User validation |

## Related Skills

- **writing-claude-md** - Structure and formatting for generated CLAUDE.md
- **contextd:cross-session-memory** - Record onboarding findings for future sessions
