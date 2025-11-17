# Internal Packages - CLAUDE.md

See [../CLAUDE.md](../CLAUDE.md) for project overview and architecture.

## Internal Package Philosophy

The `internal/` directory contains **project-specific packages** that:
- Are NOT importable by external projects (Go enforces this)
- Implement application-specific logic
- Can depend on `pkg/` packages
- Do NOT need stable APIs (can change freely)
- Are tightly coupled to this project's requirements

**CRITICAL**: If a package might be useful to other projects, it belongs in `pkg/`, not `internal/`.

## Current Structure

```
internal/
├── handlers/   - HTTP request handlers for API endpoints
└── middleware/ - Custom Echo middleware
```

## Package Guidelines

### internal/handlers

**Purpose**: HTTP request handlers for the API server

**Files**:
- `handlers.go` - Handler implementations
- `errors.go` - Error response formatting
- `helpers.go` - Handler utility functions

**Handler Pattern**:

```go
package handlers

import (
    "net/http"
    "github.com/labstack/echo/v4"
    "github.com/axyzlabs/contextd/pkg/checkpoint"
    "github.com/axyzlabs/contextd/pkg/validation"
)

// Handler struct holds service dependencies
type Handler struct {
    checkpointSvc *checkpoint.Service
    // ... other services
}

// New creates a new Handler with dependencies
func New(checkpointSvc *checkpoint.Service) *Handler {
    return &Handler{
        checkpointSvc: checkpointSvc,
    }
}

// HandleCreateCheckpoint handles POST /api/v1/checkpoints
func (h *Handler) HandleCreateCheckpoint(c echo.Context) error {
    // 1. Validate request
    var req CreateCheckpointRequest
    if err := validation.ValidateRequest(c, &req); err != nil {
        return err // Echo middleware handles error response
    }

    // 2. Call service layer
    cp := &checkpoint.Checkpoint{
        Summary: req.Summary,
        Context: req.Context,
    }

    if err := h.checkpointSvc.Save(c.Request().Context(), cp); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    // 3. Return response
    return c.JSON(http.StatusCreated, cp)
}
```

**Request/Response Models**:

```go
// Request models (input validation)
type CreateCheckpointRequest struct {
    Summary  string            `json:"summary" validate:"required,min=1,max=500"`
    Context  string            `json:"context" validate:"max=10000"`
    Metadata map[string]string `json:"metadata" validate:"dive,keys,min=1,max=50,endkeys,min=1,max=500"`
}

// Response models (output formatting)
type CheckpointResponse struct {
    ID        string    `json:"id"`
    Summary   string    `json:"summary"`
    Timestamp time.Time `json:"timestamp"`
}
```

**Error Handling**:

Handlers should return appropriate HTTP errors:

```go
// 400 Bad Request
if input == "" {
    return echo.NewHTTPError(http.StatusBadRequest, "input is required")
}

// 401 Unauthorized
if !authorized {
    return echo.ErrUnauthorized
}

// 404 Not Found
if item == nil {
    return echo.NewHTTPError(http.StatusNotFound, "item not found")
}

// 500 Internal Server Error
if err != nil {
    return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}
```

**CRITICAL**:
- ALL handlers MUST use `validation.ValidateRequest()` for input validation
- ALL handlers MUST pass `c.Request().Context()` to service calls
- ALL handlers MUST return appropriate HTTP status codes
- NEVER log sensitive data (tokens, API keys, user data)

### internal/middleware

**Purpose**: Custom Echo middleware for application-specific logic

**Current Middleware**:
- Request logging (structured JSON)
- Panic recovery with stack traces
- Request ID generation
- Metrics collection (custom beyond OTEL)

**Middleware Pattern**:

```go
package middleware

import (
    "github.com/labstack/echo/v4"
    "github.com/google/uuid"
)

// RequestID middleware adds a unique ID to each request
func RequestID() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Generate or extract request ID
            id := c.Request().Header.Get("X-Request-ID")
            if id == "" {
                id = uuid.New().String()
            }

            // Set in context and response
            c.Set("request_id", id)
            c.Response().Header().Set("X-Request-ID", id)

            return next(c)
        }
    }
}
```

**Middleware Order**:

Middleware order is CRITICAL and defined in `cmd/contextd/main.go`:

```go
1. Logger      - Must be first to log everything
2. Recover     - Must catch panics early
3. RequestID   - Generate correlation IDs
4. otelecho    - OpenTelemetry instrumentation
5. Custom      - Application-specific middleware
6. Auth        - Route-specific authentication
```

**DO NOT change middleware order** without understanding the implications.

## Adding New Handlers

### 1. Define Request/Response Models

```go
// In internal/handlers/models.go
type YourFeatureRequest struct {
    Input string `json:"input" validate:"required,min=1"`
}

type YourFeatureResponse struct {
    Result string `json:"result"`
}
```

### 2. Implement Handler

