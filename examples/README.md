# contextd Examples

Learn how to use contextd's MCP tools in Claude Code for AI agent memory, context management, and intelligent code search.

## Overview

This directory contains comprehensive guides for using contextd's features through MCP (Model Context Protocol) tools in Claude Code. Each example shows real-world usage patterns with practical scenarios you'll encounter in daily development.

**All examples demonstrate MCP tool usage** - the protocol that Claude Code uses to communicate with contextd.

## Available Examples

### Core Workflows

| Example | What You'll Learn | Key Tools |
|---------|------------------|-----------|
| **[session-lifecycle](./session-lifecycle/)** | The fundamental pattern: search past learnings → apply strategies → record new knowledge | `memory_search`, `memory_record`, `memory_feedback`, `memory_outcome` |
| **[checkpoints](./checkpoints/)** | Save conversation snapshots and restore to any point for safe refactoring and error recovery | `checkpoint_save`, `checkpoint_list`, `checkpoint_resume` |
| **[remediation](./remediation/)** | Track error patterns and their fixes to never debug the same issue twice | `remediation_record`, `remediation_search`, `remediation_feedback` |
| **[repository-indexing](./repository-indexing/)** | Semantic code search that finds code by what it does, not what it's named | `repository_index`, `semantic_search` |
| **[context-folding](./context-folding/)** | Execute complex subtasks in isolated branches with 90%+ context compression | `branch_create`, `branch_return`, `branch_status` |

### Configuration

| Example | Description |
|---------|-------------|
| **[qdrant-config](./qdrant-config/)** | Production Qdrant vector store configurations for scaling beyond local development |

## Quick Start

### 1. Install contextd

First, install contextd and configure it with Claude Code:

```bash
# Option 1: Automated (recommended)
claude plugins add fyrsmithlabs/contextd
# Then in Claude Code: /contextd:install

# Option 2: Manual
brew install fyrsmithlabs/tap/contextd
ctxd mcp install
```

See [QUICKSTART.md](../QUICKSTART.md) for detailed installation instructions.

### 2. Verify MCP Tools Available

In Claude Code, ask:
```
Claude, list your MCP tools
```

You should see contextd tools like:
- `memory_search`
- `checkpoint_save`
- `remediation_search`
- `semantic_search`
- `branch_create`
- And more...

### 3. Start with Session Lifecycle

**New to contextd?** Start here: [session-lifecycle](./session-lifecycle/)

This example teaches the fundamental pattern that powers everything else:
1. Search for past learnings
2. Apply strategies to your task
3. Record new knowledge
4. Provide feedback
5. Track outcomes

**5 minutes** to understand the core workflow.

### 4. Explore by Use Case

Pick the example that matches your current need:

#### "I'm refactoring critical code"
→ [checkpoints](./checkpoints/) - Save before/after snapshots for safe rollback

#### "This error looks familiar"
→ [remediation](./remediation/) - Search for past fixes before debugging

#### "Where's the code that handles X?"
→ [repository-indexing](./repository-indexing/) - Semantic search finds code by concept

#### "I need to explore many files but keep context clean"
→ [context-folding](./context-folding/) - Isolate exploration, return only results

#### "I'm deploying to production"
→ [qdrant-config](./qdrant-config/) - Scale beyond local embedded storage

## How MCP Tools Work

contextd runs as an **MCP server** that Claude Code connects to. When you ask Claude to perform actions, it can invoke contextd's tools:

```
┌─────────────────────────────────────────┐
│         Claude Code (Client)            │
│                                         │
│  User: "Search for past learnings      │
│         about database migrations"      │
│                                         │
│  Claude: [Invokes MCP tool]            │
│  ┌────────────────────────────────┐    │
│  │  memory_search(                │    │
│  │    project_id="my-api",        │    │
│  │    query="database migrations",│    │
│  │    limit=5                     │    │
│  │  )                             │    │
│  └────────────┬───────────────────┘    │
└───────────────┼────────────────────────┘
                │ MCP Protocol (stdio)
                │
┌───────────────▼────────────────────────┐
│       contextd (MCP Server)            │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │   Search vectorstore...          │  │
│  │   - Find 3 relevant memories     │  │
│  │   - Rank by confidence           │  │
│  │   - Return to Claude             │  │
│  └──────────────────────────────────┘  │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │  Storage (~/.local/share/contextd)  │
│  │  - Memories                      │  │
│  │  - Checkpoints                   │  │
│  │  - Remediation patterns          │  │
│  │  - Repository index              │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
                │
                ▼
        [Returns results to Claude]

Claude: "I found 3 relevant memories about
        database migrations..."
```

