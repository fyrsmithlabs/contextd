# Development Patterns

**Parent**: [Architecture Standards](../architecture.md)

This document describes how to extend and modify the contextd architecture.

---

## Adding New Endpoints

1. Define handler function in appropriate package
2. Add route in `cmd/contextd/main.go` setupRoutes()
3. Use `api.Group()` for authenticated endpoints
4. Public endpoints go directly on `e` (Echo instance)

**Example**:
```go
// In pkg/mypackage/handlers.go
func (h *Handler) HandleRequest(c echo.Context) error {
    // Handler implementation
}

// In cmd/contextd/main.go setupRoutes()
e.POST("/api/v1/myendpoint", myHandler.HandleRequest)
```

---

## Adding New Package

Follow standard Go layout:
```
pkg/
  newpackage/
    newpackage.go      # Public API
    internal.go        # Internal helpers (optional)
    newpackage_test.go # Tests
```

**Checklist**:
1. Create package directory
2. Define public API in `<package>.go`
3. Write tests in `<package>_test.go`
4. Add godoc comments for all exported types/functions
5. Update pkg/CLAUDE.md if package introduces new patterns

---

## Configuration Changes

1. Add to `pkg/config/config.go` Config struct
2. Add to Load() function with getEnv() helper
3. Document in architecture.md and README

**Example**:
```go
// In pkg/config/config.go
type Config struct {
    // ... existing fields ...
    MyNewSetting string
}

func Load() *Config {
    return &Config{
        // ... existing fields ...
        MyNewSetting: getEnv("MY_NEW_SETTING", "default-value"),
    }
}
```

---

## Middleware Order

**Current order (DO NOT CHANGE without reason):**
1. Logger - Must be first to log everything
2. Recover - Catch panics early
3. RequestID - Generate ID for correlation
4. otelecho - OTEL instrumentation
5. Route-specific (e.g., auth for /api/v1/*)

**Why this order**:
- Logger first ensures all requests are logged (including panics)
- Recover second catches panics before they escape
- RequestID third provides correlation ID for logs and traces
- OTEL fourth instruments with request ID available
- Route-specific last applies only to specific routes

---

## Error Handling Patterns

### Service Layer

```go
func (s *Service) Operation(ctx context.Context, input Input) (Output, error) {
    // Validate input
    if err := input.Validate(); err != nil {
        return Output{}, fmt.Errorf("invalid input: %w", err)
    }

    // Create span for tracing
    ctx, span := tracer.Start(ctx, "operation")
    defer span.End()

    // Perform operation
    result, err := s.store.DoSomething(ctx, input)
    if err != nil {
        span.RecordError(err)
        return Output{}, fmt.Errorf("failed to do something: %w", err)
    }

    // Return success
    return result, nil
}
```

### Handler Layer

```go
func (h *Handler) HandleRequest(c echo.Context) error {
    // Parse input
    var input Input
    if err := c.Bind(&input); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
    }

    // Call service
    output, err := h.service.Operation(c.Request().Context(), input)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    // Return response
    return c.JSON(http.StatusOK, output)
}
```

---

## Testing Strategy

### Unit Tests

- **Coverage Target**: ≥80% overall
- **Critical Paths**: 100% coverage
- **Location**: `*_test.go` files alongside implementation

### Integration Tests

- **Scope**: Service layer + vector store
- **Setup**: In-memory or test Qdrant instance
- **Teardown**: Cleanup test data

### End-to-End Tests

- **Scope**: Full request/response cycle
- **Setup**: Test server with test socket
- **Cleanup**: Remove test socket and data

**See:** `docs/standards/testing-standards.md` for complete testing requirements
