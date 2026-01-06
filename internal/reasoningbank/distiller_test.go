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

func TestParseConsolidatedMemory_ValidResponse(t *testing.T) {
	// Test parsing a valid LLM response with all fields
	llmResponse := `
TITLE: Consolidated API Error Handling Strategy

CONTENT:
When building REST APIs, implement comprehensive error handling with:
1. Structured error responses with error codes
2. Detailed error messages for developers
3. Safe, user-friendly messages for clients
4. Proper HTTP status codes

TAGS: go, api, error-handling, rest

OUTCOME: success

SOURCE_ATTRIBUTION:
Synthesized from 3 source memories about API error handling patterns.
Combines insights from authentication, validation, and database error scenarios.
`

	sourceIDs := []string{"mem-1", "mem-2", "mem-3"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	// Validate parsed fields
	assert.Equal(t, "Consolidated API Error Handling Strategy", memory.Title)
	assert.Contains(t, memory.Content, "When building REST APIs")
	assert.Contains(t, memory.Content, "Proper HTTP status codes")
	assert.Equal(t, OutcomeSuccess, memory.Outcome)
	assert.Equal(t, []string{"go", "api", "error-handling", "rest"}, memory.Tags)
	assert.Contains(t, memory.Description, "Synthesized from 3 source memories")
	assert.Equal(t, DistilledConfidence, memory.Confidence)
	assert.Equal(t, 0, memory.UsageCount)
}

func TestParseConsolidatedMemory_MinimalResponse(t *testing.T) {
	// Test parsing a response with only required fields
	llmResponse := `
TITLE: Database Connection Pattern

CONTENT:
Always use connection pooling with proper timeout configuration.
Set max connections based on workload requirements.

OUTCOME: success
`

	sourceIDs := []string{"mem-1", "mem-2"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	assert.Equal(t, "Database Connection Pattern", memory.Title)
	assert.Contains(t, memory.Content, "connection pooling")
	assert.Equal(t, OutcomeSuccess, memory.Outcome)
	assert.Empty(t, memory.Tags)
	assert.Empty(t, memory.Description) // No source attribution
}

func TestParseConsolidatedMemory_FailureOutcome(t *testing.T) {
	// Test parsing a response with failure outcome
	llmResponse := `
TITLE: Anti-pattern: Ignoring Context Cancellation

CONTENT:
Never ignore context cancellation in long-running operations.
This leads to resource leaks and hanging goroutines.

TAGS: go, concurrency, context

OUTCOME: failure

SOURCE_ATTRIBUTION:
Common mistake observed across multiple failed implementations.
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	assert.Equal(t, OutcomeFailure, memory.Outcome)
	assert.Contains(t, memory.Title, "Anti-pattern")
}

func TestParseConsolidatedMemory_MissingTitle(t *testing.T) {
	// Test error handling when TITLE is missing
	llmResponse := `
CONTENT:
Some content here

OUTCOME: success
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	assert.Error(t, err)
	assert.Nil(t, memory)
	assert.Contains(t, err.Error(), "TITLE field is required")
}

func TestParseConsolidatedMemory_MissingContent(t *testing.T) {
	// Test error handling when CONTENT is missing
	llmResponse := `
TITLE: Some Title

OUTCOME: success
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	assert.Error(t, err)
	assert.Nil(t, memory)
	assert.Contains(t, err.Error(), "CONTENT field is required")
}

func TestParseConsolidatedMemory_MissingOutcome(t *testing.T) {
	// Test error handling when OUTCOME is missing
	llmResponse := `
TITLE: Some Title

CONTENT:
Some content
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	assert.Error(t, err)
	assert.Nil(t, memory)
	assert.Contains(t, err.Error(), "OUTCOME field is required")
}

func TestParseConsolidatedMemory_InvalidOutcome(t *testing.T) {
	// Test error handling with invalid outcome value
	llmResponse := `
TITLE: Some Title

CONTENT:
Some content

OUTCOME: maybe
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	assert.Error(t, err)
	assert.Nil(t, memory)
	assert.Contains(t, err.Error(), "invalid OUTCOME value")
	assert.Contains(t, err.Error(), "maybe")
}

func TestParseConsolidatedMemory_EmptyResponse(t *testing.T) {
	// Test error handling with empty LLM response
	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory("", sourceIDs)
	assert.Error(t, err)
	assert.Nil(t, memory)
	assert.Contains(t, err.Error(), "llm response cannot be empty")
}

func TestParseConsolidatedMemory_EmptySourceIDs(t *testing.T) {
	// Test error handling with empty sourceIDs
	llmResponse := `
TITLE: Some Title

CONTENT:
Some content

OUTCOME: success
`

	memory, err := parseConsolidatedMemory(llmResponse, []string{})
	assert.Error(t, err)
	assert.Nil(t, memory)
	assert.Contains(t, err.Error(), "sourceIDs cannot be empty")
}

func TestParseConsolidatedMemory_TagsWithSpaces(t *testing.T) {
	// Test parsing tags with various spacing
	llmResponse := `
TITLE: Test Title

CONTENT:
Test content

TAGS: go, api,  error-handling  ,rest,   kubernetes

OUTCOME: success
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	// Tags should be trimmed
	assert.Equal(t, []string{"go", "api", "error-handling", "rest", "kubernetes"}, memory.Tags)
}

func TestParseConsolidatedMemory_MultiLineContent(t *testing.T) {
	// Test parsing multi-line content with formatting
	llmResponse := `
TITLE: Multi-line Example

CONTENT:
This is a multi-line content block.

It has multiple paragraphs and should preserve structure.

- Bullet point 1
- Bullet point 2

Code example:
  func example() {
      return nil
  }

OUTCOME: success
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	// Content should preserve multiple lines
	assert.Contains(t, memory.Content, "multi-line content block")
	assert.Contains(t, memory.Content, "multiple paragraphs")
	assert.Contains(t, memory.Content, "Bullet point 1")
	assert.Contains(t, memory.Content, "func example()")
}

func TestParseConsolidatedMemory_WithCodeBlockMarkers(t *testing.T) {
	// Test parsing response with markdown code block markers
	llmResponse := "```\n" + `
TITLE: Example With Code Blocks

CONTENT:
Content inside code blocks

OUTCOME: success
` + "\n```"

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	assert.Equal(t, "Example With Code Blocks", memory.Title)
	assert.Contains(t, memory.Content, "Content inside code blocks")
}

func TestParseConsolidatedMemory_CaseInsensitiveOutcome(t *testing.T) {
	// Test that outcome parsing is case-insensitive
	testCases := []struct {
		name     string
		outcome  string
		expected Outcome
	}{
		{"lowercase success", "success", OutcomeSuccess},
		{"uppercase success", "SUCCESS", OutcomeSuccess},
		{"mixed case success", "SuCcEsS", OutcomeSuccess},
		{"lowercase failure", "failure", OutcomeFailure},
		{"uppercase failure", "FAILURE", OutcomeFailure},
		{"mixed case failure", "FaIlUrE", OutcomeFailure},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			llmResponse := fmt.Sprintf(`
TITLE: Test Title

CONTENT:
Test content

OUTCOME: %s
`, tc.outcome)

			sourceIDs := []string{"mem-1"}

			memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, memory.Outcome)
		})
	}
}

