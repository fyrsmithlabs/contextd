# Feature: MCP Integration

**Version**: 2.0.0
**Status**: Complete
**Last Updated**: 2025-11-04

---

## Overview

The MCP (Model Context Protocol) integration provides a standardized interface for Claude Code and other AI assistants to interact with contextd services. It implements the MCP Streamable HTTP transport (specification version 2025-03-26), enabling remote access and multiple concurrent sessions for distributed teams.

**Purpose**: Bridge between AI assistants and contextd's core functionality.

**Key Capabilities**:
- Session Management: Save and retrieve checkpoints for resuming work
- Error Resolution: Store and search for error solutions with hybrid matching
- AI Diagnosis: Intelligent troubleshooting with hypothesis generation
- Knowledge Management: Skills system for reusable workflows and templates
- Analytics: Usage metrics and performance tracking
- Repository Indexing: Semantic search across codebase files

---

## Quick Reference

**Key Facts**:
- **Technology**: MCP Streamable HTTP (spec 2025-03-26), JSON-RPC 2.0, Go SDK
- **Transport**: HTTP/1.1 with Server-Sent Events (SSE)
- **Location**: `pkg/mcp/`, endpoint POST/GET `/mcp`
- **Port**: 8080 (configurable via CONTEXTD_HTTP_PORT)
- **SDK**: `github.com/modelcontextprotocol/go-sdk/mcp`
- **Tools**: 16 MCP tools covering all contextd functionality
- **Status**: Production-ready (add auth for production deployments)

**Components**:
- MCP Server: Tool registry, validation, rate limiting, telemetry
- Session Management Tools: checkpoint_save, checkpoint_search, checkpoint_list
- Error Resolution Tools: remediation_save, remediation_search
- AI Diagnosis Tools: troubleshoot, list_patterns
- Repository Indexing: index_repository
- Skills Management Tools: skill_create, skill_search, skill_list, skill_update, skill_delete, skill_apply
- System Operations: status, analytics_get

**Performance Targets**:
- Health checks: <10ms
- Checkpoint operations: <100ms
- Search operations: <200ms
- AI diagnosis: <2s
- Repository indexing: Variable (depends on size)

**Security**:
- Input validation: Path traversal protection, length limits
- Rate limiting: Per-connection, per-tool (default: 10 RPS, 20 burst)
- Data isolation: Project-level databases, no cross-project access
- Transport: HTTP (MVP), add TLS + auth for production

**Observability**:
- OpenTelemetry: Traces and metrics
- Structured logging: Errors with context
- Health checks: Service health monitoring

---

## Detailed Documentation

**Requirements & Design**:
@./mcp/requirements.md - Functional & non-functional requirements, compliance
@./mcp/architecture.md - Component design, transport layer, service integration

**Tool Specifications**:
@./mcp/tools.md - Complete tool catalog with input/output schemas

**Workflows & Usage**:
@./mcp/workflows.md - Lifecycle, initialization, common usage patterns

**Implementation**:
@./mcp/implementation.md - Current status, testing, limitations, roadmap

---

## Protocol Compliance

**MCP Version**: 2025-03-26 (Streamable HTTP)

**Compliance Points**:
- ✅ Tool Discovery: Server advertises all 16 tools with complete schemas
- ✅ JSON-RPC 2.0: All messages follow JSON-RPC 2.0 format
- ✅ Error Handling: Errors returned as JSON-RPC error objects
- ✅ Resource Management: Proper context handling and cancellation
- ✅ Lifecycle Management: Graceful initialization and shutdown
- ✅ Session Management: `Mcp-Session-Id` header for multi-client support

**SDK**: Official `github.com/modelcontextprotocol/go-sdk/mcp` ensures full compliance.

---

## Tool Catalog Summary

**16 Tools Total**:

1. **Session Management** (3):
   - checkpoint_save, checkpoint_search, checkpoint_list

2. **Error Resolution** (2):
   - remediation_save, remediation_search

3. **AI Diagnosis** (2):
   - troubleshoot, list_patterns

4. **Repository Indexing** (1):
   - index_repository

5. **Skills Management** (6):
   - skill_create, skill_search, skill_list, skill_update, skill_delete, skill_apply

6. **System Operations** (2):
   - status, analytics_get

**See**: @./mcp/tools.md for complete tool specifications.

---

## Error Response Format

**Structured Errors**:

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

**Error Categories**:
- `validation`: Invalid input (400 Bad Request equivalent)
- `not_found`: Resource not found (404 Not Found equivalent)
- `internal`: Internal server error (500 Internal Server Error equivalent)
- `timeout`: Operation timed out (504 Gateway Timeout equivalent)
- `unauthorized`: Unauthorized access (401 Unauthorized equivalent)

---

## Summary

The MCP integration provides a complete, production-ready interface for AI assistants to interact with contextd services. With 16 tools covering session management, error resolution, AI diagnosis, skills management, and analytics, it enables powerful context-aware workflows for developers.

**Current Status**: Complete (v2.0.0)
**Production Readiness**:
- ✅ Core functionality ready
- ⚠️ Add authentication for production
- ⚠️ Add TLS via reverse proxy for production

**Next Steps**:
- Add authentication (Bearer token, JWT, OAuth)
- Deploy behind reverse proxy with TLS
- Implement advanced rate limiting for scale
- Add caching and batching optimizations

**Related Documentation**:
- Project: [/CLAUDE.md](/home/dahendel/projects/contextd/CLAUDE.md)
- Standards: [/docs/standards/](/home/dahendel/projects/contextd/docs/standards/)
- Architecture: [/docs/architecture/](/home/dahendel/projects/contextd/docs/architecture/)
- MCP Protocol: [https://modelcontextprotocol.io/](https://modelcontextprotocol.io/)
