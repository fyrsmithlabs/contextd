# MCP Tools Implementation Plan

**Status**: Active | **Created**: 2025-01-19 | **Target**: v1.0.0

## Overview

This plan addresses the implementation gaps identified in the MCP endpoint audit (2025-01-19). The audit found that while core infrastructure is solid, only ~50% of claimed tools are fully implemented.

**Current State**: 8/16 tools working, 4 stubbed, 9 missing entirely

**Goal**: Implement all 16 MCP tools with full functionality and ≥80% test coverage

---

## Implementation Status Matrix

| Tool | Status | Service | Handler | Tests | Priority |
|------|--------|---------|---------|-------|----------|
| checkpoint_save | ✅ Complete | ✅ | ✅ | ✅ | - |
| checkpoint_search | ✅ Complete | ✅ | ✅ | ✅ | - |
| checkpoint_list | ✅ Complete | ✅ | ✅ | ✅ | - |
| remediation_save | ✅ Complete | ✅ | ✅ | ✅ | - |
| remediation_search | ✅ Complete | ✅ | ✅ | ✅ | - |
| collection_create | ✅ Complete | ✅ | ✅ | ✅ | - |
| collection_delete | ✅ Complete | ✅ | ✅ | ✅ | - |
| collection_list | ✅ Complete | ✅ | ✅ | ✅ | - |
| **skill_save** | ⚠️ Stubbed | ❌ | ⚠️ | ❌ | **P0** |
| **skill_search** | ⚠️ Stubbed | ❌ | ⚠️ | ❌ | **P0** |
| **skill_create** | ❌ Missing | ❌ | ❌ | ❌ | **P0** |
| **skill_update** | ❌ Missing | ❌ | ❌ | ❌ | **P1** |
| **skill_delete** | ❌ Missing | ❌ | ❌ | ❌ | **P1** |
| **skill_apply** | ❌ Missing | ❌ | ❌ | ❌ | **P1** |
| **skill_list** | ❌ Missing | ❌ | ❌ | ❌ | **P1** |
| **index_repository** | ⚠️ Stubbed | ⚠️ | ⚠️ | ❌ | **P0** |
| **troubleshoot** | ❌ Missing | ❌ | ❌ | ❌ | **P0** |
| **list_patterns** | ❌ Missing | ❌ | ❌ | ❌ | **P1** |
| **status** | ⚠️ Wrong | ✅ | ⚠️ | ⚠️ | **P0** |
| **analytics_get** | ❌ Missing | ❌ | ❌ | ❌ | **P2** |

---

## Phase 1: Critical Fixes (P0) - Target: Sprint 1

### 1.1 Implement Skills Service (`pkg/skills/`)

**Issue**: Package is completely empty (only documentation files)

**Tasks**:
- [ ] Create `pkg/skills/service.go` with Service struct
- [ ] Implement `Save(ctx, skill) (id, error)` - Save skill template
- [ ] Implement `Search(ctx, query, limit) ([]Skill, error)` - Semantic search
- [ ] Implement `Create(ctx, skill) (id, error)` - Alias for Save (explicit creation)
- [ ] Implement `Get(ctx, id) (*Skill, error)` - Retrieve by ID
- [ ] Create `pkg/skills/types.go` for Skill struct
- [ ] Create `pkg/skills/service_test.go` with ≥80% coverage
- [ ] Add OpenTelemetry instrumentation
- [ ] Document in `pkg/skills/CLAUDE.md`

**Dependencies**:
- Qdrant adapter (existing)
- Embedding service (existing)
- Multi-tenant isolation (project/team scope)

**Acceptance Criteria**:
- Skills saved to `shared` database (team-scoped, not project-scoped)
- Semantic search works across saved skills
- Test coverage ≥80%
- No cross-team data leakage

---

### 1.2 Fix Stubbed Skill Handlers

**File**: `pkg/mcp/server.go`

**Tasks**:
- [ ] Fix `handleSkillSave` (lines 472-476) - Integrate with skills.Service
- [ ] Fix `handleSkillSearch` (lines 478-482) - Integrate with skills.Service
- [ ] Add `handleSkillCreate` - Wire to skills.Service.Create
- [ ] Update tool routing in `pkg/mcp/protocol.go` (handleToolsCallMethod)
- [ ] Add handler tests in `pkg/mcp/server_test.go`
- [ ] Update discovery.go with correct tool schemas

