# MCP Requirements

**Parent**: [../SPEC.md](../SPEC.md)

This document defines the functional and non-functional requirements for contextd's MCP integration.

---

## Functional Requirements

### F1: Tool Catalog

**Requirement**: Provide complete coverage of contextd functionality through MCP tools.

**Tools Required**:
1. **Session Management** (3 tools):
   - `checkpoint_save`: Save session checkpoints
   - `checkpoint_search`: Search checkpoints semantically
   - `checkpoint_list`: List recent checkpoints with pagination

2. **Error Resolution** (2 tools):
   - `remediation_save`: Store error solutions
   - `remediation_search`: Find similar error solutions with hybrid matching

3. **AI Diagnosis** (2 tools):
   - `troubleshoot`: AI-powered error diagnosis
   - `list_patterns`: Browse troubleshooting patterns

4. **Repository Indexing** (1 tool):
   - `index_repository`: Index repositories for semantic search

5. **Skills Management** (6 tools):
   - `skill_create`: Create reusable workflow templates
   - `skill_search`: Search skills semantically
   - `skill_list`: List all skills with pagination
   - `skill_update`: Update existing skills
   - `skill_delete`: Delete skills
   - `skill_apply`: Apply skills and track usage

6. **System Operations** (2 tools):
   - `status`: Get service health and metrics
   - `analytics_get`: Get usage analytics and performance metrics

**Total**: 16 MCP tools

### F2: Input Validation

**Requirement**: All tools must validate inputs before processing.

**Validation Rules**:
- **Summary**: Required, max 500 chars
- **Description**: Optional, max 5000 chars
- **Error Message**: Required, max 10000 chars
- **Stack Trace**: Optional, max 50000 chars
- **Project Path**: Required, absolute path, no path traversal
- **Tags**: Max 20 tags, max 50 chars per tag
- **Query**: Required, max 1000 chars
- **Context**: Max 50 fields, max 1000 chars per value
- **Skill Content**: Required, max 50000 chars

**Path Traversal Protection**:
```go
func validateProjectPath(p string) error {
    if !filepath.IsAbs(p) {
        return fmt.Errorf("project_path must be an absolute path")
    }
    if strings.Contains(p, "..") {
        return fmt.Errorf("project_path cannot contain '..'")
    }
    cleaned := filepath.Clean(p)
    if cleaned != p {
        return fmt.Errorf("project_path must be a clean absolute path")
    }
    return nil
}
```

### F3: Error Handling

**Requirement**: Return structured errors with categories and details.

**Error Categories**:
- `validation`: Invalid input provided (400 Bad Request equivalent)
- `not_found`: Resource not found (404 Not Found equivalent)
- `internal`: Internal server error (500 Internal Server Error equivalent)
- `timeout`: Operation timed out (504 Gateway Timeout equivalent)
- `unauthorized`: Unauthorized access (401 Unauthorized equivalent)

**Error Response Format**:
```json
{
  "jsonrpc": "2.0",
  "id": "request-id",
  "error": {
    "code": -32603,
    "message": "[validation] invalid summary",
    "data": {
      "category": "validation",
      "message": "invalid summary",
      "details": {
        "field": "summary",
        "error": "summary is required"
      }
    }
  }
}
```

### F4: Type Safety

**Requirement**: Strong typing with JSON schema validation.

**Implementation**:
- Go structs with JSON schema tags
- Compile-time type checking
- Runtime JSON schema validation
- Custom validators for business rules
- Numeric value clamping to safe ranges

**Example**:
```go
type CheckpointSaveInput struct {
    Summary     string                 `json:"summary" jsonschema:"required,Brief summary (max 500 chars)"`
    Description string                 `json:"description,omitempty" jsonschema:"Detailed description"`
    ProjectPath string                 `json:"project_path" jsonschema:"required,Absolute path"`
    Context     map[string]interface{} `json:"context,omitempty" jsonschema:"Metadata"`
    Tags        []string               `json:"tags,omitempty" jsonschema:"Tags"`
}
```

---

## Non-Functional Requirements

### NF1: Performance

**Response Time Targets**:
- Health checks: <10ms
- Checkpoint save: <100ms
- Checkpoint search: <200ms
- Remediation search: <300ms (hybrid matching)
- AI troubleshoot: <2s (OpenAI API dependency)
- Repository indexing: Variable (depends on size)

**Throughput** (local Qdrant, TEI embeddings):
| Operation | p50 | p95 | p99 |
|-----------|-----|-----|-----|
| checkpoint_save | 250ms | 450ms | 600ms |
| checkpoint_search | 80ms | 150ms | 200ms |
| remediation_search | 120ms | 200ms | 280ms |
| troubleshoot | 2.5s | 4.5s | 6.0s |
| skill_search | 90ms | 160ms | 220ms |

**Concurrent Requests**: HTTP server handles multiple concurrent connections (multi-session support).

### NF2: Security

