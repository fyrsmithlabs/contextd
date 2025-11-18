---
name: contextd-creating-package
description: Use when creating any new package in pkg/ directory, before writing any package code - enforces proper package creation workflow including name validation, category classification, documentation updates, and verification to prevent duplicates and maintain consistency
---

# Creating Package Workflow

## Overview

**Every new package MUST follow this workflow.** This skill prevents duplicate packages, enforces naming conventions, maintains documentation currency, and ensures category consistency.

**Core principle**: Package creation is not just "mkdir + touch files". It's classification, documentation, pattern enforcement, and verification.

## When to Use This Skill

Use this skill when you see:
- User requests "create pkg/NAME"
- Task involves "new package"
- About to run `mkdir pkg/...`
- Adding new functionality that needs new package
- User says "add package for X"

**CRITICAL**: Use BEFORE creating any files. If files exist, STOP and delete them. Start over with this workflow.

## The Iron Law

```
NO PACKAGE FILES WITHOUT THIS WORKFLOW
```

**Violating the letter of the rules is violating the spirit of the rules.**

**Violations & Counters**:
- Created files before invoking skill? DELETE them. Start over. No "just add docs" - delete and restart.
- "User urgent, skip workflow"? NO. Workflow takes 2 minutes, prevents hours of rework.
- "User says skip the skill"? The skill IS the workflow. Cannot create packages without it.
- "Docs can be added later"? NO. pkg/CLAUDE.md updated NOW, or package is orphaned.
- "Next PR for docs"? "Later" never happens. Documentation is part of creation, not follow-up.
- "I know the patterns"? Use workflow anyway for consistency. Experts use checklists.
- "I'm the architect"? Skills ensure consistency across ALL developers, including experts.

**No exceptions:**
- Not for "simple packages"
- Not for "temporary packages"
- Not for "user is in a hurry"
- Not for "I'll document later"
- Not if user says "skip"
- Not if files already exist
- Not for partial workflow

**If user insists on skipping**: "I cannot create packages without documentation. This violates project standards and creates orphaned packages."

## 6-Step Mandatory Workflow

**Workflow is All-or-Nothing**:
- Cannot execute partial workflow (steps 1-3 without 4-6 = orphaned package)
- Cherry-picking steps defeats the purpose
- All steps required for complete, verified package
- This is like "partial airplane pre-flight check" - doesn't work

**If user requests partial execution**:
"Workflow is atomic operation. All steps required. Partial execution creates incomplete package.

Options:
A) Execute full workflow (2 minutes, recommended)
B) User handles manually (loses enforcement, not recommended)

Cannot do partial - this is non-negotiable for task completion."

### Step 1: Pre-Flight Checks

**BEFORE creating anything, answer these questions:**

```markdown
Pre-Flight Checklist:
- [ ] Does similar package already exist? (check pkg/ directory)
- [ ] Is package name valid? (lowercase, single word, no underscores, no stuttering)
- [ ] Should this extend existing package instead of creating new one?
- [ ] Is this functionality better as internal/ package (not public API)?
```

**Commands to run**:
```bash
# Check for existing packages
ls -la /home/dahendel/projects/contextd/pkg/ | grep -i <name-keyword>

# Check pkg/CLAUDE.md for existing mappings
grep -i "<name-keyword>" /home/dahendel/projects/contextd/pkg/CLAUDE.md
```

**If similar package exists**: MANDATORY review before proceeding.

1. STOP package creation immediately
2. Review existing package (read main file + godoc)
3. Compare proposed vs existing functionality
4. Present analysis to user:
   ```
   Found pkg/<existing>. Comparison:
   - Existing: [what it does]
   - Proposed: [what pkg/<new> would do]
   - Overlap: [estimated %]

   Options:
   A) Extend pkg/<existing> (recommended if >50% overlap)
   B) Create pkg/<new> (explain why separate package needed)

   Please choose and explain reasoning if B.
   ```
5. BLOCK until user provides reasoning for separate package (if choosing B)

**If package name invalid**: STOP. Suggest valid alternative:
- `file_utils` → `fileutil` or `files`
- `CachePackage` → `cache`
- `vector_store` → `vectorstore`
- `utils` → too generic, suggest specific name

### Step 2: Classify Package

**Determine category** (Security/Storage/Core/API/AI):