**Implementation Pattern** (from working handlers):
```go
func (s *Server) handleSkillSave(c echo.Context) error {
    var req SkillSaveRequest
    if err := parseToolRequest(c, &req); err != nil {
        return err
    }

    // Async operation like checkpoint_save
    opID := generateOperationID()

    go func() {
        ctx := context.Background()

        skill := &skills.Skill{
            Name:        req.Name,
            Description: req.Description,
            Content:     req.Content,
            Tags:        req.Tags,
        }

        skillID, err := s.skillsService.Save(ctx, skill)

        result := map[string]interface{}{"skill_id": skillID}
        s.operations.Complete(opID, result, err)
    }()

    return JSONRPCSuccess(c, extractRequestID(c), map[string]string{
        "operation_id": opID,
    })
}
```

**Acceptance Criteria**:
- All skill handlers return real data (no placeholders)
- Async operations tracked via operation store
- Integration tests pass

---

### 1.3 Implement Troubleshoot Service (`pkg/troubleshoot/`)

**Issue**: Service doesn't exist, tool not in discovery

**Tasks**:
- [ ] Create `pkg/troubleshoot/` directory
- [ ] Create `pkg/troubleshoot/service.go` with Service struct
- [ ] Implement `Diagnose(ctx, errorMsg, context) (*Diagnosis, error)` - AI-powered error analysis
- [ ] Implement `GetPatterns(ctx) ([]Pattern, error)` - List known error patterns
- [ ] Create `pkg/troubleshoot/types.go` for Diagnosis/Pattern structs
- [ ] Create `pkg/troubleshoot/service_test.go` with ≥80% coverage
- [ ] Add OpenTelemetry instrumentation
- [ ] Create `pkg/troubleshoot/CLAUDE.md`

**Dependencies**:
- OpenAI API (or local LLM) for error analysis
- Qdrant for pattern storage (shared database)
- Remediation service (for suggesting fixes)

**Acceptance Criteria**:
- AI analyzes error messages and provides diagnosis
- Common patterns stored and searchable
- Integration with remediation for fix suggestions
- Test coverage ≥80%

---

### 1.4 Add Troubleshoot Handlers

**File**: `pkg/mcp/server.go` (new handlers)

**Tasks**:
- [ ] Create `handleTroubleshoot` - Wire to troubleshoot.Service.Diagnose
- [ ] Create `handleListPatterns` - Wire to troubleshoot.Service.GetPatterns
- [ ] Update tool routing in `pkg/mcp/protocol.go`
- [ ] Add to discovery.go tool list with schemas
- [ ] Add handler tests

**Acceptance Criteria**:
- Tools callable via `/mcp` endpoint
- Returns structured diagnosis with suggestions
- Integration tests pass

---

### 1.5 Fix index_repository Handler

**Issue**: Returns fake operation_id, no actual indexing happens

**Tasks**:
- [ ] Verify `pkg/repository/service.go` exists and works
- [ ] Fix `handleIndexRepository` (lines 484-490) to call repository.Service
- [ ] Implement async indexing (like checkpoint_save pattern)
- [ ] Add operation tracking via operation store
- [ ] Update tests to verify actual indexing

**Implementation Pattern**:
```go
func (s *Server) handleIndexRepository(c echo.Context) error {
    var req IndexRepositoryRequest
    if err := parseToolRequest(c, &req); err != nil {
        return err
    }

    opID := generateOperationID()

    go func() {
        ctx := context.Background()
        err := s.repositoryService.IndexRepository(ctx, req.Path, req.Options)

        result := map[string]interface{}{
            "indexed_files": fileCount,
            "status":        "complete",
        }
        s.operations.Complete(opID, result, err)
    }()

    return JSONRPCSuccess(c, extractRequestID(c), map[string]string{
        "operation_id": opID,
        "status":       "pending",
    })
}
```

**Acceptance Criteria**:
- Real indexing happens (not placeholder)
- Operation status queryable via status tool
- Test coverage ≥80%

---

### 1.6 Fix status Tool

**Issue**: Returns server health instead of operation status

**File**: `pkg/mcp/server.go`, lines 492-498

**Current (WRONG)**:
```go
func (s *Server) handleStatus(c echo.Context) error {
    return JSONRPCSuccess(c, "req-mno", map[string]interface{}{
        "status":  "healthy",
        "service": "contextd",
        "version": "0.9.0-rc-1",
    })
}
```

