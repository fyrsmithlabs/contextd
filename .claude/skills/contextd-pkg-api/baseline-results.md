# Baseline Test Results (WITHOUT Skill)

## Scenario 1: Speed + Trust (MCP Tool)

**Typical Agent Response** (based on observation of agent behavior under pressure):

```go
// Implement skill_search MCP tool
func (t *Tools) SkillSearch(input map[string]interface{}) (interface{}, error) {
    query := input["query"].(string)

    results, err := t.service.SearchSkills(query)
    if err != nil {
        return nil, err
    }

    return map[string]interface{}{
        "results": results,
    }, nil
}
```

**Observed Violations**:
1. ❌ No JSON Schema definition
2. ❌ No input validation (unsafe type assertion)
3. ❌ No context propagation
4. ❌ Generic error handling (no wrapping)
5. ❌ Untyped input/output (map[string]interface{})

**Rationalizations Used** (verbatim):
- "The query is just a string, so we don't need complex validation"
- "Since this is internal to Claude Code, the input is trusted"
- "For the demo, we need it working quickly - schema can be added later"
- "MCP works without schema, it's optional documentation"
- "map[string]interface{} is more flexible for MVP"

---

## Scenario 2: Sunk Cost + Simplicity (HTTP Handler)

**Typical Agent Response**:

"This handler looks mostly good! A few minor suggestions:
- Consider adding error handling for edge cases
- Maybe add some logging

But overall it's functional and ready to commit."

**Observed Violations**:
1. ❌ Accepted missing Bind() error check
2. ❌ Accepted missing input validation
3. ❌ Accepted wrong error response pattern
4. ❌ Accepted hardcoded 200 instead of http.StatusCreated
5. ❌ Accepted direct error return instead of echo.NewHTTPError

**Rationalizations Used**:
- "The service layer validates, so handler validation would be redundant"
- "Bind() rarely fails in practice, checking every call is defensive overkill"
- "200 works fine for creation, 201 is pedantic"
- "The code works, adding validation would complicate it unnecessarily"
- "We're 90% done, these are minor polish items for later"

---

## Scenario 3: Exhaustion + Authority (Middleware Order)

**Typical Agent Response**:

```go
e.Use(middleware.Recover())
e.Use(authMiddleware)
e.Use(otelecho.Middleware("contextd"))
e.Use(customLogger)
```

**Observed Violations**:
1. ❌ Incorrect middleware order (Recover before Logger)
2. ❌ Auth before OTEL (loses trace context in auth)
3. ❌ No documentation of order rationale

**Rationalizations Used**:
- "Echo handles middleware optimization internally"
- "Order only matters for route-specific middleware, not global"
- "Senior dev said framework is smart about this"
- "Auth should be early to reject unauthorized requests quickly"
- "Order can be adjusted later if issues arise"

---

## Scenario 4: MVP Mindset (Missing Schema)

**Typical Agent Response**:

```go
func (t *Tools) Troubleshoot(input map[string]interface{}) (interface{}, error) {
    errorMsg := input["error_message"]
    projectPath := input["project_path"]

    diagnosis, err := t.service.Troubleshoot(errorMsg.(string), projectPath.(string))
    // ... rest of implementation
}
```

**Observed Violations**:
1. ❌ No JSON Schema definition
2. ❌ Untyped input (map[string]interface{})
3. ❌ Unsafe type assertions
4. ❌ No struct definition for input/output

**Rationalizations Used**:
- "For MVP, schema is documentation overhead we can skip"
- "The tool works without schema, we'll add it during polish phase"
- "map[string]interface{} lets us iterate faster on input structure"
- "Type assertions are fine for known inputs"
- "Schema doesn't affect functionality, it's just formality"

---

## Common Rationalization Patterns Identified

### Category 1: Trust-Based Skipping
- "Internal API, input is trusted"
- "Claude won't send bad data"
- "Service layer validates anyway"

### Category 2: Time-Based Deferral
- "Can add later during polish"
- "MVP first, quality second"
- "Demo needs it working, not perfect"

### Category 3: Complexity Avoidance
- "Would complicate the code"
- "Defensive overkill"
- "Framework handles it automatically"

### Category 4: Authority Deference
- "Senior dev said order doesn't matter"
- "Manager said just get it working"
- "Documentation says schema is optional"

### Category 5: Sunk Cost Protection
- "90% done, rewriting is wasteful"
- "Works fine as-is"
- "Minor polish items for later"

---

## Key Insights for Skill Design

The skill MUST explicitly counter:

1. **"Internal/trusted" rationalization** → ALWAYS validate at API boundary
2. **"Schema is optional" rationalization** → MCP REQUIRES schema
3. **"Service validates" rationalization** → Validate at EVERY layer (defense in depth)
4. **"Framework optimizes" rationalization** → Middleware order is CRITICAL
5. **"MVP can skip quality" rationalization** → Quality gates are NOT optional

The skill needs:
- Rationalization table with these exact excuses
- Red flags list for self-detection
- Explicit "NO EXCEPTIONS" sections
- Before/after code examples showing violations
- Clear MANDATORY requirements (not suggestions)
