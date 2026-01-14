# contextd Examples

Production-ready code examples demonstrating common contextd use cases and workflows.

## Overview

This directory contains standalone, executable examples that show how to use contextd's core features. Each example is self-contained with a detailed README and working Go code.

**All examples run locally with zero external dependencies** (uses embedded chromem + FastEmbed).

## Available Examples

### Core Workflows

| Example | Description | Key Tools |
|---------|-------------|-----------|
| **[session-lifecycle](./session-lifecycle/)** | Complete session pattern: search memories → perform task → record results | `memory_search`, `memory_record`, `memory_feedback`, `memory_outcome` |
| **[checkpoints](./checkpoints/)** | Context snapshot management: save, list, and resume from checkpoints | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` |
| **[remediation](./remediation/)** | Error pattern tracking: record fixes and search for similar past errors | `remediation_record`, `remediation_search` |
| **[repository-indexing](./repository-indexing/)** | Semantic code search with automatic grep fallback | `repository_index`, `semantic_search`, `repository_search` |
| **[context-folding](./context-folding/)** | Isolated subtask execution with token budgets and secret scrubbing | `branch_create`, `branch_status`, `branch_return` |

### Configuration

| Example | Description |
|---------|-------------|
| **[qdrant-config](./qdrant-config/)** | Production Qdrant vector store configurations (dev, prod, large-repos) |

## Quick Start

### 1. Choose Your Use Case

Pick the example that matches your needs:

- **First time user?** Start with [session-lifecycle](./session-lifecycle/) to see the basic memory workflow
- **Need to save context?** See [checkpoints](./checkpoints/) for snapshot management
- **Debugging errors?** Check [remediation](./remediation/) for error pattern reuse
- **Searching code?** Try [repository-indexing](./repository-indexing/) for semantic search
- **Managing token budgets?** Explore [context-folding](./context-folding/) for isolated subtasks
- **Production deployment?** Review [qdrant-config](./qdrant-config/) for external vector storage

### 2. Run an Example

Each example is a standalone Go program:

```bash
# Navigate to example directory
cd examples/session-lifecycle

# Run the example
go run main.go
```

Or build and run:

```bash
cd examples/session-lifecycle
go build -o demo
./demo
```

### 3. Customize for Your Use Case

All examples include:
- **README.md** - Detailed explanation with use cases and troubleshooting
- **main.go** - Working code with extensive comments
- **Inline documentation** - Clear explanations of each step

Copy and modify the example code for your specific needs.

## Architecture

### Local-First Design

All examples run with contextd's embedded stack:

```
┌─────────────────────────────────────┐
│     Your Go Application             │
│  (examples/session-lifecycle, etc)  │
└──────────────┬──────────────────────┘
               │ MCP Protocol (stdio)
┌──────────────▼──────────────────────┐
│          contextd Server            │
│                                     │
│  ┌──────────────────────────────┐  │
│  │   chromem (embedded)         │  │
│  │   In-process vector DB       │  │
│  └──────────────────────────────┘  │
│                                     │
│  ┌──────────────────────────────┐  │
│  │   FastEmbed (local ONNX)     │  │
│  │   No API calls               │  │
│  └──────────────────────────────┘  │
└─────────────────────────────────────┘
```

**Zero external dependencies. No API keys. No network calls.**

### Optional: Production Setup

For production deployments, see [qdrant-config](./qdrant-config/) to use external Qdrant:

```bash
# Set environment variable
export VECTORSTORE_PROVIDER=qdrant

# Run example with Qdrant config
contextd --config examples/qdrant-config/prod.yaml
```

## Use Case Guide

### When to Use Each Example

#### Session Lifecycle

**You need this when:**
- Building AI agents that learn from past experiences
- Implementing "memory" for Claude Code workflows
- Tracking which strategies worked/failed across sessions

**Real-world scenarios:**
- Agent remembers "npm install fails in Docker, use yarn instead"
- Storing "User prefers TypeScript over JavaScript"
- Recording "API rate limits resolved by exponential backoff"

**See:** [session-lifecycle](./session-lifecycle/)

---

#### Checkpoints

**You need this when:**
- Working on long-running tasks that might need rollback
- Want to save progress before trying risky changes
- Need to resume work after interruptions

**Real-world scenarios:**
- Save context before refactoring attempt, resume if it fails
- Checkpoint after each successful test suite run
- Snapshot context before deploying to production

**See:** [checkpoints](./checkpoints/)

---

#### Remediation

**You need this when:**
- Encountering the same class of errors repeatedly
- Want to build a "runbook" of fixes automatically
- Need to share error solutions across team members

**Real-world scenarios:**
- Record fix for "ECONNREFUSED on port 5432" → "Start Docker Postgres"
- Store solution for "Module not found" → "Run npm install"
- Track "CORS error" → "Add Access-Control-Allow-Origin header"

**See:** [remediation](./remediation/)

---

#### Repository Indexing

**You need this when:**
- Searching large codebases (>10K files)
- Want semantic search ("find authentication logic") not just grep
- Need to understand unfamiliar code quickly

**Real-world scenarios:**
- "Where is the payment processing code?" → semantic search finds it
- "Show me all database migration files" → finds patterns across naming conventions
- "Find error handling patterns" → discovers similar try/catch blocks

**See:** [repository-indexing](./repository-indexing/)

---

#### Context-Folding

**You need this when:**
- Token budget is constrained (approaching context limits)
- Running exploratory subtasks (trial-and-error debugging)
- Want to isolate messy work from main reasoning chain

**Real-world scenarios:**
- Search 20 files for a function → fold into branch, return "found in src/auth.go:42"
- Try 5 different API fixes → fold attempts, return only successful one
- Fetch 10 documentation pages → fold research, return summary

**See:** [context-folding](./context-folding/)

---

## Common Patterns

### Pattern 1: Search Before Record

Always search for existing memories before recording new ones:

```go
// 1. Search first
results := memorySearch("npm install errors")

