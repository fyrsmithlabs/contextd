package reasoningbank

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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