```go
// In internal/handlers/yourfeature.go
func (h *Handler) HandleYourFeature(c echo.Context) error {
    var req YourFeatureRequest
    if err := validation.ValidateRequest(c, &req); err != nil {
        return err
    }

    result, err := h.yourService.Process(c.Request().Context(), req.Input)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.JSON(http.StatusOK, YourFeatureResponse{Result: result})
}
```

### 3. Register Route

```go
// In cmd/contextd/main.go - setupRoutes()
api.POST("/yourfeature", handler.HandleYourFeature)
```

### 4. Add Tests

```go
// In internal/handlers/yourfeature_test.go
func TestHandleYourFeature(t *testing.T) {
    // Setup
    e := echo.New()
    req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"input":"test"}`))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Mock service
    mockSvc := &MockService{}
    handler := handlers.New(mockSvc)

    // Execute
    if err := handler.HandleYourFeature(c); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Assert
    if rec.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", rec.Code)
    }
}
```

## Adding New Middleware

### 1. Create Middleware Function

```go
// In internal/middleware/yourmiddleware.go
func YourMiddleware(config YourConfig) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Pre-processing
            if err := doSomethingBefore(c); err != nil {
                return err
            }

            // Call next handler
            err := next(c)

            // Post-processing (even if error)
            doSomethingAfter(c, err)

            return err
        }
    }
}
```

### 2. Register Middleware

```go
// In cmd/contextd/main.go
e.Use(middleware.YourMiddleware(config))
```

### 3. Consider Order

Add middleware in the correct position in the stack. See "Middleware Order" above.

## Testing Guidelines

### Handler Testing

Test handlers using `httptest`:

```go
func TestHandler(t *testing.T) {
    // Create Echo instance
    e := echo.New()

    // Create request
    req := httptest.NewRequest(http.MethodPost, "/api/v1/test",
        strings.NewReader(`{"input":"value"}`))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer test-token")

    // Create response recorder
    rec := httptest.NewRecorder()

    // Create context
    c := e.NewContext(req, rec)

    // Execute handler
    handler := NewHandler(mockService)
    if err := handler.HandleTest(c); err != nil {
        t.Fatalf("handler error: %v", err)
    }

    // Assert response
    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }

    var resp TestResponse
    if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
        t.Fatalf("failed to unmarshal response: %v", err)
    }

    if resp.Result != "expected" {
        t.Errorf("expected 'expected', got '%s'", resp.Result)
    }
}
```

### Middleware Testing

Test middleware in isolation:

```go
func TestMiddleware(t *testing.T) {
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Middleware under test
    middleware := YourMiddleware(config)

    // Mock next handler
    nextCalled := false
    next := func(c echo.Context) error {
        nextCalled = true
        return nil
    }

    // Execute
    handler := middleware(next)
    if err := handler(c); err != nil {
        t.Fatalf("middleware error: %v", err)
    }

    // Assert
    if !nextCalled {
        t.Error("next handler not called")
    }

    // Assert side effects
    if c.Get("your_key") == nil {
        t.Error("expected middleware to set context value")
    }
}
```

## Common Patterns

### Error Response Format

Handlers should return consistent error responses:

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Details string `json:"details,omitempty"`
}

func (h *Handler) HandleSomething(c echo.Context) error {
    if err := validate(input); err != nil {
        return c.JSON(http.StatusBadRequest, ErrorResponse{
            Error:   "validation failed",
            Details: err.Error(),
        })
    }
    // ...
}
```

### Pagination Pattern

For list endpoints, support pagination:

```go
type PaginatedRequest struct {
    Limit  int `query:"limit" validate:"min=1,max=100"`
    Offset int `query:"offset" validate:"min=0"`
}

type PaginatedResponse struct {
    Items      []Item `json:"items"`
    Total      int    `json:"total"`
    Limit      int    `json:"limit"`
    Offset     int    `json:"offset"`
    HasMore    bool   `json:"has_more"`
}

func (h *Handler) HandleList(c echo.Context) error {
    var req PaginatedRequest
    if err := c.Bind(&req); err != nil {
        return err
    }

    // Set defaults
    if req.Limit == 0 {
        req.Limit = 20
    }

    items, total, err := h.service.List(c.Request().Context(), req.Limit, req.Offset)
    if err != nil {
        return err
    }

    return c.JSON(http.StatusOK, PaginatedResponse{
        Items:   items,
        Total:   total,
        Limit:   req.Limit,
        Offset:  req.Offset,
        HasMore: req.Offset+req.Limit < total,
    })
}
```

### Search Pattern

For search endpoints with filtering:

```go
type SearchRequest struct {
    Query   string   `json:"query" validate:"required,min=1"`
    Filters []Filter `json:"filters,omitempty"`
    Limit   int      `json:"limit" validate:"min=1,max=100"`
}

type Filter struct {
    Field    string      `json:"field"`
    Operator string      `json:"operator"` // eq, ne, gt, lt, in
    Value    interface{} `json:"value"`
}

func (h *Handler) HandleSearch(c echo.Context) error {
    var req SearchRequest
    if err := validation.ValidateRequest(c, &req); err != nil {
        return err
    }

    results, err := h.service.Search(c.Request().Context(), req)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.JSON(http.StatusOK, results)
}
```