**Input Validation**:
- Path traversal protection on all file paths
- Length limits on all string inputs to prevent DoS
- Type safety to prevent injection attacks
- No user-controlled filter parameters (database-per-project prevents filter injection)

**Rate Limiting**:
- Default: 10 requests per second per tool
- Default burst: 20 requests
- Algorithm: Token bucket with per-connection, per-tool isolation
- Tool-specific overrides:
  - `troubleshoot`: 2 RPS, 5 burst (expensive AI operations)
  - `skill_create`: 5 RPS, 10 burst (large embeddings)

**Data Isolation**:
- Project-level isolation: Each project has own vector database
- Shared knowledge: Remediations and skills in shared database
- No cross-project access: Queries scoped to project_path

**Transport Security**:
- HTTP transport with remote access support
- Multiple concurrent sessions via HTTP
- Session tracking via `Mcp-Session-Id` header
- No authentication in MVP (trusted network assumption)
- Production recommendations:
  - Deploy behind reverse proxy with TLS (nginx/Caddy)
  - Add authentication (Bearer token, JWT, OAuth)
  - Use VPN or SSH tunnel for remote access
  - Implement rate limiting and DDoS protection

### NF3: Observability

**OpenTelemetry Integration**:
- Traces: OTLP/HTTP exporter, 5s batch timeout, 512 spans per batch
- Metrics: 60s export interval
- Middleware: Auto-instruments all HTTP requests
- Resource: service.name, service.version, deployment.environment
- Propagation: W3C Trace Context + Baggage

**Metrics Collected**:
- HTTP request duration (histogram)
- HTTP request count (counter)
- Active connections (gauge)
- HTTP status codes (labels)
- MCP tool call performance
- Vector store operation timing
- Embedding generation duration

**Logging**:
- Structured logging with context
- Errors logged with telemetry
- Full stack traces for debugging
- Sensitive data redacted from logs

### NF4: Reliability

**Operation Timeouts**:
| Operation | Timeout | Rationale |
|-----------|---------|-----------|
| checkpoint_save | 30s | Embedding generation + DB write |
| checkpoint_search | 10s | Vector similarity search |
| checkpoint_list | 5s | Simple database query |
| remediation_save | 30s | Embedding generation + DB write |
| remediation_search | 10s | Hybrid matching (semantic + string) |
| troubleshoot | 60s | AI diagnosis + knowledge base search |
| list_patterns | 5s | Database query with filtering |
| index_repository | 300s | Large repository indexing |
| skill_create | 120s | Large content embedding + DB write |
| skill_search | 10s | Vector similarity search |
| skill_list | 30s | Database query with pagination |
| skill_update | 120s | Re-embedding + DB update |
| skill_delete | 30s | Database deletion + vector cleanup |
| skill_apply | 30s | Database query + usage tracking |
| status | 30s | Health checks across services |
| analytics_get | 30s | Aggregated metrics calculation |

**Graceful Shutdown**:
1. Signal handling (SIGINT/SIGTERM)
2. Context cancellation for in-flight operations
3. Service cleanup (close database connections)
4. Telemetry flush (5s timeout)
5. Clean process termination

**Health Checks**:
- Service health checks on initialization
- Health status available via `status` tool during runtime

### NF5: Testability

**Unit Tests**:
- Coverage requirement: 100% for tool handlers
- Coverage requirement: 100% for validation functions
- Coverage requirement: 100% for error handling
- Coverage requirement: 100% for type conversion

**Integration Tests**:
- End-to-end tool calls (full request/response cycle)
- Service integration verification
- Timeout handling tests
- Rate limiting enforcement tests

**Performance Tests**:
- Tool call latency benchmarks
- Throughput tests (concurrent request handling)
- Memory usage tests (check for leaks)
- Rate limiter overhead measurement

---

## Compliance Requirements

### MCP Protocol Compliance

**Protocol Version**: 2025-03-26 (Streamable HTTP)

**Implementation Requirements**:
1. **Tool Discovery**: Server advertises all 16 tools with complete schemas
2. **JSON-RPC 2.0**: All messages follow JSON-RPC 2.0 format
3. **Error Handling**: Errors returned as JSON-RPC error objects
4. **Resource Management**: Proper context handling and cancellation
5. **Lifecycle Management**: Graceful initialization and shutdown

**SDK**: Use official `github.com/modelcontextprotocol/go-sdk/mcp` SDK for full compliance.

---

## Summary

**Key Requirements**:
- ✅ 16 MCP tools covering all contextd functionality
- ✅ Strong input validation with path traversal protection
- ✅ Structured error responses with categories
- ✅ Type safety at compile-time, schema, and runtime levels
- ✅ Performance targets: <100ms for checkpoint ops, <2s for AI diagnosis
- ✅ Security: Rate limiting, data isolation, transport security
- ✅ Observability: OpenTelemetry traces and metrics
- ✅ Reliability: Operation timeouts, graceful shutdown, health checks
- ✅ Testability: 100% coverage for critical paths
- ✅ MCP protocol compliance via official SDK
