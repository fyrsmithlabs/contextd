# Obsolete Specifications

This directory contains specification documents that are no longer accurate for the current
contextd architecture.

## Why These Are Archived

contextd migrated from a planned gRPC-based architecture to MCP (Model Context Protocol)
with stdio transport. These documents reference the old design that was never implemented.

## Archived Files

| File | Original Location | Reason |
|------|-------------------|--------|
| `interface-architecture.md` | `docs/spec/interface/architecture.md` | References gRPC, cmux, PolicyService, SkillService, AgentService - none exist |
| `interface-REQUIREMENTS.md` | `docs/spec/interface/REQUIREMENTS.md` | References gRPC proxy, go-plugin, mTLS - not used in MCP architecture |

## Current Architecture

The current contextd architecture uses:

- **MCP Server**: `internal/mcp/` - Model Context Protocol with stdio transport
- **VectorStore**: `internal/vectorstore/` - chromem (default) or Qdrant
- **Embeddings**: `internal/embeddings/` - FastEmbed with local ONNX models
- **No gRPC**: All communication via MCP tools

See `CLAUDE.md` in the repository root for the current architecture overview.

## Date Archived

2026-01-16
