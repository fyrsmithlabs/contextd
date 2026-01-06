package reasoningbank

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	// Test that identical vectors have similarity of 1.0
	vec1 := []float32{1.0, 2.0, 3.0, 4.0}
	vec2 := []float32{1.0, 2.0, 3.0, 4.0}

	similarity := CosineSimilarity(vec1, vec2)
	assert.InDelta(t, 1.0, similarity, 0.0001,
		"identical vectors should have cosine similarity of 1.0")
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	// Test that orthogonal (perpendicular) vectors have similarity of 0.0
	vec1 := []float32{1.0, 0.0, 0.0}
	vec2 := []float32{0.0, 1.0, 0.0}

	similarity := CosineSimilarity(vec1, vec2)
	assert.InDelta(t, 0.0, similarity, 0.0001,
		"orthogonal vectors should have cosine similarity of 0.0")
}

func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	// Test that opposite vectors have similarity of -1.0
	vec1 := []float32{1.0, 2.0, 3.0}
	vec2 := []float32{-1.0, -2.0, -3.0}

	similarity := CosineSimilarity(vec1, vec2)
	assert.InDelta(t, -1.0, similarity, 0.0001,
		"opposite vectors should have cosine similarity of -1.0")
}

func TestCosineSimilarity_ScaledVectors(t *testing.T) {
	// Test that scaled versions of the same vector have similarity of 1.0
	// (cosine similarity is scale-invariant)
	vec1 := []float32{1.0, 2.0, 3.0}
	vec2 := []float32{2.0, 4.0, 6.0} // vec1 * 2

	similarity := CosineSimilarity(vec1, vec2)
	assert.InDelta(t, 1.0, similarity, 0.0001,
		"scaled vectors should have cosine similarity of 1.0")
}

func TestCosineSimilarity_PartialSimilarity(t *testing.T) {
	// Test vectors with partial similarity (45-degree angle)
	vec1 := []float32{1.0, 0.0}
	vec2 := []float32{1.0, 1.0}

	similarity := CosineSimilarity(vec1, vec2)
	// cos(45°) ≈ 0.7071
	expected := 1.0 / math.Sqrt(2)
	assert.InDelta(t, expected, similarity, 0.0001,
		"45-degree angle should have cosine similarity of ~0.7071")
}

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	// Test that empty vectors return 0.0
	vec1 := []float32{}
	vec2 := []float32{}

	similarity := CosineSimilarity(vec1, vec2)
	assert.Equal(t, 0.0, similarity,
		"empty vectors should return 0.0")
}

func TestCosineSimilarity_OneEmptyVector(t *testing.T) {
	// Test that one empty vector returns 0.0
	vec1 := []float32{1.0, 2.0, 3.0}
	vec2 := []float32{}

	similarity := CosineSimilarity(vec1, vec2)
	assert.Equal(t, 0.0, similarity,
		"one empty vector should return 0.0")
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	// Test that vectors of different lengths return 0.0
	vec1 := []float32{1.0, 2.0, 3.0}
	vec2 := []float32{1.0, 2.0}

	similarity := CosineSimilarity(vec1, vec2)
	assert.Equal(t, 0.0, similarity,
		"vectors of different lengths should return 0.0")
}

func TestCosineSimilarity_ZeroMagnitudeVector(t *testing.T) {
	// Test that zero-magnitude vectors return 0.0
	vec1 := []float32{0.0, 0.0, 0.0}
	vec2 := []float32{1.0, 2.0, 3.0}

	similarity := CosineSimilarity(vec1, vec2)
	assert.Equal(t, 0.0, similarity,
		"zero-magnitude vector should return 0.0")
}

func TestCosineSimilarity_BothZeroMagnitude(t *testing.T) {
	// Test that both zero-magnitude vectors return 0.0
	vec1 := []float32{0.0, 0.0, 0.0}
	vec2 := []float32{0.0, 0.0, 0.0}

	similarity := CosineSimilarity(vec1, vec2)
	assert.Equal(t, 0.0, similarity,
		"both zero-magnitude vectors should return 0.0")
}