func TestParseConsolidatedMemory_ProjectIDAndIDNotSet(t *testing.T) {
	// Test that ID and ProjectID are not set (must be set by caller)
	llmResponse := `
TITLE: Test Title

CONTENT:
Test content

OUTCOME: success
`

	sourceIDs := []string{"mem-1"}

	memory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	// ID and ProjectID should be empty (caller sets them)
	assert.Empty(t, memory.ID)
	assert.Empty(t, memory.ProjectID)
}

func TestExtractField_BasicExtraction(t *testing.T) {
	// Test basic field extraction
	text := `
TITLE: Example Title

CONTENT:
Example content here
`

	title := extractField(text, "TITLE:")
	assert.Equal(t, "Example Title", title)

	content := extractField(text, "CONTENT:")
	assert.Equal(t, "Example content here", content)
}

func TestExtractField_FieldNotFound(t *testing.T) {
	// Test extraction when field doesn't exist
	text := `
TITLE: Example Title
`

	content := extractField(text, "CONTENT:")
	assert.Empty(t, content)
}

func TestExtractField_MultiLineValue(t *testing.T) {
	// Test extraction of multi-line field values
	text := `
CONTENT:
Line 1
Line 2
Line 3

TAGS: test
`

	content := extractField(text, "CONTENT:")
	assert.Contains(t, content, "Line 1")
	assert.Contains(t, content, "Line 2")
	assert.Contains(t, content, "Line 3")
}

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

// mockLLMClient is a mock LLM client for testing memory consolidation.
// It returns pre-defined synthesis responses without making real LLM API calls.
type mockLLMClient struct {
	// response is the canned response to return from Complete
	response string
	// err is the error to return (if any)
	err error
	// callCount tracks how many times Complete was called
	callCount int
	// lastPrompt stores the last prompt passed to Complete
	lastPrompt string
}

// newMockLLMClient creates a mock LLM client with a default valid response.
// The default response follows the expected format for memory consolidation.
func newMockLLMClient() *mockLLMClient {
	return &mockLLMClient{
		response: `
TITLE: Consolidated Memory Pattern

CONTENT:
This is a synthesized memory that combines insights from multiple source memories.
It represents the common patterns and key learnings extracted from the sources.

The consolidation process identified shared themes and merged them into this
more valuable, integrated understanding that's easier to retrieve and apply.

TAGS: consolidated, pattern, synthesis

OUTCOME: success

SOURCE_ATTRIBUTION:
Synthesized from multiple source memories using LLM-powered consolidation.
Combines common themes and key insights into integrated knowledge.
`,
	}
}

// newMockLLMClientWithResponse creates a mock LLM client with a custom response.
func newMockLLMClientWithResponse(response string) *mockLLMClient {
	return &mockLLMClient{
		response: response,
	}
}

// newMockLLMClientWithError creates a mock LLM client that returns an error.
func newMockLLMClientWithError(err error) *mockLLMClient {
	return &mockLLMClient{
		err: err,
	}
}

// Complete returns the pre-defined response without calling a real LLM.
func (m *mockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	m.callCount++
	m.lastPrompt = prompt

	if m.err != nil {
		return "", m.err
	}

	return m.response, nil
}

