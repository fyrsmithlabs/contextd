# Initialize Go Project Command

Initialize a Go project with language-specific agents and skills for TDD-driven development.

## Usage
```
/init-go-project [project-name]
```

## Arguments
- `[project-name]` - Optional project name (defaults to current directory name)

## What This Command Does

Sets up Go-specific development tooling and AI agents in the current repository:

### 1. Go-Specific Agents

Creates `.claude/agents/` with Go development agents:

**go-architect.md**
- Designs Go service architecture
- Makes technology stack decisions
- Defines package structure and dependencies
- Creates high-level technical designs

**go-engineer.md**
- Implements features using TDD methodology
- Writes tests BEFORE implementation (RED → GREEN → REFACTOR)
- Ensures ≥70% test coverage
- Follows Go coding standards

### 2. Go-Specific Skills

Creates `.claude/skills/` with Go development skills:

**golang-pro.md**
- Mandatory skill for ALL Go code changes
- Enforces TDD workflow
- Ensures test coverage ≥70%
- Runs quality gates (build, test, lint, race detection)
- Creates conventional commits

### 3. Project-Specific CLAUDE.md Updates

Updates `CLAUDE.md` to enforce:
- Mandatory golang-pro skill usage
- Go-specific coding standards
- TDD requirements
- Quality gate enforcement

## Process

### Step 1: Verify Current Directory

```bash
pwd
# Should be in your project root (where CLAUDE.md exists)
```

### Step 2: Create Go-Specific Agents

The command creates `.claude/agents/` with:

```
.claude/agents/
├── go-architect.md    # Architecture & design decisions
└── go-engineer.md     # TDD implementation
```

**Note**: Global agents (code-reviewer, spec-writer, orchestrator, research-analyst, test-engineer)
are available from `~/.claude/agents/` and don't need to be copied.

### Step 3: Create Go-Specific Skill

Creates `.claude/skills/golang-pro.md`:

```markdown
# Golang Pro Skill

**MANDATORY for all Go development tasks.**

## Usage Pattern
```
Use the golang-pro skill to [implement/fix/refactor] [description]
```

## TDD Workflow
1. RED: Write failing tests
2. GREEN: Implement minimal code to pass
3. REFACTOR: Improve code while tests pass

## Quality Requirements
- Coverage ≥70% (≥80% preferred)
- All tests pass
- No race conditions
- Zero linter warnings
```

### Step 4: Update CLAUDE.md

Adds Go-specific section to `CLAUDE.md`:

```markdown
## ⚠️ CRITICAL: Go Code Delegation

**ALL Go coding tasks MUST be delegated to the golang-pro skill.**

```
Use the golang-pro skill to [implement/fix/refactor] [description]
```

**Do NOT write Go code directly.** The golang-pro skill enforces:
- TDD methodology (tests first)
- Test coverage ≥70%
- Quality gates (build, test, lint, race)
- Conventional commits
```

### Step 5: Initialize Go Module (if needed)

```bash
# If go.mod doesn't exist
go mod init github.com/org/project-name
```

### Step 6: Verify Setup

```bash
# Check agents created
ls .claude/agents/
# Should see: go-architect.md, go-engineer.md

# Check skill created
ls .claude/skills/
# Should see: golang-pro.md

# Check CLAUDE.md updated
grep "CRITICAL: Go Code Delegation" CLAUDE.md
```

## Example Workflow

### Example 1: Initialize New Go Project

```bash
cd ~/projects/my-new-service
/init-go-project my-new-service
```

**Output**:
```
═══════════════════════════════════════════════════════
  Go Project Initialization
═══════════════════════════════════════════════════════

Project: my-new-service
Location: /home/user/projects/my-new-service

Creating Go-specific agents...
  ✓ .claude/agents/go-architect.md
  ✓ .claude/agents/go-engineer.md

Creating Go-specific skills...
  ✓ .claude/skills/golang-pro.md

Updating CLAUDE.md...
  ✓ Added Go delegation requirements

Initializing Go module...
  ✓ go.mod created: github.com/org/my-new-service

Setup complete!

Global agents available from ~/.claude/agents/:
  ✓ code-reviewer (language-agnostic)
  ✓ spec-writer (language-agnostic)
  ✓ orchestrator (workflow coordination)
  ✓ research-analyst (error research)
  ✓ test-engineer (language-agnostic)

Next steps:
  1. Create your first spec: /spec-writer <feature> "<description>"
  2. Convert to issues: /spec-to-issue <feature>
  3. Start coding: /start-task <issue-number>
  4. Implement: Use the golang-pro skill to implement <feature>
```

### Example 2: Add Go Support to Existing Project

```bash
cd ~/projects/existing-project
/init-go-project
```