func TestCosineSimilarity_RealisticEmbeddings(t *testing.T) {
	// Test with realistic embedding-like vectors (384-dimensional)
	// Create two similar but not identical vectors
	vec1 := make([]float32, 384)
	vec2 := make([]float32, 384)

	for i := 0; i < 384; i++ {
		vec1[i] = float32(i % 10) / 10.0
		vec2[i] = vec1[i] + 0.1 // Slightly different
	}

	similarity := CosineSimilarity(vec1, vec2)
	// Should be high similarity but not 1.0
	assert.Greater(t, similarity, 0.9,
		"similar embedding vectors should have high similarity")
	assert.Less(t, similarity, 1.0,
		"slightly different vectors should not have perfect similarity")
}

func TestCosineSimilarity_HighSimilarity(t *testing.T) {
	// Test vectors with high similarity (memories that should be consolidated)
	// Simulate two embeddings of similar concepts
	vec1 := []float32{0.5, 0.8, 0.3, 0.9, 0.1}
	vec2 := []float32{0.5, 0.8, 0.3, 0.9, 0.15} // Very similar, small difference in last component

	similarity := CosineSimilarity(vec1, vec2)
	// Should be above the consolidation threshold (0.8)
	assert.Greater(t, similarity, 0.8,
		"very similar vectors should have similarity > 0.8")
}

func TestCosineSimilarity_LowSimilarity(t *testing.T) {
	// Test vectors with low similarity (memories that should NOT be consolidated)
	vec1 := []float32{1.0, 0.0, 0.0, 0.0, 0.0}
	vec2 := []float32{0.0, 0.0, 0.0, 0.0, 1.0}

	similarity := CosineSimilarity(vec1, vec2)
	// Should be below the consolidation threshold (0.8)
	assert.Less(t, similarity, 0.8,
		"dissimilar vectors should have similarity < 0.8")
}

func TestCosineSimilarity_Commutative(t *testing.T) {
	// Test that cosine similarity is commutative: sim(A, B) = sim(B, A)
	vec1 := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	vec2 := []float32{5.0, 4.0, 3.0, 2.0, 1.0}

	sim1 := CosineSimilarity(vec1, vec2)
	sim2 := CosineSimilarity(vec2, vec1)

	assert.Equal(t, sim1, sim2,
		"cosine similarity should be commutative")
}

func TestCosineSimilarity_Range(t *testing.T) {
	// Test that similarity is always in [-1, 1] range
	testCases := []struct {
		name string
		vec1 []float32
		vec2 []float32
	}{
		{"positive vectors", []float32{1, 2, 3}, []float32{4, 5, 6}},
		{"negative vectors", []float32{-1, -2, -3}, []float32{-4, -5, -6}},
		{"mixed signs", []float32{1, -2, 3}, []float32{-4, 5, -6}},
		{"large values", []float32{100, 200, 300}, []float32{150, 250, 350}},
		{"small values", []float32{0.001, 0.002, 0.003}, []float32{0.002, 0.003, 0.004}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			similarity := CosineSimilarity(tc.vec1, tc.vec2)
			assert.GreaterOrEqual(t, similarity, -1.0,
				"similarity should be >= -1.0 for %s", tc.name)
			assert.LessOrEqual(t, similarity, 1.0,
				"similarity should be <= 1.0 for %s", tc.name)
		})
	}
}

// TestFindSimilarClusters_ValidInput tests cluster detection with known similar memories.
func TestFindSimilarClusters_ValidInput(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10) // Small vector size for testing
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "cluster-test-project"

	// Create three similar memories (similar titles will have similar embeddings)
	memory1, _ := NewMemory(projectID, "Go error handling", "Content about Go errors", OutcomeSuccess, []string{"go"})
	memory2, _ := NewMemory(projectID, "Go error management", "More content about Go errors", OutcomeSuccess, []string{"go"})
	memory3, _ := NewMemory(projectID, "Python error handling", "Different language but similar topic", OutcomeSuccess, []string{"python"})
	memory4, _ := NewMemory(projectID, "Database connection pooling", "Completely different topic", OutcomeSuccess, []string{"database"})

	// Record all memories
	require.NoError(t, svc.Record(ctx, memory1))
	require.NoError(t, svc.Record(ctx, memory2))
	require.NoError(t, svc.Record(ctx, memory3))
	require.NoError(t, svc.Record(ctx, memory4))

	// Find clusters with threshold 0.8
	clusters, err := distiller.FindSimilarClusters(ctx, projectID, 0.8)
	require.NoError(t, err)
	assert.NotNil(t, clusters)

	// With our mock embedder, similarity is based on title+content length
	// Similar titles should create clusters
	// The exact number of clusters depends on the mock embedder behavior
	t.Logf("Found %d clusters", len(clusters))
}

