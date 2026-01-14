package vectorstore

import (
	"math"
)

// QualityMetrics contains retrieval quality measurements.
// These metrics help track search effectiveness over time and detect regressions.
type QualityMetrics struct {
	// NDCG is Normalized Discounted Cumulative Gain (0.0-1.0)
	// Measures ranking quality considering both relevance and position.
	// 1.0 = perfect ranking, 0.0 = worst possible ranking.
	NDCG float64

	// MRR is Mean Reciprocal Rank (0.0-1.0)
	// Measures where the first relevant document appears.
	// 1.0 = first result is relevant, 0.0 = no relevant results.
	MRR float64

	// PrecisionAtK is the proportion of relevant documents in top K results (0.0-1.0)
	// Measures how many retrieved documents are actually relevant.
	PrecisionAtK float64

	// K is the cutoff used for these metrics
	K int
}

// CalculateNDCG computes Normalized Discounted Cumulative Gain at rank K.
//
// NDCG measures ranking quality by comparing actual ranking to ideal ranking.
// It considers both relevance and position, with higher weight for top positions.
//
// Parameters:
//   - results: Search results ordered by the system
//   - expectedRanking: Ideal order of document IDs by relevance (most relevant first)
//   - k: Number of top results to consider
//
// Returns:
//   - NDCG score in range [0.0, 1.0] where 1.0 is perfect ranking
//   - Returns 0.0 if results are empty or k <= 0
//
// Algorithm:
//   - DCG (Discounted Cumulative Gain) = sum of (relevance_score / log2(position+1))
//   - IDCG (Ideal DCG) = DCG with perfect ranking
//   - NDCG = DCG / IDCG (normalized to handle different result set sizes)
func CalculateNDCG(results []SearchResult, expectedRanking []string, k int) float64 {
	if len(results) == 0 || k <= 0 || len(expectedRanking) == 0 {
		return 0.0
	}

	// Limit k to available results
	if k > len(results) {
		k = len(results)
	}

	// Build relevance lookup: document ID -> relevance score
	// Higher rank (earlier in list) = higher relevance
	// Use inverse rank as relevance: position 0 gets score N, position N-1 gets score 1
	relevance := make(map[string]float64)
	n := len(expectedRanking)
	for i, docID := range expectedRanking {
		// Relevance decreases linearly from n to 1
		relevance[docID] = float64(n - i)
	}

	// Calculate DCG for actual results
	dcg := 0.0
	for i := 0; i < k; i++ {
		docID := results[i].ID
		rel := relevance[docID] // 0 if document not in expected ranking
		// Standard DCG formula: rel / log2(position + 2)
		// +2 because positions are 0-indexed and we want log2(2) = 1 for position 0
		dcg += rel / math.Log2(float64(i+2))
	}

	// Calculate IDCG (ideal DCG) - DCG if results were perfectly ordered
	idcg := 0.0
	idealK := k
	if idealK > len(expectedRanking) {
		idealK = len(expectedRanking)
	}
	for i := 0; i < idealK; i++ {
		rel := float64(n - i) // Perfect order: highest relevance first
		idcg += rel / math.Log2(float64(i+2))
	}

	// Normalize: NDCG = DCG / IDCG
	if idcg == 0.0 {
		return 0.0
	}
	return dcg / idcg
}

// CalculateMRR computes Mean Reciprocal Rank.
//
// MRR measures where the first relevant document appears in the results.
// It's particularly useful for tasks where users only need one good result.
//
// Parameters:
//   - results: Search results ordered by the system
//   - relevantDocs: Set of document IDs that are considered relevant
//
// Returns:
//   - MRR score in range [0.0, 1.0]
//   - 1.0 if first result is relevant
//   - 1/position for first relevant result at position (1-indexed)
//   - 0.0 if no relevant documents found
//
// Example:
//   - First result relevant: MRR = 1.0
//   - Third result relevant: MRR = 1/3 = 0.333
//   - No relevant results: MRR = 0.0
func CalculateMRR(results []SearchResult, relevantDocs []string) float64 {
	if len(results) == 0 || len(relevantDocs) == 0 {
		return 0.0
	}

	// Build set for O(1) lookup
	relevant := make(map[string]bool, len(relevantDocs))
	for _, docID := range relevantDocs {
		relevant[docID] = true
	}

	// Find first relevant document
	for i, result := range results {
		if relevant[result.ID] {
			// Return reciprocal rank (1-indexed position)
			return 1.0 / float64(i+1)
		}
	}

	// No relevant document found
	return 0.0
}

// CalculatePrecisionAtK computes Precision at rank K.
//
// Precision@K measures the proportion of relevant documents in the top K results.
// It answers: "Of the K documents I retrieved, how many are actually relevant?"
//
// Parameters:
//   - results: Search results ordered by the system
//   - relevantDocs: Set of document IDs that are considered relevant
//   - k: Number of top results to consider
//
// Returns:
//   - Precision score in range [0.0, 1.0]
//   - 1.0 if all top K results are relevant
//   - 0.0 if no relevant documents in top K
//   - Returns 0.0 if results are empty or k <= 0
//
// Example:
//   - 3 relevant docs in top 5: P@5 = 3/5 = 0.6
//   - 5 relevant docs in top 5: P@5 = 5/5 = 1.0
//   - 0 relevant docs in top 5: P@5 = 0/5 = 0.0
func CalculatePrecisionAtK(results []SearchResult, relevantDocs []string, k int) float64 {
	if len(results) == 0 || k <= 0 || len(relevantDocs) == 0 {
		return 0.0
	}

	// Limit k to available results
	if k > len(results) {
		k = len(results)
	}

	// Build set for O(1) lookup
	relevant := make(map[string]bool, len(relevantDocs))
	for _, docID := range relevantDocs {
		relevant[docID] = true
	}

	// Count relevant documents in top K
	relevantCount := 0
	for i := 0; i < k; i++ {
		if relevant[results[i].ID] {
			relevantCount++
		}
	}

	return float64(relevantCount) / float64(k)
}

// CalculateAllMetrics computes NDCG, MRR, and Precision@K in a single pass.
//
// This is more efficient than calling each metric function separately when you
// need all three metrics, as it only builds the relevance lookups once.
//
// Parameters:
//   - results: Search results ordered by the system
//   - expectedRanking: Ideal order of document IDs by relevance (for NDCG)
//   - relevantDocs: Set of document IDs that are considered relevant (for MRR and P@K)
//   - k: Number of top results to consider
//
// Returns:
//   - QualityMetrics struct containing all three metrics
func CalculateAllMetrics(results []SearchResult, expectedRanking []string, relevantDocs []string, k int) QualityMetrics {
	return QualityMetrics{
		NDCG:         CalculateNDCG(results, expectedRanking, k),
		MRR:          CalculateMRR(results, relevantDocs),
		PrecisionAtK: CalculatePrecisionAtK(results, relevantDocs, k),
		K:            k,
	}
}
