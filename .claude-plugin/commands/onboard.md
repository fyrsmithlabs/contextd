Onboard an EXISTING project to contextd by analyzing codebase and generating CLAUDE.md.

**Use `/init` instead for brand new projects starting from scratch.**

## Flags

| Flag | Description |
|------|-------------|
| `--conversations` | Index past Claude Code conversations for this project |
| `--batch` | Process offline via `ctxd onboard` CLI (no context cost, runs outside agent) |
| `--file={uuid}` | Index specific conversation file only |
| `--skip-claude-md` | Skip CLAUDE.md generation, only do indexing |

**Batch Mode**: When `--batch` is specified, the agent outputs a command for the user to run:
```
Run this command to index conversations offline:
  ctxd onboard --conversations --project=/path/to/project

This processes conversations without using agent context.
Results will be available in contextd on next session.
```

## Detection Phase

Check project status:
1. Does CLAUDE.md already exist?
2. Does the project have source code files?
3. Are there Claude Code conversations to index?

**If CLAUDE.md exists:** Ask user if they want to regenerate or enhance it.
**If no source code:** Inform user to use `/init` for new projects.
**If --conversations:** Also scan `~/.claude/projects/` for past sessions.

---

## Onboarding Workflow

Use the `project-onboarding` skill to systematically analyze the codebase.

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

Use `writing-claude-md` skill structure with DISCOVERED information:

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

## Conversation Indexing (--conversations)

When `--conversations` flag is provided, also index past Claude Code conversations.

### Phase 6: Find Conversations

```bash
# Encode project path for ~/.claude/projects/ lookup
PROJECT_PATH=$(pwd)
ENCODED_PATH=$(echo "$PROJECT_PATH" | tr '/' '-')

# Find conversation files
ls ~/.claude/projects/${ENCODED_PATH}/*.jsonl 2>/dev/null
```

### Phase 7: Context Warning

If conversations found, display warning:

```
⚠️  WARNING: Conversation indexing uses significant context.

    Found: 15 conversations for this project
    Estimated tokens: ~750k total

    Options:
    [1] Continue with context folding (recommended for <10 conversations)
    [2] Switch to batch mode (process offline, no context cost)
    [3] Index specific conversations only
    [4] Skip conversation indexing

    Choice: _
```

Use `conversation-indexing` skill for extraction.

### Phase 8: Extract Learnings

For each conversation file:

**Step 1: Scrub secrets FIRST**
```
# Read and scrub before any processing
content = Read(conversation_file)
scrubbed = POST http://localhost:9090/api/v1/scrub {"content": content}
# Verify scrubbing succeeded before proceeding
```

**Step 2: Extract and store remediations (error → fix patterns)**
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

**Step 4: Extract and store policies (user corrections)**
```
mcp__contextd__memory_record(
  project_id: "global",
  title: "POLICY: verify-file-exists",
  content: "RULE: Check if file exists before reading.\nDESCRIPTION: User corrected agent for assuming files exist.\nCATEGORY: verification\nSEVERITY: medium\nSCOPE: global",
  outcome: "success",
  tags: ["type:policy", "category:verification", "severity:medium", "scope:global", "enabled:true"]
)
```

### Phase 9: Deduplicate

Before storing, check for existing similar entries:

```
# Check for duplicate remediations
remediation_search(query: "{problem summary}", tenant_id: "user")

# Check for duplicate memories
memory_search(project_id: "{project}", query: "{learning summary}")

# Check for duplicate policies
memory_search(project_id: "global", query: "type:policy {rule summary}")
```

### Phase 10: Store Index Record

Record which files were indexed:

```json
{
  "project_id": "{project}",
  "title": "Conversation index: {file}",
  "content": "SHA256: {hash}\nExtractions: {count} remediations, {count} memories, {count} policies",
  "outcome": "success",
  "tags": ["type:index-record", "file:{filename}"]
}
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
- 3 policies created

Top findings:
1. POLICY: test-before-fix - "Always run tests before claiming fix"
2. REMEDIATION: "ENOENT errors → use path.resolve()"
3. MEMORY: "Use context folding for large tasks"
```

---

## Error Handling

If contextd unavailable:
1. Check server: `curl -s http://localhost:9090/health`
   Expected: `{"status":"ok"}`
   If different or no response: contextd is not running
2. Show: "contextd server not responding. Start with `contextd serve`."

If codebase is too complex:
- Break into sections, analyze incrementally
- Focus on most critical paths first

If conversation parsing fails:
- Log error with file path
- Continue with remaining files
- Show partial results