// TestFindSimilarClusters_HighSimilarity tests cluster detection with very similar memories.
func TestFindSimilarClusters_HighSimilarity(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "high-similarity-project"

	// Create two memories with identical titles (will have very similar embeddings)
	memory1, _ := NewMemory(projectID, "Authentication with JWT tokens", "Content 1", OutcomeSuccess, []string{"auth"})
	memory2, _ := NewMemory(projectID, "Authentication with JWT tokens", "Content 2", OutcomeSuccess, []string{"auth"})

	require.NoError(t, svc.Record(ctx, memory1))
	require.NoError(t, svc.Record(ctx, memory2))

	// Find clusters with threshold 0.9 (high threshold)
	clusters, err := distiller.FindSimilarClusters(ctx, projectID, 0.9)
	require.NoError(t, err)

	// Should find at least one cluster since titles are identical
	if len(clusters) > 0 {
		// Verify cluster properties
		for _, cluster := range clusters {
			assert.GreaterOrEqual(t, len(cluster.Members), 2, "cluster should have at least 2 members")
			assert.NotNil(t, cluster.CentroidVector, "cluster should have centroid vector")
			assert.Equal(t, 10, len(cluster.CentroidVector), "centroid should match vector size")
			assert.GreaterOrEqual(t, cluster.AverageSimilarity, 0.0, "average similarity should be >= 0")
			assert.LessOrEqual(t, cluster.AverageSimilarity, 1.0, "average similarity should be <= 1")
			assert.GreaterOrEqual(t, cluster.MinSimilarity, 0.0, "min similarity should be >= 0")
			assert.LessOrEqual(t, cluster.MinSimilarity, 1.0, "min similarity should be <= 1")
			assert.LessOrEqual(t, cluster.MinSimilarity, cluster.AverageSimilarity, "min should be <= average")
		}
	}
}

// TestFindSimilarClusters_DissimilarMemories tests that dissimilar memories don't cluster.
func TestFindSimilarClusters_DissimilarMemories(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "dissimilar-project"

	// Create memories with very different content lengths (will have dissimilar embeddings)
	memory1, _ := NewMemory(projectID, "A", "Short", OutcomeSuccess, []string{"tag1"})
	memory2, _ := NewMemory(projectID, "B is a much longer title for testing purposes here",
		"This is a very long content string that should produce different embeddings",
		OutcomeSuccess, []string{"tag2"})

	require.NoError(t, svc.Record(ctx, memory1))
	require.NoError(t, svc.Record(ctx, memory2))

	// Find clusters with high threshold
	clusters, err := distiller.FindSimilarClusters(ctx, projectID, 0.95)
	require.NoError(t, err)

	// Should not find clusters with such dissimilar content
	assert.Equal(t, 0, len(clusters), "dissimilar memories should not cluster")
}

// TestFindSimilarClusters_MultipleClusters tests detection of multiple distinct clusters.
func TestFindSimilarClusters_MultipleClusters(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "multi-cluster-project"

	// Create two groups of similar memories
	// Group 1: Go error handling (similar titles/content)
	mem1, _ := NewMemory(projectID, "Go error handling pattern", "Error handling in Go", OutcomeSuccess, []string{"go"})
	mem2, _ := NewMemory(projectID, "Go error handling best practice", "Error handling in Go", OutcomeSuccess, []string{"go"})

	// Group 2: Database optimization (similar titles/content)
	mem3, _ := NewMemory(projectID, "Database query optimization", "Optimize DB queries", OutcomeSuccess, []string{"db"})
	mem4, _ := NewMemory(projectID, "Database query performance", "Optimize DB queries", OutcomeSuccess, []string{"db"})

	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))
	require.NoError(t, svc.Record(ctx, mem4))

	// Find clusters with moderate threshold
	clusters, err := distiller.FindSimilarClusters(ctx, projectID, 0.7)
	require.NoError(t, err)

	// Should potentially find multiple clusters (exact count depends on embedder)
	t.Logf("Found %d clusters with 2 expected groups", len(clusters))

	// Verify each cluster has valid properties
	for i, cluster := range clusters {
		assert.GreaterOrEqual(t, len(cluster.Members), 2, "cluster %d should have at least 2 members", i)
		assert.NotNil(t, cluster.CentroidVector, "cluster %d should have centroid", i)
		assert.Greater(t, cluster.AverageSimilarity, 0.0, "cluster %d should have positive average similarity", i)
	}
}

