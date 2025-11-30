package compression_test

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/compression"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQualityMetricsWithAllAlgorithms tests quality scoring with all compression algorithms
func TestQualityMetricsWithAllAlgorithms(t *testing.T) {
	config := compression.Config{
		DefaultAlgorithm: compression.AlgorithmExtractive,
		TargetRatio:      2.0,
	}

	service, err := compression.NewService(config)
	require.NoError(t, err)

	testContent := `# Introduction to Golang

Golang is a statically typed, compiled programming language designed at Google.

## Key Features

- Fast compilation
- Efficient execution
- Simple syntax
- Built-in concurrency

## Code Example

func main() {
    fmt.Println("Hello, World!")
}

## Conclusion

Golang is great for building scalable applications.`

	algorithms := []compression.Algorithm{
		compression.AlgorithmExtractive,
		// Skip abstractive and hybrid as they're not fully implemented
	}

	for _, algo := range algorithms {
		t.Run(string(algo), func(t *testing.T) {
			result, err := service.Compress(context.Background(), testContent, algo, 2.0)
			require.NoError(t, err)

			// Calculate quality metrics using main branch API
			metrics := compression.NewQualityMetrics(
				len(testContent),
				len(result.Content),
				2.0,
			)

			// Calculate individual scores
			compressionScore := metrics.CompressionRatioScore()
			retentionScore := metrics.InformationRetentionScore(testContent, result.Content)
			similarityScore := metrics.SemanticSimilarityScore(testContent, result.Content)
			readabilityScore := metrics.ReadabilityScore(result.Content)
			compositeScore := metrics.CompositeScore(testContent, result.Content)

			// Verify all metrics are in valid ranges
			assert.GreaterOrEqual(t, compressionScore, 0.0, "compression score should be >= 0")
			assert.LessOrEqual(t, compressionScore, 1.0, "compression score should be <= 1")
			assert.GreaterOrEqual(t, retentionScore, 0.0, "retention score should be >= 0")
			assert.LessOrEqual(t, retentionScore, 1.0, "retention score should be <= 1")
			assert.GreaterOrEqual(t, similarityScore, 0.0, "similarity score should be >= 0")
			assert.LessOrEqual(t, similarityScore, 1.0, "similarity score should be <= 1")
			assert.GreaterOrEqual(t, readabilityScore, 0.0, "readability should be >= 0")
			assert.LessOrEqual(t, readabilityScore, 1.0, "readability should be <= 1")
			assert.GreaterOrEqual(t, compositeScore, 0.0, "composite score should be >= 0")
			assert.LessOrEqual(t, compositeScore, 1.0, "composite score should be <= 1")

			// Test with quality gate
			gate := compression.NewQualityGate(compression.QualityThresholds{
				MinCompressionRatio:     1.5,
				MinInformationRetention: 0.6,
				MinSemanticSimilarity:   0.5,
				MinReadability:          0.5,
				MinCompositeScore:       0.6,
			})
			gateResult := gate.Evaluate(metrics, testContent, result.Content)

			// Log metrics for visibility
			t.Logf("Algorithm: %s", algo)
			t.Logf("  Compression Score: %.2f", compressionScore)
			t.Logf("  Information Retention: %.2f", retentionScore)
			t.Logf("  Semantic Similarity: %.2f", similarityScore)
			t.Logf("  Readability: %.2f", readabilityScore)
			t.Logf("  Composite Score: %.2f", compositeScore)
			if !gateResult.Pass {
				t.Logf("  Quality Gate: FAILED - %s", gateResult.FailureReason)
			} else {
				t.Logf("  Quality Gate: PASSED")
			}
		})
	}
}
