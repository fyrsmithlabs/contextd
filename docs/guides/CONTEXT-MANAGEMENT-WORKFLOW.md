# Context Management Workflow Guide

## Overview

This guide clarifies **how to maintain project context** when working with Claude Code, specifically for the contextd project.

## The Problem You Encountered

You saw advice to "use the Task tool with subagent_type=Explore" for gathering codebase context. This is **outdated/incorrect for the contextd project** because:

1. **Higher context cost** - Defeats contextd's PRIMARY goal (context optimization)
2. **Slower execution** - Agent overhead vs direct queries
3. **Missed dogfooding** - We should use our own product
4. **Generic advice** - Doesn't account for contextd-specific tools

## The Right Approach for Contextd

### Tool Selection Priority

```
┌─────────────────────────────────────────────────────────┐
│ 1. Contextd MCP Tools (FIRST CHOICE)                   │
│    - Dogfooding our product                             │
│    - Lowest context cost                                │
│    - Fastest execution                                  │
└─────────────────────────────────────────────────────────┘
                         ↓ (if specific files known)
┌─────────────────────────────────────────────────────────┐
│ 2. Direct File Operations                              │
│    - Read(), Grep(), Glob()                            │
│    - When you know exact files/patterns                │
└─────────────────────────────────────────────────────────┘
                         ↓ (if nothing else works)
┌─────────────────────────────────────────────────────────┐
│ 3. Task Tool with Explore Agent (AVOID)                │
│    - Only for unfamiliar codebases                     │
│    - NOT for contextd project                          │
└─────────────────────────────────────────────────────────┘
```

## Practical Examples

### Example 1: Understanding Configuration System

**❌ Wrong (outdated advice):**
```bash
# Using Explore agent
Task(subagent_type=Explore, prompt="Explore configuration system")
```

**✅ Right (contextd tools):**
```bash
# Use contextd's semantic search
mcp__contextd__checkpoint_search
  query="configuration management YAML env files"
  top_k=5

# Then read specific files
Read(pkg/config/config.go)
Read(docs/guides/CONFIGURATION.md)
```

### Example 2: Finding Error Handling Code

**❌ Wrong:**
```bash
Task(subagent_type=Explore, prompt="Find error handling patterns")
```

**✅ Right:**
```bash
# Search for error patterns directly
Grep(pattern="return.*fmt.Errorf", path="pkg/", output_mode="files_with_matches")

# Or use contextd remediation search
mcp__contextd__remediation_search
  error_message="configuration loading error"
  limit=5
```

### Example 3: Working on New Feature

**❌ Wrong:**
```bash
# Broad exploration
Task(subagent_type=Explore, prompt="Understand codebase structure")
```

**✅ Right:**
```bash
# Index repository (once)
mcp__contextd__index_repository path="/home/dahendel/projects/contextd"

# Search for relevant context
mcp__contextd__checkpoint_search
  query="similar feature implementation"
  project_path="$(pwd)"
  top_k=3

# Read CLAUDE.md hierarchy
Read(CLAUDE.md)
Read(pkg/CLAUDE.md)
Read(docs/specs/feature/SPEC.md)
```

## When to Use Each Tool

### Use Contextd MCP Tools When:
- ✅ Working on contextd project (dogfooding)
- ✅ Need semantic search across codebase
- ✅ Looking for past solutions (checkpoints, remediations)
- ✅ Want context-efficient queries
- ✅ Need to save work for later (checkpoints)

### Use Direct File Operations When:
- ✅ You know exact file paths
- ✅ Need to read specific configurations
- ✅ Searching for specific code patterns
- ✅ Following CLAUDE.md documentation hierarchy

### Use Task/Explore Agent When:
- ⚠️ **Rare**: Working on unfamiliar codebase
- ⚠️ **Rare**: Contextd tools not available
- ❌ **Never** for contextd project itself

## Key Commands Reference

### Contextd MCP Tools

