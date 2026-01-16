package vectorstore

import (
	"math"
	"testing"
)

// =============================================================================
// QUALITY METRICS - Retrieval Quality Tests
// =============================================================================
//
// These tests verify the retrieval quality metrics used to measure search
// effectiveness and detect regressions in semantic search quality.
//
// METRICS TESTED:
//   - NDCG (Normalized Discounted Cumulative Gain): Measures ranking quality
//     considering both relevance and position. Range [0.0, 1.0].
//   - MRR (Mean Reciprocal Rank): Measures position of first relevant result.
//     Range [0.0, 1.0]. Higher = relevant result appears earlier.
//   - Precision@K: Proportion of relevant documents in top K results.
//     Range [0.0, 1.0]. Higher = more relevant results retrieved.
//
// TEST STRATEGY:
//   - Known outcomes: Test against manually calculated expected values
//   - Edge cases: Empty results, invalid k, no relevant documents
//   - Boundary conditions: k > results length, single result, etc.
//
// See quality_metrics.go for metric definitions and algorithms.
// =============================================================================

// Helper to create search results for testing
func makeResults(ids ...string) []SearchResult {
	results := make([]SearchResult, len(ids))
	for i, id := range ids {
		results[i] = SearchResult{
			ID:    id,
			Score: float32(1.0 - float64(i)*0.1), // Decreasing scores
		}
	}
	return results
}

// =============================================================================
// CalculateNDCG Tests
// =============================================================================

func TestCalculateNDCG(t *testing.T) {
	t.Run("perfect ranking returns 1.0", func(t *testing.T) {
		// Results in ideal order
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		expectedRanking := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

		got := CalculateNDCG(results, expectedRanking, 5)
		if got != 1.0 {
			t.Errorf("CalculateNDCG() perfect ranking = %v, want 1.0", got)
		}
	})

	t.Run("completely reversed ranking", func(t *testing.T) {
		// Results in worst possible order (completely reversed)
		results := makeResults("doc5", "doc4", "doc3", "doc2", "doc1")
		expectedRanking := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

		got := CalculateNDCG(results, expectedRanking, 5)
		// Reversed ranking should have low NDCG (but not 0 due to DCG formula)
		// Note: NDCG for reversed ranking can be ~0.72 for 5 items
		if got >= 0.75 {
			t.Errorf("CalculateNDCG() reversed ranking = %v, want < 0.75", got)
		}
		if got < 0 || got > 1 {
			t.Errorf("CalculateNDCG() reversed ranking = %v, should be in [0, 1]", got)
		}
	})

	t.Run("partially correct ranking", func(t *testing.T) {
		// First result correct, rest out of order
		results := makeResults("doc1", "doc5", "doc3", "doc4", "doc2")
		expectedRanking := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

		got := CalculateNDCG(results, expectedRanking, 5)
		// Should be between 0 and 1, but not perfect
		if got <= 0 || got >= 1.0 {
			t.Errorf("CalculateNDCG() partial ranking = %v, want (0, 1)", got)
		}
	})

	t.Run("k smaller than results", func(t *testing.T) {
		// Only consider top 3 results
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		expectedRanking := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

		got := CalculateNDCG(results, expectedRanking, 3)
		// Top 3 are perfect, should be 1.0
		if got != 1.0 {
			t.Errorf("CalculateNDCG() top 3 perfect = %v, want 1.0", got)
		}
	})

	t.Run("k larger than results uses available results", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3")
		expectedRanking := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

		got := CalculateNDCG(results, expectedRanking, 10)
		// Should use all 3 results, not fail
		if got != 1.0 {
			t.Errorf("CalculateNDCG() k > results = %v, want 1.0", got)
		}
	})

	t.Run("irrelevant documents get zero contribution", func(t *testing.T) {
		// Results contain documents not in expected ranking
		results := makeResults("doc1", "irrelevant1", "doc2", "irrelevant2", "doc3")
		expectedRanking := []string{"doc1", "doc2", "doc3"}

		got := CalculateNDCG(results, expectedRanking, 5)
		// Should be less than 1.0 due to irrelevant docs in top positions
		if got >= 1.0 {
			t.Errorf("CalculateNDCG() with irrelevant docs = %v, want < 1.0", got)
		}
		if got < 0 {
			t.Errorf("CalculateNDCG() with irrelevant docs = %v, should be >= 0", got)
		}
	})

	// Edge cases
	t.Run("empty results returns 0", func(t *testing.T) {
		results := []SearchResult{}
		expectedRanking := []string{"doc1", "doc2"}

		got := CalculateNDCG(results, expectedRanking, 5)
		if got != 0.0 {
			t.Errorf("CalculateNDCG() empty results = %v, want 0.0", got)
		}
	})

	t.Run("empty expected ranking returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2")
		expectedRanking := []string{}

		got := CalculateNDCG(results, expectedRanking, 5)
		if got != 0.0 {
			t.Errorf("CalculateNDCG() empty expected = %v, want 0.0", got)
		}
	})

	t.Run("k equals 0 returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2")
		expectedRanking := []string{"doc1", "doc2"}

		got := CalculateNDCG(results, expectedRanking, 0)
		if got != 0.0 {
			t.Errorf("CalculateNDCG() k=0 = %v, want 0.0", got)
		}
	})

	t.Run("negative k returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2")
		expectedRanking := []string{"doc1", "doc2"}

		got := CalculateNDCG(results, expectedRanking, -1)
		if got != 0.0 {
			t.Errorf("CalculateNDCG() k=-1 = %v, want 0.0", got)
		}
	})

	t.Run("single result perfect match", func(t *testing.T) {
		results := makeResults("doc1")
		expectedRanking := []string{"doc1"}

		got := CalculateNDCG(results, expectedRanking, 1)
		if got != 1.0 {
			t.Errorf("CalculateNDCG() single perfect = %v, want 1.0", got)
		}
	})
}

