---
name: init
description: Initialize contextd for a project repository. Creates CLAUDE.md, indexes the repo, and optionally extracts learnings from past conversations.
arguments:
  - name: flags
    description: "Flags: --full (analyze codebase), --conversations (index past conversations), --batch (offline processing), --skip-claude-md"
    required: false
---

Initialize contextd for a project repository.

## Flags

| Flag | Description |
|------|-------------|
| `--full` | Run full onboarding: analyze codebase, generate CLAUDE.md, index conversations |
| `--conversations` | Also index past Claude Code conversations (with `--full`) |
| `--batch` | Process offline via `ctxd onboard` CLI (no context cost) |
| `--skip-claude-md` | Skip CLAUDE.md generation (with `--full`) |

## Quick Init (Default)

For new projects starting from scratch:

### Detection Phase

Check project status:
1. Does CLAUDE.md exist in project root?
2. Does `mcp__contextd__checkpoint_list` return any data for this project?
3. Are there source code files?

**If existing project detected:** Suggest `--full` flag for comprehensive onboarding.

### Mini Brainstorm (1-2 questions only)

Ask the user:
1. **"What does this project do?"** (one sentence description)
2. **"Any critical conventions I should know?"** (optional - skip if they say no)

### Generate Starter CLAUDE.md

Use the `project-setup` skill to create a scaffolded CLAUDE.md:

```markdown
# CLAUDE.md - [Project Name]

**Status**: Active Development
**Last Updated**: [Today's Date]

---

## Critical Rules

**ALWAYS** [placeholder - user fills in]
**NEVER** [placeholder - user fills in]

---

## Project Overview
[User's one-sentence description]

## Architecture
<!-- Add key components and their relationships -->

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| | | |

## Commands

| Command | Purpose |
|---------|---------|
| | |

## Code Standards
[User's conventions if provided, otherwise placeholder]

## Known Pitfalls
<!-- Document gotchas as you discover them -->

## ADRs (Architectural Decisions)
<!-- Format: ADR-NNN: Title, Status, Context, Decision, Consequences -->
```

### Initial Setup

After creating CLAUDE.md:

1. **Index repository:**
   ```
   mcp__contextd__repository_index(path: ".")
   ```

2. **Record initialization memory:**
   ```
   mcp__contextd__memory_record(
     project_id: "<derived from git remote or directory>",
     title: "Project initialized",
     content: "Initialized new project with starter CLAUDE.md",
     outcome: "success",
     tags: ["init", "new-project"]
   )
   ```

3. **Confirm:** "Project initialized. CLAUDE.md created and repository indexed."

---

## Full Onboarding (--full flag)

For existing projects with source code. Analyzes codebase and generates comprehensive CLAUDE.md.

**Batch Mode**: When `--batch` is specified, output a command for the user to run:
```
Run this command to index conversations offline:
  ctxd onboard --conversations --project=/path/to/project

This processes conversations without using agent context.
Results will be available in contextd on next session.
```

### Phase 1: Discovery

Run these discovery commands (do NOT ask user - investigate first):

```bash
# Repository structure
find . -type d -name node_modules -prune -o -name .git -prune -o -type d -print | head -50

# Configuration files
ls -la *.json *.yaml *.toml Makefile go.mod requirements.txt Cargo.toml 2>/dev/null

# Entry points
ls -la cmd/ src/ main.* index.* app.* 2>/dev/null

# CI/CD
ls -la .github/workflows/ .gitlab-ci.yml Jenkinsfile 2>/dev/null
```

### Phase 2: Pattern Extraction

| Target | How to Find |
|--------|-------------|
| Language | go.mod, package.json, requirements.txt, Cargo.toml |
| Framework | Dependencies in lockfiles |
| Architecture | Directory structure patterns |
| Commands | Makefile, package.json scripts, README |
| Tests | *_test.*, *.spec.*, __tests__/ |
| Linting | .eslintrc, .golangci.yml, prettier config |

### Phase 3: Generate CLAUDE.md