// CallCount returns the number of times Complete was called.
func (m *mockLLMClient) CallCount() int {
	return m.callCount
}

// LastPrompt returns the last prompt passed to Complete.
func (m *mockLLMClient) LastPrompt() string {
	return m.lastPrompt
}

// TestMockLLMClient_DefaultResponse tests the default mock LLM client behavior.
func TestMockLLMClient_DefaultResponse(t *testing.T) {
	ctx := context.Background()
	mock := newMockLLMClient()

	// Call Complete
	response, err := mock.Complete(ctx, "test prompt")
	require.NoError(t, err)
	assert.NotEmpty(t, response)

	// Verify response contains expected fields
	assert.Contains(t, response, "TITLE:")
	assert.Contains(t, response, "CONTENT:")
	assert.Contains(t, response, "TAGS:")
	assert.Contains(t, response, "OUTCOME:")
	assert.Contains(t, response, "SOURCE_ATTRIBUTION:")

	// Verify call tracking
	assert.Equal(t, 1, mock.CallCount())
	assert.Equal(t, "test prompt", mock.LastPrompt())
}

// TestMockLLMClient_CustomResponse tests mock with custom response.
func TestMockLLMClient_CustomResponse(t *testing.T) {
	ctx := context.Background()
	customResponse := `
TITLE: Custom Test Memory

CONTENT:
This is a custom response for testing purposes.

OUTCOME: success
`
	mock := newMockLLMClientWithResponse(customResponse)

	response, err := mock.Complete(ctx, "custom prompt")
	require.NoError(t, err)
	assert.Equal(t, customResponse, response)

	// Verify call tracking
	assert.Equal(t, 1, mock.CallCount())
	assert.Equal(t, "custom prompt", mock.LastPrompt())
}

// TestMockLLMClient_Error tests mock that returns an error.
func TestMockLLMClient_Error(t *testing.T) {
	ctx := context.Background()
	expectedErr := fmt.Errorf("mock LLM error")
	mock := newMockLLMClientWithError(expectedErr)

	response, err := mock.Complete(ctx, "error prompt")
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Empty(t, response)

	// Verify call tracking (should still track call even on error)
	assert.Equal(t, 1, mock.CallCount())
	assert.Equal(t, "error prompt", mock.LastPrompt())
}

// TestMockLLMClient_MultipleCalls tests that call tracking works correctly.
func TestMockLLMClient_MultipleCalls(t *testing.T) {
	ctx := context.Background()
	mock := newMockLLMClient()

	// Make multiple calls
	for i := 1; i <= 3; i++ {
		prompt := fmt.Sprintf("prompt %d", i)
		_, err := mock.Complete(ctx, prompt)
		require.NoError(t, err)

		// Verify call count increments
		assert.Equal(t, i, mock.CallCount())
		// Verify last prompt is updated
		assert.Equal(t, prompt, mock.LastPrompt())
	}

	assert.Equal(t, 3, mock.CallCount())
	assert.Equal(t, "prompt 3", mock.LastPrompt())
}

// TestMockLLMClient_ValidResponseFormat tests that default response is parseable.
func TestMockLLMClient_ValidResponseFormat(t *testing.T) {
	ctx := context.Background()
	mock := newMockLLMClient()

	response, err := mock.Complete(ctx, "test prompt")
	require.NoError(t, err)

	// Verify the response can be parsed by parseConsolidatedMemory
	sourceIDs := []string{"mem-1", "mem-2"}
	memory, err := parseConsolidatedMemory(response, sourceIDs)
	require.NoError(t, err)
	assert.NotNil(t, memory)

	// Verify parsed fields
	assert.Equal(t, "Consolidated Memory Pattern", memory.Title)
	assert.Contains(t, memory.Content, "synthesized memory")
	assert.Equal(t, OutcomeSuccess, memory.Outcome)
	assert.Equal(t, []string{"consolidated", "pattern", "synthesis"}, memory.Tags)
	assert.Contains(t, memory.Description, "Synthesized from multiple source memories")
}