| Category | Packages | Patterns | Skill to Update |
|----------|----------|----------|-----------------|
| **Security** | auth, session, isolation, rbac | Multi-tenant, input validation, constant-time | contextd:pkg-security |
| **Storage** | checkpoint, remediation, cache, persistence | Qdrant, database-per-project, query security | contextd:pkg-storage |
| **Core** | config, telemetry, logging, health | Standard patterns, error handling, initialization | contextd:pkg-core |
| **API** | mcp, handlers, middleware, routes | Request/response, validation, MCP tools | contextd:pkg-api |
| **AI** | embedding, search, semantic, context | Embeddings, vector ops, AI integrations | contextd:pkg-ai |

**Determine security level**:
- **Critical**: auth, session, rbac (user-facing security)
- **High**: logging (secret redaction), mcp (API exposure), checkpoint (project isolation)
- **Medium**: remediation, config, embedding, search
- **Low**: telemetry, health checks

**Multi-category packages**: If package spans multiple categories (e.g., MCP + Storage), assign primary category and note secondary in pkg/CLAUDE.md.

### Step 3: Create Package Structure

**Now create files** (and ONLY now):

```bash
# Create directory
mkdir -p /home/dahendel/projects/contextd/pkg/<name>

# Create main file
cat > /home/dahendel/projects/contextd/pkg/<name>/<name>.go <<'EOF'
// Package <name> provides [one-sentence purpose].
//
// [Detailed description: what problem it solves, key features]
package <name>

// TODO: Add implementation
EOF

# Create test file
cat > /home/dahendel/projects/contextd/pkg/<name>/<name>_test.go <<'EOF'
package <name>

import "testing"

// TODO: Add tests (TDD - write tests first)
EOF
```

**Package-level godoc** (mandatory):
- First sentence: "Package <name> provides X"
- Second paragraph: Detailed description (what, why, key features)
- Example usage (if public API)

### Step 4: Update pkg/CLAUDE.md

**Add package to mapping table**:

Location: `/home/dahendel/projects/contextd/pkg/CLAUDE.md`

Find the table:
```markdown
## Package-Skill Mapping

| Package | Category | Skill to Invoke | Security Level |
|---------|----------|-----------------|----------------|
```

Add row (alphabetically by package name):
```markdown
| pkg/<name> | <Category> | contextd:pkg-<category> | <Level> |
```

**Example**:
```markdown
| pkg/cache | Storage | contextd:pkg-storage | Medium |
```

### Step 5: Update Category Skill

**Read category skill** (e.g., `.claude/skills/contextd-pkg-storage/SKILL.md`)

**Add package to "Packages in This Category" section**:
```markdown
## Packages in This Category

- **pkg/<name>** - [one-sentence purpose]
```

**If new patterns introduced**: Add to "Common Patterns" section with code example.

**If security-critical**: Add to security checklist (for Security category).

### Step 6: Verify Setup

**Run verification commands**:

```bash
# 1. Package builds
go build /home/dahendel/projects/contextd/pkg/<name>/

# 2. Tests exist and pass
go test /home/dahendel/projects/contextd/pkg/<name>/

# 3. Check pkg/CLAUDE.md updated
grep "pkg/<name>" /home/dahendel/projects/contextd/pkg/CLAUDE.md

# 4. Check category skill updated (if created)
grep "pkg/<name>" /home/dahendel/projects/contextd/.claude/skills/contextd-pkg-<category>/SKILL.md
```

**Verification checklist**:
- [ ] Package builds without errors
- [ ] Test file exists (even if TODO)
- [ ] pkg/CLAUDE.md contains package mapping
- [ ] Category skill updated (package listed)
- [ ] Package name follows Go conventions
- [ ] godoc complete (package-level documentation)

### Step 7: Completion Verification

**Invoke completion skill**: `contextd:completing-major-task`

**Required evidence**:
```markdown
Task: Create pkg/<name> package
Type: Feature
Changes:
  - pkg/<name>/<name>.go (new file, package implementation)
  - pkg/<name>/<name>_test.go (new file, tests)
  - pkg/CLAUDE.md (updated mapping table)
  - .claude/skills/contextd-pkg-<category>/SKILL.md (updated package list)
Verification Evidence:
  ✓ Build: `go build ./pkg/<name>/` - Success
  ✓ Tests: `go test ./pkg/<name>/` - PASS (or TODO marker)
  ✓ Security: [Category security requirements met]
  ✓ Functionality: [Basic structure validated]
Risk Assessment: If workflow skipped, package becomes orphaned (not in pkg/CLAUDE.md), patterns inconsistent, duplicate functionality possible.
```

## Special Cases

### Complex Packages (Needs Spec)

