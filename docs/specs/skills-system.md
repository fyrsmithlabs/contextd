# contextd Skills System Specification

**Status**: Draft
**Created**: 2024-12-02
**Author**: Claude + dahendel

---

## Overview

The contextd Skills System enables AI agents to effectively use contextd's MCP tools through teachable workflows. Unlike agent-specific skill systems, contextd provides **agent-agnostic skills infrastructure** accessible to any MCP-compatible agent.

### Design Goals

1. **Agent-agnostic** - Any MCP client (Claude, GPT, Gemini, custom agents) can access skills
2. **Searchable** - Semantic search across all indexed skills
3. **Extensible** - Index custom skills, scrape documentation
4. **Learnable** - Skills can reference memories and remediations for context

---

## Architecture

### Two Repositories

```
contextd/                      # Core MCP server
├── docs/specs/
│   └── skills-system.md       # This spec
├── internal/skills/           # Skills MCP tool implementation (Phase 7+)
└── skills/                    # Built-in skills served via MCP

contextd-marketplace/          # Claude Code plugin distribution
├── plugin.json                # Plugin manifest
├── skills/
│   ├── using-contextd/SKILL.md
│   ├── cross-session-memory/SKILL.md
│   ├── checkpoint-workflow/SKILL.md
│   └── error-remediation/SKILL.md
└── commands/
    ├── checkpoint.md
    ├── remember.md
    ├── diagnose.md
    ├── resume.md
    ├── status.md
    └── search.md
```

### How It Works

1. **Claude Code users**: Install `contextd-marketplace` as plugin, skills load via standard mechanism
2. **Other MCP agents**: Call `skills_search` and `skills_get` tools directly
3. **Future**: `skills_create` enables interactive skill creation with doc scraping

---

## Skills Definition

### Four Modular Skills

Each skill has specific triggers and covers a distinct workflow:

#### `contextd:using-contextd`

**Trigger**: Session start (like `using-superpowers`)

**Purpose**: Introduction to contextd tools, establishes the mental model.

**Teaches**:
- What contextd is (cross-session memory + context management)
- Overview of 9 MCP tools grouped by function
- When to invoke the other 3 workflow skills
- Tenant ID auto-derived from GitHub remote (no config needed)

#### `contextd:cross-session-memory`

**Trigger**: Task start (search), task completion (prompt to record)

**Purpose**: The learning loop.

**Teaches**:
- `memory_search` at task start - "Have I solved something like this before?"
- `memory_record` at task completion - "What did I learn?"
- `memory_feedback` when a memory helped/didn't help - reinforcement learning
- **Search before assuming** - Always search collections before re-deriving
- **Capture the WHY** - Design decisions need rejected alternatives, tradeoffs, consequences

#### `contextd:checkpoint-workflow`

**Trigger**: Context approaching 70%, long-running tasks, user request

**Purpose**: Context preservation.