**Correct Implementation**:
```go
func (s *Server) handleStatus(c echo.Context) error {
    var req StatusRequest
    if err := parseToolRequest(c, &req); err != nil {
        return err
    }

    op, err := s.operations.Get(req.OperationID)
    if err != nil {
        return JSONRPCError(c, extractRequestID(c), -32602,
            fmt.Sprintf("operation not found: %s", req.OperationID))
    }

    return JSONRPCSuccess(c, extractRequestID(c), map[string]interface{}{
        "operation_id": op.ID,
        "status":       op.Status,  // pending/complete/failed
        "result":       op.Result,
        "error":        op.Error,
        "created_at":   op.CreatedAt,
        "updated_at":   op.UpdatedAt,
    })
}
```

**Tasks**:
- [ ] Fix handleStatus to query operation store
- [ ] Add StatusRequest type to types.go
- [ ] Update discovery.go schema for status tool
- [ ] Add tests for operation status queries
- [ ] Test with real async operations

**Acceptance Criteria**:
- Returns operation status, not server health
- Works with checkpoint_save, skill_save, index_repository operations
- Returns proper error for unknown operation_id
- Test coverage ≥80%

---

## Phase 2: High Priority (P1) - Target: Sprint 2

### 2.1 Complete Skills CRUD Operations

**Tasks**:
- [ ] Implement `handleSkillUpdate` - Update existing skill
- [ ] Implement `handleSkillDelete` - Delete skill by ID
- [ ] Implement `handleSkillApply` - Apply skill template to context
- [ ] Implement `handleSkillList` - List all skills (with filtering)
- [ ] Add to skills.Service: `Update`, `Delete`, `Apply`, `List` methods
- [ ] Update discovery.go with new tools
- [ ] Add comprehensive tests

**Acceptance Criteria**:
- Full CRUD for skills
- skill_apply executes skill template logic
- Team-scoped (no cross-team access)
- Test coverage ≥80%

---

### 2.2 Remove Legacy REST Endpoints

**Issue**: Dual endpoint architecture (REST + MCP) creates confusion

**Tasks**:
- [ ] Add deprecation warnings to legacy endpoints
- [ ] Update client examples to use `/mcp` endpoint
- [ ] Remove legacy endpoint handlers from `server.go` (lines 133-167)
- [ ] Keep only `/mcp` endpoint in RegisterRoutes()
- [ ] Update documentation (README, SPEC.md)
- [ ] Remove legacy endpoint tests

**Migration Path**:
1. Deprecate: Add HTTP 410 Gone responses with migration instructions
2. Wait: Allow 1 sprint for client migration
3. Remove: Delete legacy handler code

**Acceptance Criteria**:
- Only `/mcp` endpoint exists
- All tools accessible via `tools/call` method
- Spec-compliant architecture
- No breaking changes for clients using `/mcp`

---

## Phase 3: Medium Priority (P2) - Target: Sprint 3

### 3.1 Implement Analytics Service (`pkg/analytics/`)

**Tasks**:
- [ ] Create `pkg/analytics/` directory
- [ ] Create `pkg/analytics/service.go` with Service struct
- [ ] Implement `GetMetrics(ctx, filters) (*Metrics, error)` - Usage metrics
- [ ] Implement aggregation logic (checkpoint count, search queries, etc.)
- [ ] Create `pkg/analytics/types.go`
- [ ] Create `pkg/analytics/service_test.go` with ≥80% coverage
- [ ] Add OpenTelemetry instrumentation
- [ ] Create `pkg/analytics/CLAUDE.md`

**Dependencies**:
- Qdrant for metrics storage
- OpenTelemetry for metrics collection

**Acceptance Criteria**:
- Returns usage statistics (tool calls, checkpoints, searches)
- Team-scoped metrics (no cross-team visibility)
- Test coverage ≥80%

---

### 3.2 Add Analytics Handlers

**Tasks**:
- [ ] Create `handleAnalyticsGet` - Wire to analytics.Service
- [ ] Update tool routing in protocol.go
- [ ] Add to discovery.go
- [ ] Add handler tests

**Acceptance Criteria**:
- analytics_get tool works via `/mcp`
- Returns structured metrics
- Integration tests pass

---

### 3.3 Add Authentication (OAuth 2.0)

**Issue**: No auth on `/mcp` endpoint (MVP mode)

**Tasks**:
- [ ] Create `pkg/auth/` package
- [ ] Implement OAuth 2.0 client credentials flow
- [ ] Add Bearer token validation middleware
- [ ] Add JWT claims extraction (user ID, team ID)
- [ ] Update ExtractOwnerID to use JWT claims
- [ ] Remove MVP auth bypass (protocol.go:128)
- [ ] Add auth tests
- [ ] Document auth setup in README