**Trigger**: Package has >1K lines planned OR security-critical OR multi-category

**Action**: Create spec BEFORE implementation:
```bash
mkdir -p /home/dahendel/projects/contextd/docs/specs/<package>/
# Invoke: contextd:creating-spec skill
```

### Extending Existing Package

**If similar package exists**, prefer extending over creating new:

```go
// GOOD - Extend existing package
// pkg/checkpoint/snapshot.go (new file in existing package)

// BAD - Create duplicate package
// pkg/checkpoints/ (conflicts with pkg/checkpoint)
```

### Internal Packages

**If functionality is NOT public API**:
```bash
# Use internal/ instead of pkg/
mkdir -p /home/dahendel/projects/contextd/internal/<name>
```

Update `internal/CLAUDE.md` instead of `pkg/CLAUDE.md` (if it exists).

### Multi-Category Packages

**Example**: pkg/persistence (Storage + API patterns)

**Action**:
1. Assign primary category (Storage)
2. Note secondary in pkg/CLAUDE.md:
   ```markdown
   | pkg/persistence | Storage (+ API) | contextd:pkg-storage | High |
   ```
3. Update BOTH category skills (storage + api)
4. Document cross-category patterns in both skills

## Red Flags - STOP Immediately

**If you hear yourself thinking**:
- "Docs can be added later"
- "Next PR for documentation"
- "User said urgent, skip workflow"
- "User says skip the skill"
- "Workflow is overkill for simple package"
- "I'll update pkg/CLAUDE.md after implementation"
- "Category skill update can wait"
- "User specified exact name, don't validate"
- "Similar package exists but this is different"
- "Already created files, just add docs"
- "User is architect/expert, trust them"
- "Partial workflow is better than none"
- "Process takes too long"

**All of these mean**: STOP. Follow the workflow. It takes 2 minutes and prevents hours of debugging orphaned packages, duplicate functionality, and inconsistent patterns.

## Common Rationalizations (Don't Fall For These)

| Excuse | Reality |
|--------|---------|
| "User urgent, skip workflow" | Workflow takes 2 min. Rework takes hours. |
| "User says skip the skill" | The skill IS the workflow. Cannot skip. |
| "Docs later" | "Later" never happens. Package becomes orphaned. |
| "Next PR for docs" | Documentation is part of creation, not follow-up. 97% of "next PRs" never happen. |
| "I know patterns" | Skills exist for consistency across packages, not just you. |
| "I'm the architect/expert" | Experts use checklists (aviation, surgery, software). Workflow faster for experts, not skipped. |
| "Already created files, just add docs" | Delete and restart. 2 min of work vs hours of wrong patterns. Sunk cost fallacy. |
| "User exact name" | Validate anyway. User may not know Go conventions. |
| "Similar package different enough" | Mandatory review first. Often extending is better than creating duplicate. |
| "One category enough" | Multi-category packages need both skills for consistency. |
| "Workflow overkill" | Workflow prevents duplicate work, not creates it. |
| "Simple package, simple process" | All packages need documentation to be discovered. |
| "Partial workflow OK" | Workflow is all-or-nothing. Partial execution = incomplete package. |

## Failure Modes & Prevention

### Failure: Package created without pkg/CLAUDE.md entry
**Prevention**: Step 4 is mandatory, verification in Step 6 checks for it

### Failure: Category skill not updated
**Prevention**: Step 5 is mandatory, verification checks skill file

### Failure: Invalid package name (underscores, mixed case)
**Prevention**: Step 1 validates name BEFORE file creation

### Failure: Duplicate package (similar one exists)
**Prevention**: Step 1 checks existing packages BEFORE creation

### Failure: Wrong category assigned
**Prevention**: Step 2 classification table guides correct category

### Failure: No verification, broken package committed
**Prevention**: Step 6 builds package, runs tests, checks updates

## Success Criteria

Package creation is complete when:
- ✅ All 6 steps executed in order
- ✅ Package builds without errors
- ✅ pkg/CLAUDE.md updated (verified with grep)
- ✅ Category skill updated (verified with grep)
- ✅ Completion template provided (contextd:completing-major-task)
- ✅ No red flags triggered
- ✅ Pre-flight checks passed

## The Bottom Line

**Creating packages IS following this workflow.**

Not following workflow = orphaned package = wasted work = inconsistent patterns = duplicate functionality.

**2 minutes now saves 2 hours later.**

If you skip steps, delete the package and start over. This is non-negotiable.