```bash
# Index repository
mcp__contextd__index_repository path="$(pwd)"

# Semantic search
mcp__contextd__checkpoint_search
  query="your search"
  project_path="$(pwd)"
  top_k=5

# Save checkpoint
mcp__contextd__checkpoint_save
  summary="Brief description"
  project_path="$(pwd)"
  tags=["feature", "config"]

# Find error solutions
mcp__contextd__remediation_search
  error_message="error text"
  limit=5

# Troubleshoot issues
mcp__contextd__troubleshoot
  error_message="problem description"
  stack_trace="stack trace if available"
```

### Direct File Operations

```bash
# Read files
Read(pkg/config/config.go)

# Search code
Grep(pattern="LoadConfig", path="pkg/", output_mode="content")

# Find files
Glob(pattern="**/*_test.go")
```

### Checkpoint System

```bash
# Check context usage
/context-check

# Manual checkpoint
/auto-checkpoint

# Checkpoint at thresholds (automatic):
# - 70% context (140K tokens) - silent save
# - 90% context (180K tokens) - save + recommend /clear
```

## Why This Matters for Contextd

### Dogfooding Benefits
1. **Real-world testing** - We use our own product
2. **Validate features** - Discover bugs and improvements
3. **Demonstrate value** - Show context efficiency gains
4. **Build confidence** - Trust our own implementation

### Context Efficiency Goals
- **Current**: 12K tokens for full context
- **Target**: <3K tokens per search
- **Measured**: 88% context reduction with checkpoints
- **Goal**: 5x reduction (v2.0 → v2.1)

### Performance Targets
- **Search latency**: <100ms
- **Checkpoint save**: <2s
- **Index repository**: One-time operation
- **Semantic search**: Near-instant

## Common Pitfalls to Avoid

### ❌ Don't Do This
```bash
# Using explore agent on contextd
Task(subagent_type=Explore, prompt="...")

# Reading entire files into context
Read(huge-file.go) # 10K+ lines

# Broad, unfocused searches
Grep(pattern=".*", path=".")
```

### ✅ Do This Instead
```bash
# Use contextd tools
mcp__contextd__checkpoint_search query="specific topic"

# Read with limits
Read(huge-file.go, offset=100, limit=50)

# Focused searches
Grep(pattern="LoadConfig", path="pkg/config/")
```

## Integration with Development Workflow

### Step 1: Start Work
```bash
# Search recent checkpoints
mcp__contextd__checkpoint_search
  query="recent work on [feature]"
  project_path="$(pwd)"
  top_k=3
```

### Step 2: Gather Context
```bash
# Read documentation hierarchy
Read(CLAUDE.md)
Read(pkg/CLAUDE.md)
Read(docs/specs/feature/SPEC.md)

# Search for relevant code
mcp__contextd__checkpoint_search query="similar implementation"
```

### Step 3: During Work
```bash
# Save progress checkpoints
mcp__contextd__checkpoint_save
  summary="Implemented YAML config loading"
  project_path="$(pwd)"
  tags=["config", "yaml"]
```

### Step 4: Context Thresholds
```bash
# At 70% - auto-save happens silently
# At 90% - save and /clear

/context-check  # Check current usage
/auto-checkpoint  # Manual checkpoint
```

## Summary

**For the contextd project:**
1. **Always use contextd MCP tools first** (dogfooding)
2. **Use direct file operations** when you know paths
3. **Never use Explore agent** (defeats our goals)
4. **Checkpoint frequently** (context efficiency)
5. **Monitor context usage** (stay under 70%)

**Remember**: We build context optimization tools. We should use them and validate they work as expected.

## Related Documentation

- [AUTO-CHECKPOINT-SYSTEM.md](./AUTO-CHECKPOINT-SYSTEM.md) - Checkpoint automation
- [DEVELOPMENT-WORKFLOW.md](./DEVELOPMENT-WORKFLOW.md) - Development process
- [GETTING-STARTED.md](./GETTING-STARTED.md) - Setup and MCP integration
- [CLAUDE.md](../../CLAUDE.md) - Project-wide policies
- [~/.claude/CLAUDE.md](~/.claude/CLAUDE.md) - Global configuration
