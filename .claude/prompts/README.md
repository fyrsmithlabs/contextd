# Claude Workflow Prompts

Pre-defined prompts for GitHub Actions workflows using the Anthropic Claude Code action.

## Overview

These prompt files provide reusable, maintainable instructions for Claude in GitHub Actions workflows. Instead of embedding large prompts in workflow YAML files, workflows reference these files and substitute variables.

## Available Prompts

| Prompt File | Purpose | Used By Workflow |
|-------------|---------|------------------|
| `code-review.md` | Automated code review | `claude-code-review.yml` |
| `auto-development.md` | Automated feature implementation | `auto-development.yml` |
| `spec-creation.md` | Create feature specifications | `spec-creation.yml` |
| `issue-grooming.md` | Issue triage and organization | `issue-grooming.yml` |
| `tdd-enforcement.md` | Enforce TDD and coverage standards | `tdd-enforcement.yml` |
| `claude-generic.md` | General purpose @claude interactions | `claude.yml` |

## Template Variables

Prompts use `{{ variable }}` syntax for workflow substitution:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{ repository }}` | GitHub repository | `axyzlabs/contextd` |
| `{{ pr_number }}` | Pull request number | `42` |
| `{{ issue_number }}` | Issue number | `123` |
| `{{ spec_path }}` | Path to spec file | `docs/specs/auth/SPEC.md` |
| `{{ feature_name }}` | Feature name | `authentication` |
| `{{ dry_run }}` | Dry run flag | `true`/`false` |
| `{{ event_type }}` | GitHub event type | `issue_comment` |
| `{{ trigger_context }}` | Event context | `@claude mentioned` |

## Usage in Workflows

### Method 1: Direct File Reference (Preferred)

```yaml
- name: Run Claude Code Review
  uses: anthropics/claude-code-action@v1
  with:
    claude_code_oauth_token: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}
    prompt: |
      ${{
        format(
          readfile('.claude/prompts/code-review.md'),
          github.repository,
          github.event.pull_request.number
        )
      }}
```

### Method 2: Variable Substitution in Workflow

```yaml
- name: Prepare prompt
  id: prompt
  run: |
    PROMPT=$(cat .claude/prompts/code-review.md)
    PROMPT="${PROMPT//\{\{ repository \}\}/${{ github.repository }}}"
    PROMPT="${PROMPT//\{\{ pr_number \}\}/${{ github.event.pull_request.number }}}"
    echo "content<<EOF" >> $GITHUB_OUTPUT
    echo "$PROMPT" >> $GITHUB_OUTPUT
    echo "EOF" >> $GITHUB_OUTPUT

- name: Run Claude
  uses: anthropics/claude-code-action@v1
  with:
    claude_code_oauth_token: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}
    prompt: ${{ steps.prompt.outputs.content }}
```

### Method 3: Inline with Substitution

```yaml
- name: Run Claude
  uses: anthropics/claude-code-action@v1
  with:
    claude_code_oauth_token: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}
    prompt: |
      $(cat .claude/prompts/code-review.md | \
        sed 's/{{ repository }}/${{ github.repository }}/g' | \
        sed 's/{{ pr_number }}/${{ github.event.pull_request.number }}/g')
```

## Benefits

### ✅ Maintainability
- Update prompts in one place
- Version control prompt changes
- Easy to review and improve

### ✅ Reusability
- Share prompts across workflows
- Consistent behavior across jobs
- DRY (Don't Repeat Yourself)

### ✅ Readability
- Workflows stay clean and focused
- Prompts are well-documented
- Easy to understand what Claude does

### ✅ Testability
- Test prompts independently
- Validate variable substitution
- Iterate without modifying workflows

## Creating New Prompts

1. **Create file** in `.claude/prompts/`
2. **Use template variables** with `{{ variable }}` syntax
3. **Document the prompt** in this README
4. **Reference agent files** for specialized behavior
5. **Test substitution** locally before deploying

### Template Structure

```markdown
# [Prompt Name] Prompt

You are [role description].

## Context
- Repository: {{ repository }}
- [Other context variables]

## Your Role
Read and follow instructions in `.claude/agents/[agent-name].md`.

## Tasks
1. [Step 1]
2. [Step 2]
...

## Output
[Expected output description]
```

## Testing Prompts

Test variable substitution locally:

```bash
# Test code-review prompt
cat .claude/prompts/code-review.md | \
  sed 's/{{ repository }}/dahendel\/contextd/g' | \
  sed 's/{{ pr_number }}/42/g'

# Verify all variables substituted
cat .claude/prompts/code-review.md | \
  sed 's/{{ repository }}/dahendel\/contextd/g' | \
  sed 's/{{ pr_number }}/42/g' | \
  grep '{{'  # Should return nothing
```

## Best Practices

1. **Keep prompts focused** - One clear purpose per prompt
2. **Reference agent files** - Leverage existing agent definitions
3. **Use project documentation** - Reference CLAUDE.md and docs/
4. **Provide context** - Always include repository and event info
5. **Clear outputs** - Specify exactly what Claude should produce
6. **Variable naming** - Use descriptive, consistent variable names
7. **Documentation** - Update this README when adding prompts

## Troubleshooting

### Variables not substituting
- Check variable syntax: `{{ variable }}` with spaces
- Verify workflow substitution step
- Test locally first

### Prompt too long
- Break into multiple steps
- Reference external docs instead of inlining
- Use agent files for complex instructions

### Claude not following prompt
- Make instructions more specific
- Reference relevant agent files
- Add examples of expected output

## Related Documentation

- [GitHub Actions Workflow Documentation](../../.github/workflows/CLAUDE.md)
- [Project Agents](../.claude/agents/README.md)
- [Project Standards](../../docs/standards/)
- [Claude Code Action Docs](https://github.com/anthropics/claude-code-action)
