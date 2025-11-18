# Parallel Execution Plan: Refactor SPEC Files (Kinney Principles)

**Date**: 2025-11-18
**Method**: Parallel task execution with subagent-driven development
**Goal**: Refactor 8 massive spec files to follow Kinney documentation principles

---

## Kinney Principles (MANDATORY)

**Core Rules**:
1. **Main file ~150 lines** (max 200 acceptable)
2. **Noun-heavy main** (what/where things are)
3. **Verb-heavy details** (how/when to do things) → @imports
4. **Modular structure** (scannable overview + on-demand details)
5. **Token-efficient** (load only what's needed)

**Main SPEC.md Template** (~150 lines):
```markdown
# Feature: [Name]

**Version**: X.Y.Z
**Status**: [Draft/Approved/Implemented/Deprecated]
**Last Updated**: YYYY-MM-DD

---

## Overview

[Brief description - what it is, why it exists]

## Quick Reference

**Key Facts**:
- Technology: [tech stack]
- Location: [where it lives]
- Dependencies: [what it needs]
- Status: [current state]

**Components**:
- [List of major components]

---

## Detailed Documentation

**Requirements & Design**:
@./[spec-name]/requirements.md - Functional & non-functional requirements
@./[spec-name]/architecture.md - System design & component interactions

**Implementation**:
@./[spec-name]/implementation.md - Phase-by-phase implementation plan
@./[spec-name]/workflows.md - Example workflows & usage patterns

**Additional**:
@./[spec-name]/api-reference.md - API endpoints, formats, error codes (if applicable)
@./[spec-name]/security.md - Security considerations (if security-critical)

---

## Summary

[Wrap-up, current status, next steps]
```

---

## Target Files (8 Specs)

### Priority 1: Auth & MCP (Security-Critical)

**1. docs/specs/auth/SPEC.md** (1,594 lines)
- **Extract to**: `docs/specs/auth/`
  - `requirements.md` - Auth requirements
  - `architecture.md` - Token generation, validation, storage
  - `implementation.md` - Implementation phases
  - `security.md` - Constant-time comparison, security model
  - `workflows.md` - Authentication workflows

**2. docs/specs/mcp/SPEC.md** (1,439 lines)
- **Extract to**: `docs/specs/mcp/`
  - `requirements.md` - MCP requirements
  - `architecture.md` - Transport, session management, JSON-RPC
  - `tools.md` - MCP tool definitions
  - `workflows.md` - Lifecycle, initialization sequence
  - `implementation.md` - Current status, refactoring plan

### Priority 2: Core Features

**3. docs/specs/troubleshooting/SPEC.md** (1,421 lines)
- **Extract to**: `docs/specs/troubleshooting/`
  - `requirements.md` - Troubleshooting requirements
  - `architecture.md` - AI-powered diagnosis system
  - `patterns.md` - Common error patterns
  - `workflows.md` - Usage examples
  - `implementation.md` - Current implementation

**4. docs/specs/indexing/SPEC.md** (1,382 lines)
- **Extract to**: `docs/specs/indexing/`
  - `requirements.md` - Indexing requirements
  - `architecture.md` - Repository indexing system
  - `implementation.md` - AST parsing, code analysis
  - `workflows.md` - Indexing workflows

**5. docs/specs/checkpoint/SPEC.md** (1,163 lines)
- **Extract to**: `docs/specs/checkpoint/`
  - `requirements.md` - Checkpoint requirements
  - `architecture.md` - Storage, search, multi-tenant isolation
  - `workflows.md` - Save, search, restore workflows
  - `api-reference.md` - API endpoints
  - `implementation.md` - Current implementation

**6. docs/specs/remediation/SPEC.md** (1,029 lines)
- **Extract to**: `docs/specs/remediation/`
  - `requirements.md` - Remediation requirements
  - `architecture.md` - Error solution storage/search
  - `workflows.md` - Save remediation, search patterns
  - `implementation.md` - Hybrid search (semantic + string matching)

### Priority 3: Infrastructure

**7. docs/specs/multi-tenant/SPEC.md** (966 lines)
- **Extract to**: `docs/specs/multi-tenant/`
  - `requirements.md` - Isolation requirements
  - `architecture.md` - Database-per-project design
  - `security.md` - Filter injection prevention, boundaries
  - `implementation.md` - Current implementation
  - `workflows.md` - Project hash generation, database routing

**8. docs/specs/config/SPEC.md** (893 lines)
- **Extract to**: `docs/specs/config/`
  - `requirements.md` - Configuration requirements
  - `architecture.md` - Environment variables, defaults
  - `reference.md` - Complete config reference table
  - `workflows.md` - Configuration examples

---

## Refactoring Process (Per Spec)

### Step 1: Read Original Spec
Read the entire original SPEC.md file to understand structure.

### Step 2: Create Directory
```bash
mkdir -p docs/specs/[spec-name]/
```

### Step 3: Extract Sections to Separate Files

**Identify major sections**:
- Overview/motivation → Keep in main
- Requirements → Extract to `requirements.md`
- Architecture/design → Extract to `architecture.md`
- Implementation details → Extract to `implementation.md`
- Workflows/examples → Extract to `workflows.md`
- API reference → Extract to `api-reference.md` (if exists)
- Security → Extract to `security.md` (if security-critical)

**Create extracted files** with proper headers:
```markdown
# [Section Name]

**Parent**: [../SPEC.md](../SPEC.md)

[Content from original spec]
```

### Step 4: Create Scannable Main SPEC.md

**Structure** (~150 lines):
```markdown
# Feature: [Name]

**Version**: X.Y.Z
**Status**: [Status]
**Last Updated**: YYYY-MM-DD

## Overview
[Brief description]

## Quick Reference
[Key facts, components]

## Detailed Documentation
@./[spec-name]/requirements.md
@./[spec-name]/architecture.md
@./[spec-name]/implementation.md
@./[spec-name]/workflows.md

## Summary
[Wrap-up]
```

### Step 5: Verify Line Count
```bash
wc -l docs/specs/[spec-name]/SPEC.md
# Target: ~150 lines (max 200 acceptable)
```

### Step 6: Commit
```bash
git add docs/specs/[spec-name]/
git commit -m "refactor(docs): modularize [spec-name] SPEC following Kinney principles

Reduced main file from [OLD] to [NEW] lines ([X]% reduction)

Created modular structure:
- [spec-name]/requirements.md
- [spec-name]/architecture.md
- [spec-name]/implementation.md
- [spec-name]/workflows.md

Main SPEC.md now scannable with @imports for details.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Parallel Execution Strategy

**Deploy 8 parallel agents** (one per spec file):

```
Agent 1: Refactor docs/specs/auth/SPEC.md
Agent 2: Refactor docs/specs/mcp/SPEC.md
Agent 3: Refactor docs/specs/troubleshooting/SPEC.md
Agent 4: Refactor docs/specs/indexing/SPEC.md
Agent 5: Refactor docs/specs/checkpoint/SPEC.md
Agent 6: Refactor docs/specs/remediation/SPEC.md
Agent 7: Refactor docs/specs/multi-tenant/SPEC.md
Agent 8: Refactor docs/specs/config/SPEC.md
```

**Each agent**:
1. Reads original spec
2. Creates directory structure
3. Extracts sections to separate files
4. Creates scannable main SPEC.md with @imports
5. Verifies line count (~150 lines)
6. Commits changes

**Coordination**: All agents work independently (no file conflicts)

---

## Success Criteria

**Per Spec**:
- [ ] Main SPEC.md ≤200 lines (target: ~150)
- [ ] Scannable overview (quick orientation)
- [ ] @imports work correctly (no broken references)
- [ ] Noun-heavy main (what/where)
- [ ] Verb-heavy details (how/when) in @imported files
- [ ] Committed with proper message

**Overall**:
- [ ] All 8 specs refactored
- [ ] Total reduction: ~7,000+ lines → ~1,200 main files + modular details
- [ ] All @imports verified
- [ ] Documentation now follows Kinney principles

---

## Prompt for Parallel Execution

**To execute this plan in parallel**:

```
I need to refactor 8 massive spec files to follow Kinney documentation principles
(scannable ~150 line main files with @imports for details).

Please deploy 8 parallel task-executor agents to refactor these specs:

1. docs/specs/auth/SPEC.md (1,594 lines)
2. docs/specs/mcp/SPEC.md (1,439 lines)
3. docs/specs/troubleshooting/SPEC.md (1,421 lines)
4. docs/specs/indexing/SPEC.md (1,382 lines)
5. docs/specs/checkpoint/SPEC.md (1,163 lines)
6. docs/specs/remediation/SPEC.md (1,029 lines)
7. docs/specs/multi-tenant/SPEC.md (966 lines)
8. docs/specs/config/SPEC.md (893 lines)

Each agent should:
1. Read the plan: docs/plans/2025-11-18-refactor-specs-kinney.md
2. Follow the Kinney principles (main ~150 lines, @imports for details)
3. Create modular structure for their assigned spec
4. Commit the refactoring

Use parallel execution to complete all 8 simultaneously.
```

---

## Notes

- **Context efficiency**: This refactoring reduces token usage when loading specs
- **Scannability**: Main files provide quick orientation, details on demand
- **Maintainability**: Easier to update specific sections without touching entire spec
- **Consistency**: All specs follow same modular structure

**Estimated time**: 15-20 minutes with parallel execution (vs 2+ hours sequential)