**You don't write MCP client code** - Claude Code handles that automatically. You just ask Claude to perform actions, and it uses the appropriate MCP tools.

## Architecture

### Local-First Design

contextd runs entirely locally with zero external dependencies:

```
Your Machine
├── Claude Code (AI agent)
├── contextd Server (MCP server)
│   ├── chromem (embedded vector database)
│   ├── FastEmbed (local ONNX embeddings)
│   └── SQLite (metadata storage)
└── ~/.local/share/contextd (data storage)
    ├── memories/
    ├── checkpoints/
    ├── remediation/
    └── repository-index/
```

**No API keys. No network calls. No cloud services.**

Everything runs on your machine. Your code and conversations never leave your computer.

### Optional: Production Scaling

For production use, scale to Qdrant:

```
Production Environment
├── Claude Code Instances (multiple agents)
├── contextd Servers (horizontal scaling)
└── Qdrant Cluster (external vector database)
    └── Multi-tenant isolation
```

See [qdrant-config](./qdrant-config/) for production deployment patterns.

## Common Patterns

### Pattern 1: Search Before Doing

Always search for past learnings before starting non-trivial work:

```
User: "Implement rate limiting for the API"

Claude: "Let me search for past learnings about rate limiting..."

[memory_search: "rate limiting API implementation"]

Found: "Used token bucket algorithm with Redis backend"

Claude: "Based on past experience, I'll use a token bucket approach..."
```

**Why**: Avoid reinventing solutions you've already developed.

---

### Pattern 2: Checkpoint Before Refactoring

Save checkpoints before risky changes:

```
User: "Refactor the authentication system"

Claude: "This is a critical system. Let me save a checkpoint first..."

[checkpoint_save: "before-auth-refactor"]

[Makes changes...]

User: "Wait, this broke production!"

[checkpoint_resume: "before-auth-refactor"]

Claude: "Restored to working state. Let me try a different approach..."
```

**Why**: Always have a rollback point for critical changes.

---

### Pattern 3: Record After Success

Save new learnings immediately after completing tasks:

```
[Completes task successfully]

Claude: "Let me record this approach for future reference..."

[memory_record: title="OAuth2 integration with Stripe", content="..."]

Claude: "Recorded for future sessions."
```

**Why**: Build knowledge over time. Today's solution helps tomorrow's problem.

---

### Pattern 4: Search Remediation for Errors

Check for known fixes before debugging:

```
User: "App crashed: nil pointer dereference"

Claude: "Let me search for similar errors we've fixed..."

[remediation_search: "nil pointer dereference"]

Found: "Add nil checks before accessing objects"

Claude: "We've seen this before. Applying the same fix..."
```

**Why**: Don't debug the same error twice.

---

### Pattern 5: Use Branches for Exploration

Keep exploration out of main context:

```
User: "Find all files that import 'stripe'"

Claude: "This requires reading many files. Using a branch..."

[branch_create: budget=5000]
  [Reads 20 files - uses 4,500 tokens]
[branch_return: "Found 8 files importing stripe: ..."]

Main context grows by: 100 tokens (not 4,500!)
```

**Why**: Achieve 90%+ context compression for verbose operations.

## Data Storage

All contextd data is stored locally:

```
~/.local/share/contextd/
├── chromem/                 # Embedded vector database
│   ├── memories.db
│   ├── remediation.db
│   └── repository-index.db
├── checkpoints/             # Conversation snapshots
│   └── [checkpoint files]
└── config/                  # User configuration
    └── config.yaml
```

### Backup Your Data

```bash
# Backup everything
tar -czf contextd-backup.tar.gz ~/.local/share/contextd

# Restore
tar -xzf contextd-backup.tar.gz -C ~/
```

### Clear All Data

```bash
# Remove all stored data (cannot be undone!)
rm -rf ~/.local/share/contextd
```

## Troubleshooting

