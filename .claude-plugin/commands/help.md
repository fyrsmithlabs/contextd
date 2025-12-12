# contextd Help

List all available skills and commands for contextd Claude Code plugin.

## Skills

Skills are activated automatically based on context or can be referenced with `@contextd:<skill-name>`.

| Skill | Description |
|-------|-------------|
| `using-contextd` | Overview of all contextd tools - use at session start |
| `session-lifecycle` | Manages session start/end, checkpoint resume, learning extraction |
| `cross-session-memory` | Learning loop: search before starting, record after completing |
| `checkpoint-workflow` | Save and resume session state when context gets full |
| `error-remediation` | Error diagnosis, past fix search, and solution recording |
| `repository-search` | Semantic code search - finds code by meaning, not keywords |
| `secret-scrubbing` | Configure PostToolUse hooks for automatic secret redaction |
| `writing-claude-md` | Best practices for creating effective CLAUDE.md files |
| `project-onboarding` | Analyze existing codebases to generate CLAUDE.md |
| `consensus-review` | Multi-agent code review with 4 parallel specialized reviewers |

## Commands

Commands are invoked with `/contextd:<command>`.

| Command | Description |
|---------|-------------|
| `/contextd:search <query>` | Search across memories and remediations |
| `/contextd:remember` | Record a learning or insight from current session |
| `/contextd:checkpoint` | Save a checkpoint of current session state |
| `/contextd:resume` | List and resume from a previous checkpoint |
| `/contextd:status` | Show contextd status for current project |
| `/contextd:diagnose <error>` | Diagnose an error using AI analysis and past fixes |
| `/contextd:init` | Initialize contextd for a new project |
| `/contextd:onboard` | Analyze existing project and generate CLAUDE.md |
| `/contextd:consensus-review <path>` | Run multi-agent code review on files/directory |
| `/contextd:test-skill <skill> <n>` | Run pressure test scenario against a skill |
| `/contextd:help` | Show this help message |

## MCP Tools

Low-level tools available via `mcp__contextd__*`:

| Tool | Purpose |
|------|---------|
| `memory_search` | Find relevant past strategies |
| `memory_record` | Save new memory explicitly |
| `memory_feedback` | Rate memory helpfulness (adjusts confidence) |
| `checkpoint_save` | Save context snapshot |
| `checkpoint_list` | List available checkpoints |
| `checkpoint_resume` | Resume from checkpoint |
| `remediation_search` | Find error fix patterns |
| `remediation_record` | Record new fix |
| `repository_index` | Index repo for semantic search |
| `repository_search` | Semantic search over indexed code |
| `troubleshoot_diagnose` | AI-powered error diagnosis |

## Quick Start

```
1. /contextd:status           - See what contextd knows
2. /contextd:search <topic>   - Find relevant memories
3. Do your work
4. /contextd:remember         - Record what you learned
5. /contextd:checkpoint       - Save before clearing context
```

## Getting Started with Skills

Reference skills in conversation:
- "Use the @contextd:error-remediation skill to diagnose this error"
- "Follow @contextd:session-lifecycle for session start"
- "Apply @contextd:writing-claude-md to create CLAUDE.md"

## Error Handling

@_error-handling.md
