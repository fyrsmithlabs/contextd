---
title: contextd Interface
status: DEPRECATED
created: 2025-11-23
updated: 2026-01-06
author: contextd team
version: 2.0.0
deprecated: true
deprecated_reason: "contextd v2 uses MCP (Model Context Protocol) via stdio transport, not gRPC. See internal/mcp/ for current implementation."
---

# contextd Interface Specification

**⚠️ DEPRECATED**: This specification describes a gRPC-based interface that was planned but never implemented. contextd v2 uses the **MCP (Model Context Protocol)** with stdio transport instead. See `internal/mcp/server.go` for the current implementation.

## Overview

~~gRPC interface for contextd server enabling Claude to use secure, scrubbed tools via programmatic code execution.~~

**Current Architecture**: MCP server with stdio transport providing all memory, checkpoint, remediation, repository, and context-folding tools.

**Purpose**: Replace built-in agent tools (Bash, Read, Write) with secure alternatives that scrub secrets and enforce capabilities.

**Design**: Based on [Anthropic Advanced Tool Use](https://www.anthropic.com/engineering/advanced-tool-use) patterns.

## Quick Reference

| Component | Technology |
|-----------|------------|
| Server | Native Go |
| API | **Dual-Protocol**: gRPC + HTTP/REST (same port via cmux) |
| HTTP Framework | Echo v4 |
| Discovery | TOOL.md + schema.json (lazy) |
| Vector DB | Qdrant |
| Scrubbing | gitleaks |
| Isolation | seccomp + Linux namespaces |

## Architecture

@./architecture.md

```
┌──────────────────────────────────────────┐
│ Claude                                   │
│  1. Read TOOL.md (discover tools)        │
│  2. Read schema.json (get examples)      │
│  3. Choose protocol:                     │
│     - gRPC: Write Python with grpcio     │
│     - HTTP: Use curl/requests            │
└──────────────────────────────────────────┘
                    │
                    │ gRPC OR HTTP (same port :50051)
                    ▼
┌──────────────────────────────────────────┐
│ contextd server (dual-protocol)          │
│  ├── cmux (port multiplexer)             │
│  │   ├── gRPC Services (HTTP/2+grpc)     │
│  │   └── Echo HTTP API (HTTP/1.1)        │
│  ├── Scrubbing Pipeline (gitleaks)       │
│  └── Tool Executors (process-isolated)   │
└──────────────────────────────────────────┘
```

## Core Components

### Tool Discovery
- **TOOL.md**: Human/AI readable catalog with gRPC endpoint info
- **schema.json**: Input/output schemas with realistic examples
- **Lazy loading**: Claude reads only what's needed

### gRPC Services

| Service | Methods | Purpose |
|---------|---------|---------|
| SafeExecService | Bash, Read, Write | Scrubbed filesystem/shell |
| MemoryService | Search, Store, Feedback, Get | ReasoningBank |
| CheckpointService | Save, List, Resume | Session persistence |
| PolicyService | Check, List, Get | Governance |
| RefService | GetContent | Resolve content refs |
| SessionService | Start, End | Session lifecycle |

### Security Model
@./security.md

- **Scrubbing**: gitleaks on all output
- **Process isolation**: seccomp profiles, namespaces
- **Audit logging**: All operations logged
- **Transport**: Localhost dual-protocol (Phase 1), mTLS (future)

### Configuration
@./configuration.md

- **Server config**: Port, limits, timeouts
- **Scrubbing config**: Patterns, rules
- **Session config**: TTL, scope

## Requirements

### Functional
- FR-1: Dual-protocol server (gRPC + HTTP) on configurable port
- FR-1a: gRPC via HTTP/2 with content-type application/grpc
- FR-1b: HTTP/REST via Echo framework for simpler integration
- FR-2: Tools execute in process-isolated sandbox
- FR-3: All tool output scrubbed before return
- FR-4: Audit log captures all operations
- FR-5: TOOL.md documents all services with gRPC examples
- FR-6: schema.json includes 1-5 realistic examples per method
- FR-7: Tiered responses (summary → preview → ref) for token efficiency
- FR-8: Session management with configurable TTL

### Non-Functional
- NFR-1: Tool latency <100ms (excluding execution time)
- NFR-2: Memory per tool execution <64MB
- NFR-3: Concurrent sessions supported
- NFR-4: Graceful degradation on scrubber failure

### Security
- SEC-1: No secrets in tool responses (gitleaks enforced)
- SEC-2: Path traversal prevented
- SEC-3: Command injection prevented
- SEC-4: Resource limits enforced (cgroups)

## Implementation Phases

### Phase 1: Core gRPC Server
- gRPC service implementation
- SafeExecService (Bash, Read, Write)
- Server-level gitleaks scrubbing
- TOOL.md + schema.json documentation

### Phase 2: Isolation + Security
- seccomp profiles
- Linux namespaces
- Dual scrubbing (tool + server)
- Audit logging

### Phase 3: Memory Services
- MemoryService (ReasoningBank)
- CheckpointService
- Qdrant integration
- RefService for content resolution

### Phase 4: Governance
- PolicyService
- Session management
- Configuration management

## Acceptance Criteria

- [ ] gRPC server functional with all services
- [ ] All tools execute in isolated processes
- [ ] Scrubbing verified (no secret leakage)
- [ ] Audit log captures all operations
- [ ] TOOL.md + schema.json complete with examples
- [ ] Test coverage ≥80%
- [ ] Claude can discover and call tools via code execution

## References

- [Anthropic Advanced Tool Use](https://www.anthropic.com/engineering/advanced-tool-use)
- [gRPC Go](https://grpc.io/docs/languages/go/)
- [gitleaks](https://github.com/gitleaks/gitleaks)
- [seccomp](https://man7.org/linux/man-pages/man2/seccomp.2.html)