// TestMergeCluster_ValidCluster tests successful cluster merging with mock LLM.
func TestMergeCluster_ValidCluster(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()
	mockLLM := newMockLLMClient()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	projectID := "merge-test-project"

	// Create cluster with similar memories
	mem1, _ := NewMemory(projectID, "Go Error Pattern 1", "Always wrap errors", OutcomeSuccess, []string{"go", "errors"})
	mem1.Confidence = 0.8
	mem1.UsageCount = 5
	require.NoError(t, svc.Record(ctx, mem1))

	mem2, _ := NewMemory(projectID, "Go Error Pattern 2", "Use fmt.Errorf for wrapping", OutcomeSuccess, []string{"go", "errors"})
	mem2.Confidence = 0.9
	mem2.UsageCount = 10
	require.NoError(t, svc.Record(ctx, mem2))

	// Get vectors for centroid calculation
	vec1, err := svc.GetMemoryVectorByProjectID(ctx, projectID, mem1.ID)
	require.NoError(t, err)
	vec2, err := svc.GetMemoryVectorByProjectID(ctx, projectID, mem2.ID)
	require.NoError(t, err)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2},
		CentroidVector:    calculateCentroid([][]float32{vec1, vec2}),
		AverageSimilarity: 0.95,
		MinSimilarity:     0.92,
	}

	// Merge the cluster
	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	require.NoError(t, err)
	assert.NotNil(t, consolidatedMem)

	// Verify consolidated memory properties
	assert.Equal(t, projectID, consolidatedMem.ProjectID)
	assert.NotEmpty(t, consolidatedMem.ID)
	assert.Equal(t, "Consolidated Memory Pattern", consolidatedMem.Title)
	assert.Contains(t, consolidatedMem.Content, "synthesized memory")
	assert.Equal(t, OutcomeSuccess, consolidatedMem.Outcome)
	assert.Equal(t, []string{"consolidated", "pattern", "synthesis"}, consolidatedMem.Tags)

	// Verify source attribution is in description
	assert.Contains(t, consolidatedMem.Description, "Synthesized from multiple source memories")

	// Verify LLM was called
	assert.Equal(t, 1, mockLLM.CallCount())
	assert.NotEmpty(t, mockLLM.LastPrompt())
	assert.Contains(t, mockLLM.LastPrompt(), "Go Error Pattern 1")
	assert.Contains(t, mockLLM.LastPrompt(), "Go Error Pattern 2")
}

// TestMergeCluster_ConfidenceCalculation tests that merged confidence is calculated correctly.
func TestMergeCluster_ConfidenceCalculation(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()
	mockLLM := newMockLLMClient()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	projectID := "confidence-test-project"

	// Create memories with different confidences and usage counts
	// High confidence, high usage (should dominate)
	mem1, _ := NewMemory(projectID, "High confidence memory", "Content 1", OutcomeSuccess, []string{"test"})
	mem1.Confidence = 0.9
	mem1.UsageCount = 10
	require.NoError(t, svc.Record(ctx, mem1))

	// Low confidence, low usage (should contribute less)
	mem2, _ := NewMemory(projectID, "Low confidence memory", "Content 2", OutcomeSuccess, []string{"test"})
	mem2.Confidence = 0.5
	mem2.UsageCount = 1
	require.NoError(t, svc.Record(ctx, mem2))

	// Medium confidence, medium usage
	mem3, _ := NewMemory(projectID, "Medium confidence memory", "Content 3", OutcomeSuccess, []string{"test"})
	mem3.Confidence = 0.7
	mem3.UsageCount = 5
	require.NoError(t, svc.Record(ctx, mem3))

	vec1, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem1.ID)
	vec2, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem2.ID)
	vec3, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem3.ID)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2, mem3},
		CentroidVector:    calculateCentroid([][]float32{vec1, vec2, vec3}),
		AverageSimilarity: 0.85,
		MinSimilarity:     0.80,
	}

	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	require.NoError(t, err)
	assert.NotNil(t, consolidatedMem)

	// Calculate expected confidence: weighted average
	// weight1 = usageCount + 1 = 11, weight2 = 2, weight3 = 6
	// expectedConfidence = (0.9*11 + 0.5*2 + 0.7*6) / (11+2+6) = (9.9 + 1.0 + 4.2) / 19 = 15.1 / 19 ≈ 0.795
	expectedConfidence := (0.9*11.0 + 0.5*2.0 + 0.7*6.0) / (11.0 + 2.0 + 6.0)

	// Verify confidence is calculated correctly (weighted by usage count)
	assert.InDelta(t, expectedConfidence, consolidatedMem.Confidence, 0.001,
		"confidence should be weighted average based on usage counts")

	// Verify confidence is in valid range
	assert.GreaterOrEqual(t, consolidatedMem.Confidence, 0.0)
	assert.LessOrEqual(t, consolidatedMem.Confidence, 1.0)

	// High-usage, high-confidence memory should dominate
	// So result should be closer to 0.9 than to 0.5
	assert.Greater(t, consolidatedMem.Confidence, 0.7,
		"high-usage high-confidence memory should dominate the score")
}

// TestMergeCluster_MemoryLinking tests that source memories are linked to consolidated version.
func TestMergeCluster_MemoryLinking(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()
	mockLLM := newMockLLMClient()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	projectID := "linking-test-project"

	// Create source memories
	mem1, _ := NewMemory(projectID, "Source Memory 1", "Content 1", OutcomeSuccess, []string{"source"})
	mem1.Confidence = 0.8
	mem1.UsageCount = 3
	require.NoError(t, svc.Record(ctx, mem1))

	mem2, _ := NewMemory(projectID, "Source Memory 2", "Content 2", OutcomeSuccess, []string{"source"})
	mem2.Confidence = 0.85
	mem2.UsageCount = 5
	require.NoError(t, svc.Record(ctx, mem2))

	// Store original IDs before merging
	originalID1 := mem1.ID
	originalID2 := mem2.ID

	vec1, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem1.ID)
	vec2, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem2.ID)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2},
		CentroidVector:    calculateCentroid([][]float32{vec1, vec2}),
		AverageSimilarity: 0.90,
		MinSimilarity:     0.88,
	}

	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	require.NoError(t, err)
	assert.NotNil(t, consolidatedMem)

	// Retrieve source memories from storage to check linking
	updatedMem1, err := svc.GetByProjectID(ctx, projectID, originalID1)
	require.NoError(t, err)
	updatedMem2, err := svc.GetByProjectID(ctx, projectID, originalID2)
	require.NoError(t, err)

	// Verify source memories have ConsolidationID set
	assert.NotNil(t, updatedMem1.ConsolidationID, "source memory 1 should have consolidation ID")
	assert.NotNil(t, updatedMem2.ConsolidationID, "source memory 2 should have consolidation ID")

	// Verify ConsolidationID points to consolidated memory
	assert.Equal(t, consolidatedMem.ID, *updatedMem1.ConsolidationID,
		"source memory 1 should link to consolidated memory")
	assert.Equal(t, consolidatedMem.ID, *updatedMem2.ConsolidationID,
		"source memory 2 should link to consolidated memory")

	// Verify original content is preserved
	assert.Equal(t, "Source Memory 1", updatedMem1.Title)
	assert.Equal(t, "Content 1", updatedMem1.Content)
	assert.Equal(t, "Source Memory 2", updatedMem2.Title)
	assert.Equal(t, "Content 2", updatedMem2.Content)
}