// =============================================================================
// CalculateMRR Tests
// =============================================================================

func TestCalculateMRR(t *testing.T) {
	t.Run("first result relevant returns 1.0", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3")
		relevantDocs := []string{"doc1"}

		got := CalculateMRR(results, relevantDocs)
		if got != 1.0 {
			t.Errorf("CalculateMRR() first relevant = %v, want 1.0", got)
		}
	})

	t.Run("second result relevant returns 0.5", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3")
		relevantDocs := []string{"doc2"}

		got := CalculateMRR(results, relevantDocs)
		want := 0.5
		if math.Abs(got-want) > 0.0001 {
			t.Errorf("CalculateMRR() second relevant = %v, want %v", got, want)
		}
	})

	t.Run("third result relevant returns 0.333", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4")
		relevantDocs := []string{"doc3"}

		got := CalculateMRR(results, relevantDocs)
		want := 1.0 / 3.0
		if math.Abs(got-want) > 0.0001 {
			t.Errorf("CalculateMRR() third relevant = %v, want %v", got, want)
		}
	})

	t.Run("multiple relevant docs uses first position", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4")
		// doc2 and doc4 are relevant, but doc2 comes first
		relevantDocs := []string{"doc2", "doc4"}

		got := CalculateMRR(results, relevantDocs)
		want := 0.5 // Second position
		if math.Abs(got-want) > 0.0001 {
			t.Errorf("CalculateMRR() multiple relevant = %v, want %v", got, want)
		}
	})

	t.Run("no relevant results returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3")
		relevantDocs := []string{"doc4", "doc5"}

		got := CalculateMRR(results, relevantDocs)
		if got != 0.0 {
			t.Errorf("CalculateMRR() no relevant = %v, want 0.0", got)
		}
	})

	// Edge cases
	t.Run("empty results returns 0", func(t *testing.T) {
		results := []SearchResult{}
		relevantDocs := []string{"doc1"}

		got := CalculateMRR(results, relevantDocs)
		if got != 0.0 {
			t.Errorf("CalculateMRR() empty results = %v, want 0.0", got)
		}
	})

	t.Run("empty relevant docs returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2")
		relevantDocs := []string{}

		got := CalculateMRR(results, relevantDocs)
		if got != 0.0 {
			t.Errorf("CalculateMRR() empty relevant = %v, want 0.0", got)
		}
	})

	t.Run("both empty returns 0", func(t *testing.T) {
		results := []SearchResult{}
		relevantDocs := []string{}

		got := CalculateMRR(results, relevantDocs)
		if got != 0.0 {
			t.Errorf("CalculateMRR() both empty = %v, want 0.0", got)
		}
	})

	t.Run("last result relevant", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		relevantDocs := []string{"doc5"}

		got := CalculateMRR(results, relevantDocs)
		want := 0.2 // 1/5
		if math.Abs(got-want) > 0.0001 {
			t.Errorf("CalculateMRR() last relevant = %v, want %v", got, want)
		}
	})
}

// =============================================================================
// CalculatePrecisionAtK Tests
// =============================================================================

