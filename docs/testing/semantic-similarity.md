# Semantic Similarity Testing

## Overview

contextd's semantic similarity testing infrastructure validates that vector search returns semantically relevant results with proper ranking. Unlike mock tests that return hardcoded scores (e.g., 0.9), these tests use **real embedding models** to verify actual search quality.

**Key Features:**
- Test fixtures with known query-document pairs and expected rankings
- Quality metrics (NDCG, MRR, Precision@K) to measure retrieval effectiveness
- Regression detection that fails CI if search quality degrades
- Baseline metrics for tracking performance over time

**Why This Matters:**
- Catches regressions when embedding models change
- Validates that "similar" documents actually score higher than "dissimilar" ones
- Ensures semantic search behaves as users expect

---

## Test Fixtures

### What Are Test Fixtures?

Test fixtures are curated query-document pairs with **known expected behavior**. Each fixture includes:

1. **Query**: The search query string
2. **Documents**: 2+ candidate documents with unique IDs
3. **Expected Ranking**: Ideal document order by relevance (most relevant first)
4. **Expected Score Ranges**: Min/max similarity scores for each document
5. **Description**: What semantic behavior this fixture validates

### Available Fixtures

| Fixture | Description | Tests For |
|---------|-------------|-----------|
| `HighSimilarityPair` | Very similar documents (Go vs Golang) | Synonym recognition, high scores (>0.7) |
| `LowSimilarityPair` | Dissimilar documents (Go programming vs Italian cooking) | Topic separation, low scores (<0.3) |
| `SynonymHandling` | Query synonyms (tutorial/guide) | Semantic equivalence |
| `MultiTopicDocuments` | Partial matches (ML + Python query vs ML-only or Python-only docs) | Multi-topic ranking |
| `GradualRelevanceDecay` | 5 docs with decreasing relevance | Score distribution |

### Fixture Structure

```go
type SemanticTestCase struct {
    Name                 string                  // Unique identifier
    Query                string                  // Search query
    Documents            []TestDocument          // Candidate documents
    ExpectedRanking      []string                // Ideal document order
    ExpectedScoreRanges  map[string]ScoreRange   // Min/max scores per doc
    Description          string                  // What this tests
}
```

### Example: HighSimilarityPair

```go
Query: "Go programming language tutorial"

Documents:
  doc1: "Go programming language tutorial for beginners"
  doc2: "Golang programming guide and best practices"
  doc3: "Python machine learning tutorial with examples"

ExpectedRanking: ["doc1", "doc2", "doc3"]

ExpectedScoreRanges:
  doc1: {Min: 0.7, Max: 1.0}  // Exact match
  doc2: {Min: 0.7, Max: 1.0}  // Go = Golang (synonym)
  doc3: {Min: 0.0, Max: 0.6}  // Different language
```

**Validation:** Tests assert that doc1/doc2 both score >0.7 (high similarity) while doc3 scores <0.6 (low similarity).

### Location

- Fixtures: `internal/vectorstore/testdata/fixtures.go`
- Fixture tests: `internal/vectorstore/testdata/fixtures_test.go`

---

## Quality Metrics

contextd uses three standard information retrieval metrics to measure search effectiveness:

### NDCG (Normalized Discounted Cumulative Gain)

**What It Measures:** Ranking quality considering both relevance and position.

**Range:** 0.0 (worst) to 1.0 (perfect)

**Algorithm:**
1. DCG = sum of (relevance_score / log2(position + 1))
2. IDCG = DCG with perfect ranking
3. NDCG = DCG / IDCG

**Why It Matters:** Penalizes relevant documents appearing low in results. A document at position 10 contributes less than the same document at position 1.

**Example:**
```
Perfect ranking:       NDCG = 1.0
Reversed ranking:      NDCG < 0.7
Random ranking:        NDCG ≈ 0.5
```

### MRR (Mean Reciprocal Rank)

**What It Measures:** Where the first relevant document appears.

**Range:** 0.0 (no relevant docs) to 1.0 (first result relevant)

**Formula:** MRR = 1 / position_of_first_relevant_doc

**Why It Matters:** Optimized for tasks where users need one good result (e.g., "find the authenticate() function").

**Example:**
```
First result relevant:     MRR = 1.0
Third result relevant:     MRR = 0.333
No relevant results:       MRR = 0.0
```

### Precision@K

**What It Measures:** Proportion of relevant documents in top K results.

**Range:** 0.0 (none relevant) to 1.0 (all relevant)

**Formula:** Precision@K = (relevant_docs_in_top_K) / K

**Why It Matters:** Measures accuracy of top results. High precision means users see fewer irrelevant results.

**Example:**
```
3 relevant in top 5:   P@5 = 0.6
5 relevant in top 5:   P@5 = 1.0
0 relevant in top 5:   P@5 = 0.0
```

