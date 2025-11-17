# OpenCode Integration

OpenCode is an AI coding agent that can automatically fix issues, implement features, and assist with development tasks.

## Quick Start

### Triggering OpenCode

**Manual Trigger** (comment on PR or issue):
```
/oc help with this implementation
/opencode fix the test failures
```

**Important**: Use comments on the **Conversation** tab of PRs, not inline code review comments. The `issue_comment` event works for both issues and PR conversation comments, but NOT for inline review comments in the "Files changed" tab.

**Label Trigger**:
- Add the `ai:opencode` label to any issue
- OpenCode will automatically research and propose solutions
- After completion, the issue is automatically labeled `ai:needs-spec`
- Great for batch processing or scheduled tasks
- Note: PR label triggers are not supported by opencode

**Automatic Trigger**:
- Failed check runs automatically trigger OpenCode
- Failed TDD or Go Test Matrix workflows trigger auto-fix attempts

### Using Claude Agents

OpenCode has access to specialized Claude agents matching our development workflow:

```
@go-engineer implement the hybrid search feature
@golang-reviewer review this PR for security issues
@qa-engineer run comprehensive tests on this feature
@security-auditor audit this code for vulnerabilities
@performance-engineer optimize this database query
@qdrant-specialist help with vector search configuration
@documentation-engineer update the API documentation
```

## Available Agents

### Implementation Agents (Full Access)

**@go-engineer**
- Expert Go developer following TDD and contextd standards
- Can write, edit code, and run bash commands
- Use for: Feature implementation, bug fixes, refactoring

**@qa-engineer**
- Comprehensive testing specialist
- Executes test skills and creates test cases
- Can write tests and run test commands
- Use for: Test creation, test skill execution, QA validation

**@qdrant-specialist**
- Qdrant vector database expert
- Can modify Qdrant schemas and run operations
- Use for: Vector search optimization, collection management

- Can design schemas and optimize performance

**@documentation-engineer**
- Technical documentation specialist
- Can create and update documentation (no bash)
- Use for: API docs, developer guides, README updates

### Analysis Agents (Read-Only)

**@golang-reviewer**
- Security and performance code reviewer
- Read-only analysis with bash for investigation
- Use for: Code review, security audits, pattern analysis

**@security-auditor**
- Security vulnerability specialist
- Read-only with investigation tools
- Use for: Security audits, compliance checks, threat analysis

**@performance-engineer**
- Performance optimization expert
- Read-only with profiling tools
- Use for: Bottleneck identification, performance analysis

## Configuration

### opencode.json

Located at the repository root, this file configures:
- Instructions: References to CLAUDE.md files
- Agents: Custom agent definitions with prompts and permissions
- Tools: Which tools each agent can access

Example agent configuration:
```json
{
  "agent": {
    "go-engineer": {
      "description": "Expert Go developer for implementing features",
      "mode": "subagent",
      "model": "anthropic/claude-sonnet-4-20250514",
      "prompt": "{file:./.claude/agents/go-engineer.md}",
      "tools": {
        "write": true,
        "edit": true,
        "bash": true
      }
    }
  }
}
```

### Workflow Triggers

The `.github/workflows/opencode.yml` workflow triggers on:

1. **Issue Comments**: Comments containing `/oc` or `/opencode` (works on both issues and PRs)
2. **Label Added**: `ai:opencode` label added to issues (triggers research → labels `ai:needs-spec`)
3. **Failed Check Runs**: Automatically attempts to fix failures
4. **Failed Workflows**: Auto-fix for TDD and Go Test Matrix failures

**Note**: PR review comments and pull_request events are not supported by opencode. Use regular PR comments instead (comment on the "Conversation" tab, not in the "Files changed" review).

### PR Code Review Issues

**Current Limitation**: OpenCode cannot automatically fix PR code review issues because it doesn't support `pull_request_review` or `pull_request_review_comment` events.

**Workarounds**:
- Use `/oc fix this review comment` in PR conversation comments
- Manually trigger OpenCode on specific PRs using issue comments
- Request support for PR events in [OpenCode issue #2152](https://github.com/sst/opencode/issues/2152)

### Required Secrets

- `OPENCODE_API_KEY`: OpenCode API key for authentication
- `GITHUB_TOKEN`: Automatically provided, used for GitHub API access

## Workflow Integration

### With TDD Enforcement

When TDD workflow fails:
1. OpenCode automatically triggers
2. Analyzes test failures
3. Can invoke @go-engineer to fix issues
4. Pushes fixes to the branch
5. Triggers re-run of tests

### With Code Review

In PR comments:
```
/oc @golang-reviewer review this PR for security
/oc @performance-engineer analyze this database query
```

OpenCode will:
1. Invoke the specified agent
2. Analyze the code
3. Post review comments
4. Suggest improvements

### With Documentation

```
/oc @documentation-engineer update API docs for the new endpoint
```

OpenCode will:
1. Invoke documentation-engineer agent
2. Generate or update documentation
3. Commit changes to the branch

## Best Practices

### Effective Agent Usage

1. **Be Specific**: `@go-engineer fix the race condition in pkg/vectorstore/hybrid_search.go`
2. **One Task Per Request**: Don't combine multiple unrelated tasks
3. **Reference Files**: Include file paths for context
4. **Use Appropriate Agent**: Match agent expertise to task

### Security Considerations

- Read-only agents (reviewer, auditor, performance) cannot modify code
- Implementation agents require explicit invocation
- All changes are committed with clear attribution
- Review auto-fixes before merging

### Performance Tips

- Use specific agents to avoid context switching
- Provide file paths to reduce search time
- Reference existing tests/docs for consistency
- Let auto-fix handle simple test failures

## Troubleshooting

### OpenCode Not Triggering

**Check**:
1. Comment starts with `/oc` or `/opencode` (or has space before)
2. Workflow has proper permissions in repo settings
3. `OPENCODE_API_KEY` secret is set
4. Workflow file is on the target branch

### Agent Not Found

**Check**:
1. Agent name matches opencode.json exactly
2. Agent .md file exists in `.claude/agents/`
3. opencode.json is valid JSON
4. File path in prompt is correct

### Permission Errors

**Check**:
1. Workflow has `contents: write` permission
2. Branch protection rules allow bot commits
3. `GITHUB_TOKEN` has appropriate scopes
4. Agent has required tool permissions

## Examples

### Fix Test Failure
```
/oc the test_hybrid_search is failing - fix it
```

### Implement Feature
```
/oc @go-engineer implement BM25 scoring as described in issue #123
```

### Code Review
```
/oc @golang-reviewer check for race conditions in the new checkpoint code
```

### Security Audit
```
/oc @security-auditor audit the authentication middleware
```

### Performance Analysis
```
```

### Update Documentation
```
/oc @documentation-engineer add API examples for the new remediation endpoint
```

### Research and Spec Creation
- Label any issue with `ai:opencode`
- OpenCode automatically researches the problem and proposes solutions
- Issue is then labeled `ai:needs-spec` for specification work

## Integration with Claude Code

OpenCode agents use the same prompt files as Claude Code agents (`.claude/agents/*.md`), ensuring:
- Consistent behavior across tools
- Shared knowledge base
- Unified development patterns
- Same quality standards

Agents follow all CLAUDE.md guidelines and contextd development standards.

## Further Reading

- [OpenCode Documentation](https://opencode.ai/docs/)
- [Claude Code Integration](https://docs.claude.com/en/docs/claude-code/)
- [Project CLAUDE.md](../../CLAUDE.md)
- [Agent Definitions](.claude/agents/)
