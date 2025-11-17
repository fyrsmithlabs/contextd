# Slash Commands

This directory contains slash command definitions for contextd development workflows.

## Available Commands

Type `/` in Claude Code to see all available commands.

### Repository Management

| Command | Purpose | Example |
|---------|---------|---------|
| `/configure-repo [phase]` | Configure repository with security and development best practices | `/configure-repo security` |
| `/setup-github` | Configure GitHub repository settings | `/setup-github` |

### Specification & Planning

| Command | Purpose | Example |
|---------|---------|---------|
| `/create-spec-issue <name>` | Create GitHub issue for missing spec | `/create-spec-issue authentication` |
| `/spec-to-issue <name>` | Convert spec to GitHub issues with tasks | `/spec-to-issue checkpoint-tagging` |
| `/review-priorities [timeframe]` | Review completed work and update project priorities | `/review-priorities 24h` |

### Development Workflow

| Command | Purpose | Example |
|---------|---------|---------|
| `/start-task <issue-number>` | Initialize development environment with TDD setup | `/start-task 42` |
| `/run-quality-gates [scope]` | Execute quality checks (quick\|full\|coverage\|security) | `/run-quality-gates full` |
| `/check-dependencies [--update]` | Vulnerability scanning and dependency management | `/check-dependencies --update` |
| `/debug-issue "<description>"` | AI-assisted debugging with research-analyst | `/debug-issue "race condition in cache"` |
| `/init-go-project <name>` | Initialize new Go project structure | `/init-go-project myservice` |

### Contextd-Specific Commands

| Command | Purpose | Example |
|---------|---------|---------|

### MCP and Testing (contextd)

| Command | Purpose | Example |
|---------|---------|---------|
| `/checkpoint save "<summary>"` | Quick checkpoint (via contextd MCP) | `/checkpoint save "implemented auth"` |
| `/checkpoint search "<query>"` | Search past checkpoints | `/checkpoint search "authentication"` |
| `/checkpoint list` | List recent checkpoints | `/checkpoint list` |
| `/remediation search "<error>"` | Find error solutions | `/remediation search "connection refused"` |
| `/troubleshoot "<error>"` | AI-powered error diagnosis | `/troubleshoot "panic: runtime error"` |
| `/index repository path=<path>` | Index repository for search | `/index repository path=/home/user/projects/myapp` |
| `/status` | Check contextd service status | `/status` |

## Command Categories

### Daily Development
- `/start-task` - Begin work on issue
- `/run-quality-gates quick` - Quick checks before commit
- `/checkpoint save` - Save progress
- `/check-dependencies` - Weekly dependency check

### Before PR
- `/run-quality-gates full` - Comprehensive quality check
- `/checkpoint save` - Document final state
- PR creation (automated via workflow)

### Troubleshooting
- `/debug-issue` - Get AI assistance
- `/remediation search` - Find known solutions
- `/troubleshoot` - Diagnose errors

### Planning
- `/review-priorities` - Update priorities
- `/spec-to-issue` - Break down specs into tasks
- `/create-spec-issue` - Request missing spec

## Usage Notes

### Required Parameters
- `<parameter>` - Required argument
- `[parameter]` - Optional argument
- `<option1|option2>` - Choose one option

### Common Patterns

**Starting a new feature:**
```bash
/start-task 42
# Creates branch, draft PR, TDD setup
```

**Before committing:**
```bash
/run-quality-gates quick
# Runs tests, coverage, linting
```

**Debugging an error:**
```bash
/debug-issue "race condition in checkpoint service"
# Uses @agent-research-analyst to find solutions
```

**Saving progress:**
```bash
/checkpoint save "implemented multi-tenant isolation"
# Creates checkpoint in contextd
```

## Creating New Commands

To add a new slash command:

1. Create `.claude/commands/<command-name>.md`
2. Document:
   - Usage syntax (with `$ARGUMENTS` keyword)
   - Process steps
   - Examples
   - Notes and requirements
3. Add to this README in appropriate category
4. Update root `CLAUDE.md` if widely used
5. Create supporting scripts in `scripts/` if needed

### Command Template

```markdown
# Command: /my-command

## Usage
/my-command <required-arg> [optional-arg]

## Description
Brief description of what this command does.

## Process
1. Step 1
2. Step 2
3. Step 3

## Examples
/my-command value1
/my-command value1 value2

## Notes
- Important note 1
- Important note 2
```

## Automation Hooks

Commands can be triggered automatically via hooks in `.claude/settings.toml`:

- **On file save**: Run tests, format code
- **On commit**: Run quality gates
- **On new package**: Check for spec
- **On errors**: Suggest using debug-issue
- **At context 70%**: Remind to checkpoint+clear
- **On credential files**: Set 0600 permissions

See `.claude/settings.toml` for hook configuration.

## Command Shortcuts

Frequently used commands:

```bash
# Quick aliases (if your shell supports)
alias task="/start-task"
alias qa="/run-quality-gates quick"
alias ckpt="/checkpoint save"
alias fix="/debug-issue"
```

## Best Practices

### Command Naming
- Use kebab-case
- Be descriptive but concise
- Use verbs for actions

### Command Documentation
- Always provide examples
- Document all parameters
- Include common pitfalls
- Link to relevant specs

### Command Implementation
- Use existing scripts where possible
- Keep commands focused
- Handle errors gracefully
- Provide clear feedback

## Integration with Workflow

Commands integrate with the development workflow:

1. **Issue Selection**: Browse issues, select priority
2. **Start Work**: `/start-task <issue>`
3. **Research**: `/create-spec-issue` if spec missing
4. **Implement**: Use golang-pro skill (TDD)
5. **Quality Check**: `/run-quality-gates`
6. **Save Progress**: `/checkpoint save`
7. **Debug**: `/debug-issue` if errors
8. **Create PR**: Automated workflow
9. **Review**: Code review loop
10. **Merge**: After approval

## Related Documentation

- **Root CLAUDE.md**: Project-wide instructions
- **.claude/agents/**: Agent definitions
- **.claude/skills/**: Reusable skills
- **docs/standards/**: Development standards
- **scripts/**: Automation scripts

## Token Efficiency

Commands support token efficiency by:
- **Clear syntax**: Minimal explanation needed
- **Automation**: Reduce manual steps
- **Checkpointing**: Save context efficiently
- **References**: Link to specs, don't duplicate

## Troubleshooting

### Command Not Found
- Ensure `.claude/commands/<name>.md` exists
- Restart Claude Code to reload commands
- Check for syntax errors in command file

### Command Fails
- Check command documentation for requirements
- Verify parameters are correct
- Check related scripts exist
- Review error messages

### Command Slow
- Use `background = true` in settings.toml for long-running commands
- Break into smaller commands if appropriate
- Optimize underlying scripts

## Questions?

If you have questions about commands:
1. Check command documentation (`.claude/commands/<name>.md`)
2. Check this README for usage patterns
3. Review root CLAUDE.md for workflow context
4. Ask before proceeding

## Summary

**Commands provide**:
- Quick access to common workflows
- Automation of repetitive tasks
- Integration with contextd MCP tools
- Consistency across team

**Remember**:
- Type `/` to see all commands
- Read command docs before using
- Use automation hooks for efficiency
- Create new commands for repeated tasks