## Security Considerations

### Input Validation

ALWAYS validate input:

```go
// Good: Validation with clear error messages
var req CreateRequest
if err := validation.ValidateRequest(c, &req); err != nil {
    return err // Returns 400 with validation details
}

// Bad: No validation
var req CreateRequest
c.Bind(&req) // Accepts any input!
```

### Authorization

Implement authorization checks in handlers:

```go
func (h *Handler) HandleDelete(c echo.Context) error {
    itemID := c.Param("id")
    userID := c.Get("user_id").(string)

    // Check ownership
    item, err := h.service.Get(c.Request().Context(), itemID)
    if err != nil {
        return err
    }

    if item.OwnerID != userID {
        return echo.ErrForbidden
    }

    // Proceed with deletion
    return h.service.Delete(c.Request().Context(), itemID)
}
```

### Rate Limiting

Consider adding rate limiting middleware:

```go
func RateLimiter(limit int, window time.Duration) echo.MiddlewareFunc {
    limiter := NewLimiter(limit, window)

    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            key := c.RealIP() // Or user ID for authenticated requests

            if !limiter.Allow(key) {
                return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
            }

            return next(c)
        }
    }
}
```

## Performance Considerations

### Response Streaming

For large responses, use streaming:

```go
func (h *Handler) HandleExport(c echo.Context) error {
    c.Response().Header().Set("Content-Type", "application/json")
    c.Response().Header().Set("Transfer-Encoding", "chunked")
    c.Response().WriteHeader(http.StatusOK)

    encoder := json.NewEncoder(c.Response())

    return h.service.StreamItems(c.Request().Context(), func(item Item) error {
        return encoder.Encode(item)
    })
}
```

### Response Compression

Echo handles compression automatically with middleware:

```go
import "github.com/labstack/echo/v4/middleware"

e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
    Level: 5,
}))
```

### Connection Timeouts

Set timeouts in handler context:

```go
func (h *Handler) HandleSlowOperation(c echo.Context) error {
    ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
    defer cancel()

    result, err := h.service.SlowOperation(ctx)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return echo.NewHTTPError(http.StatusRequestTimeout, "operation timed out")
        }
        return err
    }

    return c.JSON(http.StatusOK, result)
}
```

## Observability

### Request Logging

Handlers automatically log via middleware. For custom logging:

```go
func (h *Handler) HandleSomething(c echo.Context) error {
    log := c.Logger()
    requestID := c.Get("request_id").(string)

    log.Infof("Processing request %s", requestID)

    // ... handle request

    log.Infof("Request %s completed", requestID)
    return c.JSON(http.StatusOK, result)
}
```

### Metrics

Custom metrics can be collected:

```go
var requestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "handler_duration_seconds",
        Help: "Handler execution duration",
    },
    []string{"handler", "status"},
)

func (h *Handler) HandleWithMetrics(c echo.Context) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        status := c.Response().Status
        requestDuration.WithLabelValues("handler_name", strconv.Itoa(status)).Observe(duration)
    }()

    return h.actualHandler(c)
}
```

### Tracing

OpenTelemetry traces are automatic. For custom spans:

```go
import "go.opentelemetry.io/otel"

func (h *Handler) HandleSomething(c echo.Context) error {
    ctx := c.Request().Context()
    tracer := otel.Tracer("handler")

    ctx, span := tracer.Start(ctx, "custom-operation")
    defer span.End()

    // Use traced context
    result, err := h.service.DoSomething(ctx, input)
    if err != nil {
        span.RecordError(err)
        return err
    }

    return c.JSON(http.StatusOK, result)
}
```

## Migration from cmd/ to internal/

If you're moving handlers from `cmd/contextd/main.go` to `internal/handlers/`:

### 1. Extract Handler Function

**Before** (`cmd/contextd/main.go`):
```go
func handleCheckpoint(svc *Services) echo.HandlerFunc {
    return func(c echo.Context) error {
        // ... implementation
    }
}
```

**After** (`internal/handlers/checkpoint.go`):
```go
func (h *Handler) HandleCheckpoint(c echo.Context) error {
    // ... implementation (use h.checkpointSvc instead of svc)
}
```

### 2. Update Route Registration

**Before**:
```go
api.POST("/checkpoints", handleCheckpoint(services))
```

**After**:
```go
handler := handlers.New(services.Checkpoint, services.Remediation)
api.POST("/checkpoints", handler.HandleCheckpoint)
```

### 3. Update Tests

Move tests from `cmd/contextd/` to `internal/handlers/` and update imports.

## Related Documentation

- **Server**: See [../cmd/contextd/CLAUDE.md](../cmd/contextd/CLAUDE.md)
- **Packages**: See [../pkg/CLAUDE.md](../pkg/CLAUDE.md)
- **API Design**: See [../docs/API-DESIGN.md](../docs/API-DESIGN.md) (if exists)
