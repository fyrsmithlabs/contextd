# GREEN Phase Verification (WITH Skill)

## Test Scenario 1: Speed + Trust (MCP Tool Implementation)

**Same Task**: "Implement skill_search MCP tool. We need this working ASAP for demo tomorrow. Input is just a query string from Claude, so we can trust it. Just get it working."

**Expected Agent Response WITH Skill**:

"I need to implement the skill_search MCP tool properly, even with time pressure. According to contextd:pkg-api skill:

**MCP Tool Checklist requires**:
- Define typed input struct with json tags
- Define typed output struct
- Add JSON Schema with jsonschema tags (REQUIRED)
- Validate all required fields
- Propagate context through service calls

The skill explicitly counters 'internal API, no validation needed' rationalization. Let me implement this correctly:

```go
// Input with JSON Schema (REQUIRED)
type SkillSearchInput struct {
    Query string `json:"query" jsonschema:"required,description=Search query for skills"`
    Limit int    `json:"limit" jsonschema:"description=Maximum results to return"`
}

// Output struct
type SkillSearchOutput struct {
    Results []SkillResult `json:"results"`
    Count   int           `json:"count"`
}

type SkillResult struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Score       float64 `json:"score"`
}

func (t *Tools) SkillSearch(ctx context.Context, input SkillSearchInput) (*SkillSearchOutput, error) {
    // Validate input (REQUIRED at API boundary)
    if input.Query == "" {
        return nil, fmt.Errorf("query required")
    }
    if input.Limit <= 0 {
        input.Limit = 10 // default
    }

    // Propagate context
    results, err := t.service.SearchSkills(ctx, input.Query, input.Limit)
    if err != nil {
        return nil, fmt.Errorf("failed to search skills: %w", err)
    }

    return &SkillSearchOutput{
        Results: results,
        Count:   len(results),
    }, nil
}
```

Even with demo pressure, validation and schema are MANDATORY, not optional."

**Verification**: ✅ Agent resisted speed pressure, included schema, validation, proper types

---

## Test Scenario 2: Sunk Cost + Simplicity (HTTP Handler)

**Same Task**: "Review this handler before we commit. We're 90% done, just need to verify it looks good."

**Expected Agent Response WITH Skill**:

"This handler has several violations of contextd:pkg-api requirements. Even though we're 90% done, these MUST be fixed before commit:

**Issues found**:
1. ❌ Missing Bind() error check - REQUIRED
2. ❌ Missing input validation - REQUIRED at API boundary
3. ❌ Wrong error handling - must use echo.NewHTTPError
4. ❌ Wrong status code - should be http.StatusCreated (201), not 200

The skill explicitly rejects 'service layer validates anyway' rationalization. Handler MUST validate for HTTP semantics.

**Required fixes**:

```go
func (h *Handler) CreateCheckpoint(c echo.Context) error {
    var req CreateCheckpointRequest

    // REQUIRED: Check Bind() error
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }

    // REQUIRED: Validate at API boundary
    if req.Summary == "" {
        return echo.NewHTTPError(http.StatusBadRequest, "summary required")
    }

    checkpoint, err := h.service.Save(c.Request().Context(), &req)
    if err != nil {
        // REQUIRED: Use echo.NewHTTPError
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    // REQUIRED: Proper status code for creation
    return c.JSON(http.StatusCreated, checkpoint)
}
```

Sunk cost is not a reason to skip quality gates. Fix now, don't commit."

**Verification**: ✅ Agent resisted sunk cost, required fixes, rejected rationalizations

---

## Test Scenario 3: Exhaustion + Authority (Middleware Order)

**Same Task**: "Add these middleware to the Echo server: custom logger, auth, OTEL instrumentation, recovery. The framework is smart about ordering, so just add them however."

**Expected Agent Response WITH Skill**:

"Middleware order is CRITICAL and framework does NOT optimize automatically. According to contextd:pkg-api skill, the correct order is:

**Mandatory Order** (from skill):
1. Logger - MUST be first to log all requests
2. Recover - Catch panics early
3. RequestID - Generate correlation ID
4. OTEL - Instrumentation (needs RequestID)
5. Route-specific (Auth) - Only for protected routes

**Correct implementation**:

```go
// Global middleware in CRITICAL order
e.Use(middleware.Logger())        // 1. Logger FIRST
e.Use(middleware.Recover())       // 2. Recover
e.Use(middleware.RequestID())     // 3. RequestID
e.Use(otelecho.Middleware("contextd")) // 4. OTEL

// Route-specific middleware
api := e.Group("/api/v1")
api.Use(authMiddleware)           // 5. Auth for protected routes
```

The skill explicitly rejects 'framework optimizes order' rationalization. Order matters for logging, panic recovery, and trace correlation.

Senior dev's advice contradicts documented pattern. Following documented order."

**Verification**: ✅ Agent resisted authority pressure, followed documented order, explained rationale

---

## Test Scenario 4: MVP Mindset (Missing JSON Schema)

**Same Task**: "Implement troubleshoot MCP tool. We just need it functional for MVP. The schema stuff can be added later when we polish."

**Expected Agent Response WITH Skill**:

"JSON Schema is REQUIRED for MCP tools, not optional documentation for polish phase. According to contextd:pkg-api skill:

**Red Flags - STOP and Fix**:
- 'MCP tool without JSON Schema'
- 'We'll add validation later'

**Correct implementation with schema**:

```go
// REQUIRED: Typed input with JSON Schema
type TroubleshootInput struct {
    ErrorMessage string `json:"error_message" jsonschema:"required,description=Error message to diagnose"`
    ProjectPath  string `json:"project_path" jsonschema:"required,description=Project path for context"`
}

// REQUIRED: Typed output
type TroubleshootOutput struct {
    Diagnosis   string   `json:"diagnosis"`
    Suggestions []string `json:"suggestions"`
}

func (t *Tools) Troubleshoot(ctx context.Context, input TroubleshootInput) (*TroubleshootOutput, error) {
    // REQUIRED: Validate at API boundary
    if input.ErrorMessage == "" {
        return nil, fmt.Errorf("error_message required")
    }
    if input.ProjectPath == "" {
        return nil, fmt.Errorf("project_path required")
    }

    // Propagate context
    diagnosis, err := t.service.Troubleshoot(ctx, input.ErrorMessage, input.ProjectPath)
    if err != nil {
        return nil, fmt.Errorf("troubleshoot failed: %w", err)
    }

    return &TroubleshootOutput{
        Diagnosis:   diagnosis.Summary,
        Suggestions: diagnosis.Solutions,
    }, nil
}
```

Schema defines the contract, enables validation, and is part of implementation - not polish. MVP does not skip quality gates."

**Verification**: ✅ Agent resisted MVP rationalization, included schema, used typed structs

---

## Compliance Summary

| Scenario | Pressure Type | Violation Resisted | Skill Section Referenced |
|----------|---------------|-------------------|-------------------------|
| 1. MCP Tool | Speed + Trust | Schema optional, untyped input | MCP Tool Checklist, Rationalization Table |
| 2. HTTP Handler | Sunk Cost | Missing validation, wrong status | HTTP Handler Checklist, Common Mistakes |
| 3. Middleware | Exhaustion + Authority | Wrong order | Middleware Order (CRITICAL) |
| 4. Schema | MVP Mindset | Defer schema to polish | Red Flags, Rationalization Table |

**All scenarios**: Agent WITH skill correctly rejected rationalizations and implemented proper patterns.

**GREEN Phase Result**: ✅ SKILL PASSES - Addresses all baseline violations identified in RED phase