### Implementation

```go
// Calculate all metrics at once
metrics := vectorstore.CalculateAllMetrics(
    results,          // Search results from store
    expectedRanking,  // Ideal document order
    relevantDocs,     // IDs of relevant documents
    k,                // Top K cutoff
)

fmt.Printf("NDCG: %.3f\n", metrics.NDCG)
fmt.Printf("MRR:  %.3f\n", metrics.MRR)
fmt.Printf("P@%d:  %.3f\n", k, metrics.PrecisionAtK)
```

**Location:** `internal/vectorstore/quality_metrics.go`

---

## Running Tests

### Quick Test (Fake Embeddings)

Fast tests with deterministic fake embedder (no ONNX runtime required):

```bash
# Unit tests: fixtures and metrics
go test ./internal/vectorstore/testdata
go test ./internal/vectorstore -run QualityMetrics

# Integration tests: fake embeddings
go test ./test/integration/framework -run SemanticSimilarity
```

**Use Case:** Fast CI/CD checks, local development

### Real Embedding Tests

Tests with actual FastEmbed models (requires ONNX runtime):

```bash
# Real embedding tests (unit-level)
go test -v ./internal/vectorstore -run SemanticReal

# Real embedding tests (integration-level)
USE_REAL_EMBEDDINGS=1 go test -v ./test/integration/framework -run SemanticSimilarity
```

**Requirements:**
- ONNX runtime: `/usr/lib/libonnxruntime.so` or `ONNX_PATH` env var
- First run downloads embedding models (~100MB)
- Skipped automatically in short mode: `go test -short`

**Use Case:** Pre-release validation, embedding model upgrades

### Test Suites

| Test Suite | Command | Embeddings | Purpose |
|------------|---------|------------|---------|
| Fixture validation | `go test ./internal/vectorstore/testdata` | None | Validate fixture structure |
| Quality metrics | `go test ./internal/vectorstore -run QualityMetrics` | None | Test metric calculations |
| Real semantic tests | `go test ./internal/vectorstore -run SemanticReal` | Real | Validate search quality |
| Regression detection | `go test ./internal/vectorstore -run RegressionDetection` | Real | Block quality degradation |
| Integration tests | `go test ./test/integration/framework -run SemanticSimilarity` | Fake (default) or Real (opt-in) | End-to-end search |

---

## Baseline Metrics

### What Are Baselines?

Baseline metrics represent **expected performance** of the embedding model on known test fixtures. They serve as a quality threshold: if current metrics fall below baseline, tests fail.

**Location:** `internal/vectorstore/testdata/baseline_metrics.json`

### Baseline Structure

```json
{
  "version": "1.0.0",
  "model": "BAAI/bge-small-en-v1.5",
  "tolerance": 0.05,
  "test_cases": [
    {
      "name": "high_similarity_pair",
      "k": 3,
      "metrics": {
        "ndcg": 0.95,
        "mrr": 1.0,
        "precision_at_k": 0.667
      }
    }
  ],
  "aggregate_targets": {
    "min_avg_ndcg": 0.90,
    "min_avg_mrr": 0.95,
    "min_avg_precision": 0.65
  },
  "regression_thresholds": {
    "ndcg_threshold_multiplier": 0.95,
    "note": "Metrics must be >= 95% of baseline (5% degradation allowed)"
  }
}
```

### Current Baselines

| Fixture | NDCG | MRR | Precision@K | K |
|---------|------|-----|-------------|---|
| `high_similarity_pair` | 0.95 | 1.0 | 0.667 | 3 |
| `low_similarity_pair` | 0.95 | 1.0 | 0.667 | 3 |
| `synonym_handling` | 0.92 | 1.0 | 0.667 | 3 |
| `multi_topic_documents` | 0.88 | 1.0 | 0.667 | 3 |
| `gradual_relevance_decay` | 0.93 | 1.0 | 0.60 | 5 |

**Aggregate Targets:**
- Average NDCG ≥ 0.90
- Average MRR ≥ 0.95
- Average Precision ≥ 0.65

---

## Regression Detection

### How It Works

The `TestChromemStore_SemanticReal_RegressionDetection` test:

1. Loads `baseline_metrics.json`
2. Runs all 5 fixtures with real FastEmbed embeddings
3. Calculates NDCG, MRR, and Precision@K for each fixture
4. **Fails** if any metric falls below `baseline * 0.95` (95% threshold)

### Threshold Calculation

```go
threshold := baseline * thresholdMultiplier

// Example:
// baseline NDCG = 0.90
// threshold multiplier = 0.95
// threshold = 0.90 * 0.95 = 0.855
// Test FAILS if actual NDCG < 0.855
```

**Tolerance:** 5% degradation allowed to handle minor fluctuations.

### Why This Matters

