# Gap Resolution Roadmap

**Purpose**: Multi-session workflow to resolve ALL implementation gaps identified in spec vs code analysis.
**Created**: 2025-11-26
**Status**: Active

---

## Gap Summary

| Category | Count | Priority |
|----------|-------|----------|
| Critical MVP | 3 | P0 |
| gRPC Services | 6 | P1 |
| Infrastructure | 4 | P1-P2 |
| Code TODOs | 3 | P2 |
| Future | 3 | P3 |

**Total**: 19 gaps across 5 categories

---

## Dependency Graph

```
                    ┌─────────────────┐
                    │ Session 1: MVP  │
                    │ Memory Core     │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
    ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
    │ Session 2:  │  │ Session 3:  │  │ Session 4:  │
    │ Memory Svc  │  │ Checkpoint  │  │ Audit Log   │
    └──────┬──────┘  └──────┬──────┘  └─────────────┘
           │                │
           ▼                ▼
    ┌─────────────┐  ┌─────────────┐
    │ Session 5:  │  │ Session 6:  │
    │ Remediation │  │ Policy Svc  │
    └──────┬──────┘  └──────┬──────┘
           │                │
           └───────┬────────┘
                   ▼
           ┌─────────────┐
           │ Session 7:  │
           │ Multi-tenant│
           └──────┬──────┘
                  │
    ┌─────────────┼─────────────┐
    │             │             │
    ▼             ▼             ▼
┌─────────┐ ┌─────────┐ ┌─────────────┐
│Session 8│ │Session 9│ │ Session 10: │
│ Skill   │ │ Agent   │ │ Isolation   │
└─────────┘ └─────────┘ └─────────────┘
                              │
                              ▼
                       ┌─────────────┐
                       │ Session 11: │
                       │ Context-Fold│
                       └─────────────┘
```

---

## Session 1: Memory Service Core (P0 - Critical)

**Gaps Addressed**:
1. Memory search on session start (`internal/grpc/session.go:73`)
2. Distillation pipeline (`internal/grpc/session.go:97, 105`)
3. Qdrant TLS (`internal/qdrant/client.go:123`)

**Prerequisites**: None (foundational)

**Deliverables**:
- [ ] `internal/memory/service.go` - Memory service implementation
- [ ] `internal/memory/embedding.go` - Embedding generation (OpenAI/local)
- [ ] `internal/memory/distiller.go` - Async distillation worker
- [ ] `internal/qdrant/client.go` - Add TLS configuration
- [ ] Update `internal/grpc/session.go` - Wire memory search on Start()
- [ ] Update `internal/grpc/session.go` - Wire distillation on End()

**Tests**:
- [ ] Memory service unit tests (>80% coverage)
- [ ] Distillation queue tests
- [ ] Integration test: session start → memory injection
- [ ] Integration test: session end → distillation triggered

**Acceptance Criteria**:
```gherkin
Given a session starts for project with existing memories
When Session.Start is called
Then relevant memories are searched
And injected_context includes up to 3 relevant memories
And total injection is <500 tokens

Given a session ends with outcome="success"
When Session.End is called with distill=true
Then distillation job is queued
And returns immediately (async)
```

**Estimated Effort**: 1 session (~2-4 hours)

---

## Session 2: MemoryService gRPC (P1)

**Gaps Addressed**:
- MemoryService (Search, Store, Feedback, Get)

**Prerequisites**: Session 1

**Deliverables**:
- [ ] `internal/grpc/memory_service.go` - gRPC service implementation
- [ ] `internal/grpc/grpc_services.go` - Add MemoryGRPCService wrapper
- [ ] `internal/grpc/server.go` - Add HTTP handlers for memory routes
- [ ] Update `servers/contextd/schema.json` - Add Memory.* schemas

**Tests**:
- [ ] gRPC Memory.Search test
- [ ] gRPC Memory.Store test
- [ ] gRPC Memory.Feedback test
- [ ] gRPC Memory.Get test
- [ ] HTTP memory endpoint tests
- [ ] Dual-protocol memory integration test

