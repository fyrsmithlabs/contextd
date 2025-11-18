# Remediation Implementation

**Parent**: [../SPEC.md](../SPEC.md)

This document describes the implementation details of the hybrid matching algorithm.

---

## Hybrid Matching Algorithm

### Algorithm Overview

The remediation system uses a sophisticated hybrid matching algorithm that combines semantic understanding with syntactic similarity:

**Formula**:
```
hybrid_score = (semantic_score × 0.7) + (string_score × 0.3)

With boost factors:
  if error_type_match:   hybrid_score × 1.10 (+10%)
  if stack_trace_match:  hybrid_score × 1.15 (+15%)

Final score capped at 1.0
```

---

## Phase 1: Error Normalization

Remove variable parts to enable pattern matching:

**Transformations**:
- Line numbers: `line 42` → `LINE_NUM`
- Memory addresses: `0x7f3b4c1234a0` → `MEM_ADDR`
- Timestamps: `2025-01-15 14:30:45` → `TIMESTAMP`
- File paths: `/home/user/project/main.go` → `main.go`
- UUIDs: `550e8400-e29b-41d4-a716-446655440000` → `UUID`
- Process IDs: `PID 12345` → `PID`
- Whitespace: Multiple spaces → Single space

**Example**:
```
Input:  "SyntaxError at line 42 in /home/user/app.py (PID 1234) at 0x7f3b4c1234a0"
Output: "SyntaxError LINE_NUM in app.py (PID) at MEM_ADDR"
```

---

## Phase 2: Signature Generation

**Components**:
1. **Normalized Error**: Cleaned error message
2. **Error Type**: Extracted error class (e.g., "ImportError", "NullPointerException")
3. **Stack Signature**: Normalized function names + file names from stack trace
4. **Hash**: SHA256 hash of normalized error + type + stack signature

**Error Type Extraction Patterns**:
- Python: `ValueError:` → `valueerror`
- Java: `NullPointerException` → `nullpointerexception`
- Go: `error:` → `error`
- Generic: First `*Error` or `*Exception` word

**Stack Signature Extraction**:
```
Input stack trace:
  at main.processRequest (main.go:42)
  at runtime.goexit (runtime.go:1234)

Output signature: "main.processrequest|main.go|runtime.goexit|runtime.go"
```

---

## Phase 3: Semantic Similarity

**Vector Embedding Generation**:
- Model: BAAI/bge-large-en-v1.5 (TEI) or text-embedding-3-small (OpenAI)
- Dimension: 1536 (configurable based on model)
- Input text: `"{error_type}: {error_message}"`

**Semantic Search**:
- Similarity metric: Cosine similarity
- Candidates: Top 2×N results (for re-ranking)
- Database: "shared" (global knowledge)

**Score Calculation**:
```
semantic_score = 1.0 / (1.0 + distance)
```

---

## Phase 4: String Similarity

**Algorithm**: Levenshtein Distance (edit distance)

**Implementation**:
```go
func CalculateStringSimilarity(text1, text2 string) float64 {
    if text1 == text2 {
        return 1.0
    }

    distance := LevenshteinDistance(text1, text2)
    maxLen := max(len(text1), len(text2))

    similarity := 1.0 - float64(distance)/float64(maxLen)
    return max(0.0, similarity)
}
```

**Why String Similarity?**
- Catches syntactic patterns that embeddings might miss
- Better at matching error codes and identifiers
- Complements semantic understanding

---

## Phase 5: Hybrid Score Computation

**Weighted Combination**:
```go
func CalculateHybridScore(semanticScore, stringScore float64) float64 {
    return (semanticScore * 0.7) + (stringScore * 0.3)
}
```

**Default Weights**:
- Semantic: 70% (understanding meaning)
- String: 30% (syntactic patterns)

**Configurable**:
```go
matcher := NewMatcherWithWeights(
    0.8,  // semantic weight (80%)
    0.2,  // string weight (20%)
    0.6,  // min semantic score
    0.4,  // min string score
    0.7,  // min hybrid score
)
```

---

## Phase 6: Boost Factors

**Error Type Match (+10%)**:
- Exact match: `importerror` == `importerror`
- Fuzzy match: Levenshtein distance allows minor differences
- Examples: `ImportError` matches `ModuleNotFoundError` (both import-related)

**Stack Trace Match (+15%)**:
- Compare normalized stack signatures
- Require 50% overlap of function/file names
- Fuzzy matching on individual components

**Boost Application**:
```go
func BoostScore(result MatchResult) MatchResult {
    boostFactor := 1.0

    if result.ErrorTypeMatch {
        boostFactor += 0.1
    }

    if result.StackTraceMatch {
        boostFactor += 0.15
    }

    result.HybridScore *= boostFactor
    if result.HybridScore > 1.0 {
        result.HybridScore = 1.0  // Cap at 1.0
    }

    return result
}
```

