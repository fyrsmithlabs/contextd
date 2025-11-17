# Agent Coordination Documentation Updates - 2025-11-17

## Summary

Updated CLAUDE.md and MULTI-AGENT-ORCHESTRATION.md to establish clear coordination rules between specialized agents, particularly for MCP implementation work.

## Changes Made

### 1. CLAUDE.md - New Section 5a: Multi-Agent Coordination

**Location**: After Section 5 "Go Code Delegation" (line 142)

**Added**:
- **MCP Implementation Pattern (MANDATORY)**: Two-phase workflow
  - Phase 1: mcp-developer for protocol research and design
  - Phase 2: golang-pro for Go implementation with TDD
- **Example Coordination Flow**: Concrete example showing handoff between agents
- **General Agent Coordination Rules**: Sequential vs Parallel patterns
- **Key Principles**:
  - Never skip specialist agents
  - Always pass context forward
  - One agent per domain
  - Maintain security context

**Why This Matters**:
- Prevents asking golang-pro to research MCP specs (out of domain)
- Prevents asking mcp-developer to write Go code (out of domain)
- Ensures protocol expertise (mcp-developer) informs implementation (golang-pro)
- Maintains security requirements throughout handoff

### 2. MULTI-AGENT-ORCHESTRATION.md - Enhanced Delegation Table

**Location**: Agent Delegation section (line 42)

**Changes**:
- Added "Notes" column to delegation table
- Highlighted **mcp-developer** for protocol design
- Clarified **golang-pro** enforces TDD + security
- Added **test-strategist** to table

**Added New Section**: Multi-Agent Coordination Patterns (line 59)

**Includes**:
1. **MCP Implementation Pattern** - 3-phase workflow with visualization:
   ```
   Phase 1: mcp-developer (design)
   Phase 2: golang-pro (implement)
   Phase 3: code-reviewer (verify)
   ```

2. **Other Common Patterns**:
   - Security-Critical Feature pattern
   - Performance Optimization pattern
   - Documentation Generation pattern

**Example Usage**:
```
Step 1: "Use mcp-developer agent to research MCP Streamable HTTP
        specification and design /mcp endpoint implementation"

Step 2: "Use golang-pro skill to implement /mcp endpoint with:
        - Protocol requirements from mcp-developer
        - Security requirements from CLAUDE.md Section 1
        - TDD with ≥80% test coverage"

Step 3: "Use superpowers:requesting-code-review to verify implementation"
```

## Problem This Solves

**Before**:
- Unclear when to use mcp-developer vs golang-pro
- Risk of asking wrong agent to do cross-domain work
- No clear handoff pattern for complex tasks
- Security requirements might be lost in handoff

**After**:
- Clear 3-phase pattern: design → implement → verify
- Explicit domain boundaries (protocol vs code)
- Documented handoff with context passing
- Security requirements mandated in every phase

## Real-World Application

**Current MCP Connection Issue**:

Following the new pattern, the correct approach is:

1. ✅ **DONE**: mcp-developer researched spec and identified gaps
   - Found: We implement custom REST API not MCP protocol
   - Gap: Missing `/mcp` endpoint, no `initialize` method, wrong SSE signature
   - Output: MCP_HTTP_GAP_ANALYSIS.md with requirements

2. **NEXT**: golang-pro implements proper MCP endpoint
   - Input: Gap analysis from mcp-developer
   - Requirements: POST/GET/DELETE /mcp, JSON-RPC routing, session management
   - Security: Section 1 requirements (auth, input validation, multi-tenant)
   - Output: Production code with ≥80% test coverage

3. **THEN**: code-reviewer validates against MCP spec
   - Verify protocol compliance
   - Verify security requirements met
   - Verify test coverage adequate

## Files Modified

1. `CLAUDE.md` - Added Section 5a (68 lines)
2. `docs/guides/MULTI-AGENT-ORCHESTRATION.md` - Enhanced delegation table and added patterns (68 lines)

## Benefits

1. **Prevents Domain Confusion**: Each agent stays in expertise zone
2. **Ensures Quality**: Specialist design → professional implementation → rigorous review
3. **Maintains Security**: Security context passed through entire chain
4. **Reusable Pattern**: Not just MCP - works for any complex feature
5. **Clear Accountability**: Each phase has clear deliverables

## Next Steps

With this documentation in place:

1. Follow the MCP Implementation Pattern to fix current connection issue
2. Use as template for future complex features (context-folding, etc.)
3. Reference in CLAUDE.md when delegating multi-agent work

## Related Documents

- `MCP_HTTP_GAP_ANALYSIS.md` - mcp-developer's output (phase 1 complete)
- `CLAUDE.md` Section 1 - Security requirements (must pass to golang-pro)
- `docs/specs/context-folding/` - Next major feature that will use this pattern
