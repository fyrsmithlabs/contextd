# Feature: Repository Indexing System

**Version**: 0.x
**Status**: Implemented
**Last Updated**: 2025-11-18

---

## Overview

The repository indexing system provides semantic search capabilities over existing repositories and directories. It walks file trees, filters files based on include/exclude patterns and size limits, reads file contents, and creates searchable checkpoints with vector embeddings.

**Key Capabilities**:
- Semantic search across codebases using natural language
- Pattern-based file filtering (include/exclude)
- File size limits (1MB default, 10MB max)
- Path traversal prevention for security
- Integration with checkpoint system for storage

**Status**: Core implementation complete in v0.x with basic indexing, pattern matching, and security features.

---

## Quick Reference

**Technology**:
- Language: Go
- MCP Tool: `index_repository`
- CLI Command: `ctxd index`

**Location**:
- MCP Handler: `pkg/mcp/tools.go` (handleIndexRepository)
- Indexer Logic: `pkg/mcp/tools.go` (indexRepositoryFiles)
- CLI: `cmd/ctxd/index.go`

**Dependencies**:
- Checkpoint service (storage)
- Embedding service (vector generation)
- Vector store (Qdrant, project-specific databases)

**Key Metrics**:
- Default max file size: 1MB
- Maximum file size: 10MB
- Timeout: 5 minutes
- Throughput: ~16 files/second (TEI), ~2 files/second (OpenAI)

---

## Components

**File Tree Traversal**:
- Recursive directory walking via `filepath.Walk`
- Symlink support with cycle detection
- Permission-aware file reading

**Pattern Matching**:
- Include patterns (whitelist files)
- Exclude patterns (blacklist files)
- Glob-style syntax (`*.go`, `node_modules/**`)

**Security**:
- Path traversal prevention
- Input validation (paths, patterns, size)
- File system permission respect
- Multi-tenant isolation (project-specific databases)

**Checkpoint Integration**:
- One checkpoint per indexed file
- Full file contents in description
- Tags: `["indexed", "repository", "<extension>"]`
- Semantic search enabled via embeddings

---

## Detailed Documentation

**Requirements & Design**:
@./indexing/requirements.md - Functional and non-functional requirements, use cases
@./indexing/architecture.md - System components, data flow, performance characteristics

**Implementation**:
@./indexing/implementation.md - API specification, pattern matching, error handling, testing
@./indexing/workflows.md - Indexing phases, usage examples, CLI commands

---

## Current Status

**Phase 1: Core Implementation** (Complete)
- [x] MCP tool handler
- [x] File tree traversal
- [x] Pattern matching (include/exclude)
- [x] File size filtering
- [x] Path traversal prevention
- [x] Checkpoint creation integration
- [x] CLI command

**Phase 2: Optimizations** (Future)
- [ ] Batch embedding generation (10x speedup)
- [ ] Parallel processing (20x speedup)
- [ ] Incremental indexing
- [ ] Progress reporting

**Phase 3: Enhanced Error Handling** (Future)
- [ ] Continue on file errors
- [ ] Retry with backoff
- [ ] Per-file error reporting
- [ ] Binary file detection

**Phase 4: Advanced Features** (Future)
- [ ] AST-based code indexing
- [ ] Large file chunking
- [ ] Metadata extraction
- [ ] De-duplication

---

## Related Documentation

**Standards**:
- [docs/standards/architecture.md](../../standards/architecture.md) - Multi-tenant architecture
- [docs/standards/coding-standards.md](../../standards/coding-standards.md) - Go coding patterns
- [docs/standards/testing-standards.md](../../standards/testing-standards.md) - TDD requirements

**Specifications**:
- [docs/specs/checkpoint/SPEC.md](../checkpoint/SPEC.md) - Checkpoint system
- [docs/specs/multi-tenant/SPEC.md](../multi-tenant/SPEC.md) - Multi-tenant isolation

**ADRs**:
- [docs/architecture/adr/002-universal-multi-tenant-architecture.md](../../architecture/adr/002-universal-multi-tenant-architecture.md) - Database isolation

---

## Summary

Repository indexing enables semantic search over codebases by creating searchable checkpoints from files. The system supports flexible pattern-based filtering, respects file size limits, and maintains security through path validation and multi-tenant isolation.

**Next Steps**:
1. Optimize performance with batch embedding and parallel processing
2. Enhance error handling for partial success scenarios
3. Add advanced features (AST indexing, chunking, de-duplication)
4. Implement incremental indexing for large repositories
