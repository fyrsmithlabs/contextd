# Debug Issue

**Command**: `/debug-issue "<description>" [--file <path>]`

**Description**: AI-assisted debugging with research-analyst integration for rapid issue resolution.

**Usage**:
```
/debug-issue "panic: runtime error"
/debug-issue "race condition in cache" --file pkg/cache/cache.go
/debug-issue "connection timeout" --context "Redis client initialization"
```

## Purpose

Follows user preference to use @agent-research-analyst for error resolution by:
- Researching the error across multiple sources
- Finding proven solutions from documentation and community
- Creating resolution specification documents
- Providing implementation guidance
- Building organizational knowledge base

## Agent Workflow

When this command is invoked, delegate to the research-analyst agent:

```
@agent-research-analyst search for solution to: [error description]

Context:
- Language: Go
- Error: [full error message]
- File: [file path if provided]
- Additional Context: [any context provided]

Please provide:
1. Root cause analysis
2. Proven solutions from official sources
3. Implementation steps
4. Prevention strategies
5. Related issues and patterns
```

## Expected Workflow

### 1. Research Phase
Research-analyst will:
- Search official Go documentation
- Review GitHub issues and discussions
- Check Stack Overflow and community forums
- Analyze similar resolved issues
- Consult package-specific documentation

### 2. Analysis Phase
Research-analyst will provide:
- **Root Cause**: What's causing the error
- **Impact**: Severity and scope
- **Solutions**: Ranked by reliability and complexity
- **Trade-offs**: Pros and cons of each approach

### 3. Resolution Spec Creation
Create a specification document:

**Location**: `docs/specs/resolutions/<error-name>.md`

**Template**:
```markdown
# Resolution: [Error Name]

**Date**: [YYYY-MM-DD]
**Severity**: [Critical|High|Medium|Low]
**Status**: [Identified|In Progress|Resolved]

## Problem Statement

[Clear description of the issue]

## Context

- **File**: [path]
- **Function/Method**: [name]
- **Error Message**: [full error]
- **When it occurs**: [conditions]

## Root Cause

[Technical explanation of why this happens]

## Solution

### Recommended Approach

[Detailed implementation steps]

```go
// Example code showing the fix
```

### Alternative Approaches

1. **Approach 1**: [Description]
   - Pros: [benefits]
   - Cons: [drawbacks]

2. **Approach 2**: [Description]
   - Pros: [benefits]
   - Cons: [drawbacks]

## Testing

### Test Cases

```go
// Tests to verify the fix
func TestErrorResolved(t *testing.T) {
    // Test implementation
}
```

### Verification Steps

1. [Step 1]
2. [Step 2]
3. [Step 3]

## Prevention

[How to prevent this in the future]

## References

- [Official documentation links]
- [GitHub issues]
- [Community discussions]
- [Related CVEs if applicable]

## Implementation Checklist

- [ ] Implement solution
- [ ] Add tests
- [ ] Update documentation
- [ ] Run quality gates
- [ ] Code review
- [ ] Deploy and monitor
```

### 4. Implementation Phase
After spec is created:
1. **DELEGATE to golang-pro** for all Go code fixes:
   ```
   Use the golang-pro skill to implement the resolution from docs/specs/resolutions/<error-name>.md
   ```
2. golang-pro will handle: implementation with TDD, tests, quality gates
3. Create PR with reference to resolution spec

## Example Usage

### Example 1: Panic Error
```bash
/debug-issue "panic: runtime error: invalid memory address or nil pointer dereference"

# Research-analyst will:
# 1. Identify common causes (nil pointer, uninitialized variable)
# 2. Search for Go-specific patterns
# 3. Provide nil-checking strategies
# 4. Create resolution spec
```

### Example 2: Race Condition
```bash
/debug-issue "race condition detected" --file pkg/cache/cache.go

# Research-analyst will:
# 1. Analyze concurrent access patterns
# 2. Review mutex/sync usage
# 3. Suggest proper synchronization
# 4. Provide race-free implementation
```

### Example 3: Performance Issue
```bash
/debug-issue "high memory usage in production" --context "After 24h uptime"

# Research-analyst will:
# 1. Investigate memory leak patterns
# 2. Review garbage collection
# 3. Suggest profiling approaches
# 4. Provide optimization strategies
```

## Integration with Development Workflow

### When to Use

1. **Compilation Errors**: Complex errors not immediately obvious
2. **Runtime Errors**: Panics, crashes, unexpected behavior
3. **Race Conditions**: Data races detected by `-race` flag
4. **Performance Issues**: Memory leaks, high CPU, slow responses
5. **Integration Issues**: Third-party package problems
6. **Test Failures**: Flaky or failing tests

### After Debugging

1. **Spec Created**: `docs/specs/resolutions/[error].md`
2. **Issue Created**: Link to resolution spec
3. **Implementation**: Follow resolution spec
4. **Tests Added**: Prevent regression
5. **Documentation Updated**: Add to knowledge base

## Benefits

### Short-term
- ✅ Faster error resolution
- ✅ Access to proven solutions
- ✅ Reduced debugging time
- ✅ Better root cause understanding

### Long-term
- ✅ Knowledge base of resolutions
- ✅ Pattern recognition for similar issues
- ✅ Team learning and growth
- ✅ Reduced recurrence of issues

## Success Criteria

- ✅ Research-analyst provides comprehensive analysis
- ✅ Resolution spec created with clear steps
- ✅ Solution successfully implemented
- ✅ Tests prevent regression
- ✅ Knowledge documented for future reference

## Example Resolution Spec

**File**: `docs/specs/resolutions/nil-pointer-cache.md`

```markdown
# Resolution: Nil Pointer in Cache Access

**Date**: 2025-10-23
**Severity**: High
**Status**: Resolved

## Problem Statement

Application panics with "runtime error: invalid memory address or nil pointer dereference" when accessing cache in pkg/cache/cache.go:45.

## Context

- **File**: pkg/cache/cache.go
- **Function**: Get()
- **Error**: panic: runtime error: invalid memory address or nil pointer dereference
- **When it occurs**: During concurrent access to cache

## Root Cause

Cache is not initialized before first use. Multiple goroutines attempt to access the cache map before initialization completes.

## Solution

### Recommended Approach

Use sync.Once to ensure single initialization:

```go
type Cache struct {
    once  sync.Once
    store map[string]interface{}
    mu    sync.RWMutex
}

func (c *Cache) init() {
    c.once.Do(func() {
        c.store = make(map[string]interface{})
    })
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.init()
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.store[key]
    return val, ok
}
```

## Testing

```go
func TestCacheConcurrent(t *testing.T) {
    cache := &Cache{}

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            cache.Get(fmt.Sprintf("key-%d", id))
        }(i)
    }
    wg.Wait()
}
```

## Prevention

- Always initialize maps before use
- Use sync.Once for thread-safe initialization
- Run tests with -race flag
- Document initialization requirements

## References

- https://go.dev/blog/race-detector
- https://pkg.go.dev/sync#Once
```

## Notes

- **User Preference**: Always use research-analyst for errors
- **Knowledge Building**: Each resolution adds to organizational knowledge
- **Prevention Focus**: Not just fixing, but preventing recurrence
- **Documentation**: Critical for team learning
