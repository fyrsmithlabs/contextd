Onboard an EXISTING project to contextd by analyzing codebase and generating CLAUDE.md.

**Use `/init` instead for brand new projects starting from scratch.**

## Detection Phase

Check project status:
1. Does CLAUDE.md already exist?
2. Does the project have source code files?

**If CLAUDE.md exists:** Ask user if they want to regenerate or enhance it.
**If no source code:** Inform user to use `/init` for new projects.

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

## Present to User

Show the generated CLAUDE.md and ask:
"I've analyzed the codebase and generated this CLAUDE.md. Want me to write it, or would you like to adjust anything first?"

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
