---
name: contextd-pkg-api
description: Use when working with MCP tools, HTTP handlers, or middleware in contextd API packages (pkg/mcp, pkg/handlers, pkg/middleware) - enforces JSON Schema for MCP tools, input validation at every API boundary, proper error handling, correct HTTP status codes, and critical middleware ordering patterns
---

# API Package Development (MCP, Handlers, Middleware)

## Overview

API code is the security boundary between external input and internal systems. Every API package MUST validate inputs, define schemas, handle errors properly, and follow established patterns. No exceptions for "internal APIs", "trusted input", or "MVP".

**Core Principle**: Validate at EVERY boundary, regardless of trust assumptions.

## When to Use This Skill

Use when:
- Creating or modifying MCP tools (pkg/mcp)
- Implementing HTTP handlers (pkg/handlers)
- Adding middleware (pkg/middleware)
- Receiving API-related code review feedback
- Debugging input validation issues
- Setting up request/response patterns

## API Package Types

### MCP Tools (pkg/mcp)

JSON-RPC tools for Claude Code integration via stdio transport.

**Mandatory Requirements**:
- ✅ JSON Schema definition for input (REQUIRED, not optional)
- ✅ Typed input/output structs (no map[string]interface{})
- ✅ Input validation before processing
- ✅ Context propagation through service calls
- ✅ Wrapped errors with context
- ✅ Tool name: lowercase with underscores (e.g., "checkpoint_save")

### HTTP Handlers (pkg/handlers)

Echo framework handlers for REST API endpoints.

**Mandatory Requirements**:
- ✅ Check Bind() errors (ALWAYS)
- ✅ Validate input after binding
- ✅ Proper HTTP status codes (200, 201, 400, 401, 404, 500)
- ✅ Use echo.NewHTTPError for errors
- ✅ Return c.JSON() for success responses
- ✅ Context propagation to services

### Middleware (pkg/middleware)

Request processing pipeline components.

