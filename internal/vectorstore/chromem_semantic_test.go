package vectorstore_test

import (
	"context"
	"math"
	"os"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// semanticTestEmbedder creates embeddings with predictable semantic similarity.
// This allows testing that the vector store actually performs semantic search correctly.
type semanticTestEmbedder struct {
	vectorSize int
}

func (e *semanticTestEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = e.makeSemanticEmbedding(text)
	}
	return embeddings, nil
}

func (e *semanticTestEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.makeSemanticEmbedding(text), nil
}

// makeSemanticEmbedding creates embeddings where semantic similarity can be controlled.
// Documents with overlapping keywords get similar embeddings.
func (e *semanticTestEmbedder) makeSemanticEmbedding(text string) []float32 {
	embedding := make([]float32, e.vectorSize)
	
	// Create semantic features based on keywords
	// Each keyword activates specific dimensions
	keywords := map[string][]int{
		"go":          {0, 1, 2, 10, 11, 12},
		"golang":      {0, 1, 2, 10, 11, 12}, // Similar to "go"
		"programming": {3, 4, 5, 13, 14, 15},
		"language":    {3, 4, 5, 13, 14, 15}, // Similar to "programming"
		"python":      {6, 7, 8, 16, 17, 18},
		"database":    {20, 21, 22, 30, 31, 32},
		"vector":      {20, 21, 22, 30, 31, 32}, // Similar to "database"
		"search":      {23, 24, 25, 33, 34, 35},
		"tutorial":    {40, 41, 42, 50, 51, 52},
		"guide":       {40, 41, 42, 50, 51, 52}, // Similar to "tutorial"
		"machine":     {60, 61, 62, 70, 71, 72},
		"learning":    {60, 61, 62, 70, 71, 72}, // Similar to "machine"
	}
	
	// Activate dimensions for each keyword found
	textLower := toLower(text)
	for keyword, dims := range keywords {
		if contains(textLower, keyword) {
			for _, dim := range dims {
				if dim < e.vectorSize {
					embedding[dim] = 1.0
				}
			}
		}
	}
	
	// Normalize to unit vector (required for cosine similarity)
	var sumSq float32
	for _, val := range embedding {
		sumSq += val * val
	}
	if sumSq > 0 {
		norm := float32(1.0) / sqrt32(sumSq)
		for i := range embedding {
			embedding[i] *= norm
		}
	}
	
	return embedding
}

// Helper functions
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// TestChromemStore_SemanticSimilarity_HighScoresForSimilarContent verifies that
// semantically similar documents get high similarity scores (>0.7).
func TestChromemStore_SemanticSimilarity_HighScoresForSimilarContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_semantic_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	embedder := &semanticTestEmbedder{vectorSize: 384}
	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add documents with known semantic relationships
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Go programming language tutorial"},
		{ID: "doc2", Content: "Golang programming guide"},        // Very similar to doc1
		{ID: "doc3", Content: "Python machine learning tutorial"}, // Different topic
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search for "Go programming" - should find doc1 and doc2 with high scores
	results, err := store.Search(ctx, "Go programming language", 3)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Find doc1 or doc2 in results - should have high similarity score
	foundHighScore := false
	for _, r := range results {
		if r.ID == "doc1" || r.ID == "doc2" {
			// Semantically similar content should score > 0.7
			assert.Greater(t, r.Score, float32(0.7), 
				"Similar content (ID: %s) should have similarity > 0.7, got %.3f", r.ID, r.Score)
			foundHighScore = true
		}
	}
	assert.True(t, foundHighScore, "Should find at least one similar document with high score")
}

// TestChromemStore_SemanticSimilarity_LowScoresForDissimilarContent verifies that
// semantically dissimilar documents get low similarity scores (<0.5).
func TestChromemStore_SemanticSimilarity_LowScoresForDissimilarContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_semantic_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	embedder := &semanticTestEmbedder{vectorSize: 384}
	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add documents with different semantic content
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Go programming language tutorial"},
		{ID: "doc2", Content: "Database vector search guide"},    // Completely different
		{ID: "doc3", Content: "Python machine learning tutorial"}, // Also different
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search for "Go programming" - doc2 and doc3 should have low scores if returned
	results, err := store.Search(ctx, "Go programming language", 3)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Check dissimilar documents have low scores
	for _, r := range results {
		if r.ID == "doc2" || r.ID == "doc3" {
			// Semantically dissimilar content should score < 0.5
			assert.Less(t, r.Score, float32(0.5), 
				"Dissimilar content (ID: %s) should have similarity < 0.5, got %.3f", r.ID, r.Score)
		}
	}
}

