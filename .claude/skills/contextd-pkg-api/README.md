# contextd:pkg-api Skill

**Status**: Production Ready
**Version**: 1.0.0
**Created**: 2025-11-18
**Testing Methodology**: RED-GREEN-REFACTOR (TDD for skills)

## Purpose

Enforces API development patterns for contextd packages:
- MCP tools (pkg/mcp): JSON Schema, typed I/O, validation
- HTTP handlers (pkg/handlers): Bind() checks, proper status codes, echo.NewHTTPError
- Middleware (pkg/middleware): Critical ordering, error handling

## When to Use

Use this skill when:
- Implementing MCP tools for Claude Code
- Writing HTTP handlers with Echo framework
- Adding middleware to the server
- Reviewing API code for quality/security
- Debugging input validation issues

## Testing Documentation

This skill was developed using Test-Driven Development principles adapted to process documentation:

### RED Phase (Baseline Testing)
- **test-scenarios.md**: 4 pressure scenarios (speed, trust, sunk cost, authority, exhaustion, MVP)
- **baseline-results.md**: Agent behavior WITHOUT skill (documented rationalizations)
- **Key finding**: Agents skip validation under pressure with 5 rationalization patterns

### GREEN Phase (Skill Implementation)
- **SKILL.md**: Main skill file addressing baseline failures
- **green-verification.md**: Verification that skill addresses all baseline violations
- **Result**: ✅ All scenarios pass with skill present

### REFACTOR Phase (Loophole Closing)
- **refactor-loopholes.md**: 10 additional loopholes discovered (6 new + 3 meta)
- **refactor-verification.md**: Verification all loopholes closed
- **Result**: ✅ Bulletproof against 15 rationalization patterns

### Quality Assurance
- **cso-verification.md**: Claude Search Optimization compliance
- **Result**: ✅ Excellent discoverability and keyword coverage

## Skill Statistics

- **Rationalization Patterns Covered**: 15
- **Pressure Scenarios Tested**: 4 baseline + 6 refactor = 10 total
- **Word Count**: 1911 (acceptable for comprehensive discipline skill)
- **Character Count (description)**: 297 (under 500 preferred)
- **Loopholes Closed**: 10/10 (100%)

## Rationalization Coverage

The skill explicitly counters these 15 rationalizations:

1. Schema is optional for MCP tools
2. Internal API, input is trusted
3. Service layer validates anyway
4. Bind() rarely fails in practice
5. 200 works fine for creation
6. Framework optimizes middleware order
7. We're 90% done, validation is polish
8. MVP can skip quality gates
9. We'll add schema during polish phase
10. Validation adds overhead
11. Tests prove validation unnecessary
12. OpenAPI/Proto schema covers this
13. Optional fields don't need validation
14. Generic errors are good enough
15. Operation too fast for context

## Integration with contextd Skills Ecosystem

**Dependencies**:
- contextd:completing-major-task (verification)
- contextd:code-review (pre-PR check)

**Related Skills**:
- contextd:pkg-security (auth, session middleware)
- contextd:pkg-storage (API + storage integration)
- contextd:pkg-ai (API + embeddings integration)

## Files in This Directory

- **SKILL.md**: Main skill reference (deploy this to .claude/skills/)
- **README.md**: This file (deployment documentation)
- **test-scenarios.md**: RED phase pressure scenarios
- **baseline-results.md**: RED phase agent behavior (without skill)
- **green-verification.md**: GREEN phase verification (with skill)
- **refactor-loopholes.md**: REFACTOR phase loophole discovery
- **refactor-verification.md**: REFACTOR phase loophole closure verification
- **cso-verification.md**: Claude Search Optimization compliance check

## Deployment Checklist

- [x] RED Phase: Baseline testing complete
- [x] GREEN Phase: Skill addresses baseline failures
- [x] REFACTOR Phase: All loopholes closed
- [x] CSO: Description optimized for discovery
- [x] Testing: Comprehensive pressure scenario coverage
- [x] Documentation: README and test artifacts
- [ ] Git: Committed to repository
- [ ] Validation: Skill available to agents

## Usage Example

```
# Agent working on MCP tool implementation
Task: "Implement skill_search MCP tool quickly for demo"

Agent loads contextd:pkg-api skill

Agent response:
"According to contextd:pkg-api, I need to:
1. Define typed input struct with JSON Schema (REQUIRED by MCP)
2. Validate ALL fields (required AND optional)
3. Propagate context (ALWAYS, even for fast operations)

The skill explicitly rejects 'schema is optional' and 'demo pressure'
rationalizations. Implementing correctly..."
```

## Success Metrics

**Skill is successful if**:
- Agents implement proper JSON Schema for MCP tools (100% compliance)
- Agents validate input at API boundaries (no "trusted input" skipping)
- Agents use proper HTTP status codes (no hardcoded 200)
- Agents check Bind() errors (no unchecked errors)
- Agents follow middleware order (Logger → Recover → RequestID → OTEL → Auth)
- Agents resist all 15 rationalization patterns

## Maintenance

**Update skill when**:
- New API patterns emerge in contextd
- New rationalization patterns discovered
- MCP protocol requirements change
- Echo framework best practices update
- New loopholes identified in testing

**Testing required for updates**: YES (re-run pressure scenarios)

## License

Part of contextd project. See root LICENSE file.
