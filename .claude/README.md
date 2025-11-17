# .claude Directory

This directory contains Claude Code configuration for multi-agent orchestration and automation.

## Structure

```
.claude/
├── README.md           # This file
├── settings.toml       # Hook configurations and automation
├── mcp.json            # MCP server configurations
├── agents/             # Agent definitions for delegation
│   ├── orchestrator.md # Main workflow coordinator
│   └── [agent].md      # Specialized agents (add as needed)
├── commands/           # Slash command definitions
│   ├── create-spec-issue.md
│   ├── spec-to-issue.md
│   ├── create-repo.md
│   └── [command].md    # Custom commands
└── skills/             # Skill definitions (optional)
    └── [skill].md      # Custom skills
```

## Components

### mcp.json
Configures MCP (Model Context Protocol) server connections:
- **github-bot**: Creates PRs and commits code (BOT_GITHUB_TOKEN)
- **github-reviewer**: Approves PRs created by bot (REVIEWER_GITHUB_TOKEN)
- Two-bot system satisfies GitHub ruleset requirements (no self-approval)
- See `docs/setup/GITHUB_BOT_SETUP.md` for complete setup instructions

### settings.toml
Configures automation hooks that run at specific events:
- **PostToolUse**: After a tool completes (e.g., run tests after file save)
- **PreToolUse**: Before a tool runs (e.g., checks before commit)
- **Notification**: When Claude sends notifications
- **UserPromptSubmit**: When user submits a prompt

### agents/
Contains agent definitions for multi-agent orchestration:
- **orchestrator.md**: Main coordinator that delegates to specialized agents
- Additional agents can be added as needed (go-architect, go-engineer, etc.)

**Usage Pattern**: "Have the [agent-type] agent [specific task]"

### commands/
Markdown files that become slash commands in Claude Code:
- Files become available as `/command-name`
- Use `$ARGUMENTS` keyword for parameters
- Can execute scripts or provide workflows

**Usage**: Type `/` in Claude Code to see available commands

### skills/
Optional directory for custom skill definitions:
- Skills are specialized capabilities
- Can be invoked by Claude Code
- Project-specific workflows and automations

## Creating New Agents

1. Create `.claude/agents/<agent-name>.md`
2. Define:
   - Role and responsibilities
   - Activation criteria
   - Key specifications to reference
   - Decision frameworks
   - Examples (good/bad)
3. Update `orchestrator.md` delegation table
4. Update root `CLAUDE.md` agent list

## Creating New Commands

1. Create `.claude/commands/<command-name>.md`
2. Document:
   - Usage syntax
   - Process steps
   - Examples
   - Notes and requirements
3. Add to root `CLAUDE.md` "Available Commands" section
4. Create supporting scripts in `.scripts/` if needed

## Creating New Skills

1. Create `.claude/skills/<skill-name>.md`
2. Define:
   - Skill purpose
   - When to use
   - How to invoke
   - Dependencies
3. Document in root `CLAUDE.md` if widely used

## Automation Hooks

Current hooks configured in `settings.toml`:

### On File Save (Go files)
- Run tests (`go test ./...`)
- Format code (`go fmt $FILE`)
- Run linters (`golint`, `go vet`)

### On Commit
- Run pre-task completion checks (`.scripts/pre-task-complete.sh`)
- Verify build, tests, coverage, race conditions

### On New Package Files
- Check for spec existence
- Warn if spec missing
- Suggest creating spec issue

### On Errors
- Detect errors in command output
- Suggest using @agent-research-analyst
- Recommend creating resolution spec

## Best Practices

### Agents
- Keep agent files focused on single responsibilities
- Reference specs rather than duplicating content
- Document decision frameworks clearly
- Include activation criteria

### Commands
- Use clear, descriptive names
- Document all parameters
- Provide examples
- Include error handling notes

### Skills
- Define clear scope and purpose
- Document dependencies
- Provide usage examples
- Keep skills focused

### Hooks
- Use `background = true` for non-blocking operations
- Test hooks thoroughly before committing
- Document hook behavior in comments
- Consider performance impact

## Example: Adding a New Agent

```markdown
# Package: my-agent

## Role
Brief description of agent's purpose

## Responsibilities
- Task 1
- Task 2

## Activation
When to delegate to this agent

## Key Specifications
- docs/specs/relevant-spec.md

## Decision Framework
How this agent makes decisions

## Examples
### Good
[Example]

### Bad
[Example]
```

Then update `orchestrator.md`:
```markdown
| Task Type | Agent | Command Pattern |
|-----------|-------|----------------|
| My task | my-agent | "Have the my-agent agent..." |
```

## Token Efficiency

The .claude directory structure supports token efficiency:

1. **Small files**: Each agent/command is focused
2. **On-demand loading**: Only load what's needed
3. **Reference specs**: Link to details, don't duplicate
4. **Progressive disclosure**: Load metadata first, details later

## Version Control

All files in `.claude/` should be committed to git:
- Settings are project-specific
- Agents define project workflows
- Commands are team-shared
- Skills can be project or team level

Use `.gitignore` for:
- Local overrides (if supported)
- Temporary files
- Secrets (never commit in .claude/)

## Related Documentation

- Root `CLAUDE.md` - Project-wide instructions
- `docs/specs/` - Technical specifications
- `.scripts/` - Automation scripts
- Package `CLAUDE.md` files - Package-specific guidance

---

**Note**: This directory structure follows the multi-agent orchestration pattern while maintaining token efficiency through hierarchical documentation.