// TestChromemStore_SemanticSimilarity_RankingByRelevance verifies that
// search results are ranked by semantic relevance (most similar first).
func TestChromemStore_SemanticSimilarity_RankingByRelevance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_semantic_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	embedder := &semanticTestEmbedder{vectorSize: 384}
	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add documents with varying relevance to our query
	docs := []vectorstore.Document{
		{ID: "exact_match", Content: "Go programming language tutorial guide"},     // Exact match
		{ID: "close_match", Content: "Golang programming tutorial"},                // Very similar
		{ID: "partial_match", Content: "Programming language guide"},               // Partial overlap
		{ID: "unrelated", Content: "Database vector search"},                       // Unrelated
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search for "Go programming language tutorial"
	results, err := store.Search(ctx, "Go programming language tutorial", 4)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Verify results are sorted by descending similarity score
	for i := 0; i < len(results)-1; i++ {
		assert.GreaterOrEqual(t, results[i].Score, results[i+1].Score,
			"Results should be ranked by similarity: %s (%.3f) should rank >= %s (%.3f)",
			results[i].ID, results[i].Score, results[i+1].ID, results[i+1].Score)
	}

	// Verify most relevant document is first
	if len(results) > 0 {
		topResult := results[0]
		// Should be either exact_match or close_match
		assert.Contains(t, []string{"exact_match", "close_match"}, topResult.ID,
			"Most relevant document should be ranked first, got: %s", topResult.ID)
	}
}

// TestChromemStore_SemanticSimilarity_ScoreRange verifies that
// similarity scores are in valid range [0, 1].
func TestChromemStore_SemanticSimilarity_ScoreRange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_semantic_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	embedder := &semanticTestEmbedder{vectorSize: 384}
	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add diverse documents
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Go programming language"},
		{ID: "doc2", Content: "Python machine learning"},
		{ID: "doc3", Content: "Database vector search"},
		{ID: "doc4", Content: "Tutorial guide learning"},
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search and verify all scores are in valid range
	results, err := store.Search(ctx, "Go programming", 10)
	require.NoError(t, err)

	for _, r := range results {
		assert.GreaterOrEqual(t, r.Score, float32(0.0),
			"Score should be >= 0.0, got %.3f for doc %s", r.Score, r.ID)
		assert.LessOrEqual(t, r.Score, float32(1.0),
			"Score should be <= 1.0, got %.3f for doc %s", r.Score, r.ID)
		
		// Also verify score is not NaN or Inf
		assert.False(t, math.IsNaN(float64(r.Score)),
			"Score should not be NaN for doc %s", r.ID)
		assert.False(t, math.IsInf(float64(r.Score), 0),
			"Score should not be Inf for doc %s", r.ID)
	}
}

// TestChromemStore_SemanticSimilarity_SynonymHandling verifies that
// synonyms (words with similar meanings) are treated as semantically similar.
func TestChromemStore_SemanticSimilarity_SynonymHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_semantic_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "semantic_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(),
	}

	embedder := &semanticTestEmbedder{vectorSize: 384}
	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Add documents using synonyms
	docs := []vectorstore.Document{
		{ID: "tutorial_doc", Content: "Programming language tutorial"},
		{ID: "guide_doc", Content: "Programming language guide"}, // "guide" is synonym of "tutorial"
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search for "tutorial" - should find both with similar scores
	results, err := store.Search(ctx, "tutorial", 2)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Both should have reasonably high scores since "tutorial" and "guide" are synonyms
	tutorialScore := float32(0.0)
	guideScore := float32(0.0)
	for _, r := range results {
		if r.ID == "tutorial_doc" {
			tutorialScore = r.Score
		} else if r.ID == "guide_doc" {
			guideScore = r.Score
		}
	}

	// Scores should be similar (within 0.3 of each other) since they're synonyms
	scoreDiff := abs32(tutorialScore - guideScore)
	assert.Less(t, scoreDiff, float32(0.3),
		"Synonym scores should be similar: tutorial=%.3f, guide=%.3f, diff=%.3f",
		tutorialScore, guideScore, scoreDiff)
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