### MCP Tools Not Available

**Symptom**: Claude says "I don't have access to that tool"

**Fix**:
1. Verify contextd is installed: `which contextd`
2. Check MCP configuration: `cat ~/.claude/settings.json | grep contextd`
3. Restart Claude Code
4. Run validation: `ctxd mcp status`

---

### "Connection refused" errors

**Symptom**: contextd fails to start or Claude can't connect

**Fix**:
1. Check if contextd is running: `ps aux | grep contextd`
2. Test manually: `contextd --mcp --no-http`
3. Check logs: `tail -f ~/.local/share/contextd/logs/contextd.log`
4. Verify MCP config in `~/.claude/settings.json`

---

### Search returns no results

**Symptom**: `memory_search` or `semantic_search` finds nothing

**Possible causes**:
1. **Nothing recorded yet**: Use `memory_record` to save learnings
2. **Query too specific**: Try broader search terms
3. **Wrong project_id**: Ensure you're searching the right project
4. **min_confidence too high**: Lower the threshold

**Fix**: See troubleshooting sections in individual examples.

---

### "Project not indexed" error

**Symptom**: `semantic_search` fails with indexing error

**Fix**:
```
[repository_index: project_path="/path/to/project"]
```

Must index before searching. See [repository-indexing](./repository-indexing/) for details.

## Best Practices

### ✅ DO

- **Search first**: Always check for past learnings before starting work
- **Record learnings**: Save strategies after completing tasks
- **Save checkpoints**: Before refactoring or risky changes
- **Use branches**: For verbose exploration or trial-and-error
- **Provide feedback**: Mark memories/remediations as helpful/not helpful
- **Track outcomes**: Report success/failure for confidence tuning

### ❌ DON'T

- **Don't skip searches**: Missing past learnings means repeating work
- **Don't record trivia**: Only save reusable strategies
- **Don't forget checkpoints**: You'll regret it when things break
- **Don't overuse branches**: Simple tasks don't need isolation
- **Don't ignore feedback**: The system learns from your ratings

## Integration Examples

### Combining Tools

The real power comes from combining tools:

```
# Complex refactoring workflow:

1. Search for past refactoring strategies
   [memory_search: "refactoring patterns"]

2. Save checkpoint before starting
   [checkpoint_save: "before-refactor"]

3. Use branch for code exploration
   [branch_create: "find-related-code"]
   [semantic_search: "authentication logic"]
   [branch_return: "Found in 3 files"]

4. Apply refactoring

5. If errors occur, search for fixes
   [remediation_search: error_message]

6. Record the refactoring approach
   [memory_record: "Successful auth refactor pattern"]

7. Provide feedback on helpful memories
   [memory_feedback: helpful=true]
```

See individual examples for detailed integration patterns.

## Next Steps

### For New Users

1. **Read**: [session-lifecycle](./session-lifecycle/) (5 minutes)
2. **Try**: Ask Claude to search for past learnings about something you've worked on
3. **Practice**: Record a new learning after completing your next task
4. **Explore**: Try checkpoints on your next refactoring

### For Power Users

- **Optimize**: Tune confidence thresholds and search parameters
- **Scale**: Set up [Qdrant](./qdrant-config/) for production
- **Automate**: Configure auto-checkpoints at context thresholds
- **Integrate**: Build workflows combining multiple tools

### For Teams

- **Share**: Export/import memories across team members
- **Standardize**: Use consistent categories and tags
- **Document**: Record team patterns and conventions
- **Monitor**: Track which memories are most helpful

## Contributing

Found a useful pattern? Want to improve an example?

1. Test your pattern in real usage
2. Document the scenario and solution
3. Submit a PR with:
   - Clear use case description
   - Example MCP tool invocations
   - Expected outcomes
   - Common pitfalls

## Support

- **Documentation**: [Full docs](../docs/)
- **Issues**: [GitHub Issues](https://github.com/fyrsmithlabs/contextd/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fyrsmithlabs/contextd/discussions)

---

**Start with**: [session-lifecycle](./session-lifecycle/) - Master the fundamental pattern in 5 minutes.

**Remember**: contextd is a tool for continuous improvement. Every task completed with contextd makes the next task easier. The system learns from your work and helps you avoid repeating yourself.