// TestMergeCluster_SourceAttribution tests that source attribution is properly stored.
func TestMergeCluster_SourceAttribution(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	// Create mock LLM with custom response that includes specific attribution
	customResponse := `
TITLE: Consolidated API Error Handling

CONTENT:
Comprehensive error handling strategy combining multiple approaches.
Use structured errors with proper HTTP status codes.

TAGS: api, errors, go

OUTCOME: success

SOURCE_ATTRIBUTION:
Synthesized from 3 source memories covering authentication errors,
validation errors, and database error handling patterns.
`
	mockLLM := newMockLLMClientWithResponse(customResponse)

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	projectID := "attribution-test-project"

	// Create source memories
	mem1, _ := NewMemory(projectID, "Auth Errors", "Handle auth errors", OutcomeSuccess, []string{"auth"})
	mem2, _ := NewMemory(projectID, "Validation Errors", "Handle validation errors", OutcomeSuccess, []string{"validation"})
	mem3, _ := NewMemory(projectID, "DB Errors", "Handle database errors", OutcomeSuccess, []string{"database"})

	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))
	require.NoError(t, svc.Record(ctx, mem3))

	vec1, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem1.ID)
	vec2, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem2.ID)
	vec3, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem3.ID)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2, mem3},
		CentroidVector:    calculateCentroid([][]float32{vec1, vec2, vec3}),
		AverageSimilarity: 0.85,
		MinSimilarity:     0.80,
	}

	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	require.NoError(t, err)
	assert.NotNil(t, consolidatedMem)

	// Verify source attribution is stored in Description field
	assert.Contains(t, consolidatedMem.Description, "Synthesized from 3 source memories")
	assert.Contains(t, consolidatedMem.Description, "authentication errors")
	assert.Contains(t, consolidatedMem.Description, "validation errors")
	assert.Contains(t, consolidatedMem.Description, "database error handling patterns")

	// Verify title and content are also set correctly
	assert.Equal(t, "Consolidated API Error Handling", consolidatedMem.Title)
	assert.Contains(t, consolidatedMem.Content, "Comprehensive error handling strategy")
}

// TestMergeCluster_NilCluster tests error handling with nil cluster.
func TestMergeCluster_NilCluster(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	logger := zap.NewNop()
	mockLLM := newMockLLMClient()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	// Try to merge nil cluster
	consolidatedMem, err := distiller.MergeCluster(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, consolidatedMem)
	assert.Contains(t, err.Error(), "cluster cannot be nil")

	// Verify LLM was not called
	assert.Equal(t, 0, mockLLM.CallCount())
}

// TestMergeCluster_InsufficientMembers tests error handling with cluster < 2 members.
func TestMergeCluster_InsufficientMembers(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()
	mockLLM := newMockLLMClient()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	projectID := "insufficient-members-project"

	// Create cluster with only 1 member
	mem, _ := NewMemory(projectID, "Single Memory", "Content", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, mem))

	vec, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem.ID)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem},
		CentroidVector:    vec,
		AverageSimilarity: 0.0,
		MinSimilarity:     0.0,
	}

	// Try to merge cluster with insufficient members
	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	assert.Error(t, err)
	assert.Nil(t, consolidatedMem)
	assert.Contains(t, err.Error(), "cluster must have at least 2 members")

	// Verify LLM was not called
	assert.Equal(t, 0, mockLLM.CallCount())
}

// TestMergeCluster_NoLLMClient tests error handling when LLM client is not configured.
func TestMergeCluster_NoLLMClient(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	// Create distiller WITHOUT LLM client
	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	projectID := "no-llm-project"

	// Create cluster
	mem1, _ := NewMemory(projectID, "Memory 1", "Content 1", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Memory 2", "Content 2", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	vec1, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem1.ID)
	vec2, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem2.ID)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2},
		CentroidVector:    calculateCentroid([][]float32{vec1, vec2}),
		AverageSimilarity: 0.90,
		MinSimilarity:     0.85,
	}

	// Try to merge without LLM client
	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	assert.Error(t, err)
	assert.Nil(t, consolidatedMem)
	assert.Contains(t, err.Error(), "LLM client not configured")
}

