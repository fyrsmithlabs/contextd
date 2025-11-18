# Remediation Requirements

**Parent**: [../SPEC.md](../SPEC.md)

This document defines the functional and non-functional requirements for the remediation system.

---

## Functional Requirements

### FR1: Error Solution Storage

**Requirement**: The system MUST store error solutions with rich context.

**Details**:
- Error message (required)
- Error type/class (required)
- Solution description (required)
- Stack trace (optional)
- Tags for categorization (optional, max 10)
- Severity level (optional: low, medium, high, critical)
- Additional context metadata (optional, max 20 entries)
- Project path (optional, for reference)

### FR2: Intelligent Error Matching

**Requirement**: The system MUST find similar errors using hybrid matching.

**Algorithm**:
- **Hybrid Score**: 70% semantic similarity + 30% string similarity
- **Boost Factors**:
  - +10% for error type match
  - +15% for stack trace match
- **Minimum Thresholds**:
  - Semantic score ≥ 0.5
  - String score ≥ 0.3
  - Hybrid score ≥ 0.6

### FR3: Error Normalization

**Requirement**: The system MUST normalize errors for pattern matching.

**Transformations**:
- Line numbers: `line 42` → `LINE_NUM`
- Memory addresses: `0x7f3b4c1234a0` → `MEM_ADDR`
- Timestamps: `2025-01-15 14:30:45` → `TIMESTAMP`
- File paths: `/home/user/project/main.go` → `main.go`
- UUIDs: `550e8400-e29b-41d4-a716-446655440000` → `UUID`
- Process IDs: `PID 12345` → `PID`
- Whitespace: Multiple spaces → Single space

### FR4: Global Knowledge Sharing

**Requirement**: Remediations MUST be stored in shared database accessible to all projects.

**Rationale**:
- Error solutions are universal knowledge
- No project-level isolation for remediations
- Enables cross-project learning

### FR5: Vector Embeddings

**Requirement**: The system MUST generate vector embeddings for semantic search.

**Supported Models**:
- **TEI (Recommended)**: BAAI/bge-large-en-v1.5 (1024 dimensions)
- **OpenAI**: text-embedding-3-small (1536 dimensions)

**Embedding Format**: `"{error_type}: {error_message}"`

---

## Non-Functional Requirements

### NFR1: Performance

**Latency Targets** (P95):
- Create remediation: ≤ 150ms
- Search remediations: ≤ 250ms

**Throughput Targets**:
- Creates: ≥ 200/min
- Searches: ≥ 600/min

### NFR2: Accuracy

**Matching Quality**:
- Precision: ≥ 0.8 (80% of matches are relevant)
- Recall: ≥ 0.7 (70% of relevant solutions found)
- Match score correlation with relevance: ≥ 0.75

### NFR3: Scalability

**Collection Size**:
- Current: ~1,000 remediations
- Target: ~100,000 remediations (100x growth)
- Query performance: Sub-linear complexity (HNSW)

**Resource Usage**:
- Memory: ~50MB per 10,000 vectors
- Disk: ~200MB per 10,000 vectors (compressed)

### NFR4: Reliability

**Availability**: 99.9% uptime
**Data Durability**: No data loss on crashes
**Error Handling**: Graceful degradation, no panics

### NFR5: Security

**Data Privacy**:
- User responsible for sanitizing sensitive data
- No automatic PII detection
- No credential scanning (use pre-commit hooks)

**Input Validation**:
- Size limits enforced (error message: 10KB, solution: 10KB, stack trace: 50KB)
- No SQL injection risk (vector database)
- No command injection (no shell execution)

### NFR6: Observability

**Tracing**:
- Span per operation (create, search)
- Attributes: error_type, project_path, database
- Error recording on failures

**Metrics**:
- `remediation.create.total` - Counter
- `remediation.search.total` - Counter
- `remediation.search.duration` - Histogram
- `remediation.match.quality` - Histogram

---

## Validation Rules

### CreateRemediationRequest

- `error_message`: Required, non-empty, max 10KB
- `error_type`: Required, non-empty, max 500 chars
- `solution`: Required, non-empty, max 10KB
- `severity`: Optional, must be: low|medium|high|critical
- `tags`: Optional, max 10 tags, each max 50 chars
- `context`: Optional, max 20 entries, keys max 50 chars, values max 500 chars
- `stack_trace`: Optional, max 50KB

### SearchRequest

- `error_message`: Required, non-empty, max 10KB
- `limit`: Required, 1-100
- `min_score`: Optional, 0.0-1.0
- `tags`: Optional, max 10 tags
