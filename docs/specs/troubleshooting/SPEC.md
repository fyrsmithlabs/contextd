# Feature: Troubleshooting Service

**Package**: `pkg/troubleshooting`
**Version**: 1.0.0
**Status**: Implemented
**Last Updated**: 2025-11-18

---

## Overview

The troubleshooting service provides AI-powered error diagnosis and troubleshooting capabilities for contextd. It analyzes error messages and stack traces, identifies root causes, generates hypotheses, and recommends diagnostic steps and solutions based on historical knowledge and semantic pattern matching.

**Purpose**: Enable automated learning from resolved errors, intelligent diagnosis of new issues, and context-efficient debugging through semantic pattern matching with safety-first approach.

---

## Quick Reference

**Key Technologies**:
- AI-powered diagnosis (semantic search + hybrid scoring)
- Vector embeddings (1536 dimensions)
- Qdrant vector database (shared database)
- OpenTelemetry instrumentation
- MCP tool integration

**Location**:
- Package: `pkg/troubleshooting`
- Collection: `troubleshooting_knowledge` (shared database)
- Handlers: `pkg/mcp/handlers/troubleshooting.go`
- API: `/api/v1/troubleshoot`, `/api/v1/troubleshoot/patterns`

**Components**:
- Diagnosis Engine (5-step troubleshooting process)
- Pattern Retrieval (hybrid semantic search)
- Session Management (track diagnostic sessions)
- Safety Detection (destructive operation warnings)

**Performance Targets**:
- Diagnose (full): < 2s (typical: 1.5s)
- Search Similar: < 300ms (typical: 200ms)
- Store Resolution: < 500ms (typical: 300ms)
- List Patterns: < 100ms (typical: 50ms)

**MCP Tools**:
1. `troubleshoot` - AI-powered error diagnosis
2. `list_patterns` - Browse troubleshooting patterns

---

## Key Features

### AI-Powered Diagnosis

- 5-step troubleshooting process (symptom collection → pattern recognition → hypothesis formation → ranking → action generation)
- Semantic error pattern matching using vector embeddings
- Hybrid scoring: semantic (60%) + success rate (30%) + usage (10%)
- Confidence levels: high (≥0.8), medium (0.5-0.79), low (<0.5)
- Progressive disclosure based on confidence

### Safety-First Approach

- Automatic detection of destructive operations
- Safety warnings for actions (delete, remove, kill, restart, etc.)
- Verification steps before critical actions
- Expected outcome documentation

### Multi-Tenant Architecture

- Shared database for global troubleshooting knowledge
- All projects access universal patterns
- No project-specific filtering needed
- Database-level isolation prevents filter injection

### Feedback Loop

- Success rate tracking (0.0 - 1.0)
- Usage count monitoring
- Pattern evolution and learning
- Continuous improvement through feedback

---

## Detailed Documentation

**Requirements & Design**:
@./troubleshooting/requirements.md - Functional & non-functional requirements, features, security
@./troubleshooting/architecture.md - System design, components, data models, hybrid scoring
@./troubleshooting/patterns.md - 5-step process, confidence levels, safety detection

**Implementation**:
@./troubleshooting/workflows.md - API specs, HTTP/MCP endpoints, usage examples
@./troubleshooting/implementation.md - Testing requirements, optimization, future enhancements

---

## Data Categories

**Error Categories**:
- Configuration (config errors, missing env vars)
- Resource (out of memory, disk full)
- Dependency (missing library, version mismatch)
- Permission (access denied, file permissions)
- Logic (nil pointer, index out of bounds)
- Network (connection refused, timeout)
- Storage (database errors, file I/O)
- General (uncategorized)

**Severity Levels**:
- Critical (service crash, data loss, security breach)
- High (major feature broken, workaround exists)
- Medium (minor feature broken, inconvenient)
- Low (cosmetic, edge case, minor annoyance)

---

## Integration Points

**Internal Dependencies**:
- `pkg/vectorstore` - Universal vector store interface
- `pkg/embedding` - Embedding generation service
- `pkg/telemetry` - OpenTelemetry instrumentation
- `pkg/validation` - Request validation

**External Dependencies**:
- Vector Database: Qdrant (local instance)
- Embedding Service: OpenAI API or TEI (local)
- Monitoring: OpenTelemetry collector (optional)

**API Surface**:
- HTTP: `/api/v1/troubleshoot`, `/api/v1/troubleshoot/patterns`
- MCP: `troubleshoot`, `list_patterns`
- Service: `Diagnose()`, `SearchSimilarIssues()`, `StoreResolution()`, `ListPatterns()`

---

## Quick Start

### Diagnose an Error

```go
req := &troubleshooting.DiagnosisRequest{
    ErrorMessage: "panic: runtime error: invalid memory address",
    Context: map[string]string{"file": "main.go", "line": "42"},
    Mode: troubleshooting.ModeAuto,
}

session, err := service.Diagnose(ctx, req)
// Returns: Session with diagnosis, similar issues, recommended actions
```

### Store a Resolution

```go
req := &troubleshooting.StoreKnowledgeRequest{
    ErrorPattern:    "connection refused localhost:5432",
    RootCause:       "PostgreSQL service not running",
    Solution:        "sudo systemctl start postgresql",
    Severity:        "high",
    Category:        troubleshooting.CategoryNetwork,
    Tags:            []string{"postgresql", "database"},
}

knowledge, err := service.StoreResolution(ctx, req)
```

### MCP Tool Usage

```bash
# From Claude Code
/troubleshoot "Failed to connect to database: SQLSTATE[HY000] [2002]"
/list_patterns category=database severity=high
```

---

## Summary

The troubleshooting service provides intelligent error diagnosis through AI-powered semantic pattern matching. It combines vector similarity search with success rate and usage tracking to deliver high-confidence recommendations with safety warnings for destructive operations.

**Current Status**: Fully implemented with 5-step diagnosis process, hybrid scoring algorithm, safety detection, and comprehensive MCP/HTTP API integration.

**Next Steps**: Session persistence, automatic feedback loop, pattern evolution, interactive/guided modes, and AI enhancement for novel error analysis.

**Related Specs**: Vector Store (`docs/specs/vectorstore/SPEC.md`), Remediation (`docs/specs/remediation/SPEC.md`)
