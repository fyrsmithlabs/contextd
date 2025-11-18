# Feature: Remediation System

**Version**: 1.0.0
**Status**: Implemented
**Last Updated**: 2025-11-04

---

## Overview

The remediation system provides intelligent error solution storage and retrieval using hybrid matching algorithms. It enables developers to save error solutions with context and later find similar errors using a combination of semantic similarity (vector embeddings) and string matching techniques.

**Purpose**: Enable global knowledge sharing of error solutions with intelligent matching.

**Design Goals**:
1. **Intelligent Matching**: Combine semantic and syntactic similarity for accurate error matching
2. **Context-Aware**: Store rich context including stack traces, error types, and metadata
3. **Global Knowledge**: Share error solutions across all projects (stored in shared database)
4. **Fast Retrieval**: Efficient hybrid search with configurable thresholds
5. **Developer-Friendly**: Clear match scores and detailed match explanations

---

## Quick Reference

**Key Technologies**:
- Hybrid matching: 70% semantic + 30% string similarity
- Vector embeddings: BAAI/bge-large-en-v1.5 (TEI) or text-embedding-3-small (OpenAI)
- Storage: Qdrant shared database (global knowledge)
- Algorithm: HNSW for sub-linear vector search

**Location**:
- Package: `pkg/remediation`
- MCP Tools: `remediation_save`, `remediation_search`
- Database: `shared` (remediations collection)

**Components**:
- **Service**: Create, search, CRUD operations
- **Matcher**: Hybrid matching algorithm with boost factors
- **Embedder**: Vector embedding generation
- **VectorStore**: Qdrant storage with HNSW indexing

**Key Features**:
- Error normalization (removes line numbers, addresses, timestamps)
- Stack trace matching (+15% boost)
- Error type matching (+10% boost)
- Configurable weights and thresholds
- Global knowledge sharing across projects

**Performance Targets** (P95):
- Create: ≤ 150ms
- Search: ≤ 250ms
- Throughput: 200+ creates/min, 600+ searches/min

---

## Detailed Documentation

**Requirements & Design**:
@./remediation/requirements.md - Functional & non-functional requirements
@./remediation/architecture.md - System design & component interactions

**Implementation**:
@./remediation/implementation.md - Hybrid matching algorithm details
@./remediation/workflows.md - MCP tools, usage examples, error handling

---

## Hybrid Matching Algorithm (Summary)

**Formula**:
```
hybrid_score = (semantic_score × 0.7) + (string_score × 0.3)

Boost factors:
  Error type match:   +10%
  Stack trace match:  +15%

Final score capped at 1.0
```

**Phases**:
1. **Normalization**: Remove variable parts (line numbers, addresses, timestamps)
2. **Signature Generation**: Extract error type, stack signature, hash
3. **Semantic Similarity**: Vector search with cosine similarity
4. **String Similarity**: Levenshtein distance for syntactic matching
5. **Hybrid Score**: Weighted combination (70/30)
6. **Boost Factors**: Apply error type and stack trace boosts
7. **Filtering & Ranking**: Apply thresholds, sort by final score

**Thresholds**:
- Semantic: ≥ 0.5 (50% similarity)
- String: ≥ 0.3 (30% similarity)
- Hybrid: ≥ 0.6 (60% overall match)

---

## Data Models (Summary)

**Remediation**:
- Core: ID, error_message, error_type, solution
- Context: project_path, tags, severity, metadata
- Debug: stack_trace
- Generated: timestamp, signature (normalized error + type + hash)

**MatchResult**:
- Scores: semantic_score, string_score, hybrid_score
- Match details: error_type_match, stack_trace_match

**SimilarError**:
- Complete remediation + match_score + match_details

---

## MCP Tools (Summary)

### remediation_save

**Purpose**: Store error solution with vector embeddings

**Required**: error_message, error_type, solution
**Optional**: project_path, tags, severity, stack_trace, context

**Output**: Remediation ID + timestamp

### remediation_search

**Purpose**: Find similar errors using hybrid matching

**Required**: error_message
**Optional**: stack_trace (for boost), limit (1-100), min_score (0-1), tags

**Output**: Ranked results with match scores and details

---

## Security & Privacy

**Data Sharing**:
- Remediations stored in **shared database** (global knowledge)
- Accessible to all projects (no project-level isolation)

**User Responsibility**:
- Sanitize stack traces (remove credentials, tokens, keys)
- Redact PII before saving
- No automatic credential/PII detection

**Input Validation**:
- Size limits enforced (error: 10KB, solution: 10KB, stack trace: 50KB)
- No SQL/XSS/command injection risk (vector database, backend service)

**Authentication**: None required (localhost-only access for MVP)

---

## Testing Coverage

**Requirements**:
- Minimum: 80% overall
- Core matching: 100%
- Normalization: 100%
- Signature generation: 100%

**Test Categories**:
- Normalization (~20 cases)
- Signature generation (~15 cases)
- Hybrid matching (~25 cases)
- Validation (~15 cases)
- Integration (end-to-end, multi-tenant, error cases)
- Performance (load/stress testing)

---

## Summary

The remediation system is a production-ready intelligent error solution database with hybrid matching that combines semantic understanding (70%) and syntactic similarity (30%). It stores solutions globally with rich context and enables cross-project learning.

**Current Status**: Implemented, production-ready (v1.0.0)

**Future Enhancements**: Update/delete operations (v2.1), batch operations (v2.2), advanced filters (v2.3), feedback learning (0.9.0-rc-1), pattern detection (v3.1)

**Package**: `pkg/remediation`
**Tests**: `pkg/remediation/*_test.go`
