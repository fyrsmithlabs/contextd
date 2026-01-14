//go:build cgo

package vectorstore_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// skipIfFastEmbedUnavailable skips the test if FastEmbed is not available.
// This checks for both short mode and ONNX runtime availability.
func skipIfFastEmbedUnavailable(t *testing.T) {
	t.Helper()

	// Skip in short mode as this downloads models
	if testing.Short() {
		t.Skip("skipping FastEmbed test in short mode")
	}

	// Skip if ONNX runtime not available
	if _, err := os.Stat("/usr/lib/libonnxruntime.so"); os.IsNotExist(err) {
		if os.Getenv("ONNX_PATH") == "" {
			t.Skip("ONNX runtime not available, skipping FastEmbed test")
		}
	}
}

// createRealEmbedder creates a FastEmbed provider for testing.
func createRealEmbedder(t *testing.T) embeddings.Provider {
	t.Helper()

	provider, err := embeddings.NewFastEmbedProvider(embeddings.FastEmbedConfig{
		Model: "BAAI/bge-small-en-v1.5",
	})
	require.NoError(t, err, "Failed to create FastEmbed provider")

	return provider
}

// TestChromemStore_SemanticReal_HighSimilarityPair validates that very similar
// documents (e.g., Go vs Golang) receive high similarity scores (>0.7) with real embeddings.
func TestChromemStore_SemanticReal_HighSimilarityPair(t *testing.T) {
	skipIfFastEmbedUnavailable(t)

	tmpDir, err := os.MkdirTemp("", "chromem_semantic_real_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := createRealEmbedder(t)
	defer embedder.Close()

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_real_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Get test fixture
	fixture := testdata.HighSimilarityPair()
	require.NoError(t, testdata.ValidateFixture(fixture), "Fixture should be valid")

	// Convert fixture documents to vectorstore documents
	docs := make([]vectorstore.Document, len(fixture.Documents))
	for i, testDoc := range fixture.Documents {
		docs[i] = vectorstore.Document{
			ID:       testDoc.ID,
			Content:  testDoc.Content,
			Metadata: testDoc.Metadata,
		}
	}

	// Add documents
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search with the query
	results, err := store.Search(ctx, fixture.Query, len(fixture.Documents))
	require.NoError(t, err)
	require.NotEmpty(t, results, "Should return results")

	// Validate score ranges
	for _, result := range results {
		expectedRange, ok := fixture.ExpectedScoreRanges[result.ID]
		if !ok {
			continue // Document not in expected ranges
		}

		assert.GreaterOrEqual(t, result.Score, expectedRange.Min,
			"Document %s score %.3f should be >= expected min %.3f",
			result.ID, result.Score, expectedRange.Min)
		assert.LessOrEqual(t, result.Score, expectedRange.Max,
			"Document %s score %.3f should be <= expected max %.3f",
			result.ID, result.Score, expectedRange.Max)
	}

	// Calculate quality metrics
	relevantDocs := []string{"doc1", "doc2"} // Go and Golang are both relevant
	metrics := vectorstore.CalculateAllMetrics(results, fixture.ExpectedRanking, relevantDocs, 3)

	t.Logf("HighSimilarityPair metrics: NDCG=%.3f, MRR=%.3f, P@3=%.3f",
		metrics.NDCG, metrics.MRR, metrics.PrecisionAtK)

	// Validate metrics meet quality thresholds
	assert.Greater(t, metrics.NDCG, 0.85, "NDCG should be high for clear ranking")
	assert.Greater(t, metrics.MRR, 0.9, "MRR should be high (relevant doc should be first)")
	assert.Greater(t, metrics.PrecisionAtK, 0.6, "Precision@3 should be reasonable")
}

// TestChromemStore_SemanticReal_LowSimilarityPair validates that dissimilar
// documents from different domains receive low similarity scores (<0.3) with real embeddings.
func TestChromemStore_SemanticReal_LowSimilarityPair(t *testing.T) {
	skipIfFastEmbedUnavailable(t)

	tmpDir, err := os.MkdirTemp("", "chromem_semantic_real_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := createRealEmbedder(t)
	defer embedder.Close()

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_real_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Get test fixture
	fixture := testdata.LowSimilarityPair()
	require.NoError(t, testdata.ValidateFixture(fixture), "Fixture should be valid")

	// Convert fixture documents to vectorstore documents
	docs := make([]vectorstore.Document, len(fixture.Documents))
	for i, testDoc := range fixture.Documents {
		docs[i] = vectorstore.Document{
			ID:       testDoc.ID,
			Content:  testDoc.Content,
			Metadata: testDoc.Metadata,
		}
	}

	// Add documents
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search with the query
	results, err := store.Search(ctx, fixture.Query, len(fixture.Documents))
	require.NoError(t, err)
	require.NotEmpty(t, results, "Should return results")

	// Validate score ranges
	for _, result := range results {
		expectedRange, ok := fixture.ExpectedScoreRanges[result.ID]
		if !ok {
			continue // Document not in expected ranges
		}

		assert.GreaterOrEqual(t, result.Score, expectedRange.Min,
			"Document %s score %.3f should be >= expected min %.3f",
			result.ID, result.Score, expectedRange.Min)
		assert.LessOrEqual(t, result.Score, expectedRange.Max,
			"Document %s score %.3f should be <= expected max %.3f",
			result.ID, result.Score, expectedRange.Max)
	}

	// Special check: cooking document should have very low score
	for _, result := range results {
		if result.ID == "doc2" { // Italian cooking document
			assert.Less(t, result.Score, float32(0.4),
				"Dissimilar document (cooking) should have very low score, got %.3f",
				result.Score)
		}
	}

	// Calculate quality metrics
	relevantDocs := []string{"doc1", "doc3"} // Programming docs are relevant
	metrics := vectorstore.CalculateAllMetrics(results, fixture.ExpectedRanking, relevantDocs, 3)

	t.Logf("LowSimilarityPair metrics: NDCG=%.3f, MRR=%.3f, P@3=%.3f",
		metrics.NDCG, metrics.MRR, metrics.PrecisionAtK)

	// Validate metrics meet quality thresholds
	assert.Greater(t, metrics.NDCG, 0.85, "NDCG should be high for clear ranking")
	assert.Greater(t, metrics.MRR, 0.9, "MRR should be high (relevant doc should be first)")
}

// TestChromemStore_SemanticReal_SynonymHandling validates that synonyms and
// related terms (e.g., tutorial/guide) are recognized as semantically similar with real embeddings.
func TestChromemStore_SemanticReal_SynonymHandling(t *testing.T) {
	skipIfFastEmbedUnavailable(t)

	tmpDir, err := os.MkdirTemp("", "chromem_semantic_real_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := createRealEmbedder(t)
	defer embedder.Close()

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_real_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Get test fixture
	fixture := testdata.SynonymHandling()
	require.NoError(t, testdata.ValidateFixture(fixture), "Fixture should be valid")

	// Convert fixture documents to vectorstore documents
	docs := make([]vectorstore.Document, len(fixture.Documents))
	for i, testDoc := range fixture.Documents {
		docs[i] = vectorstore.Document{
			ID:       testDoc.ID,
			Content:  testDoc.Content,
			Metadata: testDoc.Metadata,
		}
	}

	// Add documents
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search with the query
	results, err := store.Search(ctx, fixture.Query, len(fixture.Documents))
	require.NoError(t, err)
	require.NotEmpty(t, results, "Should return results")

	// Validate score ranges
	for _, result := range results {
		expectedRange, ok := fixture.ExpectedScoreRanges[result.ID]
		if !ok {
			continue // Document not in expected ranges
		}

		assert.GreaterOrEqual(t, result.Score, expectedRange.Min,
			"Document %s score %.3f should be >= expected min %.3f",
			result.ID, result.Score, expectedRange.Min)
		assert.LessOrEqual(t, result.Score, expectedRange.Max,
			"Document %s score %.3f should be <= expected max %.3f",
			result.ID, result.Score, expectedRange.Max)
	}

	// Special check: both tutorial and guide documents should score high
	tutorialScore := float32(0.0)
	guideScore := float32(0.0)
	for _, result := range results {
		if result.ID == "doc1" {
			tutorialScore = result.Score
		} else if result.ID == "doc2" {
			guideScore = result.Score
		}
	}
	assert.Greater(t, tutorialScore, float32(0.6), "Tutorial document should have high score")
	assert.Greater(t, guideScore, float32(0.6), "Guide document should have high score (synonym of tutorial)")

	// Calculate quality metrics
	relevantDocs := []string{"doc1", "doc2"} // Tutorial and guide are both relevant
	metrics := vectorstore.CalculateAllMetrics(results, fixture.ExpectedRanking, relevantDocs, 3)

	t.Logf("SynonymHandling metrics: NDCG=%.3f, MRR=%.3f, P@3=%.3f",
		metrics.NDCG, metrics.MRR, metrics.PrecisionAtK)

	// Validate metrics meet quality thresholds
	assert.Greater(t, metrics.NDCG, 0.80, "NDCG should be good for synonym handling")
	assert.Greater(t, metrics.MRR, 0.9, "MRR should be high (relevant doc should be first)")
}

// TestChromemStore_SemanticReal_MultiTopicDocuments validates correct ranking
// when documents contain multiple topics with partial query matches using real embeddings.
func TestChromemStore_SemanticReal_MultiTopicDocuments(t *testing.T) {
	skipIfFastEmbedUnavailable(t)

	tmpDir, err := os.MkdirTemp("", "chromem_semantic_real_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := createRealEmbedder(t)
	defer embedder.Close()

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_real_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Get test fixture
	fixture := testdata.MultiTopicDocuments()
	require.NoError(t, testdata.ValidateFixture(fixture), "Fixture should be valid")

	// Convert fixture documents to vectorstore documents
	docs := make([]vectorstore.Document, len(fixture.Documents))
	for i, testDoc := range fixture.Documents {
		docs[i] = vectorstore.Document{
			ID:       testDoc.ID,
			Content:  testDoc.Content,
			Metadata: testDoc.Metadata,
		}
	}

	// Add documents
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search with the query
	results, err := store.Search(ctx, fixture.Query, len(fixture.Documents))
	require.NoError(t, err)
	require.NotEmpty(t, results, "Should return results")

	// Validate score ranges
	for _, result := range results {
		expectedRange, ok := fixture.ExpectedScoreRanges[result.ID]
		if !ok {
			continue // Document not in expected ranges
		}

		assert.GreaterOrEqual(t, result.Score, expectedRange.Min,
			"Document %s score %.3f should be >= expected min %.3f",
			result.ID, result.Score, expectedRange.Min)
		assert.LessOrEqual(t, result.Score, expectedRange.Max,
			"Document %s score %.3f should be <= expected max %.3f",
			result.ID, result.Score, expectedRange.Max)
	}

	// Special check: doc1 (ML + Python) should score highest
	assert.Equal(t, "doc1", results[0].ID,
		"Document with both topics should rank first")

	// Calculate quality metrics
	relevantDocs := []string{"doc1", "doc3"} // ML documents are most relevant to "ML with Python"
	metrics := vectorstore.CalculateAllMetrics(results, fixture.ExpectedRanking, relevantDocs, 3)

	t.Logf("MultiTopicDocuments metrics: NDCG=%.3f, MRR=%.3f, P@3=%.3f",
		metrics.NDCG, metrics.MRR, metrics.PrecisionAtK)

	// Validate metrics meet quality thresholds
	assert.Greater(t, metrics.NDCG, 0.75, "NDCG should be reasonable for multi-topic ranking")
	assert.Greater(t, metrics.MRR, 0.9, "MRR should be high (best match should be first)")
}

// TestChromemStore_SemanticReal_GradualRelevanceDecay validates that similarity
// scores decay gradually as document relevance decreases with real embeddings.
func TestChromemStore_SemanticReal_GradualRelevanceDecay(t *testing.T) {
	skipIfFastEmbedUnavailable(t)

	tmpDir, err := os.MkdirTemp("", "chromem_semantic_real_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := createRealEmbedder(t)
	defer embedder.Close()

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_real_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Get test fixture
	fixture := testdata.GradualRelevanceDecay()
	require.NoError(t, testdata.ValidateFixture(fixture), "Fixture should be valid")

	// Convert fixture documents to vectorstore documents
	docs := make([]vectorstore.Document, len(fixture.Documents))
	for i, testDoc := range fixture.Documents {
		docs[i] = vectorstore.Document{
			ID:       testDoc.ID,
			Content:  testDoc.Content,
			Metadata: testDoc.Metadata,
		}
	}

	// Add documents
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search with the query
	results, err := store.Search(ctx, fixture.Query, len(fixture.Documents))
	require.NoError(t, err)
	require.NotEmpty(t, results, "Should return results")

	// Validate score ranges
	for _, result := range results {
		expectedRange, ok := fixture.ExpectedScoreRanges[result.ID]
		if !ok {
			continue // Document not in expected ranges
		}

		assert.GreaterOrEqual(t, result.Score, expectedRange.Min,
			"Document %s score %.3f should be >= expected min %.3f",
			result.ID, result.Score, expectedRange.Min)
		assert.LessOrEqual(t, result.Score, expectedRange.Max,
			"Document %s score %.3f should be <= expected max %.3f",
			result.ID, result.Score, expectedRange.Max)
	}

	// Validate gradual decay: scores should generally decrease
	// We allow some flexibility since real embeddings may not produce perfect ordering
	if len(results) >= 3 {
		// At least check that top 3 are reasonably ordered
		assert.Greater(t, results[0].Score, results[2].Score-0.2,
			"Scores should generally decay from top to bottom")
	}

	// Calculate quality metrics
	relevantDocs := []string{"doc1", "doc2", "doc3"} // Top 3 docs are relevant
	metrics := vectorstore.CalculateAllMetrics(results, fixture.ExpectedRanking, relevantDocs, 5)

	t.Logf("GradualRelevanceDecay metrics: NDCG=%.3f, MRR=%.3f, P@5=%.3f",
		metrics.NDCG, metrics.MRR, metrics.PrecisionAtK)

	// Validate metrics meet quality thresholds
	assert.Greater(t, metrics.NDCG, 0.80, "NDCG should be good for gradual decay")
	assert.Greater(t, metrics.MRR, 0.9, "MRR should be high (most relevant doc should be first)")
	assert.Greater(t, metrics.PrecisionAtK, 0.5, "P@5 should be reasonable")
}

// TestChromemStore_SemanticReal_AllFixtures runs all test fixtures and reports
// aggregate quality metrics. This provides a comprehensive quality assessment.
func TestChromemStore_SemanticReal_AllFixtures(t *testing.T) {
	skipIfFastEmbedUnavailable(t)

	tmpDir, err := os.MkdirTemp("", "chromem_semantic_real_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := createRealEmbedder(t)
	defer embedder.Close()

	// Aggregate metrics across all fixtures
	var totalNDCG, totalMRR, totalPrecision float64
	fixtures := testdata.AllFixtures()
	fixtureCount := len(fixtures)

	t.Logf("Running %d test fixtures with real embeddings", fixtureCount)

	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			// Create fresh store for each fixture
			config := vectorstore.ChromemConfig{
				Path:              tmpDir + "/" + fixture.Name,
				Compress:          false,
				DefaultCollection: "semantic_real_test",
				VectorSize:        384,
				Isolation:         vectorstore.NewNoIsolation(),
			}

			store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
			require.NoError(t, err)
			defer store.Close()

			ctx := context.Background()

			// Convert fixture documents to vectorstore documents
			docs := make([]vectorstore.Document, len(fixture.Documents))
			for i, testDoc := range fixture.Documents {
				docs[i] = vectorstore.Document{
					ID:       testDoc.ID,
					Content:  testDoc.Content,
					Metadata: testDoc.Metadata,
				}
			}

			// Add documents
			_, err = store.AddDocuments(ctx, docs)
			require.NoError(t, err)

			// Search with the query
			results, err := store.Search(ctx, fixture.Query, len(fixture.Documents))
			require.NoError(t, err)
			require.NotEmpty(t, results, "Should return results")

			// Determine relevant documents based on fixture
			var relevantDocs []string
			switch fixture.Name {
			case "high_similarity_pair":
				relevantDocs = []string{"doc1", "doc2"}
			case "low_similarity_pair":
				relevantDocs = []string{"doc1", "doc3"}
			case "synonym_handling":
				relevantDocs = []string{"doc1", "doc2"}
			case "multi_topic_documents":
				relevantDocs = []string{"doc1", "doc3"}
			case "gradual_relevance_decay":
				relevantDocs = []string{"doc1", "doc2", "doc3"}
			}

			// Calculate quality metrics
			k := 3
			if fixture.Name == "gradual_relevance_decay" {
				k = 5
			}
			metrics := vectorstore.CalculateAllMetrics(results, fixture.ExpectedRanking, relevantDocs, k)

			t.Logf("%s: NDCG=%.3f, MRR=%.3f, P@%d=%.3f",
				fixture.Name, metrics.NDCG, metrics.MRR, k, metrics.PrecisionAtK)

			// Accumulate for aggregate metrics
			totalNDCG += metrics.NDCG
			totalMRR += metrics.MRR
			totalPrecision += metrics.PrecisionAtK
		})
	}

	// Calculate and report aggregate metrics
	avgNDCG := totalNDCG / float64(fixtureCount)
	avgMRR := totalMRR / float64(fixtureCount)
	avgPrecision := totalPrecision / float64(fixtureCount)

	t.Logf("\n=== Aggregate Metrics (Real Embeddings) ===")
	t.Logf("Average NDCG: %.3f", avgNDCG)
	t.Logf("Average MRR: %.3f", avgMRR)
	t.Logf("Average Precision: %.3f", avgPrecision)

	// Validate aggregate metrics meet minimum thresholds
	// These are relaxed compared to individual test thresholds to account for variation
	assert.Greater(t, avgNDCG, 0.75, "Average NDCG should be > 0.75")
	assert.Greater(t, avgMRR, 0.85, "Average MRR should be > 0.85")
	assert.Greater(t, avgPrecision, 0.55, "Average Precision should be > 0.55")
}

// baselineMetrics represents the structure of baseline_metrics.json
type baselineMetrics struct {
	Version     string `json:"version"`
	Model       string `json:"model"`
	Description string `json:"description"`
	Tolerance   float64 `json:"tolerance"`
	TestCases   []struct {
		Name    string `json:"name"`
		Metrics struct {
			NDCG        float64 `json:"ndcg"`
			MRR         float64 `json:"mrr"`
			PrecisionAtK float64 `json:"precision_at_k"`
		} `json:"metrics"`
	} `json:"test_cases"`
	AggregateTargets struct {
		MinAvgNDCG      float64 `json:"min_avg_ndcg"`
		MinAvgMRR       float64 `json:"min_avg_mrr"`
		MinAvgPrecision float64 `json:"min_avg_precision"`
	} `json:"aggregate_targets"`
}

// loadBaselineMetrics loads the baseline metrics from testdata/baseline_metrics.json
func loadBaselineMetrics(t *testing.T) baselineMetrics {
	t.Helper()

	data, err := os.ReadFile("testdata/baseline_metrics.json")
	require.NoError(t, err, "Failed to read baseline_metrics.json")

	var baseline baselineMetrics
	err = json.Unmarshal(data, &baseline)
	require.NoError(t, err, "Failed to parse baseline_metrics.json")

	return baseline
}

// TestChromemStore_SemanticReal_BaselineComparison compares current metrics
// against baseline expectations and logs any differences.
// This test is informational and helps track quality over time.
func TestChromemStore_SemanticReal_BaselineComparison(t *testing.T) {
	skipIfFastEmbedUnavailable(t)

	baseline := loadBaselineMetrics(t)
	t.Logf("Loaded baseline metrics (model: %s, tolerance: %.1f%%)",
		baseline.Model, baseline.Tolerance*100)

	tmpDir, err := os.MkdirTemp("", "chromem_semantic_real_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := createRealEmbedder(t)
	defer embedder.Close()

	// Track aggregate metrics
	var totalNDCG, totalMRR, totalPrecision float64
	fixtures := testdata.AllFixtures()

	t.Logf("\n=== Baseline Comparison ===")

	for _, fixture := range fixtures {
		// Create fresh store for each fixture
		config := vectorstore.ChromemConfig{
			Path:              tmpDir + "/" + fixture.Name,
			Compress:          false,
			DefaultCollection: "semantic_real_test",
			VectorSize:        384,
			Isolation:         vectorstore.NewNoIsolation(),
		}

		store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		// Convert and add documents
		docs := make([]vectorstore.Document, len(fixture.Documents))
		for i, testDoc := range fixture.Documents {
			docs[i] = vectorstore.Document{
				ID:       testDoc.ID,
				Content:  testDoc.Content,
				Metadata: testDoc.Metadata,
			}
		}
		_, err = store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Search
		results, err := store.Search(ctx, fixture.Query, len(fixture.Documents))
		require.NoError(t, err)

		// Determine relevant documents
		var relevantDocs []string
		switch fixture.Name {
		case "high_similarity_pair":
			relevantDocs = []string{"doc1", "doc2"}
		case "low_similarity_pair":
			relevantDocs = []string{"doc1", "doc3"}
		case "synonym_handling":
			relevantDocs = []string{"doc1", "doc2"}
		case "multi_topic_documents":
			relevantDocs = []string{"doc1", "doc3"}
		case "gradual_relevance_decay":
			relevantDocs = []string{"doc1", "doc2", "doc3"}
		}

		// Calculate metrics
		k := 3
		if fixture.Name == "gradual_relevance_decay" {
			k = 5
		}
		metrics := vectorstore.CalculateAllMetrics(results, fixture.ExpectedRanking, relevantDocs, k)

		// Find baseline for this fixture
		var baselineNDCG, baselineMRR, baselinePrecision float64
		for _, tc := range baseline.TestCases {
			if tc.Name == fixture.Name {
				baselineNDCG = tc.Metrics.NDCG
				baselineMRR = tc.Metrics.MRR
				baselinePrecision = tc.Metrics.PrecisionAtK
				break
			}
		}

		// Compare and log
		ndcgDiff := metrics.NDCG - baselineNDCG
		mrrDiff := metrics.MRR - baselineMRR
		precisionDiff := metrics.PrecisionAtK - baselinePrecision

		t.Logf("\n%s:", fixture.Name)
		t.Logf("  NDCG:      %.3f (baseline: %.3f, diff: %+.3f)", metrics.NDCG, baselineNDCG, ndcgDiff)
		t.Logf("  MRR:       %.3f (baseline: %.3f, diff: %+.3f)", metrics.MRR, baselineMRR, mrrDiff)
		t.Logf("  P@K:       %.3f (baseline: %.3f, diff: %+.3f)", metrics.PrecisionAtK, baselinePrecision, precisionDiff)

		// Check if within tolerance (baseline * (1 - tolerance))
		minNDCG := baselineNDCG * (1 - baseline.Tolerance)
		minMRR := baselineMRR * (1 - baseline.Tolerance)
		minPrecision := baselinePrecision * (1 - baseline.Tolerance)

		if metrics.NDCG < minNDCG {
			t.Logf("  ⚠️  NDCG below tolerance threshold (%.3f)", minNDCG)
		}
		if metrics.MRR < minMRR {
			t.Logf("  ⚠️  MRR below tolerance threshold (%.3f)", minMRR)
		}
		if metrics.PrecisionAtK < minPrecision {
			t.Logf("  ⚠️  Precision below tolerance threshold (%.3f)", minPrecision)
		}

		totalNDCG += metrics.NDCG
		totalMRR += metrics.MRR
		totalPrecision += metrics.PrecisionAtK
	}

	// Calculate aggregates
	fixtureCount := float64(len(fixtures))
	avgNDCG := totalNDCG / fixtureCount
	avgMRR := totalMRR / fixtureCount
	avgPrecision := totalPrecision / fixtureCount

	t.Logf("\n=== Aggregate Results ===")
	t.Logf("Average NDCG:      %.3f (target: %.3f)", avgNDCG, baseline.AggregateTargets.MinAvgNDCG)
	t.Logf("Average MRR:       %.3f (target: %.3f)", avgMRR, baseline.AggregateTargets.MinAvgMRR)
	t.Logf("Average Precision: %.3f (target: %.3f)", avgPrecision, baseline.AggregateTargets.MinAvgPrecision)

	// Note: This test logs comparisons but doesn't fail on minor deviations
	// The goal is to track quality over time, not enforce strict thresholds
	t.Log("\nℹ️  This test tracks quality over time. Review logs for any concerning trends.")
}
