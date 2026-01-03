# Ralph Wiggum Loop Prompt: Issue #54

## Task: Optimize repository_search Response Size

Implement `content_mode` parameter for `repository_search` MCP tool to prevent context bloat.

---

## Context

**Issue:** #54 - Optimize repository_search response size to prevent context bloat

**Problem:** The `repository_search` tool returns complete file content for all results, consuming 15-20k tokens per 10 results. This rapidly exhausts agent context.

**Solution:** Add a `content_mode` parameter with three levels:
- `minimal` (default): File path, score, branch only (~100 tokens/result)
- `preview`: First 200 characters (~150 tokens/result)
- `full`: Complete content (~2k tokens/result)

---

## Files to Modify

1. `internal/mcp/tools.go` - Add ContentMode field and conditional response logic
2. `.claude-plugin/schemas/contextd-mcp-tools.schema.json` - Update schema
3. `docs/api/mcp-tools.md` - Document the new parameter (create if needed)
4. `internal/mcp/tools_repository_test.go` - Add tests for content_mode behavior

---

## Implementation Requirements

### 1. Update `repositorySearchInput` struct (tools.go:474-480)

Add field:
```go
ContentMode string `json:"content_mode,omitempty" jsonschema:"Content mode: minimal (default), preview, or full"`
```

### 2. Update response building logic (tools.go:634-647)

Replace unconditional content inclusion with conditional logic:
```go
// Determine content mode (default: minimal)
contentMode := args.ContentMode
if contentMode == "" {
    contentMode = "minimal"
}

outputResults := make([]map[string]interface{}, 0, len(results))
for _, r := range results {
    result := map[string]interface{}{
        "file_path": r.FilePath,
        "score":     r.Score,
        "branch":    r.Branch,
    }

    switch contentMode {
    case "full":
        scrubbedContent := s.scrubber.Scrub(r.Content).Scrubbed
        result["content"] = scrubbedContent
        result["metadata"] = r.Metadata
    case "preview":
        scrubbedContent := s.scrubber.Scrub(r.Content).Scrubbed
        preview := scrubbedContent
        if len(preview) > 200 {
            preview = preview[:200] + "..."
        }
        result["content_preview"] = preview
    // case "minimal": no content added
    }

    outputResults = append(outputResults, result)
}
```

### 3. Update `repositorySearchOutput` struct

Add content_mode to output:
```go
ContentMode string `json:"content_mode" jsonschema:"Content mode used"`
```

### 4. Update JSON Schema

Add content_mode to `repository_search_input` definition:
```json
"content_mode": {
  "type": "string",
  "enum": ["minimal", "preview", "full"],
  "default": "minimal",
  "description": "Content inclusion mode: minimal (path/score/branch only), preview (first 200 chars), full (complete content)"
}
```

Update `repository_search_output` results schema to reflect optional content fields.

### 5. Write Tests

Test each mode returns expected fields:
- `TestRepositorySearch_ContentMode_Minimal` - only file_path, score, branch
- `TestRepositorySearch_ContentMode_Preview` - includes content_preview (max 200 chars)
- `TestRepositorySearch_ContentMode_Full` - includes full content and metadata
- `TestRepositorySearch_ContentMode_Default` - empty string defaults to minimal

---

## TDD Workflow

1. **RED**: Write tests first that will fail
2. **GREEN**: Implement just enough to pass
3. **REFACTOR**: Clean up while keeping tests green

---

## Success Criteria

- [ ] `content_mode` parameter added to `repositorySearchInput`
- [ ] Default mode is `minimal` when not specified
- [ ] `minimal` mode returns only: file_path, score, branch
- [ ] `preview` mode returns: file_path, score, branch, content_preview (max 200 chars)
- [ ] `full` mode returns: file_path, score, branch, content, metadata
- [ ] JSON schema updated with new parameter and enum values
- [ ] All existing tests still pass
- [ ] New tests cover all three content modes
- [ ] `go test ./...` passes
- [ ] Documentation updated (if docs/api/mcp-tools.md exists)

---

## Completion Promise

When ALL success criteria are met and `go test ./...` passes, output:

```
<promise>ISSUE-54-COMPLETE</promise>
```

---

## Iteration Guidelines

Each iteration should:
1. Check current state of implementation
2. Run `go test ./internal/mcp/... -v` to see what's passing/failing
3. Make incremental progress toward success criteria
4. Document what was done in git commits
5. Only emit promise when ALL criteria are verified

Do NOT emit the promise until:
- All tests pass
- Schema is valid JSON
- Default behavior is minimal mode
