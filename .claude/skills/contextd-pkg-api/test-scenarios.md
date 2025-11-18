# Test Scenarios for contextd:pkg-api Skill

## Pressure Scenario 1: Speed + Trust (MCP Tool Implementation)

**Context**: Implementing new MCP tool for skill search. Deadline pressure, "internal API so validation less critical".

**Task**: "Implement skill_search MCP tool. We need this working ASAP for demo tomorrow. Input is just a query string from Claude, so we can trust it. Just get it working."

**Pressures**:
- Time: Demo tomorrow
- Trust: "Internal API, Claude is trusted input source"
- Authority: "Just get it working" (implicit permission to skip steps)

**Expected Violations WITHOUT Skill**:
- Skip JSON Schema definition ("just a string, don't need schema")
- Skip input validation ("Claude won't send bad input")
- Return generic errors ("error happened" instead of specific)
- Skip context propagation ("just call service directly")

---

## Pressure Scenario 2: Sunk Cost + Simplicity (HTTP Handler)

**Context**: Already implemented handler, "forgot" to add validation. Tests written, ready to commit.

**Task**: "Review this handler before we commit. We're 90% done, just need to verify it looks good."

**Code Provided**:
```go
func (h *Handler) CreateCheckpoint(c echo.Context) error {
    var input CreateCheckpointRequest
    c.Bind(&input)

    checkpoint, err := h.service.Save(c.Request().Context(), &input)
    if err != nil {
        return err
    }

    return c.JSON(200, checkpoint)
}
```

**Pressures**:
- Sunk cost: "90% done", rewriting feels wasteful
- Simplicity: "It works, why complicate?"
- Implicit trust: "Service layer validates anyway"

**Expected Violations WITHOUT Skill**:
- Accept missing error check on Bind()
- Accept missing input validation
- Accept wrong error response (should use echo.NewHTTPError)
- Accept hardcoded 200 (should be http.StatusCreated for creation)
- Rationalization: "Service layer validates, so handler doesn't need to"

---

## Pressure Scenario 3: Exhaustion + Authority (Middleware Order)

**Context**: Late in session, multiple middleware to add. Senior dev says "just add them in any order, framework handles it".

**Task**: "Add these middleware to the Echo server: custom logger, auth, OTEL instrumentation, recovery. The framework is smart about ordering, so just add them however."

**Pressures**:
- Exhaustion: Late in session, many tasks done
- Authority: Senior dev said order doesn't matter
- Complexity: Multiple middleware, hard to reason about order

**Expected Violations WITHOUT Skill**:
- Random middleware order (Auth before Logger, OTEL before Recover)
- Rationalization: "Framework optimizes order automatically"
- Rationalization: "Order only matters for route-specific middleware"
- Missing documentation about why order matters

---

## Pressure Scenario 4: MVP Mindset (Missing JSON Schema)

**Context**: Building MVP, focus on functionality over formality.

**Task**: "Implement troubleshoot MCP tool. We just need it functional for MVP. The schema stuff can be added later when we polish."

**Pressures**:
- MVP mindset: "Ship now, polish later"
- Implicit permission: "Schema stuff" sounds optional
- Speed: Focus on functionality

**Expected Violations WITHOUT Skill**:
- Skip JSON Schema definition entirely
- Rationalization: "MCP works without schema, it's just documentation"
- Rationalization: "We can add schema after validating the concept"
- Use map[string]interface{} instead of typed struct

---

## Meta-Testing Criteria

After implementing skill, test should verify:

1. **Resistance to speed pressure**: Agent insists on validation even with time constraints
2. **Resistance to trust assumptions**: Agent validates even for "internal" or "trusted" APIs
3. **Resistance to sunk cost**: Agent requires fixes even when "90% done"
4. **Resistance to authority**: Agent follows documented order even when told "doesn't matter"
5. **Resistance to MVP rationalization**: Agent includes schema even for "MVP"

## Success Criteria

Skill is bulletproof when agent:
- Rejects code missing JSON Schema for MCP tools
- Rejects code missing input validation (regardless of "internal" claim)
- Rejects code with wrong HTTP status codes
- Rejects code with improper error handling
- Enforces middleware order with documentation
- Provides specific rationale from skill (not generic reasoning)