---

## Phase 7: Filtering & Ranking

**Minimum Thresholds**:
- Semantic score: ≥ 0.5 (50% semantic similarity)
- String score: ≥ 0.3 (30% string similarity)
- Hybrid score: ≥ 0.6 (60% overall match)

**Re-ranking**:
1. Filter results below thresholds
2. Sort by final hybrid score (descending)
3. Return top N results

**Why These Thresholds?**
- Semantic 0.5: Ensures some conceptual similarity
- String 0.3: Allows for reasonable syntactic variation
- Hybrid 0.6: High-confidence matches only (balances precision/recall)

---

## Testing Requirements

### Unit Tests

**Coverage Requirements**:
- Minimum: 80% overall
- Core matching: 100%
- Normalization: 100%
- Signature generation: 100%

**Test Categories**:

1. **Normalization Tests** (~20 test cases)
   - Line numbers
   - Memory addresses
   - Timestamps
   - File paths
   - UUIDs, PIDs
   - Complex combinations

2. **Signature Tests** (~15 test cases)
   - Error type extraction
   - Stack signature extraction
   - Hash generation
   - Edge cases

3. **Matching Tests** (~25 test cases)
   - String similarity
   - Fuzzy error type matching
   - Stack trace matching
   - Hybrid score calculation
   - Boost factors
   - Threshold filtering

4. **Validation Tests** (~15 test cases)
   - Required fields
   - Invalid severity
   - Tag limits
   - Context limits

### Integration Tests

**Scenarios**:

1. **End-to-End Flow**
   - Create remediation
   - Search with similar error
   - Verify match score
   - Verify match details

2. **Hybrid Matching**
   - Create multiple remediations
   - Search with variations
   - Verify semantic ranking
   - Verify string matching contribution

3. **Multi-Tenant Isolation**
   - Create in shared database
   - Search from different projects
   - Verify all projects see same remediations

4. **Error Cases**
   - Invalid input
   - Timeout scenarios
   - Vector store failures

### Performance Tests

**Load Testing**:
- 100 creates/minute sustained
- 500 searches/minute sustained
- Latency P95 < 250ms

**Stress Testing**:
- 1000 creates/minute burst
- 5000 searches/minute burst
- No crashes or data loss

---

## Security Considerations

### Data Privacy

**Remediations are Global Knowledge**:
- Stored in "shared" database
- Accessible to all projects
- No project-level isolation for remediation data

**User Responsibility**:
- Don't include sensitive data in error messages
- Sanitize stack traces (remove credentials, tokens, keys)
- Redact PII before saving

**Validation**:
- No automatic PII detection (user responsibility)
- No credential scanning (use pre-commit hooks)

### Authentication

**HTTP Transport**:
- Service uses HTTP transport on port 8080
- No authentication required (localhost-only access)
- Intended for local development and single-user environments

### Input Sanitization

**SQL Injection**: N/A (vector database, no SQL)
**XSS**: N/A (backend service, no HTML rendering)
**Command Injection**: N/A (no shell command execution)

**Size Limits**:
- Error message: 10KB
- Solution: 10KB
- Stack trace: 50KB
- Context values: 500 chars each

---

## Observability

**Traces**:
- Span per operation (create, search)
- Attributes: error_type, project_path, database
- Error recording on failures

**Metrics**:
- `remediation.create.total` - Counter (by error_type)
- `remediation.search.total` - Counter
- `remediation.search.duration` - Histogram (ms)
- `remediation.match.quality` - Histogram (score)

---

## Future Enhancements

### Planned Features

1. **Update/Delete Operations** (v2.1)
   - Implement vector store delete support
   - Update-in-place or delete+insert pattern

2. **Batch Operations** (v2.2)
   - Batch create (multiple remediations)
   - Batch search (multiple errors)

3. **Advanced Filters** (v2.3)
   - Filter by severity
   - Filter by date range
   - Filter by project (optional isolation)

4. **Learning from Feedback** (0.9.0-rc-1)
   - Track solution effectiveness
   - Boost popular solutions
   - Deprecate outdated solutions

5. **Pattern Detection** (v3.1)
   - Automatic pattern extraction
   - Common error categories
   - Trend analysis

---

## References

- **Package Implementation**: `pkg/remediation/`
- **MCP Tools**: `pkg/mcp/tools.go`
- **Testing**: `pkg/remediation/*_test.go`
- **Multi-Tenant Architecture**: `docs/adr/002-universal-multi-tenant-architecture.md`