func TestCalculatePrecisionAtK(t *testing.T) {
	t.Run("all results relevant returns 1.0", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		relevantDocs := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

		got := CalculatePrecisionAtK(results, relevantDocs, 5)
		if got != 1.0 {
			t.Errorf("CalculatePrecisionAtK() all relevant = %v, want 1.0", got)
		}
	})

	t.Run("3 of 5 relevant returns 0.6", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		relevantDocs := []string{"doc1", "doc3", "doc5"}

		got := CalculatePrecisionAtK(results, relevantDocs, 5)
		want := 0.6 // 3/5
		if math.Abs(got-want) > 0.0001 {
			t.Errorf("CalculatePrecisionAtK() 3/5 relevant = %v, want %v", got, want)
		}
	})

	t.Run("none relevant returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		relevantDocs := []string{"doc6", "doc7"}

		got := CalculatePrecisionAtK(results, relevantDocs, 5)
		if got != 0.0 {
			t.Errorf("CalculatePrecisionAtK() none relevant = %v, want 0.0", got)
		}
	})

	t.Run("k smaller than results considers only top k", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		// doc1 and doc2 are relevant, doc5 is relevant but not in top 3
		relevantDocs := []string{"doc1", "doc2", "doc5"}

		got := CalculatePrecisionAtK(results, relevantDocs, 3)
		want := 2.0 / 3.0 // Only doc1 and doc2 count in top 3
		if math.Abs(got-want) > 0.0001 {
			t.Errorf("CalculatePrecisionAtK() k=3 = %v, want %v", got, want)
		}
	})

	t.Run("k larger than results uses available results", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3")
		relevantDocs := []string{"doc1", "doc2"}

		got := CalculatePrecisionAtK(results, relevantDocs, 10)
		want := 2.0 / 3.0 // 2 relevant out of 3 available
		if math.Abs(got-want) > 0.0001 {
			t.Errorf("CalculatePrecisionAtK() k > results = %v, want %v", got, want)
		}
	})

	t.Run("precision at 1", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3")
		relevantDocs := []string{"doc1"}

		got := CalculatePrecisionAtK(results, relevantDocs, 1)
		if got != 1.0 {
			t.Errorf("CalculatePrecisionAtK() P@1 = %v, want 1.0", got)
		}
	})

	t.Run("precision at 1 with irrelevant first", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3")
		relevantDocs := []string{"doc2", "doc3"}

		got := CalculatePrecisionAtK(results, relevantDocs, 1)
		if got != 0.0 {
			t.Errorf("CalculatePrecisionAtK() P@1 irrelevant = %v, want 0.0", got)
		}
	})

	// Edge cases
	t.Run("empty results returns 0", func(t *testing.T) {
		results := []SearchResult{}
		relevantDocs := []string{"doc1"}

		got := CalculatePrecisionAtK(results, relevantDocs, 5)
		if got != 0.0 {
			t.Errorf("CalculatePrecisionAtK() empty results = %v, want 0.0", got)
		}
	})

	t.Run("empty relevant docs returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2")
		relevantDocs := []string{}

		got := CalculatePrecisionAtK(results, relevantDocs, 5)
		if got != 0.0 {
			t.Errorf("CalculatePrecisionAtK() empty relevant = %v, want 0.0", got)
		}
	})

	t.Run("k equals 0 returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2")
		relevantDocs := []string{"doc1"}

		got := CalculatePrecisionAtK(results, relevantDocs, 0)
		if got != 0.0 {
			t.Errorf("CalculatePrecisionAtK() k=0 = %v, want 0.0", got)
		}
	})

	t.Run("negative k returns 0", func(t *testing.T) {
		results := makeResults("doc1", "doc2")
		relevantDocs := []string{"doc1"}

		got := CalculatePrecisionAtK(results, relevantDocs, -1)
		if got != 0.0 {
			t.Errorf("CalculatePrecisionAtK() k=-1 = %v, want 0.0", got)
		}
	})
}

// =============================================================================
// CalculateAllMetrics Tests
// =============================================================================