// TestMergeCluster_LLMError tests error handling when LLM call fails.
func TestMergeCluster_LLMError(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	// Create mock LLM that returns an error
	llmError := fmt.Errorf("LLM API error: rate limit exceeded")
	mockLLM := newMockLLMClientWithError(llmError)

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	projectID := "llm-error-project"

	// Create cluster
	mem1, _ := NewMemory(projectID, "Memory 1", "Content 1", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Memory 2", "Content 2", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	vec1, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem1.ID)
	vec2, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem2.ID)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2},
		CentroidVector:    calculateCentroid([][]float32{vec1, vec2}),
		AverageSimilarity: 0.90,
		MinSimilarity:     0.85,
	}

	// Try to merge - should fail with LLM error
	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	assert.Error(t, err)
	assert.Nil(t, consolidatedMem)
	assert.Contains(t, err.Error(), "LLM synthesis failed")
	assert.Contains(t, err.Error(), "rate limit exceeded")

	// Verify LLM was called
	assert.Equal(t, 1, mockLLM.CallCount())
}

// TestMergeCluster_InvalidLLMResponse tests error handling with malformed LLM response.
func TestMergeCluster_InvalidLLMResponse(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(10)
	logger := zap.NewNop()

	// Create mock LLM with invalid response (missing required fields)
	invalidResponse := `
TITLE: Incomplete Response

CONTENT:
This response is missing the OUTCOME field.
`
	mockLLM := newMockLLMClientWithResponse(invalidResponse)

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	projectID := "invalid-response-project"

	// Create cluster
	mem1, _ := NewMemory(projectID, "Memory 1", "Content 1", OutcomeSuccess, []string{"test"})
	mem2, _ := NewMemory(projectID, "Memory 2", "Content 2", OutcomeSuccess, []string{"test"})
	require.NoError(t, svc.Record(ctx, mem1))
	require.NoError(t, svc.Record(ctx, mem2))

	vec1, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem1.ID)
	vec2, _ := svc.GetMemoryVectorByProjectID(ctx, projectID, mem2.ID)

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2},
		CentroidVector:    calculateCentroid([][]float32{vec1, vec2}),
		AverageSimilarity: 0.90,
		MinSimilarity:     0.85,
	}

	// Try to merge - should fail during parsing
	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	assert.Error(t, err)
	assert.Nil(t, consolidatedMem)
	assert.Contains(t, err.Error(), "parsing LLM response")
	assert.Contains(t, err.Error(), "OUTCOME field is required")

	// Verify LLM was called
	assert.Equal(t, 1, mockLLM.CallCount())
}

// TestMergeCluster_EmptyProjectID tests error handling with empty project ID.
func TestMergeCluster_EmptyProjectID(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	logger := zap.NewNop()
	mockLLM := newMockLLMClient()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger, WithLLMClient(mockLLM))
	require.NoError(t, err)

	// Create cluster with memories that have empty project ID
	mem1 := &Memory{
		ID:         "mem-1",
		ProjectID:  "", // Empty project ID
		Title:      "Memory 1",
		Content:    "Content 1",
		Outcome:    OutcomeSuccess,
		Confidence: 0.8,
	}
	mem2 := &Memory{
		ID:         "mem-2",
		ProjectID:  "", // Empty project ID
		Title:      "Memory 2",
		Content:    "Content 2",
		Outcome:    OutcomeSuccess,
		Confidence: 0.8,
	}

	cluster := &SimilarityCluster{
		Members:           []*Memory{mem1, mem2},
		CentroidVector:    make([]float32, 10),
		AverageSimilarity: 0.90,
		MinSimilarity:     0.85,
	}

	// Try to merge - should fail with empty project ID error
	consolidatedMem, err := distiller.MergeCluster(ctx, cluster)
	assert.Error(t, err)
	assert.Nil(t, consolidatedMem)
	assert.Contains(t, err.Error(), "project ID cannot be empty")

	// Verify LLM was not called
	assert.Equal(t, 0, mockLLM.CallCount())
}