**Acceptance Criteria**:
```gherkin
Given memories exist in Qdrant
When Memory.Search is called with query="error handling"
Then semantically similar memories are returned
And results are filtered by scope hierarchy
And confidence scores are included

Given a new memory to store
When Memory.Store is called
Then memory is embedded and stored in Qdrant
And returns the memory ID
```

**Estimated Effort**: 1 session (~2-3 hours)

---

## Session 3: CheckpointService (P1)

**Gaps Addressed**:
- CheckpointService (Save, List, Resume)

**Prerequisites**: Session 1 (Qdrant patterns)

**Deliverables**:
- [ ] `internal/checkpoint/service.go` - Checkpoint service implementation
- [ ] `internal/grpc/checkpoint_service.go` - gRPC service implementation
- [ ] `internal/grpc/server.go` - Add HTTP handlers for checkpoint routes
- [ ] Update `servers/contextd/schema.json` - Add Checkpoint.* schemas

**Tests**:
- [ ] Checkpoint save/list/resume unit tests
- [ ] Context summarization tests
- [ ] gRPC + HTTP dual-protocol tests

**Acceptance Criteria**:
```gherkin
Given an active session with context
When Checkpoint.Save is called with summary
Then context is stored in Qdrant
And checkpoint_id is returned

Given existing checkpoints
When Checkpoint.Resume is called with level="summary"
Then only summary is returned (token-efficient)
When level="full"
Then full context is available via ref
```

**Estimated Effort**: 1 session (~2-3 hours)

---

## Session 4: Audit Logging (P1)

**Gaps Addressed**:
- Audit logging for all operations

**Prerequisites**: None (can run in parallel with Sessions 2-3)

**Deliverables**:
- [ ] `internal/audit/logger.go` - Audit log interface and implementation
- [ ] `internal/audit/middleware.go` - gRPC interceptor + Echo middleware
- [ ] `internal/grpc/dual_server.go` - Wire audit interceptors
- [ ] Update `docs/spec/interface/security.md` - Document audit format

**Tests**:
- [ ] Audit logger unit tests
- [ ] Interceptor tests (gRPC + HTTP)
- [ ] Verify all operations logged

**Acceptance Criteria**:
```gherkin
Given any tool call (gRPC or HTTP)
When the operation completes
Then an audit log entry is created
And includes: timestamp, session_id, operation, outcome, duration
And sensitive data is NOT logged
```

**Estimated Effort**: 1 session (~2 hours)

---

## Session 5: RemediationService (P1)

**Gaps Addressed**:
- RemediationService (Search, Record)

**Prerequisites**: Session 1 (Qdrant patterns)

**Deliverables**:
- [ ] `internal/remediation/service.go` - Remediation service implementation
- [ ] `internal/grpc/remediation_service.go` - gRPC service implementation
- [ ] `internal/grpc/server.go` - Add HTTP handlers
- [ ] Update `servers/contextd/schema.json` - Add Remediation.* schemas

**Tests**:
- [ ] Remediation search/record unit tests
- [ ] Error pattern matching tests
- [ ] Dual-protocol integration tests

**Acceptance Criteria**:
```gherkin
Given an error occurs
When Remediation.Search is called with error message
Then similar past remediations are returned
And ranked by relevance and confidence

Given a successful fix
When Remediation.Record is called
Then the fix pattern is stored
And becomes searchable for future errors
```

**Estimated Effort**: 1 session (~2 hours)

---

## Session 6: PolicyService (P1)

**Gaps Addressed**:
- PolicyService (Check, List, Get)

**Prerequisites**: Session 7 (multi-tenant for scope)

**Deliverables**:
- [ ] `internal/policy/service.go` - Policy service implementation
- [ ] `internal/policy/engine.go` - Policy evaluation engine
- [ ] `internal/grpc/policy_service.go` - gRPC service implementation
- [ ] `internal/grpc/server.go` - Add HTTP handlers
- [ ] Update `servers/contextd/schema.json` - Add Policy.* schemas

**Tests**:
- [ ] Policy check/list/get unit tests
- [ ] Policy evaluation tests (compliance/violation)
- [ ] Scope hierarchy tests (project < team < org)

