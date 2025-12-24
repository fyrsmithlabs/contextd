package compression

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Compress_Extractive(t *testing.T) {
	config := Config{
		DefaultAlgorithm:  AlgorithmExtractive,
		TargetRatio:       2.0,
		QualityThreshold:  0.5,
		MaxProcessingTime: time.Second * 5,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	content := "This is a test document. It contains multiple sentences. Each sentence has some content. The compression algorithm should work on this text."

	result, err := service.Compress(context.Background(), content, AlgorithmExtractive, 2.0)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Content)
	assert.True(t, len(result.Content) < len(content))
	assert.Equal(t, AlgorithmExtractive, Algorithm(result.Metadata.Algorithm))
	assert.Equal(t, "folded", string(result.Metadata.Level))
	assert.True(t, result.Metadata.CompressionRatio > 1.0)
	assert.True(t, result.QualityScore >= 0.0 && result.QualityScore <= 1.0)
	assert.NotNil(t, result.Metadata.CompressedAt)
}

func TestService_Compress_Abstractive(t *testing.T) {
	// Skip if no API key available
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping abstractive compression test")
	}

	config := Config{
		DefaultAlgorithm:  AlgorithmAbstractive,
		TargetRatio:       3.0,
		QualityThreshold:  0.3,
		MaxProcessingTime: time.Second * 30,
		AnthropicAPIKey:   apiKey,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	content := "This is a longer test document. It contains multiple sentences with various content. The abstractive compression algorithm should reduce this text significantly. This approach uses different techniques compared to extractive methods."

	result, err := service.Compress(context.Background(), content, AlgorithmAbstractive, 3.0)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Content)
	assert.Equal(t, AlgorithmAbstractive, Algorithm(result.Metadata.Algorithm))
	assert.Equal(t, "summary", string(result.Metadata.Level))
	assert.True(t, result.Metadata.CompressionRatio >= 1.0)
	assert.True(t, result.QualityScore >= 0.0 && result.QualityScore <= 1.0)
}

func TestService_Compress_Hybrid(t *testing.T) {
	// Skip if no API key available (hybrid uses abstractive internally)
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping hybrid compression test")
	}

	config := Config{
		DefaultAlgorithm:  AlgorithmHybrid,
		TargetRatio:       2.5,
		QualityThreshold:  0.4,
		MaxProcessingTime: time.Second * 30,
		AnthropicAPIKey:   apiKey,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	content := "This is a comprehensive test document. It has many sentences with different types of content. The hybrid compression combines extractive and abstractive approaches. This should provide good compression ratios with reasonable quality."

	result, err := service.Compress(context.Background(), content, AlgorithmHybrid, 2.5)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Content)
	assert.Equal(t, AlgorithmHybrid, Algorithm(result.Metadata.Algorithm))
	assert.Equal(t, "summary", string(result.Metadata.Level))
	assert.True(t, result.Metadata.CompressionRatio >= 1.0)
	assert.True(t, result.QualityScore >= 0.0 && result.QualityScore <= 1.0)
}

func TestService_Compress_Validation(t *testing.T) {
	config := Config{}
	service, err := NewService(config)
	require.NoError(t, err)

	// Test empty content
	_, err = service.Compress(context.Background(), "", AlgorithmExtractive, 2.0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be empty")

	// Test invalid target ratio
	_, err = service.Compress(context.Background(), "test content", AlgorithmExtractive, 0.5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target ratio must be greater than 1.0")
}

func TestService_GetCapabilities(t *testing.T) {
	config := Config{}
	service, err := NewService(config)
	require.NoError(t, err)

	caps := service.GetCapabilities(context.Background())

	assert.Len(t, caps, 3)
	assert.Contains(t, caps, AlgorithmExtractive)
	assert.Contains(t, caps, AlgorithmAbstractive)
	assert.Contains(t, caps, AlgorithmHybrid)

	extractiveCaps := caps[AlgorithmExtractive]
	assert.Contains(t, extractiveCaps.SupportedAlgorithms, AlgorithmExtractive)
	assert.True(t, extractiveCaps.MaxContentLength > 0)
	assert.True(t, extractiveCaps.SupportsTargetRatio)
}