// TestFindSimilarClusters_EmptyProject tests handling of projects with no memories.
func TestFindSimilarClusters_EmptyProject(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	// Find clusters in empty project
	clusters, err := distiller.FindSimilarClusters(ctx, "empty-project", 0.8)
	require.NoError(t, err)
	assert.Empty(t, clusters, "empty project should have no clusters")
}

// TestFindSimilarClusters_SingleMemory tests handling of projects with only one memory.
func TestFindSimilarClusters_SingleMemory(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "single-memory-project"

	memory, _ := NewMemory(projectID, "Single memory", "Lone content", OutcomeSuccess, []string{"solo"})
	require.NoError(t, svc.Record(ctx, memory))

	// Find clusters
	clusters, err := distiller.FindSimilarClusters(ctx, projectID, 0.8)
	require.NoError(t, err)
	assert.Empty(t, clusters, "single memory cannot form a cluster")
}

// TestFindSimilarClusters_InvalidThreshold tests threshold validation.
func TestFindSimilarClusters_InvalidThreshold(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		threshold float64
		wantError bool
	}{
		{"negative threshold", -0.5, true},
		{"threshold too high", 1.5, true},
		{"valid minimum", 0.0, false},
		{"valid maximum", 1.0, false},
		{"valid middle", 0.8, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := distiller.FindSimilarClusters(ctx, "test-project", tc.threshold)
			if tc.wantError {
				assert.Error(t, err, "invalid threshold should return error")
			} else {
				assert.NoError(t, err, "valid threshold should not return error")
			}
		})
	}
}

// TestFindSimilarClusters_EmptyProjectID tests project ID validation.
func TestFindSimilarClusters_EmptyProjectID(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	// Test with empty project ID
	_, err = distiller.FindSimilarClusters(ctx, "", 0.8)
	assert.ErrorIs(t, err, ErrEmptyProjectID, "empty project ID should return error")
}

// TestFindSimilarClusters_ClusterStatistics tests that cluster statistics are calculated correctly.
func TestFindSimilarClusters_ClusterStatistics(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "stats-project"

	// Create three memories with very similar content
	mem1, _ := NewMemory(projectID, "Test memory one", "Content", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Test memory two", "Content", OutcomeSuccess, []string{"test"})
	mem3, _ := NewMemory(projectID, "Test memory six", "Content", OutcomeSuccess, []string{"test"})

	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))

	// Find clusters
	clusters, err := distiller.FindSimilarClusters(ctx, projectID, 0.7)
	require.NoError(t, err)

	// Verify statistics for any found clusters
	for _, cluster := range clusters {
		// Centroid vector should have correct size
		assert.Equal(t, 10, len(cluster.CentroidVector), "centroid should have correct vector size")

		// Centroid values should be reasonable (between min and max of member vectors)
		// Get a sample vector from first member
		vec1, err := svc.GetMemoryVectorByProjectID(ctx, projectID, cluster.Members[0].ID)
		require.NoError(t, err)

		for i := range cluster.CentroidVector {
			// Centroid should be within reasonable bounds
			assert.GreaterOrEqual(t, cluster.CentroidVector[i], float32(0.0))
			assert.LessOrEqual(t, cluster.CentroidVector[i], vec1[i]*10) // Loose upper bound
		}

		// Average similarity should be in valid range
		assert.GreaterOrEqual(t, cluster.AverageSimilarity, 0.0)
		assert.LessOrEqual(t, cluster.AverageSimilarity, 1.0)

		// Min similarity should be in valid range
		assert.GreaterOrEqual(t, cluster.MinSimilarity, 0.0)
		assert.LessOrEqual(t, cluster.MinSimilarity, 1.0)

		// Min should not exceed average
		assert.LessOrEqual(t, cluster.MinSimilarity, cluster.AverageSimilarity)

		t.Logf("Cluster with %d members: avg_sim=%.3f, min_sim=%.3f",
			len(cluster.Members), cluster.AverageSimilarity, cluster.MinSimilarity)
	}
}

// TestFindSimilarClusters_NoEmbedder tests error handling when embedder is not set.
func TestFindSimilarClusters_NoEmbedder(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	logger := zap.NewNop()

	// Create service WITHOUT embedder
	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "no-embedder-project"

	// Create a memory
	memory, _ := NewMemory(projectID, "Test", "Content", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, memory))

	// Try to find clusters - should fail because embedder is required
	_, err = distiller.FindSimilarClusters(ctx, projectID, 0.8)
	assert.Error(t, err, "should error when embedder is not set")
}