**Acceptance Criteria**:
- All `/mcp` requests require valid Bearer token
- JWT claims used for multi-tenant isolation
- No MVP bypass code remains
- Test coverage ≥80%

---

### 3.4 Add Rate Limiting

**Issue**: Spec claims "10 RPS, 20 burst" but not implemented

**Tasks**:
- [ ] Add rate limiting middleware (use golang.org/x/time/rate)
- [ ] Configure per-user rate limits (10 RPS, 20 burst)
- [ ] Add rate limit headers (X-RateLimit-*)
- [ ] Return 429 Too Many Requests when exceeded
- [ ] Add metrics for rate limit hits
- [ ] Add tests

**Acceptance Criteria**:
- Rate limiting enforced per user (from JWT)
- Proper HTTP 429 responses
- Metrics available in analytics
- Test coverage ≥80%

---

## Phase 4: Testing & Documentation - Continuous

### 4.1 Integration Tests

**Tasks**:
- [ ] Create `pkg/mcp/integration_test.go`
- [ ] Test each tool end-to-end (full request/response cycle)
- [ ] Test error scenarios (invalid input, missing service, etc.)
- [ ] Test multi-tenant isolation (no cross-team/project leaks)
- [ ] Test with real Qdrant instance (not mocks)

**Acceptance Criteria**:
- All 16 tools have E2E tests
- Tests run in CI/CD
- Coverage ≥80% overall

---

### 4.2 Documentation Updates

**Tasks**:
- [ ] Update `docs/specs/mcp/SPEC.md` with accurate status
- [ ] Update tool list in SPEC.md (match implementation)
- [ ] Update README.md with current feature set
- [ ] Update `pkg/mcp/CLAUDE.md` with architecture
- [ ] Add examples for each tool in `examples/mcp/`
- [ ] Update CHANGELOG.md

**Acceptance Criteria**:
- Spec matches implementation exactly
- No false "Status: Complete" claims
- Examples runnable by users

---

## Success Criteria

### Definition of Done (for entire plan)

- [ ] All 16 tools fully implemented (no stubs, no placeholders)
- [ ] All services created (skills, troubleshoot, analytics)
- [ ] Test coverage ≥80% across all packages
- [ ] Integration tests passing for all tools
- [ ] SPEC.md accurate (no divergence)
- [ ] Legacy REST endpoints removed
- [ ] Authentication implemented
- [ ] Rate limiting implemented
- [ ] Documentation complete

### Quality Gates

**Per Tool**:
- Service layer implemented with business logic
- Handler integrated with service
- Tests written (unit + integration)
- Coverage ≥80%
- OpenTelemetry instrumentation added
- Multi-tenant isolation verified
- Security review passed (no data leakage)

**Per Service**:
- Package structure follows guidelines
- CLAUDE.md documentation created
- All public APIs have godoc
- Error handling follows standards
- Context propagation throughout

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Skills service complexity | High | Start with minimal MVP, iterate |
| Troubleshoot AI integration | Medium | Use OpenAI API initially, abstract for future |
| Analytics performance | Medium | Use Qdrant aggregations, cache results |
| Auth breaking changes | High | Add auth in parallel, deprecate MVP mode gradually |
| Timeline slippage | Medium | Prioritize P0 items, defer P2 to later sprints |

---

## Timeline Estimate

**Sprint 1 (2 weeks)**: Phase 1 (P0 items)
- Skills service + handlers
- Troubleshoot service + handlers
- Fix stubbed/wrong implementations

**Sprint 2 (2 weeks)**: Phase 2 (P1 items)
- Complete skills CRUD
- Remove legacy endpoints
- Add list_patterns

**Sprint 3 (2 weeks)**: Phase 3 (P2 items)
- Analytics service
- Authentication
- Rate limiting

**Continuous**: Phase 4 (Testing & Docs)
- Run alongside each sprint
- Final verification in Sprint 3

**Total**: 6 weeks to complete all phases

---

## References

- MCP Audit Report: Internal analysis (2025-01-19)
- MCP Streamable HTTP Spec: 2025-03-26
- Current SPEC.md: `docs/specs/mcp/SPEC.md`
- Package Guidelines: `docs/standards/package-guidelines.md`
- Testing Standards: `docs/standards/testing-standards.md`

---

**Next Steps**:
1. Review and approve this plan
2. Update SPEC.md to reflect "In Progress" status
3. Create GitHub issues for each phase
4. Start Sprint 1 with skills service implementation