// TestCalculateMergedConfidence tests the confidence calculation helper function.
func TestCalculateMergedConfidence(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	distiller, err := NewDistiller(svc, logger)
	require.NoError(t, err)

	testCases := []struct {
		name               string
		memories           []*Memory
		expectedConfidence float64
		description        string
	}{
		{
			name: "equal weights",
			memories: []*Memory{
				{Confidence: 0.8, UsageCount: 0},
				{Confidence: 0.6, UsageCount: 0},
			},
			// Both have weight 1 (usageCount+1): (0.8*1 + 0.6*1) / 2 = 0.7
			expectedConfidence: 0.7,
			description:        "equal usage should average confidences",
		},
		{
			name: "weighted by usage",
			memories: []*Memory{
				{Confidence: 0.9, UsageCount: 10}, // weight 11
				{Confidence: 0.5, UsageCount: 0},  // weight 1
			},
			// (0.9*11 + 0.5*1) / 12 = 10.4 / 12 = 0.8666...
			expectedConfidence: 0.8666666666666667,
			description:        "high usage should dominate",
		},
		{
			name: "single memory",
			memories: []*Memory{
				{Confidence: 0.75, UsageCount: 5},
			},
			expectedConfidence: 0.75,
			description:        "single memory should return its confidence",
		},
		{
			name:               "empty slice",
			memories:           []*Memory{},
			expectedConfidence: DistilledConfidence,
			description:        "empty slice should return default",
		},
		{
			name: "multiple memories with varying usage",
			memories: []*Memory{
				{Confidence: 0.9, UsageCount: 10}, // weight 11
				{Confidence: 0.7, UsageCount: 5},  // weight 6
				{Confidence: 0.5, UsageCount: 1},  // weight 2
			},
			// (0.9*11 + 0.7*6 + 0.5*2) / (11+6+2) = (9.9 + 4.2 + 1.0) / 19 = 15.1 / 19 = 0.794736...
			expectedConfidence: 0.7947368421052632,
			description:        "multiple memories should use weighted average",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confidence := distiller.calculateMergedConfidence(tc.memories)
			assert.InDelta(t, tc.expectedConfidence, confidence, 0.0001,
				"%s: got %.4f, expected %.4f", tc.description, confidence, tc.expectedConfidence)

			// Verify confidence is in valid range
			assert.GreaterOrEqual(t, confidence, 0.0, "confidence should be >= 0.0")
			assert.LessOrEqual(t, confidence, 1.0, "confidence should be <= 1.0")
		})
	}
}

// TestCalculateConsolidatedConfidence tests the calculateConsolidatedConfidence function.
func TestCalculateConsolidatedConfidence(t *testing.T) {
	testCases := []struct {
		name               string
		memories           []*Memory
		expectedMin        float64 // minimum expected confidence
		expectedMax        float64 // maximum expected confidence
		description        string
	}{
		{
			name:               "empty slice",
			memories:           []*Memory{},
			expectedMin:        DistilledConfidence,
			expectedMax:        DistilledConfidence,
			description:        "empty slice should return default",
		},
		{
			name: "single memory",
			memories: []*Memory{
				{Confidence: 0.75, UsageCount: 5},
			},
			expectedMin: 0.75,
			expectedMax: 0.75,
			description: "single memory should return its confidence (no consensus bonus)",
		},
		{
			name: "perfect consensus - two memories",
			memories: []*Memory{
				{Confidence: 0.8, UsageCount: 0},
				{Confidence: 0.8, UsageCount: 0},
			},
			// Base: 0.8, Consensus bonus: (1.0 - 0.0) * (2/10) * 0.1 = 1.0 * 0.2 * 0.1 = 0.02
			// Final: 0.8 + 0.02 = 0.82
			expectedMin: 0.819,
			expectedMax: 0.821,
			description: "perfect consensus with 2 memories should add small bonus",
		},
		{
			name: "perfect consensus - ten memories",
			memories: []*Memory{
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
			},
			// Base: 0.9, Consensus bonus: (1.0 - 0.0) * (10/10) * 0.1 = 1.0 * 1.0 * 0.1 = 0.1
			// Final: 0.9 + 0.1 = 1.0
			expectedMin: 0.999,
			expectedMax: 1.0,
			description: "perfect consensus with 10 memories should give maximum bonus",
		},
		{
			name: "high consensus - similar confidences",
			memories: []*Memory{
				{Confidence: 0.8, UsageCount: 0},
				{Confidence: 0.82, UsageCount: 0},
				{Confidence: 0.79, UsageCount: 0},
				{Confidence: 0.81, UsageCount: 0},
			},
			// Base: (0.8 + 0.82 + 0.79 + 0.81) / 4 = 0.805
			// Small variance, so consensus bonus should be significant
			expectedMin: 0.81,
			expectedMax: 0.84,
			description: "high consensus (low variance) should add noticeable bonus",
		},
		{
			name: "low consensus - divergent confidences",
			memories: []*Memory{
				{Confidence: 0.2, UsageCount: 0},
				{Confidence: 0.9, UsageCount: 0},
				{Confidence: 0.5, UsageCount: 0},
			},
			// Base: (0.2 + 0.9 + 0.5) / 3 = 0.533
			// High variance, so consensus bonus should be minimal
			expectedMin: 0.53,
			expectedMax: 0.56,
			description: "low consensus (high variance) should add minimal bonus",
		},
		{
			name: "weighted by usage - equal confidence",
			memories: []*Memory{
				{Confidence: 0.8, UsageCount: 10}, // weight 11
				{Confidence: 0.8, UsageCount: 0},  // weight 1
			},
			// Base: (0.8*11 + 0.8*1) / 12 = 0.8
			// Perfect consensus bonus applies: (1.0 - 0.0) * (2/10) * 0.1 = 0.02
			// Final: 0.8 + 0.02 = 0.82
			expectedMin: 0.819,
			expectedMax: 0.821,
			description: "weighted calculation with perfect consensus",
		},
		{
			name: "weighted by usage - different confidence",
			memories: []*Memory{
				{Confidence: 0.9, UsageCount: 10}, // weight 11
				{Confidence: 0.5, UsageCount: 0},  // weight 1
			},
			// Base: (0.9*11 + 0.5*1) / 12 = 10.4 / 12 = 0.8666...
			// High variance (0.9 vs 0.5), minimal consensus bonus
			expectedMin: 0.86,
			expectedMax: 0.88,
			description: "high usage should dominate, low consensus gives small bonus",
		},
		{
			name: "all zeros",
			memories: []*Memory{
				{Confidence: 0.0, UsageCount: 0},
				{Confidence: 0.0, UsageCount: 0},
				{Confidence: 0.0, UsageCount: 0},
			},
			// Base: 0.0, Consensus bonus: (1.0 - 0.0) * (3/10) * 0.1 = 0.03
			// Final: 0.0 + 0.03 = 0.03
			expectedMin: 0.029,
			expectedMax: 0.031,
			description: "all zeros with perfect consensus should add bonus",
		},
		{
			name: "near max - should clamp at 1.0",
			memories: []*Memory{
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
				{Confidence: 0.95, UsageCount: 0},
			},
			// Base: 0.95, Consensus bonus: 0.1, Final: 1.05 -> clamped to 1.0
			expectedMin: 1.0,
			expectedMax: 1.0,
			description: "should clamp at 1.0",
		},
		{
			name: "mixed usage and confidence",
			memories: []*Memory{
				{Confidence: 0.85, UsageCount: 8},
				{Confidence: 0.88, UsageCount: 5},
				{Confidence: 0.83, UsageCount: 12},
				{Confidence: 0.86, UsageCount: 3},
				{Confidence: 0.87, UsageCount: 6},
			},
			// Base is weighted average, variance is relatively low
			// Should get a decent consensus bonus
			expectedMin: 0.85,
			expectedMax: 0.91,
			description: "real-world scenario with mixed usage and similar confidences",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confidence := calculateConsolidatedConfidence(tc.memories)

			// Check if within expected range
			assert.GreaterOrEqual(t, confidence, tc.expectedMin,
				"%s: got %.4f, expected >= %.4f", tc.description, confidence, tc.expectedMin)
			assert.LessOrEqual(t, confidence, tc.expectedMax,
				"%s: got %.4f, expected <= %.4f", tc.description, confidence, tc.expectedMax)

			// Verify confidence is in valid range [0.0, 1.0]
			assert.GreaterOrEqual(t, confidence, 0.0, "confidence should be >= 0.0")
			assert.LessOrEqual(t, confidence, 1.0, "confidence should be <= 1.0")

			// For non-empty slices, verify consensus bonus is applied correctly
			if len(tc.memories) > 1 {
				// Calculate base confidence (weighted average)
				var weightedSum float64
				var totalWeight float64
				for _, mem := range tc.memories {
					weight := float64(mem.UsageCount + 1)
					weightedSum += mem.Confidence * weight
					totalWeight += weight
				}
				baseConfidence := weightedSum / totalWeight

				// Confidence with bonus should be >= base (unless clamped)
				if baseConfidence <= 0.9 {
					assert.GreaterOrEqual(t, confidence, baseConfidence,
						"consensus bonus should increase or maintain confidence")
				}
			}
		})
	}
}

