package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockAbstractiveCompressor_BasicFunctionality verifies the mock works correctly
func TestMockAbstractiveCompressor_BasicFunctionality(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmAbstractive,
		TargetRatio:      2.0,
	}
	mock := NewMockAbstractiveCompressor(config)
	ctx := context.Background()

	content := `This is a test document with multiple sentences.
It contains several paragraphs of text.
The mock compressor should reduce the size of this content.
It simulates what an LLM-based abstractive compressor would do.
The compression should preserve key information.
While removing less important details.
This helps test the hybrid compressor without requiring an API key.`

	result, err := mock.Compress(ctx, content, AlgorithmAbstractive, 2.0)
	require.NoError(t, err)

	// Verify basic properties
	assert.NotEmpty(t, result.Content)
	assert.True(t, len(result.Content) < len(content), "compressed content should be shorter")
	assert.True(t, result.Metadata.CompressionRatio >= 1.0, "compression ratio should be >= 1.0")
	assert.True(t, result.QualityScore >= 0.4 && result.QualityScore <= 0.7, "quality score should be in expected range")
	assert.Equal(t, string(AlgorithmAbstractive), result.Metadata.Algorithm)
}

// TestMockAbstractiveCompressor_ShortContent verifies short content handling
func TestMockAbstractiveCompressor_ShortContent(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmAbstractive,
		TargetRatio:      2.0,
	}
	mock := NewMockAbstractiveCompressor(config)
	ctx := context.Background()

	shortContent := "This is very short."
	result, err := mock.Compress(ctx, shortContent, AlgorithmAbstractive, 2.0)
	require.NoError(t, err)

	// Short content should be returned as-is
	assert.Equal(t, shortContent, result.Content)
	assert.Equal(t, 1.0, result.Metadata.CompressionRatio)
	assert.Equal(t, 1.0, result.QualityScore)
}

// TestMockAbstractiveCompressor_DifferentTargetRatios verifies target ratio behavior
func TestMockAbstractiveCompressor_DifferentTargetRatios(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmAbstractive,
	}
	mock := NewMockAbstractiveCompressor(config)
	ctx := context.Background()

	content := strings.Repeat("This is a sentence. ", 50) // 1000 chars

	tests := []struct {
		name        string
		targetRatio float64
		minRatio    float64
		maxRatio    float64
	}{
		{
			name:        "low compression (1.5x)",
			targetRatio: 1.5,
			minRatio:    1.3,
			maxRatio:    2.0,
		},
		{
			name:        "medium compression (2.0x)",
			targetRatio: 2.0,
			minRatio:    1.5,
			maxRatio:    3.0,
		},
		{
			name:        "high compression (3.0x)",
			targetRatio: 3.0,
			minRatio:    2.0,
			maxRatio:    4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mock.Compress(ctx, content, AlgorithmAbstractive, tt.targetRatio)
			require.NoError(t, err)

			assert.True(t, result.Metadata.CompressionRatio >= tt.minRatio,
				"compression ratio %.2f should be >= %.2f",
				result.Metadata.CompressionRatio, tt.minRatio)
		})
	}
}

// TestMockAbstractiveCompressor_GetCapabilities verifies capabilities
func TestMockAbstractiveCompressor_GetCapabilities(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmAbstractive,
	}
	mock := NewMockAbstractiveCompressor(config)
	ctx := context.Background()

	caps := mock.GetCapabilities(ctx)

	assert.Contains(t, caps.SupportedAlgorithms, AlgorithmAbstractive)
	assert.True(t, caps.SupportsTargetRatio)
	assert.Equal(t, 50000, caps.MaxContentLength)
	assert.Equal(t, 0.4, caps.QualityScoreRange.Min)
	assert.Equal(t, 0.7, caps.QualityScoreRange.Max)
}

// TestMockAbstractiveCompressor_Deterministic verifies consistent results
func TestMockAbstractiveCompressor_Deterministic(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmAbstractive,
		TargetRatio:      2.0,
	}
	mock := NewMockAbstractiveCompressor(config)
	ctx := context.Background()

	content := `First sentence. Second sentence. Third sentence. Fourth sentence. Fifth sentence.`

	// Run compression twice
	result1, err1 := mock.Compress(ctx, content, AlgorithmAbstractive, 2.0)
	require.NoError(t, err1)

	result2, err2 := mock.Compress(ctx, content, AlgorithmAbstractive, 2.0)
	require.NoError(t, err2)

	// Results should be identical (deterministic)
	assert.Equal(t, result1.Content, result2.Content)
	assert.Equal(t, result1.Metadata.CompressionRatio, result2.Metadata.CompressionRatio)
}