**Acceptance Criteria**:
```gherkin
Given policies exist at org/team/project level
When Policy.Check is called with an action
Then all applicable policies are evaluated
And violations are returned with severity

Given a policy violation
Then suggestions for compliance are included
```

**Estimated Effort**: 1 session (~2-3 hours)

---

## Session 7: Multi-Tenant Routing (P1)

**Gaps Addressed**:
- Multi-tenant collection routing
- Session TTL/expiration

**Prerequisites**: Sessions 1-3 (services to route)

**Deliverables**:
- [ ] `internal/tenant/router.go` - Collection routing by tenant/team/project
- [ ] `internal/session/manager.go` - Add TTL enforcement
- [ ] `internal/session/cleanup.go` - Background cleanup worker
- [ ] Update all services to use tenant router

**Tests**:
- [ ] Tenant isolation tests (tenant A can't see tenant B)
- [ ] Session expiration tests
- [ ] Cleanup worker tests

**Acceptance Criteria**:
```gherkin
Given tenant_id, team_id, project in session
When any memory/checkpoint/policy query runs
Then only collections for that scope are searched
And cross-tenant access is impossible

Given a session with TTL
When TTL expires
Then session is automatically cleaned up
And associated temp data is removed
```

**Estimated Effort**: 1 session (~3 hours)

---

## Session 8: SkillService (P2)

**Gaps Addressed**:
- SkillService (CRUD)

**Prerequisites**: Session 7 (multi-tenant)

**Deliverables**:
- [ ] `internal/skill/service.go` - Skill service implementation
- [ ] `internal/grpc/skill_service.go` - gRPC service implementation
- [ ] `internal/grpc/server.go` - Add HTTP handlers
- [ ] Update `servers/contextd/schema.json` - Add Skill.* schemas

**Tests**:
- [ ] Skill CRUD unit tests
- [ ] Skill prompt template tests
- [ ] Scope isolation tests

**Estimated Effort**: 1 session (~2 hours)

---

## Session 9: AgentService (P2)

**Gaps Addressed**:
- AgentService (CRUD)

**Prerequisites**: Session 7 (multi-tenant)

**Deliverables**:
- [ ] `internal/agent/service.go` - Agent config service
- [ ] `internal/grpc/agent_service.go` - gRPC service implementation
- [ ] `internal/grpc/server.go` - Add HTTP handlers
- [ ] Update `servers/contextd/schema.json` - Add Agent.* schemas

**Tests**:
- [ ] Agent CRUD unit tests
- [ ] Agent config validation tests
- [ ] Collection access control tests

**Estimated Effort**: 1 session (~2 hours)

---

## Session 10: Process Isolation (P2)

**Gaps Addressed**:
- seccomp profiles
- Linux namespaces
- `internal/isolation/` implementation

**Prerequisites**: Sessions 1-9 (functional services first)

**Deliverables**:
- [ ] `internal/isolation/seccomp.go` - seccomp profile loading
- [ ] `internal/isolation/namespace.go` - Namespace creation
- [ ] `internal/isolation/executor.go` - Isolated command executor
- [ ] Update `internal/grpc/safeexec.go` - Use isolated executor

**Tests**:
- [ ] seccomp profile tests
- [ ] Namespace isolation tests
- [ ] Security boundary tests (can't escape sandbox)

**Acceptance Criteria**:
```gherkin
Given a Bash command to execute
When SafeExec.Bash runs
Then command runs in isolated namespace
And syscalls are restricted by seccomp profile
And filesystem access is limited to project path
```

**Estimated Effort**: 1-2 sessions (~4-6 hours)

---

## Session 11: Context-Folding (P3 - Future)

**Gaps Addressed**:
- Context-folding branch/return tools

**Prerequisites**: All P0-P2 complete

**Deliverables**:
- [ ] `internal/folding/branch.go` - Branch creation
- [ ] `internal/folding/return.go` - Return handling
- [ ] `internal/grpc/folding_service.go` - gRPC service
- [ ] New proto definitions for FoldingService

**Tests**:
- [ ] Branch/return lifecycle tests
- [ ] Budget enforcement tests
- [ ] Nested branch tests

**Estimated Effort**: 1-2 sessions (~4-6 hours)

---

## Code TODOs (Can be addressed during other sessions)

### TODO 1: Load config from file
**Location**: `cmd/contextd/main.go:80`
**Effort**: 30 min
**Assign to**: Session 1 (as part of setup)

### TODO 2: ToolHost implementation
**Location**: `internal/plugin/serve.go:51,53`
**Effort**: 1 hour
**Assign to**: Session 8 or 9 (plugin support)

### TODO 3: Telemetry TLS/version configurable
**Location**: `internal/telemetry/provider.go:23,31,67`
**Effort**: 30 min
**Assign to**: Session 4 (observability work)

---

## Execution Strategy

### Parallel Tracks

**Track A** (Core Services):
```
Session 1 → Session 2 → Session 5
```

**Track B** (Persistence):
```
Session 1 → Session 3 → Session 7
```

**Track C** (Governance):
```
Session 7 → Session 6 → Sessions 8,9
```

**Track D** (Security):
```
Session 4 → Session 10
```

### Recommended Order (Sequential)

1. **Session 1**: Memory Core (foundation for everything)
2. **Session 4**: Audit Logging (can run early, needed for compliance)
3. **Session 2**: MemoryService gRPC
4. **Session 3**: CheckpointService
5. **Session 5**: RemediationService
6. **Session 7**: Multi-Tenant Routing (needed for Policy)
7. **Session 6**: PolicyService
8. **Sessions 8,9**: Skill + Agent (parallel)
9. **Session 10**: Process Isolation
10. **Session 11**: Context-Folding (future)

### Per-Session Checklist

Before starting each session:
- [ ] Read relevant spec files
- [ ] Review dependency sessions are complete
- [ ] Check `servers/contextd/schema.json` is current

After completing each session:
- [ ] Run `go test ./...` - all pass
- [ ] Run `go build ./...` - no errors
- [ ] Coverage check for new code (>80%)
- [ ] Update `schema.json` with new endpoints
- [ ] Update `TOOL.md` if new tools added
- [ ] Update this roadmap with completion status

---

## Progress Tracking

| Session | Status | Started | Completed | Notes |
|---------|--------|---------|-----------|-------|
| 1: Memory Core | **DONE** | 2025-11-26 | 2025-11-26 | Memory search, distiller, TLS |
| 2: MemoryService | **DONE** | 2025-11-26 | 2025-11-26 | gRPC + HTTP endpoints |
| 3: CheckpointService | **DONE** | 2025-11-26 | 2025-11-26 | Save/List/Resume, 48% coverage |
| 4: Audit Logging | **DONE** | 2025-11-26 | 2025-11-26 | 92.5% coverage |
| 5: RemediationService | **DONE** | 2025-11-26 | 2025-11-26 | Semantic search, categorization |
| 6: PolicyService | **DONE** | 2025-11-26 | 2025-11-26 | Check/List/Get |
| 7: Multi-Tenant | **DONE** | 2025-11-26 | 2025-11-26 | 93.5% router coverage |
| 8: SkillService | **DONE** | 2025-11-26 | 2025-11-26 | Full CRUD |
| 9: AgentService | **DONE** | 2025-11-26 | 2025-11-26 | Full CRUD + validation |
| 10: Isolation | **DONE** | 2025-11-26 | 2025-11-26 | seccomp + namespaces (Phase 1 MVP) |
| 11: Context-Folding | Not Started | - | - | Future |

---

## References

- Interface spec: `docs/spec/interface/SPEC.md`
- Architecture: `docs/spec/interface/architecture.md`
- Proto definitions: `api/proto/contextd/v1/contextd.proto`
- ReasoningBank spec: `docs/spec/reasoning-bank/SPEC.md`
- Context-Folding spec: `docs/spec/context-folding/SPEC.md`
- Collection architecture: `docs/spec/collection-architecture/SPEC.md`