func TestCalculateAllMetrics(t *testing.T) {
	t.Run("calculates all metrics correctly", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		expectedRanking := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}
		relevantDocs := []string{"doc1", "doc2", "doc3"}

		got := CalculateAllMetrics(results, expectedRanking, relevantDocs, 5)

		// NDCG should be 1.0 (perfect ranking)
		if got.NDCG != 1.0 {
			t.Errorf("CalculateAllMetrics() NDCG = %v, want 1.0", got.NDCG)
		}

		// MRR should be 1.0 (first result is relevant)
		if got.MRR != 1.0 {
			t.Errorf("CalculateAllMetrics() MRR = %v, want 1.0", got.MRR)
		}

		// Precision@5 should be 0.6 (3 out of 5)
		want := 0.6
		if math.Abs(got.PrecisionAtK-want) > 0.0001 {
			t.Errorf("CalculateAllMetrics() PrecisionAtK = %v, want %v", got.PrecisionAtK, want)
		}

		// K should match input
		if got.K != 5 {
			t.Errorf("CalculateAllMetrics() K = %v, want 5", got.K)
		}
	})

	t.Run("handles poor ranking", func(t *testing.T) {
		// Worst case: relevant docs at the end
		results := makeResults("doc5", "doc4", "doc3", "doc2", "doc1")
		expectedRanking := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}
		relevantDocs := []string{"doc1", "doc2"}

		got := CalculateAllMetrics(results, expectedRanking, relevantDocs, 5)

		// NDCG should be low (poor ranking)
		// Note: NDCG for reversed ranking can be ~0.72 for 5 items
		if got.NDCG >= 0.75 {
			t.Errorf("CalculateAllMetrics() NDCG = %v, should be < 0.75 for poor ranking", got.NDCG)
		}

		// MRR should be low (first relevant at position 4)
		if got.MRR > 0.3 {
			t.Errorf("CalculateAllMetrics() MRR = %v, should be low for late relevant doc", got.MRR)
		}

		// Precision should be 0.4 (2 out of 5)
		want := 0.4
		if math.Abs(got.PrecisionAtK-want) > 0.0001 {
			t.Errorf("CalculateAllMetrics() PrecisionAtK = %v, want %v", got.PrecisionAtK, want)
		}
	})

	t.Run("handles empty results", func(t *testing.T) {
		results := []SearchResult{}
		expectedRanking := []string{"doc1", "doc2"}
		relevantDocs := []string{"doc1"}

		got := CalculateAllMetrics(results, expectedRanking, relevantDocs, 5)

		if got.NDCG != 0.0 {
			t.Errorf("CalculateAllMetrics() empty NDCG = %v, want 0.0", got.NDCG)
		}
		if got.MRR != 0.0 {
			t.Errorf("CalculateAllMetrics() empty MRR = %v, want 0.0", got.MRR)
		}
		if got.PrecisionAtK != 0.0 {
			t.Errorf("CalculateAllMetrics() empty PrecisionAtK = %v, want 0.0", got.PrecisionAtK)
		}
	})

	t.Run("k is preserved in result", func(t *testing.T) {
		results := makeResults("doc1", "doc2", "doc3", "doc4", "doc5")
		expectedRanking := []string{"doc1", "doc2"}
		relevantDocs := []string{"doc1"}

		got := CalculateAllMetrics(results, expectedRanking, relevantDocs, 3)

		if got.K != 3 {
			t.Errorf("CalculateAllMetrics() K = %v, want 3", got.K)
		}
	})
}

// =============================================================================
// QualityMetrics Struct Tests
// =============================================================================

func TestQualityMetrics_Struct(t *testing.T) {
	t.Run("struct fields are accessible", func(t *testing.T) {
		metrics := QualityMetrics{
			NDCG:         0.95,
			MRR:          0.85,
			PrecisionAtK: 0.75,
			K:            10,
		}

		if metrics.NDCG != 0.95 {
			t.Errorf("QualityMetrics.NDCG = %v, want 0.95", metrics.NDCG)
		}
		if metrics.MRR != 0.85 {
			t.Errorf("QualityMetrics.MRR = %v, want 0.85", metrics.MRR)
		}
		if metrics.PrecisionAtK != 0.75 {
			t.Errorf("QualityMetrics.PrecisionAtK = %v, want 0.75", metrics.PrecisionAtK)
		}
		if metrics.K != 10 {
			t.Errorf("QualityMetrics.K = %v, want 10", metrics.K)
		}
	})

	t.Run("zero value initialization", func(t *testing.T) {
		var metrics QualityMetrics

		if metrics.NDCG != 0.0 {
			t.Errorf("QualityMetrics zero NDCG = %v, want 0.0", metrics.NDCG)
		}
		if metrics.MRR != 0.0 {
			t.Errorf("QualityMetrics zero MRR = %v, want 0.0", metrics.MRR)
		}
		if metrics.PrecisionAtK != 0.0 {
			t.Errorf("QualityMetrics zero PrecisionAtK = %v, want 0.0", metrics.PrecisionAtK)
		}
		if metrics.K != 0 {
			t.Errorf("QualityMetrics zero K = %v, want 0", metrics.K)
		}
	})
}