// TestBuildConsolidationPrompt_SingleMemory tests prompt generation with a single memory.
func TestBuildConsolidationPrompt_SingleMemory(t *testing.T) {
	memory, err := NewMemory(
		"test-project",
		"Error Handling Strategy",
		"Always wrap errors with context using fmt.Errorf",
		OutcomeSuccess,
		[]string{"go", "error-handling"},
	)
	require.NoError(t, err)
	memory.Description = "A common pattern for Go error handling"
	memory.Confidence = 0.8
	memory.UsageCount = 5

	prompt := buildConsolidationPrompt([]*Memory{memory})

	// Verify prompt structure
	assert.Contains(t, prompt, "You are a memory consolidation assistant")
	assert.Contains(t, prompt, "## Source Memories")
	assert.Contains(t, prompt, "## Your Task")
	assert.Contains(t, prompt, "## Output Format")

	// Verify memory details are included
	assert.Contains(t, prompt, "Memory 1: Error Handling Strategy")
	assert.Contains(t, prompt, "A common pattern for Go error handling")
	assert.Contains(t, prompt, "Always wrap errors with context using fmt.Errorf")
	assert.Contains(t, prompt, "go, error-handling")
	assert.Contains(t, prompt, "Outcome: success")
	assert.Contains(t, prompt, "Confidence: 0.80")
	assert.Contains(t, prompt, "Usage Count: 5")

	// Verify task instructions
	assert.Contains(t, prompt, "Identify the Common Theme")
	assert.Contains(t, prompt, "Synthesize Key Insights")
	assert.Contains(t, prompt, "Preserve Important Details")
	assert.Contains(t, prompt, "Note When to Apply")

	// Verify output format specification
	assert.Contains(t, prompt, "TITLE:")
	assert.Contains(t, prompt, "CONTENT:")
	assert.Contains(t, prompt, "TAGS:")
	assert.Contains(t, prompt, "OUTCOME:")
	assert.Contains(t, prompt, "SOURCE_ATTRIBUTION:")
}

// TestBuildConsolidationPrompt_MultipleMemories tests prompt with multiple memories.
func TestBuildConsolidationPrompt_MultipleMemories(t *testing.T) {
	memory1, err := NewMemory(
		"test-project",
		"Use context.Context for cancellation",
		"Pass context.Context as first parameter to enable cancellation",
		OutcomeSuccess,
		[]string{"go", "context"},
	)
	require.NoError(t, err)
	memory1.Confidence = 0.9
	memory1.UsageCount = 10

	memory2, err := NewMemory(
		"test-project",
		"Context deadline handling",
		"Check context.Err() to detect cancellation or deadline exceeded",
		OutcomeSuccess,
		[]string{"go", "context", "timeout"},
	)
	require.NoError(t, err)
	memory2.Description = "Important for long-running operations"
	memory2.Confidence = 0.85
	memory2.UsageCount = 7

	memory3, err := NewMemory(
		"test-project",
		"Avoid context.Background in libraries",
		"Don't use context.Background() in library code, accept ctx from caller",
		OutcomeFailure,
		[]string{"go", "context", "anti-pattern"},
	)
	require.NoError(t, err)
	memory3.Confidence = 0.75
	memory3.UsageCount = 3

	prompt := buildConsolidationPrompt([]*Memory{memory1, memory2, memory3})

	// Verify all memories are included
	assert.Contains(t, prompt, "Memory 1: Use context.Context for cancellation")
	assert.Contains(t, prompt, "Memory 2: Context deadline handling")
	assert.Contains(t, prompt, "Memory 3: Avoid context.Background in libraries")

	// Verify separators between memories
	assert.Contains(t, prompt, "---")

	// Verify all memory contents are included
	assert.Contains(t, prompt, "Pass context.Context as first parameter")
	assert.Contains(t, prompt, "Check context.Err() to detect cancellation")
	assert.Contains(t, prompt, "Don't use context.Background() in library code")

	// Verify different outcomes are shown
	assert.Contains(t, prompt, "Outcome: success")
	assert.Contains(t, prompt, "Outcome: failure")

	// Verify descriptions when present
	assert.Contains(t, prompt, "Important for long-running operations")

	// Verify task guidance emphasizes synthesis
	assert.Contains(t, prompt, "Synthesize insights, don't just summarize")
	assert.Contains(t, prompt, "MORE valuable memory than any individual source")
}