**Teaches**:
- `checkpoint_save` with good summaries (what was done, what's next)
- `checkpoint_list` to find previous work
- `checkpoint_resume` at appropriate level (summary/context/full)

#### `contextd:error-remediation`

**Trigger**: Any error/exception encountered

**Purpose**: Error pattern matching.

**Teaches**:
- `troubleshoot_diagnose` first - get AI-powered analysis
- `remediation_search` - check if this error was fixed before
- `remediation_record` after fixing - save for future

---

## Slash Commands

Six user-triggered commands for Claude Code:

| Command | Purpose |
|---------|---------|
| `/contextd:checkpoint` | Quick save - auto-generates summary from recent context |
| `/contextd:remember` | Record a learning/insight from current session |
| `/contextd:diagnose` | Troubleshoot an error (user pastes error message) |
| `/contextd:resume` | List and resume from a previous checkpoint |
| `/contextd:status` | Show session info, memories, checkpoints, tenant ID |
| `/contextd:search` | Direct search across memories and remediations |

---

## MCP Tools (Future)

New tools to add to contextd MCP server:

### `skills_get`

Retrieve a skill by name from built-in or indexed sources.

```go
type skillsGetInput struct {
    Name   string `json:"name" jsonschema:"required,Skill name (e.g. contextd:checkpoint-workflow)"`
    Source string `json:"source,omitempty" jsonschema:"Source filter: builtin, indexed, or all (default: all)"`
}

type skillsGetOutput struct {
    Name        string   `json:"name" jsonschema:"Skill name"`
    Description string   `json:"description" jsonschema:"Skill description"`
    Content     string   `json:"content" jsonschema:"Full skill content as markdown"`
    Source      string   `json:"source" jsonschema:"Where skill came from"`
    Tags        []string `json:"tags" jsonschema:"Skill tags"`
}
```

### `skills_search`

Semantic search across all indexed skills.

```go
type skillsSearchInput struct {
    Query string `json:"query" jsonschema:"required,Search query"`
    Limit int    `json:"limit,omitempty" jsonschema:"Max results (default: 5)"`
}

type skillsSearchOutput struct {
    Skills []struct {
        Name        string  `json:"name"`
        Description string  `json:"description"`
        Score       float64 `json:"score"`
        Source      string  `json:"source"`
    } `json:"skills"`
    Count int `json:"count"`
}
```

### `skills_index`

Index skills from filesystem path.

```go
type skillsIndexInput struct {
    Path    string   `json:"path" jsonschema:"required,Directory containing SKILL.md files"`
    Pattern string   `json:"pattern,omitempty" jsonschema:"Glob pattern (default: **/SKILL.md)"`
    Tags    []string `json:"tags,omitempty" jsonschema:"Tags to apply to indexed skills"`
}

type skillsIndexOutput struct {
    Path          string `json:"path"`
    SkillsIndexed int    `json:"skills_indexed"`
}
```

### `skills_create`

Create a new skill with interactive research workflow.

```go
type skillsCreateInput struct {
    Name        string   `json:"name" jsonschema:"required,Skill name"`
    Description string   `json:"description" jsonschema:"required,When to use this skill"`
    DocURLs     []string `json:"doc_urls,omitempty" jsonschema:"Documentation URLs to scrape for content"`
    Selectors   []string `json:"selectors,omitempty" jsonschema:"CSS selectors for content extraction"`
    MaxDepth    int      `json:"max_depth,omitempty" jsonschema:"Link follow depth (default: 2)"`
    Content     string   `json:"content,omitempty" jsonschema:"Direct skill content (if not scraping)"`
    Tags        []string `json:"tags,omitempty" jsonschema:"Tags for categorization"`
}

type skillsCreateOutput struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Source      string `json:"source"`
}
```

**Interactive workflow** (modeled after `superpowers:writing-skills`):

1. User invokes `skills_create` with name + description
2. Agent prompts: "Should I search for best practices and common workflows?"
   - Web search for patterns/docs
   - Search existing memories for related learnings
   - Check remediation history for related errors
3. If doc URLs provided:
   - Scrape with go-colly
   - Extract relevant sections
   - Agent asks: "What aspects should this skill cover?"
4. Generate skill draft following SKILL.md structure
5. TDD validation (optional but recommended):
   - Run pressure scenario WITHOUT skill (baseline)
   - Run WITH skill (verify compliance)
   - Iterate until bulletproof
6. Store and index in vectorstore

---

## Skill File Format

Standard format compatible with Claude Code and other agents:

```markdown
---
name: skill-name-with-hyphens
description: Use when [triggers] - [what it does in third person]
---

# Skill Name

## Overview
Core principle in 1-2 sentences.

## When to Use
- Symptoms and triggers
- When NOT to use

## The Process / Quick Reference
Steps or table for scanning

## Common Mistakes
What goes wrong + fixes
```

### Frontmatter Rules

- Only two fields: `name` and `description`
- Max 1024 characters total
- `name`: Letters, numbers, hyphens only
- `description`: Third-person, starts with "Use when..."

### Claude Search Optimization (CSO)

For discovery, include:
- Concrete triggers and symptoms in description
- Keywords throughout (error messages, tool names)
- Technology-agnostic language unless skill is tech-specific

---

## Implementation Phases

### Phase 1: contextd-marketplace (Immediate)

- Create `contextd-marketplace` repository
- Write plugin.json manifest
- Write 4 skills (using-contextd, cross-session-memory, checkpoint-workflow, error-remediation)
- Write 6 slash commands
- Publish to claudemarketplaces.com

### Phase 2: Skills MCP Tools (contextd Phase 7)

- Implement `skills_get` tool
- Implement `skills_search` tool with vectorstore
- Implement `skills_index` tool
- Add built-in skills to contextd repo

### Phase 3: Skills Creation (contextd Phase 8)

- Implement `skills_create` tool
- Integrate go-colly for documentation scraping
- Add interactive research prompts
- Vectorstore indexing for skill content

### Phase 4: Agent-Agnostic Distribution

- Document MCP skills API for other agents
- Create example integrations (Codex, custom agents)
- Publish skills API specification

---

## contextd MCP Tools Reference

Skills teach agents when/how to use these tools:

| Tool | Category | Purpose |
|------|----------|---------|
| `checkpoint_save` | Checkpoint | Save context snapshot |
| `checkpoint_list` | Checkpoint | List available checkpoints |
| `checkpoint_resume` | Checkpoint | Resume from checkpoint |
| `memory_search` | Memory | Find relevant past strategies |
| `memory_record` | Memory | Save new memory |
| `memory_feedback` | Memory | Rate memory helpfulness |
| `remediation_search` | Remediation | Find error fix patterns |
| `remediation_record` | Remediation | Record new fix |
| `repository_index` | Repository | Index repo for semantic search |
| `troubleshoot_diagnose` | Troubleshoot | AI-powered error diagnosis |

---

## Related Documents

- `CONTEXTD.md` - User-facing briefing document
- `docs/spec/` - Original architecture specifications