**Mandatory Requirements**:
- ✅ Correct order: Logger → Recover → RequestID → OTEL → Route-specific
- ✅ Return errors (don't panic)
- ✅ Use typed context keys
- ✅ Document order rationale

## MCP Tool Checklist

**ALL items REQUIRED, not optional suggestions.**

**BEFORE writing MCP tool code**:

- [ ] Define typed input struct with json tags
- [ ] Define typed output struct
- [ ] Add JSON Schema with jsonschema tags (REQUIRED by MCP protocol)
- [ ] Validate ALL fields (required AND optional) - check type, format, range, length
- [ ] Propagate context through service calls (ALWAYS, even for fast operations)
- [ ] Wrap errors with context (specific messages, not generic)
- [ ] Write tests for valid input, invalid input, error cases

**Example - GOOD**:

```go
// Input with JSON Schema
type CheckpointSaveInput struct {
    Summary     string `json:"summary" jsonschema:"required"`
    ProjectPath string `json:"project_path" jsonschema:"required"`
    Content     string `json:"content" jsonschema:"required"`
}

// Output struct
type CheckpointSaveOutput struct {
    ID        string `json:"id"`
    CreatedAt string `json:"created_at"`
}

func (t *Tools) CheckpointSave(ctx context.Context, input CheckpointSaveInput) (*CheckpointSaveOutput, error) {
    // Validate input
    if input.Summary == "" {
        return nil, fmt.Errorf("summary required")
    }
    if input.ProjectPath == "" {
        return nil, fmt.Errorf("project_path required")
    }

    // Call service with context
    checkpoint, err := t.service.Save(ctx, &checkpoint.Checkpoint{
        Summary:     input.Summary,
        ProjectPath: input.ProjectPath,
        Content:     input.Content,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to save checkpoint: %w", err)
    }

    return &CheckpointSaveOutput{
        ID:        checkpoint.ID,
        CreatedAt: checkpoint.CreatedAt.Format(time.RFC3339),
    }, nil
}
```

**Example - WRONG** (These are COMMON mistakes in production code, not rare edge cases):

```go
// ❌ No schema, untyped input/output, no validation
func (t *Tools) CheckpointSave(input map[string]interface{}) (interface{}, error) {
    summary := input["summary"].(string) // unsafe type assertion
    checkpoint, err := t.service.Save(summary) // no context
    if err != nil {
        return nil, err // no error wrapping
    }
    return checkpoint, nil
}
```

## HTTP Handler Checklist

**ALL items REQUIRED, not optional suggestions.**

**BEFORE writing handler code**:

- [ ] Define request struct with validation tags
- [ ] Define response struct
- [ ] Check Bind() error (ALWAYS, Bind() fails for malformed JSON, wrong content-type, size limits)
- [ ] Validate request fields (specific error messages: which field, why invalid)
- [ ] Use proper HTTP status code constants (not hardcoded 200)
- [ ] Use echo.NewHTTPError for errors (not direct error return)
- [ ] Propagate context to service (ALWAYS, even for fast operations)
- [ ] Write tests for valid request, invalid request, service errors

**Example - GOOD**:

```go
type CreateCheckpointRequest struct {
    Summary     string `json:"summary" validate:"required"`
    ProjectPath string `json:"project_path" validate:"required"`
}

func (h *Handler) CreateCheckpoint(c echo.Context) error {
    var req CreateCheckpointRequest

    // Check Bind() error
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }

    // Validate
    if req.Summary == "" {
        return echo.NewHTTPError(http.StatusBadRequest, "summary required")
    }
    if req.ProjectPath == "" {
        return echo.NewHTTPError(http.StatusBadRequest, "project_path required")
    }

    // Call service with context
    checkpoint, err := h.service.Save(c.Request().Context(), &req)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    // Return with proper status code
    return c.JSON(http.StatusCreated, checkpoint)
}
```

**Example - WRONG** (These are COMMON mistakes in production code, not rare edge cases):

```go
// ❌ No error check, no validation, wrong status code, direct error return
func (h *Handler) CreateCheckpoint(c echo.Context) error {
    var req CreateCheckpointRequest
    c.Bind(&req) // unchecked error

    checkpoint, err := h.service.Save(c.Request().Context(), &req)
    if err != nil {
        return err // should use echo.NewHTTPError
    }

    return c.JSON(200, checkpoint) // should be http.StatusCreated
}
```

## Middleware Order (CRITICAL)

**Middleware order is NOT optional. Framework does NOT optimize automatically.**

**Correct Order** (DO NOT CHANGE):

```go
// 1. Logger - MUST be first to log all requests
e.Use(middleware.Logger())

// 2. Recover - Catch panics early
e.Use(middleware.Recover())

// 3. RequestID - Generate correlation ID
e.Use(middleware.RequestID())

// 4. OTEL - Instrumentation (needs RequestID)
e.Use(otelecho.Middleware("contextd"))

// 5. Route-specific (e.g., Auth for /api/v1/*)
api := e.Group("/api/v1")
api.Use(authMiddleware)
```

**Why Order Matters**:
- Logger first → logs all requests including panics
- Recover second → catches panics before they kill server
- RequestID third → provides correlation for OTEL
- OTEL fourth → traces include request ID
- Auth last → only for protected routes

**WRONG Order** (DO NOT USE):

```go
// ❌ Auth before Logger → unauthenticated requests not logged
e.Use(authMiddleware)
e.Use(middleware.Logger())

// ❌ OTEL before RequestID → traces lack correlation
e.Use(otelecho.Middleware("contextd"))
e.Use(middleware.RequestID())

// ❌ Recover before Logger → panics not logged
e.Use(middleware.Recover())
e.Use(middleware.Logger())
```

## HTTP Status Code Reference

| Code | Constant | When to Use |
|------|----------|-------------|
| 200 | StatusOK | Successful GET, PUT, DELETE |
| 201 | StatusCreated | Successful POST (resource created) |
| 400 | StatusBadRequest | Invalid input, validation failure |
| 401 | StatusUnauthorized | Missing or invalid auth token |
| 404 | StatusNotFound | Resource doesn't exist |
| 500 | StatusInternalServerError | Service error, unexpected failure |

**Use constants, not magic numbers**: `http.StatusCreated` not `201`

## Common Mistakes

### Mistake 1: "Schema is Optional for MCP Tools"

❌ **Wrong**: "MCP works without schema, we'll add it later"

✅ **Right**: JSON Schema is REQUIRED for MCP tools. It defines the contract, enables validation, and provides documentation. Add it BEFORE implementing tool logic.

### Mistake 2: "Internal API, No Validation Needed"

❌ **Wrong**: "Claude is trusted input, skip validation"

✅ **Right**: ALWAYS validate at API boundaries. Input can be malformed, assumptions change, and defense-in-depth requires validation at every layer.

### Mistake 3: "Service Layer Validates, Handler Doesn't Need To"

❌ **Wrong**: "Validation in service is enough"

✅ **Right**: Handler validates for HTTP semantics (400 Bad Request), service validates for business logic. Both are required.

### Mistake 4: "200 Works for Everything"

❌ **Wrong**: `return c.JSON(200, result)` for all success cases

✅ **Right**: Use proper status codes: 201 for creation, 200 for retrieval/update/delete.

### Mistake 5: "Middleware Order Doesn't Matter"

❌ **Wrong**: "Framework optimizes order automatically"

✅ **Right**: Order is CRITICAL. Logger must be first, Recover second, RequestID third, OTEL fourth, route-specific last.

### Mistake 6: "Bind() Rarely Fails, No Need to Check"

❌ **Wrong**: `c.Bind(&req)` without error check

✅ **Right**: ALWAYS check Bind() errors. Malformed JSON, wrong content type, and size limits all cause Bind() to fail.

## Red Flags - STOP and Fix Immediately (Do NOT Commit)

If you see these patterns, STOP and fix immediately:

- MCP tool without JSON Schema
- map[string]interface{} for MCP input/output
- Unchecked Bind() error in handler
- Missing input validation ("internal API" rationalization)
- Hardcoded HTTP status codes (200, 400, 500 as numbers)
- Direct error return instead of echo.NewHTTPError
- Middleware order different from documented pattern
- Generic error messages ("invalid input" without specifics)
- "We'll add validation later" or "MVP can skip schema"
- "Operation too fast for context" or "Tests prove validation unnecessary"

**All of these mean: Fix now, don't commit. No exceptions.**

## Testing Requirements

**Every API component MUST have**:

- Unit tests for valid input
- Unit tests for invalid input (validation)
- Unit tests for service errors
- Integration tests for full request/response cycle
- Table-driven tests for multiple scenarios

**Example Test**:

```go
func TestCheckpointSave_ValidInput_Success(t *testing.T) {
    tool := setupTestTool(t)
    input := CheckpointSaveInput{
        Summary:     "Test checkpoint",
        ProjectPath: "/test/project",
        Content:     "Test content",
    }

    output, err := tool.CheckpointSave(context.Background(), input)

    assert.NoError(t, err)
    assert.NotEmpty(t, output.ID)
}

func TestCheckpointSave_EmptySummary_ReturnsError(t *testing.T) {
    tool := setupTestTool(t)
    input := CheckpointSaveInput{
        Summary:     "", // invalid
        ProjectPath: "/test/project",
        Content:     "Test content",
    }

    output, err := tool.CheckpointSave(context.Background(), input)

    assert.Error(t, err)
    assert.Nil(t, output)
    assert.Contains(t, err.Error(), "summary required")
}
```

## Integration with Other Skills

**Before completing API work**:
- Use contextd:completing-major-task for verification template
- Use contextd:code-review before creating PR
- Use contextd:pkg-security if working with auth or session middleware

**For multi-package work**:
- Use contextd:pkg-storage if API calls storage layer
- Use contextd:pkg-ai if API uses embeddings or search

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "Schema is optional for MCP tools" | JSON Schema is REQUIRED by MCP protocol, not optional documentation |
| "Internal API, input is trusted" | ALWAYS validate at boundaries, regardless of trust |
| "Service layer validates anyway" | Defense-in-depth requires validation at EVERY layer |
| "Bind() rarely fails in practice" | Malformed input, wrong content-type, size limits all cause failures |
| "200 works fine for creation" | HTTP semantics: 201 for creation, 200 for other success |
| "Framework optimizes middleware order" | Order is CRITICAL and framework does NOT reorder |
| "We're 90% done, validation is polish" | Validation is REQUIRED, not optional polish |
| "MVP can skip quality gates" | Quality gates are MANDATORY, even for MVP |
| "We'll add schema during polish phase" | Schema is part of implementation, not polish |
| "Validation adds overhead" | Validation is microseconds, not milliseconds. Security > marginal performance |
| "Tests prove validation unnecessary" | Tests validate behavior, runtime validates input. Both required |
| "OpenAPI/Proto schema covers this" | MCP requires JSON Schema in tool definition. OpenAPI/Proto are separate |
| "Optional fields don't need validation" | Optional ≠ unvalidated. Validate ALL fields (type, format, range) |
| "Generic errors are good enough" | Specific errors required (which field, why). Generic errors hide root cause |
| "Operation too fast for context" | Context provides tracing, cancellation, values - not just timeouts. ALWAYS propagate |

## Summary

**API Package Development Rules**:

1. ✅ ALWAYS define JSON Schema for MCP tools
2. ✅ ALWAYS use typed structs (no map[string]interface{})
3. ✅ ALWAYS validate input at API boundary
4. ✅ ALWAYS check Bind() errors
5. ✅ ALWAYS use proper HTTP status codes
6. ✅ ALWAYS use echo.NewHTTPError for errors
7. ✅ ALWAYS follow middleware order: Logger → Recover → RequestID → OTEL → Route-specific
8. ✅ ALWAYS propagate context
9. ✅ ALWAYS wrap errors with context
10. ✅ ALWAYS write tests for valid, invalid, and error cases

**No exceptions for "internal APIs", "trusted input", "MVP", or "time pressure".**