Use `project-setup` skill structure with DISCOVERED information:

1. **Status** - Based on commit frequency (Active/Maintenance)
2. **Critical Rules** - Extracted from linter configs, pre-commit hooks
3. **Architecture** - Directory tree with purposes (from analysis)
4. **Tech Stack** - Exact versions from lockfiles
5. **Commands** - From Makefile/package.json with purposes
6. **Code Standards** - From config files
7. **Known Pitfalls** - From TODOs, FIXMEs found in codebase

### Phase 4: Verification

Before presenting CLAUDE.md:
1. Verify at least one command works (e.g., `npm run build`, `go build ./...`)
2. Check that paths in architecture section exist

### Phase 5: Index and Record

```
mcp__contextd__repository_index(path: ".")

mcp__contextd__memory_record(
  project_id: "<derived from git remote>",
  title: "Project onboarded",
  content: "Analyzed existing codebase, generated CLAUDE.md with [key findings]",
  outcome: "success",
  tags: ["onboard", "existing-project"]
)
```

---

## Conversation Indexing (--conversations with --full)

When `--conversations` flag is provided, also index past Claude Code conversations.

### Find Conversations

```bash
# Encode project path for ~/.claude/projects/ lookup
PROJECT_PATH=$(pwd)
ENCODED_PATH=$(echo "$PROJECT_PATH" | tr '/' '-')

# Find conversation files
ls ~/.claude/projects/${ENCODED_PATH}/*.jsonl 2>/dev/null
```

### Context Warning

If conversations found, display warning:

```
WARNING: Conversation indexing uses significant context.

    Found: 15 conversations for this project
    Estimated tokens: ~750k total

    Options:
    [1] Continue with context folding (recommended for <10 conversations)
    [2] Switch to batch mode (process offline, no context cost)
    [3] Index specific conversations only
    [4] Skip conversation indexing

    Choice: _
```

### Extract Learnings

For each conversation file:

**Step 1: Scrub secrets FIRST**
```
# Read and scrub before any processing
content = Read(conversation_file)
scrubbed = POST http://localhost:9090/api/v1/scrub {"content": content}
# Verify scrubbing succeeded before proceeding
```

**Step 2: Extract and store remediations (error -> fix patterns)**
```
# For each error/fix pair found:
mcp__contextd__remediation_record(
  title: "ENOENT when reading config",
  problem: "Error: ENOENT: no such file or directory",
  root_cause: "Relative path used instead of absolute",
  solution: "Use path.resolve() before file operations",
  category: "runtime",
  tenant_id: "user",
  scope: "project",
  tags: ["nodejs", "filesystem"]
)
```

**Step 3: Extract and store memories (learnings)**
```
mcp__contextd__memory_record(
  project_id: "{project}",
  title: "Always use absolute paths for file ops",
  content: "Relative paths break when cwd changes. Use path.resolve().",
  outcome: "success",
  tags: ["learning", "extracted", "nodejs"]
)
```

### Deduplicate

Before storing, check for existing similar entries:

```
# Check for duplicate remediations
remediation_search(query: "{problem summary}", tenant_id: "user")

# Check for duplicate memories
memory_search(project_id: "{project}", query: "{learning summary}")
```

---

## Present to User

Show the generated CLAUDE.md and ask:
"I've analyzed the codebase and generated this CLAUDE.md. Want me to write it, or would you like to adjust anything first?"

If `--conversations` was used, also show:
```
Conversation Indexing Results:
- 5 remediations extracted
- 12 memories recorded

Top findings:
1. REMEDIATION: "ENOENT errors -> use path.resolve()"
2. MEMORY: "Use context folding for large tasks"
```

---

## Error Handling

@_error-handling.md

**Init-specific errors:**

| Error | Resolution |
|-------|------------|
| Codebase too complex | Break into sections, analyze incrementally |
| Conversation parsing fails | Log error, continue with remaining files |
| CLAUDE.md already exists | Ask user if they want to regenerate or enhance |