// TestCalculateConsolidatedConfidence_ConsensusBonus verifies consensus bonus calculation.
func TestCalculateConsolidatedConfidence_ConsensusBonus(t *testing.T) {
	// Test that consensus bonus increases with:
	// 1. Lower variance (higher consensus)
	// 2. More sources

	// Same base confidence (0.8), varying consensus
	perfectConsensus := []*Memory{
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
	}

	moderateConsensus := []*Memory{
		{Confidence: 0.75, UsageCount: 0},
		{Confidence: 0.85, UsageCount: 0},
	}

	lowConsensus := []*Memory{
		{Confidence: 0.6, UsageCount: 0},
		{Confidence: 1.0, UsageCount: 0},
	}

	perfectConf := calculateConsolidatedConfidence(perfectConsensus)
	moderateConf := calculateConsolidatedConfidence(moderateConsensus)
	lowConf := calculateConsolidatedConfidence(lowConsensus)

	// Perfect consensus should have highest confidence
	assert.Greater(t, perfectConf, moderateConf,
		"perfect consensus should yield higher confidence than moderate")
	assert.Greater(t, moderateConf, lowConf,
		"moderate consensus should yield higher confidence than low")

	// Test that more sources increase bonus (with same variance)
	twoSources := []*Memory{
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
	}

	fiveSources := []*Memory{
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
	}

	tenSources := []*Memory{
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
		{Confidence: 0.8, UsageCount: 0},
	}

	twoConf := calculateConsolidatedConfidence(twoSources)
	fiveConf := calculateConsolidatedConfidence(fiveSources)
	tenConf := calculateConsolidatedConfidence(tenSources)

	// More sources should increase confidence (with perfect consensus)
	assert.Greater(t, fiveConf, twoConf,
		"5 agreeing sources should yield higher confidence than 2")
	assert.Greater(t, tenConf, fiveConf,
		"10 agreeing sources should yield higher confidence than 5")
}

// TestClampConfidence tests the clampConfidence helper function.
func TestClampConfidence(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"below minimum", -0.5, 0.0},
		{"at minimum", 0.0, 0.0},
		{"normal value", 0.5, 0.5},
		{"at maximum", 1.0, 1.0},
		{"above maximum", 1.5, 1.0},
		{"way below", -100.0, 0.0},
		{"way above", 100.0, 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := clampConfidence(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