**What Happens**:
- Detects existing `go.mod` (skips module initialization)
- Creates Go agents and skills
- Updates CLAUDE.md
- Preserves existing project structure

## Files Created

### .claude/agents/go-architect.md (13KB)
```markdown
# Go Architect Agent

Expert Go architect specializing in service design, package structure,
and technology stack decisions.

## Responsibilities
- Design service architecture
- Define package boundaries
- Select dependencies
- Create technical specifications
```

### .claude/agents/go-engineer.md (17KB)
```markdown
# Go Engineer Agent

Expert Go developer specializing in TDD implementation with comprehensive
test coverage and quality enforcement.

## Responsibilities
- Write tests FIRST (TDD)
- Implement minimal code
- Ensure ≥70% coverage
- Follow coding standards
```

### .claude/skills/golang-pro.md (9KB)
```markdown
# Golang Pro Skill

Mandatory skill for all Go development tasks.

## TDD Workflow
1. RED: Write failing tests
2. GREEN: Minimal implementation
3. REFACTOR: Improve code

## Quality Gates
- Build succeeds
- Tests pass
- Coverage ≥70%
- No race conditions
- Linters clean
```

## Integration with Existing Workflows

### Works With

**Global Commands** (from ~/.claude/commands/):
- ✅ `/create-repo` - Create new repository
- ✅ All other global commands available

**Template Commands**:
- ✅ `/configure-repo` - Repository setup
- ✅ `/spec-writer` - Create specifications
- ✅ `/spec-to-issue` - Generate issues
- ✅ `/start-task` - TDD workflow
- ✅ `/run-quality-gates` - Quality checks

**Global Agents** (from ~/.claude/agents/):
- ✅ code-reviewer - Code review
- ✅ spec-writer - Specifications
- ✅ orchestrator - Workflow coordination
- ✅ research-analyst - Error research
- ✅ test-engineer - Testing verification

### Complete Setup Flow

```bash
# 1. Create repository (global command)
/create-repo my-go-service "A new Go microservice"

# 2. Navigate to repository
cd my-go-service

# 3. Initialize Go project (this command)
/init-go-project my-go-service

# 4. Configure repository
/configure-repo phase-1

# 5. Create first spec (uses global spec-writer agent)
/spec-writer authentication "JWT-based authentication"

# 6. Convert to issues
/spec-to-issue authentication

# 7. Start development (uses golang-pro skill)
/start-task <issue-number>
Use the golang-pro skill to implement authentication
```

## Language-Specific Variants

This command is for **Go projects only**. For other languages, create similar commands:

- `/init-python-project` - Python with pytest, mypy, black
- `/init-rust-project` - Rust with cargo, clippy
- `/init-node-project` - Node.js with jest, eslint
- `/init-java-project` - Java with Maven/Gradle, JUnit

Each would create language-specific agents/skills while reusing global agents.

## Benefits

### 1. Template Flexibility
- Template is language-agnostic
- Can be used for any project type
- Language-specific setup is opt-in

### 2. Reduced Template Size
- Global agents not duplicated in every project
- Only language-specific agents in project
- Smaller repository clones

### 3. Consistent Global Agents
- code-reviewer works for all languages
- spec-writer works for all projects
- Updates to global agents benefit all projects

### 4. Language-Specific Enforcement
- golang-pro enforces Go best practices
- python-pro would enforce Python best practices
- Each language gets proper tooling

## Troubleshooting

### Command Not Found

**Error**: `/init-go-project: command not found`

**Solution**:
```bash
# Verify you're in a project created from git-template
ls .claude/commands/init-go-project.md

# If missing, update from template
/update-from-template
```

### Go Module Already Exists

**Message**: `go.mod already exists, skipping module initialization`

**This is normal!** The command detects existing `go.mod` and preserves it.

### Agents Already Exist

**Message**: `.claude/agents/go-architect.md already exists, skipping`

**This is safe!** The command won't overwrite existing agents. To reset:
```bash
rm .claude/agents/go-*.md
/init-go-project
```

## Related Commands

- `/create-repo` - Create new repository (global)
- `/configure-repo` - Repository configuration
- `/spec-writer` - Create specifications (uses global agent)
- `/start-task` - Start TDD workflow (uses golang-pro)

## Prerequisites

- Git repository initialized
- CLAUDE.md exists (from template)
- `.claude/` directory structure exists

## Notes

- **Global agents** are in `~/.claude/agents/` (code-reviewer, spec-writer, orchestrator, research-analyst, test-engineer)
- **Language-specific agents** are in `.claude/agents/` (go-architect, go-engineer)
- **Language-specific skills** are in `.claude/skills/` (golang-pro)
- This separation keeps projects language-agnostic until initialized

---

**Pro Tip**: Run this command immediately after `/create-repo` for Go projects.
