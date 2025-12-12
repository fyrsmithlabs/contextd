# Issue: Secret scrubbing ineffective for Claude Code built-in tools (architectural limitation)

**To be created on GitHub when token permissions allow**

## Summary

PostToolUse hooks cannot prevent secrets from entering Claude's context when using built-in tools (Read, Bash, Grep, WebFetch). The scrubbing only affects what's displayed back to the user, not what Claude sees.

## Current Behavior

```
1. User asks Claude to read .mcp.json
2. Claude calls Read tool
3. Tool returns file content WITH secrets
4. Claude receives full content (secrets in context) ← PROBLEM
5. PostToolUse hook fires, runs `ctxd scrub -`
6. Scrubbed output shown to user
7. But Claude already saw the secrets and may reference them
```

## Evidence

In session on 2025-12-11, a GitHub PAT was exposed **4 times** because:
- Read tool showed the raw `.mcp.json` content to Claude
- Claude then referenced the token in subsequent responses
- PostToolUse scrubbing only affected user-visible output

## Technical Analysis

### Hook Lifecycle (from Claude Code docs)

| Hook | When it Fires | Can Modify |
|------|---------------|------------|
| **PreToolUse** | Before tool executes | Tool **input** only (via `updatedInput`) |
| **PostToolUse** | After tool executes | Cannot modify output, only `additionalContext` |

### Why Current Approach Fails

PostToolUse hooks run after the tool response is already in Claude's conversation context. The `ctxd scrub -` command:
- ✅ Scrubs secrets from user-visible output
- ❌ Does NOT scrub secrets from Claude's context
- ❌ Cannot prevent Claude from referencing secrets it already saw

### PreToolUse Limitations

PreToolUse could block sensitive file reads entirely, but:
- Cannot modify tool output (no output exists yet)
- Binary allow/deny - no transformation capability
- Would break legitimate use cases (checking if `.env` exists)

## Proposed Solutions

### Option A: Sensitive File Blocking (PreToolUse)

Block reads of known sensitive files entirely:

```json
{
  "PreToolUse": [
    {
      "matcher": "Read",
      "hooks": [
        {
          "type": "command",
          "command": "ctxd check-sensitive --file \"$TOOL_INPUT_FILE_PATH\""
        }
      ]
    }
  ]
}
```

**Pros:** Prevents exposure completely
**Cons:** Can't read any part of sensitive files, even safely

### Option B: Request Claude Code Architecture Change

Request Anthropic add a hook point that can **transform tool output before Claude sees it**:

```
PreToolUse → Tool Executes → [NEW: OutputTransform] → Claude Receives → PostToolUse
```

This would require changes to Claude Code itself.

### Option C: MCP Proxy Layer

Create an MCP server that wraps built-in tools and scrubs before returning:

```
Claude → contextd-proxy MCP → Calls built-in tools → Scrubs → Returns to Claude
```

**Pros:** Works with current architecture
**Cons:** Significant complexity, duplicates tool implementations

### Option D: Custom Read/Bash MCP Tools

Implement `contextd_read`, `contextd_bash` etc. that scrub internally:

```go
func (s *Server) handleContextdRead(ctx context.Context, input ReadInput) (string, error) {
    content, err := os.ReadFile(input.Path)
    if err != nil {
        return "", err
    }
    scrubbed := s.scrubber.Scrub(string(content))
    return scrubbed, nil
}
```

**Pros:** Full control over output
**Cons:** Users must use contextd tools instead of built-in

## Current Workaround

Secret scrubbing **does work** for contextd's own MCP tools:
- `memory_search` - responses are scrubbed
- `repository_search` - responses are scrubbed
- `checkpoint_resume` - responses are scrubbed

But **does NOT work** for built-in Claude Code tools:
- `Read` - secrets visible to Claude
- `Bash` - secrets visible to Claude
- `Grep` - secrets visible to Claude
- `WebFetch` - secrets visible to Claude

## Recommendations

1. **Immediate**: Update documentation to clarify limitation
2. **Short-term**: Implement PreToolUse blocking for known sensitive file patterns
3. **Medium-term**: Consider Option D (custom MCP tools with built-in scrubbing)
4. **Long-term**: Engage with Anthropic on Option B (architecture change)

## Related

- Memory recorded: `7346b305-24bc-4a5a-9fb0-259103e62097` (CRITICAL: Read tool exposes secrets before scrubbing hooks can intercept)

## Labels

- `bug`
- `security`
- `architecture`
- `documentation`