Without regression detection:
- Embedding model upgrades could silently break search quality
- Code changes might degrade retrieval without notice
- Users experience worse search results

With regression detection:
- CI fails immediately if quality degrades
- Forces investigation before merge
- Baselines can be updated intentionally (see below)

### Test Output (Failure)

```
--- FAIL: TestChromemStore_SemanticReal_RegressionDetection
    chromem_semantic_real_test.go:XXX:
        REGRESSION DETECTED: high_similarity_pair NDCG below threshold
        Actual:   0.830
        Expected: >= 0.9025 (0.95 * 0.95)

        This indicates semantic search quality has degraded.
        Investigate embedding model or code changes.
```

---

## Updating Baselines

### When to Update

Update baselines when:
- ✅ **Embedding model changes** (e.g., upgrade to new model version)
- ✅ **Intentional algorithm improvements** (e.g., better ranking logic)
- ❌ **NOT when tests fail due to regression** (fix the code instead!)

### How to Update

1. **Run tests to get current metrics:**

```bash
go test -v ./internal/vectorstore -run SemanticReal 2>&1 | grep "Metrics:"
```

Example output:
```
Metrics: NDCG=0.934, MRR=1.000, Precision@3=0.667
```

2. **Update `baseline_metrics.json`:**

```json
{
  "test_cases": [
    {
      "name": "high_similarity_pair",
      "metrics": {
        "ndcg": 0.93,   // <- Update with new value
        "mrr": 1.0,
        "precision_at_k": 0.667
      }
    }
  ]
}
```

3. **Update metadata:**

```json
{
  "version": "1.1.0",              // Increment version
  "model": "BAAI/bge-base-en-v1.5", // Update model name
  "created_at": "2026-01-20",       // Update date
  "notes": "Updated for bge-base model upgrade"
}
```

4. **Commit with clear message:**

```bash
git add internal/vectorstore/testdata/baseline_metrics.json
git commit -m "Update semantic similarity baselines for bge-base-en-v1.5

- Upgraded embedding model from bge-small to bge-base
- NDCG improved from 0.90 to 0.93 (avg)
- All fixtures re-validated with new model"
```

5. **Document in PR:**

Include before/after metrics comparison and rationale for changes.

---

## Test Coverage

| Component | Test Cases | Coverage |
|-----------|------------|----------|
| Fixtures | 5 semantic test cases | 100% |
| Fixture validation | 11 edge cases + individual tests | 100% |
| Quality metrics | NDCG (12), MRR (10), P@K (12) | 100% |
| Real semantic tests | 7 tests (5 individual + 1 aggregate + 1 regression) | All fixtures |
| Integration tests | 4 tests with optional real embeddings | Both modes |

---

## CI/CD Integration

### Recommended CI Pipeline

```yaml
# Fast tests (always run)
- name: Unit Tests
  run: |
    go test ./internal/vectorstore/testdata
    go test ./internal/vectorstore -run QualityMetrics
    go test ./test/integration/framework -run SemanticSimilarity

# Real embedding tests (pre-merge, nightly)
- name: Semantic Quality Tests
  run: |
    go test -v ./internal/vectorstore -run SemanticReal
  # Only on: main branch, PRs to main, nightly schedule
```

### Makefile Target

```bash
# Fast tests
make test

# Real embedding tests
make test-semantic-real
```

---

## Troubleshooting

### Tests Skip: "ONNX runtime not available"

**Cause:** FastEmbed requires ONNX runtime for real embedding tests.

**Fix:**
```bash
# Install ONNX runtime
sudo apt-get install libonnxruntime

# Or set ONNX_PATH
export ONNX_PATH=/path/to/libonnxruntime.so
```

### Tests Fail: "Similarity score out of range"

**Cause:** Actual similarity scores don't match fixture expectations.

**Fix:**
1. Check if embedding model changed (requires baseline update)
2. Verify fixture expectations are realistic
3. Investigate code changes that might affect search

### Tests Fail: "REGRESSION DETECTED"

**Cause:** Quality metrics fell below baseline thresholds.

**Fix:**
1. **DO NOT update baselines to make tests pass** (defeats the purpose)
2. Investigate what changed: embedding model, ranking logic, preprocessing
3. Fix the regression or document why degradation is acceptable
4. Only then update baselines with clear justification

---

## See Also

- **Fixture API:** `internal/vectorstore/testdata/fixtures.go`
- **Quality Metrics:** `internal/vectorstore/quality_metrics.go`
- **Real Tests:** `internal/vectorstore/chromem_semantic_real_test.go`
- **Integration Tests:** `test/integration/framework/semantic_similarity_test.go`
- **Embeddings Guide:** `test/integration/framework/README_EMBEDDINGS.md`
- **Vector Storage Testing:** `docs/spec/vector-storage/testing.md`
