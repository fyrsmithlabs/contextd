package compression

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractiveCompressor_SelectSentencesBug reproduces the bug where
// selectSentences returns empty results when sentences are longer than targetLength
func TestExtractiveCompressor_SelectSentencesBug(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmExtractive,
		TargetRatio:      2.0,
	}
	compressor := NewExtractiveCompressor(config)

	// Short code content where sentences might be longer than target
	shortCode := `package main
import "fmt"
func main() { fmt.Println("test") }`

	ctx := context.Background()
	result, err := compressor.Compress(ctx, shortCode, AlgorithmExtractive, 2.0)
	require.NoError(t, err)

	// BUG: This should NOT be empty
	assert.NotEmpty(t, result.Content, "compressed content should not be empty")
	assert.Greater(t, result.QualityScore, 0.0, "quality score should be > 0")
	assert.Greater(t, result.Metadata.CompressionRatio, 0.0, "compression ratio should be > 0")
	// Check compression ratio is finite (not +Inf from division by zero)
	assert.False(t, result.Metadata.CompressedSize == 0, "compressed size should not be zero")
	assert.False(t, math.IsInf(result.Metadata.CompressionRatio, 0), "compression ratio should not be +Inf")
}

// TestExtractiveCompressor_SelectSentencesSkipsLongIncludesShort verifies that
// when a high-scoring sentence is too long, smaller sentences are still considered.
// This tests the specific bug fix where the loop continued instead of breaking.
func TestExtractiveCompressor_SelectSentencesSkipsLongIncludesShort(t *testing.T) {
	compressor := NewExtractiveCompressor(Config{
		DefaultAlgorithm: AlgorithmExtractive,
		TargetRatio:      2.0,
	})

	// Test the internal selectSentences function directly
	// First sentence is long and high-scoring (position bonus)
	// Second sentence is short enough to fit
	sentences := []string{
		"This is a very long first sentence that should have a high position score but will not fit in the target length because it exceeds the limit significantly.",
		"Short one.",
	}

	// Score the sentences - first will have higher position bonus
	scores := compressor.scoreSentences(sentences)

	// Target length that only fits the short sentence
	targetLength := 20

	selected := compressor.selectSentences(sentences, scores, targetLength)

	// The short sentence should be selected even though the long one scored higher
	assert.NotEmpty(t, selected, "should select at least one sentence")
	assert.Contains(t, selected, "Short one.", "should include the shorter sentence that fits")
}

// TestExtractiveCompressor_SelectSentencesMinimum verifies at least one sentence is always selected
func TestExtractiveCompressor_SelectSentencesMinimum(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmExtractive,
		TargetRatio:      5.0, // Very aggressive compression
	}
	compressor := NewExtractiveCompressor(config)

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "short code",
			content: `package main\nimport "fmt"\nfunc main() { fmt.Println("test") }`,
		},
		{
			name:    "single sentence",
			content: "This is a single sentence that is reasonably long.",
		},
		{
			name:    "very short",
			content: "Hello!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := compressor.Compress(ctx, tt.content, AlgorithmExtractive, 5.0)
			require.NoError(t, err)

			// Should always have SOME content
			assert.NotEmpty(t, result.Content, "should never return completely empty content")
			assert.Greater(t, len(result.Content), 0, "content length should be > 0")

			// Quality score should be reasonable
			assert.GreaterOrEqual(t, result.QualityScore, 0.0)
			assert.LessOrEqual(t, result.QualityScore, 1.0)

			// Compression ratio should not be infinity
			assert.False(t, result.Metadata.CompressedSize == 0, "compressed size should not be zero")
			assert.False(t, math.IsInf(result.Metadata.CompressionRatio, 0), "compression ratio should not be infinity")
		})
	}
}
