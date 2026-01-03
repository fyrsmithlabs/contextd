# Ralph Wiggum Plugin for contextd

## What is Ralph Wiggum?

Ralph Wiggum is a Claude Code plugin that implements iterative task loops with completion promises. It's useful for complex multi-step tasks that require persistent iteration until specific success criteria are met.

## Installation

The ralph-wiggum plugin is available in the Claude Code plugin marketplace:

```bash
# Install via Claude Code CLI
claude plugins add claude-plugins-official/ralph-wiggum
```

Or install via Claude Code UI:
1. Open Claude Code settings
2. Navigate to Plugins
3. Search for "ralph-wiggum"
4. Click Install

## PR #12642 Fixes (Required)

**Important**: Ensure you have ralph-wiggum v1.1.0 or later, which includes the PR #12642 fixes for multi-line bash command issues.

### What Was Fixed

PR #12642 resolved critical issues:
- ✅ Multi-line bash scripts moved from slash commands to `setup-ralph-loop.sh`
- ✅ Removed problematic ` ```! ` (auto-execute) syntax
- ✅ Changed to ` ```bash ` (display) with explicit execution
- ✅ Fixed "Command contains newlines" security errors
- ✅ Fixed "requires approval" permission check bugs

### Verify You Have the Fix

After installing, run:
```
/ralph-wiggum:help
```

If you get newline errors, you have the old version. Update with:
```bash
claude plugins update ralph-wiggum
```

## Permissions Setup

Add to `.claude/settings.json` to allow ralph-wiggum scripts to run:

```json
{
  "permissions": {
    "allow": [
      "Bash:**/.claude/plugins/cache/*/scripts/*.sh *",
      "Bash:**/setup-ralph-loop.sh *",
      "Bash:**/ralph-wiggum/*/scripts/*"
    ],
    "defaultMode": "dontAsk"
  }
}
```

**Note**: This project's `.claude/settings.json` already includes these permissions.

## Usage with contextd

### Basic Loop

```bash
/ralph-wiggum:ralph-loop "Task description" --completion-promise "Task is complete" --max-iterations 10
```

### Example: Fix All Critical Issues

```bash
/ralph-wiggum:ralph-loop "Fix all 9 critical issues from #69" --completion-promise "All 9 critical issues fixed and tests pass" --max-iterations 20
```

### Example: Grafana Integration

```bash
/ralph-wiggum:ralph-loop "Create Grafana dashboards for contextd using @.claude/ralph-prompts/grafana-integration.md" --completion-promise "All dashboards created and tested" --max-iterations 15
```

## Completion Promise Rules

The loop will continue until the completion promise is TRUE:

✅ **DO**:
- Make promises specific and verifiable
- Use objective criteria (tests pass, files exist, etc.)
- Let the promise become true naturally

❌ **DON'T**:
- Output false promises to exit the loop
- Use vague promises like "Done" or "Finished"
- Circumvent the loop by lying

### Good Promises

```
"All tests in ./internal/vectorstore pass"
"README.md contains installation instructions"
"All 5 dashboards created and returning data"
```

### Bad Promises

```
"Done"
"Task complete"
"I'm finished"
```

## Canceling a Loop

If you need to stop a Ralph loop:

```bash
/ralph-wiggum:cancel-ralph
```

This safely exits the loop and cleans up the `.claude/ralph-loop.local.md` file.

## How It Works (Post-PR #12642)

1. **Loop Setup**: Ralph creates `.claude/ralph-loop.local.md` with task context
2. **Script Execution**: The `setup-ralph-loop.sh` script handles multi-line logic
3. **Display Instructions**: Commands use ` ```bash ` blocks to show what to run
4. **Claude Executes**: Claude explicitly runs the displayed commands
5. **Promise Checking**: After each iteration, checks if promise is true
6. **Auto-Cleanup**: Loop exits and cleans up when promise becomes true

## Troubleshooting

### "Command contains newlines" Error

**Cause**: You have ralph-wiggum version < 1.1.0 (before PR #12642)

**Fix**:
```bash
claude plugins update ralph-wiggum
```

### "Permission denied" for setup-ralph-loop.sh

**Cause**: Bash permissions not configured

**Fix**: Add permissions to `.claude/settings.json` (see Permissions Setup section above)

### Loop Not Exiting

**Symptom**: Loop continues even though task seems done

**Cause**: Completion promise is not ACTUALLY true yet

**Fix**:
1. Check what the promise requires
2. Verify all criteria are met (run tests, check files, etc.)
3. Don't force exit by lying - fix the underlying issue

## Integration with contextd

Ralph Wiggum works well with contextd features:

- **Memory Search**: Loop can search past strategies with `memory_search`
- **Checkpoints**: Save progress at each iteration with `checkpoint_save`
- **Remediation**: Record fixes with `remediation_record` as issues are resolved

### Example: Combined Workflow

```bash
# Start Ralph loop with contextd integration
/ralph-wiggum:ralph-loop "Implement feature X using contextd best practices" --completion-promise "Feature X complete, tests pass, documented" --max-iterations 15

# Inside the loop, Claude will:
# 1. Search memories for similar features
# 2. Work on implementation
# 3. Save checkpoints at key points
# 4. Record solutions to any errors encountered
# 5. Verify completion promise (tests + docs)
```

## References

- [PR #12642](https://github.com/anthropics/claude-code/pull/12642) - Multi-line bash fixes
- [Issue #12170](https://github.com/anthropics/claude-code/issues/12170) - Original bug report
- Ralph Wiggum Plugin: `claude-plugins-official/ralph-wiggum`
