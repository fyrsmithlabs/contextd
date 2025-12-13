# Known Testing Gaps

**Status**: Active
**Last Updated**: 2025-12-11

---

## Critical: Mock Store Does Not Test Semantic Similarity

### The Problem

The mock vector store returns all documents regardless of query content:

```go
// developer.go:139-143
results = append(results, vectorstore.SearchResult{
    ID:       doc.ID,
    Content:  doc.Content,
    Metadata: doc.Metadata,
    Score:    0.9, // Always 0.9, ignores query
})
```

### Impact

| Test | What It Claims to Test | What It Actually Tests |
|------|------------------------|------------------------|
| C.1 Same Bug Retrieval | Exact match finds fix | Document exists |
| C.2 Similar Bug Adaptation | Similar query finds related fix | Document exists |
| C.3 False Positive Prevention | Unrelated query returns nothing | Nothing (passes trivially) |

### Evidence

Test C.3 logs this warning:
```
Note: Got 1 results (mock store behavior). In production, semantic similarity would filter these.
```

### Risk Level: HIGH

Production could:
- Return irrelevant memories for queries
- Miss relevant memories due to poor similarity scoring
- Surface wrong bug fixes for unrelated problems

### Mitigation (Future)

1. Add integration tests with real chromem store and real embeddings
2. Create negative test cases that MUST return empty results
3. Test with queries at varying semantic distances

---

## Medium: Confidence Scores Are Synthetic

### The Problem

Confidence scores come from hardcoded constants, not from actual similarity:

```go
// reasoningbank/service.go
const ExplicitRecordConfidence = 0.8
const DistilledConfidence = 0.6
```

### Impact

- Confidence thresholds (>= 0.7) filter based on source, not quality
- Cannot validate that high-confidence memories are actually relevant
- Feedback affects confidence, but initial values are arbitrary

### Risk Level: MEDIUM

Production could:
- Surface low-quality memories with high confidence
- Filter out valuable memories with low initial confidence

### Mitigation (Future)

1. Test confidence calibration: memories with confidence >0.8 should have >75% helpful rating
2. Add signal tracking tests to validate Bayesian weight updates
3. Test confidence decay curves after negative feedback

---

## Low: No Load Testing

### The Problem

Tests run with single developers and small datasets.

### Impact

- Unknown performance at scale (1000+ memories)
- Unknown behavior with concurrent developers
- Unknown vector store query latency under load

### Risk Level: LOW

Production impact unclear until real usage patterns emerge.

### Mitigation (Future)

1. Add benchmark tests with large memory collections
2. Test concurrent developer scenarios
3. Measure query latency percentiles

---

## Low: Temporal Workflows Not Integration Tested

### The Problem

Temporal workflows are tested with mocked activities, not real services.

### Impact

- Workflow orchestration logic validated
- Actual service integration not validated in workflow context

### Risk Level: LOW

Standard Go tests cover service integration directly.

### Mitigation (Future)

1. Add integration tests that run workflows against real services
2. Test failure recovery scenarios

---

## Coverage Summary

| Area | Test Confidence | Gap Severity |
|------|-----------------|--------------|
| Secret scrubbing | 95% | None |
| Checkpoint persistence | 90% | None |
| Cross-developer sharing | 85% | None |
| API contracts | 90% | None |
| **Semantic relevance** | **60%** | **HIGH** |
| Confidence calibration | 70% | MEDIUM |
| Load/performance | 0% | LOW |
| Workflow integration | 80% | LOW |

---

## Recommended Priority

1. **HIGH**: Add chromem integration tests for semantic search
2. **MEDIUM**: Add confidence calibration tests
3. **LOW**: Add load tests when scaling becomes relevant
4. **LOW**: Add Temporal integration tests when using workflows in production

---

## Related Documents

- [SPEC.md](SPEC.md) - Test specification
- [ARCH.md](ARCH.md) - Architecture
