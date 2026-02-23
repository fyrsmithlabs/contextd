# Contextd Onboarding Guide

Welcome to contextd! This guide will walk you through setup, first use, and daily workflows.

## What is Contextd?

Contextd gives your AI coding assistant **persistent memory**. It remembers:

- ✅ Solutions that worked ("last time X failed, Y fixed it")
- ✅ Patterns you've used ("when doing Z, use approach A")
- ✅ Project context ("this codebase uses pattern P")
- ✅ Session state ("pause here, resume later")

Think of it as giving Claude a **searchable memory** that improves over time.

---

## Quick Setup (5 minutes)

### Step 1: Install Claude Code

Contextd extends [Claude Code](https://claude.ai/claude-code), so install that first:

```bash
# macOS/Linux
curl -fsSL https://claude.ai/install.sh | bash

# Verify
claude --version
```

### Step 2: Install Contextd Plugin

```bash
# Add the plugin (skills, commands, agents)
claude plugins add fyrsmithlabs/marketplace
```

### Step 3: Initialize Contextd

Run this command in Claude Code:

```
/contextd-init
```

This automatically:
- Downloads the contextd binary (or uses Docker)
- Configures MCP settings
- Validates the setup

**That's it!** Restart Claude Code and you're ready.

---

## Verify Installation

After restart, check that contextd is connected:

```
/mcp
```

You should see:
```
✓ contextd - connected
```

---

## First Use Tutorial

Let's walk through the core workflows.

### 1. Starting a New Project

When you start working on a new codebase:

```
/contextd-init
```

This:
- Creates project metadata
- Primes contextd with project context
- Sets up memory tracking

**Or** if joining an existing project:

```
/contextd-onboard
```

This:
- Indexes the repository for semantic search
- Analyzes codebase patterns
- Records architectural insights

### 2. During Development

Contextd works automatically in the background:

**Search Past Learnings:**
```
/contextd-search authentication bug
```
Finds: "Last time auth failed, it was JWT expiry - check token refresh"

**Get Error Help:**
```
/contextd-diagnose "null pointer dereference in auth.go:42"
```
Gets: AI diagnosis + past fixes for similar errors

**Record Learnings:**
After solving something, save it:
```
/contextd-remember
```
Claude will ask what you learned and save it for next time.

### 3. Managing Context

Claude Code has a context window limit. When you hit ~70% capacity:

**Save Your Progress:**
```
/contextd-checkpoint
```
Creates a snapshot of your current session state.

**Clear Context:**
```
/clear
```
Resets the context window.

**Resume Work:**
```
/contextd-resume
```
Restores your session from the checkpoint.

---

## Daily Workflow

Here's a typical day using contextd:

```
┌─ Morning ────────────────────────────────────────┐
│                                                  │
│  1. Open Claude Code                             │
│     → Auto-searches memories                     │
│     → Lists available checkpoints                │
│                                                  │
│  2. Resume from yesterday (if offered)           │
│     /contextd-resume                             │
│                                                  │
└──────────────────────────────────────────────────┘

┌─ During Work ────────────────────────────────────┐
│                                                  │
│  3. Stuck on error?                              │
│     /contextd-diagnose <error>                   │
│     → Finds past fixes                           │
│                                                  │
│  4. Need code reference?                         │
│     Semantic search happens automatically        │
│     → Better than grep, understands meaning      │
│                                                  │
│  5. Solved something?                            │
│     /contextd-remember                           │
│     → Saves for next time                        │
│                                                  │
└──────────────────────────────────────────────────┘

┌─ Context Getting Full (70%+) ────────────────────┐
│                                                  │
│  6. Save checkpoint                              │
│     /contextd-checkpoint                         │
│                                                  │
│  7. Clear context                                │
│     /clear                                       │
│                                                  │
│  8. Resume work                                  │
│     /contextd-resume                             │
│     → Picks up where you left off               │
│                                                  │
└──────────────────────────────────────────────────┘

┌─ End of Day ─────────────────────────────────────┐
│                                                  │
│  9. Review what was learned                      │
│     /contextd-reflect                            │
│     → Analyzes patterns, improves docs           │
│                                                  │
│ 10. Save final checkpoint (optional)             │
│     /contextd-checkpoint                         │
│                                                  │
└──────────────────────────────────────────────────┘
```

---

## Understanding the Tools

Contextd provides these MCP tools to Claude Code:

### Memory Tools

| Tool | What It Does | When Claude Uses It |
|------|--------------|---------------------|
| `memory_search` | Finds relevant past learnings | Start of every task |
| `memory_record` | Saves a new learning | After solving problems |
| `memory_feedback` | Rates memory helpfulness | When a memory helps/doesn't help |
| `memory_outcome` | Reports task success | After completing a task |
| `memory_consolidate` | Merges related memories | Periodic cleanup |

### Checkpoint Tools

| Tool | What It Does | When to Use |
|------|--------------|-------------|
| `checkpoint_save` | Saves session state | Context nearing 70% |
| `checkpoint_list` | Shows available checkpoints | Session start |
| `checkpoint_resume` | Restores saved state | After clearing context |

### Error Remediation Tools

| Tool | What It Does | When to Use |
|------|--------------|-------------|
| `remediation_search` | Finds fixes for similar errors | When debugging |
| `remediation_record` | Records a new error fix | After fixing a bug |
| `remediation_feedback` | Rates fix helpfulness | When a fix works/doesn't |
| `troubleshoot_diagnose` | AI-powered diagnosis | Stuck on an error |

### Code Search Tools

| Tool | What It Does | When Claude Uses It |
|------|--------------|---------------------|
| `semantic_search` | Meaning-based code search | Before reading files |
| `repository_index` | Indexes codebase | Project onboarding |
| `repository_search` | Searches indexed code | Finding code patterns |

### Context Folding Tools

| Tool | What It Does | When to Use |
|------|--------------|-------------|
| `branch_create` | Creates isolated sub-task | Complex analysis |
| `branch_return` | Returns with summary | Sub-task complete |
| `branch_status` | Checks branch progress | Monitor long tasks |

### Conversation Tools

| Tool | What It Does | When Claude Uses It |
|------|--------------|---------------------|
| `conversation_index` | Indexes past conversations | Project onboarding |
| `conversation_search` | Finds past decisions/context | Researching history |

### Reflection Tools

| Tool | What It Does | When to Use |
|------|--------------|-------------|
| `reflect_report` | Generates self-reflection report | End of day/sprint |
| `reflect_analyze` | Analyzes behavioral patterns | Periodic review |

---

## Advanced Features

### Context Folding

For complex tasks that would bloat context, Claude can create "branches":

**Example:**
```
You: "Analyze the entire auth module"

Claude internally:
1. branch_create(description: "Auth module analysis", budget: 4096)
2. Reads all auth files in the branch
3. Analyzes patterns and structure
4. branch_return("Auth uses JWT, 3 handlers, ...")
5. Branch auto-cleans up
```

**Benefits:**
- Deep analysis doesn't pollute main context
- Token budget prevents runaway exploration
- Results automatically scrubbed for secrets

You don't need to manage this - Claude uses it automatically for complex tasks.

### Specialized Agents

Contextd integrates with specialized agents for complex workflows:

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| `contextd:orchestrator` | Multi-agent coordination with context-folding | Large complex tasks |
| `contextd:task-agent` | Debugging, refactoring, and architecture analysis | General development |

These agents leverage:
- **ReasoningBank** - Cross-session learning
- **Context Folding** - Isolated sub-tasks
- **Checkpoints** - Rollback safety

### Self-Reflection

Periodically run:
```
/contextd-reflect
```

This analyzes Claude's behavior patterns and:
- Identifies repeated mistakes
- Suggests documentation improvements
- Records effective strategies

---

## Configuration

### Automated (Recommended)

Use the install command:
```
/contextd-install
```

Or via ctxd CLI:
```bash
ctxd mcp install    # Auto-configure
ctxd mcp status     # Verify setup
ctxd mcp uninstall  # Remove config
```

### Manual Configuration

If you prefer manual setup, add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "contextd",
      "args": ["--mcp", "--no-http"]
    }
  }
}
```

**For Docker:**
```json
{
  "mcpServers": {
    "contextd": {
      "type": "stdio",
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "${HOME}/.config/contextd:/root/.config/contextd",
        "ghcr.io/fyrsmithlabs/contextd:latest",
        "--mcp", "--no-http"
      ]
    }
  }
}
```

---

## Data & Privacy

### Where Data Lives

All data stays on your machine:

```
~/.config/contextd/
├── vectorstore/          # Memories, checkpoints, remediations
├── lib/                  # ONNX runtime (auto-downloaded)
└── models/               # Embedding models (auto-downloaded)
```

**Nothing is sent to external servers.**

### Backup & Restore

**Backup:**
```bash
tar czf contextd-backup-$(date +%Y%m%d).tar.gz ~/.config/contextd/
```

**Restore:**
```bash
tar xzf contextd-backup-20250123.tar.gz -C ~/
```

### Multi-Tenancy

Contextd automatically isolates data by project using git:

- **Tenant ID** = Git remote URL (e.g., `github.com/username`)
- **Project ID** = Repository name

Different repositories have completely isolated memories.

---

## Troubleshooting

### "contextd not found"

Ensure binary is in PATH:
```bash
which contextd
# If not found:
export PATH="$HOME/.local/bin:$PATH"
```

### MCP Server Not Connecting

1. Check settings.json syntax (valid JSON?)
2. Verify contextd path: `which contextd`
3. Test manually: `contextd --version`
4. Restart Claude Code

### First Run is Slow

Normal! Contextd downloads:
- ONNX runtime (~50MB)
- Embedding model (~50MB)

This only happens once. Subsequent runs are instant.

### Still Stuck?

Check:
- [Main Documentation](./CONTEXTD.md)
- [Troubleshooting Guide](./troubleshooting.md)
- [GitHub Issues](https://github.com/fyrsmithlabs/contextd/issues)

Or run:
```
/contextd-help
```

---

## Next Steps

Now that you're set up:

1. **Try it out** - Start a coding session and see memory search in action
2. **Create a checkpoint** - Practice the checkpoint workflow
3. **Record learnings** - Use `/contextd-remember` after solving something
4. **Explore skills** - Run `/contextd-help` to see all available skills
5. **Read the docs** - Check [CONTEXTD.md](./CONTEXTD.md) for advanced features

Welcome to contextd! Your AI assistant just got a memory upgrade.

---

## Related Documentation

- [Main Documentation](./CONTEXTD.md) - Quick start and overview
- [Architecture Overview](./architecture.md) - Detailed component descriptions
- [Configuration Reference](./configuration.md) - All configuration options
- [Troubleshooting](./troubleshooting.md) - Common issues and fixes