// 2. Use existing knowledge if found
if len(results) > 0 {
    applyExistingSolution(results[0])
    return
}

// 3. Try new approach
solution := tryNewFix()

// 4. Record for next time
memoryRecord(solution)
```

**See:** [session-lifecycle](./session-lifecycle/), [remediation](./remediation/)

---

### Pattern 2: Checkpoint Before Risky Operations

Save context before operations that might fail:

```go
// 1. Save checkpoint before risky change
checkpointSave("before-refactor", "Safe state before refactoring auth")

// 2. Attempt risky operation
err := dangerousRefactor()

// 3. Resume from checkpoint if it failed
if err != nil {
    checkpointResume("before-refactor")
}
```

**See:** [checkpoints](./checkpoints/)

---

### Pattern 3: Index Once, Search Many

Index repositories once, then run multiple searches:

```go
// 1. Index repository (expensive, do once)
repositoryIndex("/path/to/repo")

// 2. Run multiple searches (cheap, do many times)
auth := semanticSearch("authentication logic")
db := semanticSearch("database queries")
api := semanticSearch("REST endpoints")
```

**See:** [repository-indexing](./repository-indexing/)

---

### Pattern 4: Fold Exploration, Not Implementation

Use context-folding for exploration, not final implementation:

```go
// ✅ GOOD: Fold exploration
branchCreate("find-config-file", "Search 10 config locations", 5000)
location := searchManyFiles()
branchReturn(location) // Returns: "config in ~/.app/config.yaml"

// ❌ BAD: Don't fold implementation
// Implementation should stay in main context for visibility
```

**See:** [context-folding](./context-folding/)

---

## Prerequisites

### Required

- **Go 1.21+** - All examples are Go programs
- **contextd binary** - Install with `brew install fyrsmithlabs/tap/contextd` or see [QUICKSTART.md](../QUICKSTART.md)

### Optional

- **Docker** - Only if using [qdrant-config](./qdrant-config/) examples
- **Claude Code** - For MCP integration examples

## Installation

### Quick Install

```bash
# Install contextd
brew install fyrsmithlabs/tap/contextd

# Verify installation
contextd --version

# Clone examples (if not already in repo)
git clone https://github.com/fyrsmithlabs/contextd.git
cd contextd/examples
```

### Running Examples

Each example is independent:

```bash
# Pick any example
cd session-lifecycle  # or checkpoints, remediation, etc.

# Read the README first
cat README.md

# Run the example
go run main.go
```

## Data Storage

Examples use contextd's default data directory:

```
~/.config/contextd/
├── vectorstore/          # Embedded chromem database
│   └── memories/         # Memory collection
├── checkpoints/          # Saved context snapshots
├── remediation/          # Error fix patterns
└── repositories/         # Indexed code repositories
```

**Backup your data:**

```bash
# Backup all contextd data
tar czf contextd-backup-$(date +%Y%m%d).tar.gz ~/.config/contextd/

# Restore from backup
tar xzf contextd-backup-20260114.tar.gz -C ~/
```

## Troubleshooting

### Error: "contextd: command not found"

**Cause**: contextd binary not in PATH

**Solution**:
```bash
# Install with Homebrew
brew install fyrsmithlabs/tap/contextd

# Or download binary from releases
# https://github.com/fyrsmithlabs/contextd/releases/latest
```

---

### Error: "go: cannot find main module"

**Cause**: Not in example directory, or go.mod missing

**Solution**:
```bash
# Ensure you're in the example directory
cd examples/session-lifecycle  # or other example

# Verify go.mod exists
ls go.mod

# If missing, initialize (shouldn't be needed)
go mod init example.com/demo
```

---

### Error: "connection refused" to contextd

**Cause**: contextd server not running

**Solution**:
```bash
# Start contextd in separate terminal
contextd --mcp

# Or run in background
contextd --mcp &

# Verify it's running
ps aux | grep contextd
```

---

### Examples run slowly on first execution

**Cause**: FastEmbed downloading ONNX model on first run (~50MB)

**Solution**: This is normal. Subsequent runs will be fast (model is cached).

```bash
# First run (slow - downloads model)
go run main.go  # ~30 seconds

# Subsequent runs (fast - uses cached model)
go run main.go  # ~2 seconds
```

---

### Error: "no such file or directory: ~/.config/contextd"

**Cause**: contextd data directory doesn't exist yet

**Solution**: contextd creates it automatically on first run. If you get this error:

```bash
# Manually create directory
mkdir -p ~/.config/contextd/vectorstore

# Run contextd once to initialize
contextd --version
```

---

## Contributing Examples

Want to add a new example? Follow this structure:

```
examples/your-example/
├── README.md          # Detailed guide with Overview, Quick Start, Examples, Troubleshooting
├── main.go            # Working code with extensive comments
└── go.mod             # Go module definition
```

**README template:**

```markdown
# Your Example Name

Brief description.

## Overview

Explain what this example demonstrates.

## Quick Start

Step-by-step instructions.

## Examples

Show code examples with output.

## Troubleshooting

Common issues and solutions.
```

See existing examples for patterns to follow.

## Resources

- **Documentation**: [docs/](../docs/)
- **Quick Start**: [QUICKSTART.md](../QUICKSTART.md)
- **Project README**: [README.md](../README.md)
- **MCP Protocol**: [Model Context Protocol](https://modelcontextprotocol.io/)

## License

Apache 2.0 - See [LICENSE](../LICENSE)

---

**Need help?** Open an issue: https://github.com/fyrsmithlabs/contextd/issues