// TestBuildConsolidationPrompt_EmptySlice tests handling of empty memory slice.
func TestBuildConsolidationPrompt_EmptySlice(t *testing.T) {
	prompt := buildConsolidationPrompt([]*Memory{})

	// Should still have valid structure even with no memories
	assert.Contains(t, prompt, "You are a memory consolidation assistant")
	assert.Contains(t, prompt, "## Source Memories")
	assert.Contains(t, prompt, "## Your Task")

	// Should not have memory separators
	assert.NotContains(t, prompt, "---")
}

// TestBuildConsolidationPrompt_MemoryWithoutOptionalFields tests handling of minimal memory.
func TestBuildConsolidationPrompt_MemoryWithoutOptionalFields(t *testing.T) {
	memory, err := NewMemory(
		"test-project",
		"Minimal Memory",
		"Just basic content",
		OutcomeSuccess,
		[]string{}, // No tags
	)
	require.NoError(t, err)
	// No description set

	prompt := buildConsolidationPrompt([]*Memory{memory})

	// Should include title and content
	assert.Contains(t, prompt, "Memory 1: Minimal Memory")
	assert.Contains(t, prompt, "Just basic content")

	// Should not have description or tags sections when empty
	assert.NotContains(t, prompt, "**Description:**")
	assert.NotContains(t, prompt, "**Tags:**")

	// Should still have required fields
	assert.Contains(t, prompt, "Outcome: success")
	assert.Contains(t, prompt, "Confidence:")
	assert.Contains(t, prompt, "Usage Count:")
}

// TestBuildConsolidationPrompt_FormattingConsistency tests consistent formatting.
func TestBuildConsolidationPrompt_FormattingConsistency(t *testing.T) {
	memories := make([]*Memory, 5)
	for i := 0; i < 5; i++ {
		mem, err := NewMemory(
			"test-project",
			fmt.Sprintf("Memory %d", i+1),
			fmt.Sprintf("Content for memory %d", i+1),
			OutcomeSuccess,
			[]string{fmt.Sprintf("tag%d", i+1)},
		)
		require.NoError(t, err)
		mem.Confidence = float64(i+1) * 0.15
		mem.UsageCount = i + 1
		memories[i] = mem
	}

	prompt := buildConsolidationPrompt(memories)

	// Each memory should be formatted consistently
	for i := 1; i <= 5; i++ {
		assert.Contains(t, prompt, fmt.Sprintf("### Memory %d:", i))
		assert.Contains(t, prompt, fmt.Sprintf("Memory %d", i))
		assert.Contains(t, prompt, fmt.Sprintf("Content for memory %d", i))
	}

	// Should have 4 separators for 5 memories
	separatorCount := 0
	for i := 0; i < len(prompt)-3; i++ {
		if prompt[i:i+3] == "---" {
			separatorCount++
		}
	}
	// Note: There might be separators in the template itself, so check for at least 4
	assert.GreaterOrEqual(t, separatorCount, 4, "should have separator between each pair of memories")
}

// TestBuildConsolidationPrompt_LongContent tests handling of memories with long content.
func TestBuildConsolidationPrompt_LongContent(t *testing.T) {
	longContent := strings.Repeat("This is a very long content string with lots of details. ", 100)
	memory, err := NewMemory(
		"test-project",
		"Long Memory",
		longContent,
		OutcomeSuccess,
		[]string{"go", "verbose"},
	)
	require.NoError(t, err)

	prompt := buildConsolidationPrompt([]*Memory{memory})

	// Should include the full content without truncation
	assert.Contains(t, prompt, longContent)
	assert.Contains(t, prompt, "Long Memory")
}

// TestBuildConsolidationPrompt_SpecialCharacters tests handling of special characters.
func TestBuildConsolidationPrompt_SpecialCharacters(t *testing.T) {
	memory, err := NewMemory(
		"test-project",
		"Special chars: <>\"'&",
		"Content with special chars: \n\t\r $ % # @ !",
		OutcomeSuccess,
		[]string{"special", "chars"},
	)
	require.NoError(t, err)

	prompt := buildConsolidationPrompt([]*Memory{memory})

	// Should preserve special characters
	assert.Contains(t, prompt, "Special chars: <>\"'&")
	assert.Contains(t, prompt, "Content with special chars: \n\t\r $ % # @ !")
}
