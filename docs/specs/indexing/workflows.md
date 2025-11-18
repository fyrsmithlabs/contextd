# Repository Indexing Workflows

**Parent**: [../SPEC.md](../SPEC.md)

## Indexing Workflow

### Phase 1: Validation

1. **Path validation**:
   ```go
   if _, err := os.Stat(input.Path); os.IsNotExist(err) {
       return ValidationError("path does not exist")
   }
   ```

2. **Pattern validation**:
   ```go
   for _, pattern := range input.IncludePatterns {
       if _, err := filepath.Match(pattern, "test"); err != nil {
           return ValidationError("invalid include pattern")
       }
   }
   ```

3. **Size validation**:
   ```go
   if input.MaxFileSize > 10*1024*1024 {
       return ValidationError("max_file_size too large")
   }
   ```

### Phase 2: Tree Traversal

Uses `filepath.Walk` for recursive directory traversal:

```go
err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
    // 1. Check for walk errors
    if err != nil {
        return err
    }

    // 2. Skip directories
    if info.IsDir() {
        return nil
    }

    // 3. Check file size
    if info.Size() > maxFileSize {
        return nil
    }

    // 4. Check exclude patterns
    for _, pattern := range excludePatterns {
        if matched, _ := filepath.Match(pattern, info.Name()); matched {
            return nil
        }
        if matched, _ := filepath.Match(pattern, path); matched {
            return nil
        }
    }

    // 5. Check include patterns (if specified)
    if len(includePatterns) > 0 {
        included := false
        for _, pattern := range includePatterns {
            if matched, _ := filepath.Match(pattern, info.Name()); matched {
                included = true
                break
            }
        }
        if !included {
            return nil
        }
    }

    // 6. Validate path within repository (security)
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("failed to resolve file path: %w", err)
    }
    absRepoPath, err := filepath.Abs(repoPath)
    if err != nil {
        return fmt.Errorf("failed to resolve repository path: %w", err)
    }
    if !strings.HasPrefix(absPath, absRepoPath) {
        return fmt.Errorf("path traversal detected: %s", path)
    }

    // 7. Read file contents
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    // 8. Create checkpoint
    relPath, _ := filepath.Rel(repoPath, path)
    req := &validation.CreateCheckpointRequest{
        Summary:     fmt.Sprintf("Indexed file: %s", relPath),
        Description: string(content),
        ProjectPath: repoPath,
        Context:     map[string]string{"indexed_file": relPath},
        Tags:        []string{"indexed", "repository", filepath.Ext(path)},
    }

    _, err = s.services.Checkpoint.Create(ctx, req)
    if err != nil {
        return err
    }

    filesIndexed++
    return nil
})
```

### Phase 3: Checkpoint Creation

For each file:

1. **Generate relative path**: `filepath.Rel(repoPath, path)`
2. **Build checkpoint request**:
   - Summary: `"Indexed file: <relative-path>"`
   - Description: Full file contents as string
   - Project path: Repository root (for database isolation)
   - Context: `{"indexed_file": "<relative-path>"}`
   - Tags: `["indexed", "repository", "<extension>"]`
3. **Call checkpoint service**: `s.services.Checkpoint.Create(ctx, req)`
4. **Increment counter**: `filesIndexed++`

### Phase 4: Return Results

```go
output := IndexRepositoryOutput{
    Path:            input.Path,
    FilesIndexed:    filesIndexed,
    IncludePatterns: input.IncludePatterns,
    ExcludePatterns: input.ExcludePatterns,
    MaxFileSize:     input.MaxFileSize,
    IndexedAt:       time.Now(),
}
```

---

## Usage Examples

### Example 1: Index all markdown files

Request:
```json
{
  "path": "/home/user/docs",
  "include_patterns": ["*.md"]
}
```

Response:
```json
{
  "path": "/home/user/docs",
  "files_indexed": 42,
  "include_patterns": ["*.md"],
  "exclude_patterns": [],
  "max_file_size": 1048576,
  "indexed_at": "2025-11-04T12:00:00Z"
}
```

### Example 2: Index code with exclusions

Request:
```json
{
  "path": "/home/user/project",
  "include_patterns": ["*.go", "*.js", "*.py"],
  "exclude_patterns": ["*_test.go", "node_modules/**", "__pycache__/**"],
  "max_file_size": 524288
}
```

Response:
```json
{
  "path": "/home/user/project",
  "files_indexed": 157,
  "include_patterns": ["*.go", "*.js", "*.py"],
  "exclude_patterns": ["*_test.go", "node_modules/**", "__pycache__/**"],
  "max_file_size": 524288,
  "indexed_at": "2025-11-04T12:05:00Z"
}
```

### Example 3: Validation error

Request:
```json
{
  "path": "/nonexistent/path"
}
```

Response (Error):
```json
{
  "error": "validation_error",
  "message": "path does not exist",
  "details": {
    "field": "path",
    "path": "/nonexistent/path"
  }
}
```

---

## CLI Command: `ctxd index`

The CLI provides a convenience wrapper for the MCP tool.

**Usage**:
```bash
ctxd index <path> [flags]
```

**Flags**:
- `--include <patterns>`: File patterns to include (repeatable)
- `--exclude <patterns>`: File patterns to exclude (repeatable)
- `--max-size <bytes>`: Maximum file size (default: 1048576)

**Examples**:
```bash
# Index all files
ctxd index /path/to/repo

# Index markdown files
ctxd index . --include "*.md"

# Index with exclusions
ctxd index /repo --include "*.go" --exclude "vendor/**" --exclude "*_test.go"

# Custom max size (512KB)
ctxd index /docs --max-size 524288
```

**Note**: The CLI currently only provides usage instructions and debugging commands. It does not directly invoke the MCP tool (requires contextd service running and MCP integration).
