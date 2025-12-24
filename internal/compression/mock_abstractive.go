package compression

import (
	"context"
	"strings"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// MockAbstractiveCompressor implements a mock abstractive compressor for testing
// It simulates abstractive compression by applying simple text reduction rules
// without requiring an actual Anthropic API key.
type MockAbstractiveCompressor struct {
	config Config
}

// NewMockAbstractiveCompressor creates a new mock abstractive compressor
func NewMockAbstractiveCompressor(config Config) *MockAbstractiveCompressor {
	return &MockAbstractiveCompressor{
		config: config,
	}
}

// Compress implements the Compressor interface with mock abstractive compression
// It simulates API-based compression by applying deterministic reduction rules
func (m *MockAbstractiveCompressor) Compress(ctx context.Context, content string, algorithm Algorithm, targetRatio float64) (*Result, error) {
	start := time.Now()

	// For very short content, return as-is
	if len(content) < 100 {
		return &Result{
			Content:        content,
			ProcessingTime: time.Since(start),
			QualityScore:   1.0,
			Metadata: vectorstore.CompressionMetadata{
				Level:            vectorstore.CompressionLevelSummary,
				Algorithm:        string(algorithm),
				OriginalSize:     len(content),
				CompressedSize:   len(content),
				CompressionRatio: 1.0,
				CompressedAt:     &start,
			},
		}, nil
	}

	// Simulate abstractive compression by extracting key sentences
	// and removing filler words to achieve target ratio
	compressedContent := m.simulateAbstractiveSummary(content, targetRatio)

	// Calculate metrics
	originalSize := len(content)
	compressedSize := len(compressedContent)
	compressionRatio := float64(originalSize) / float64(compressedSize)
	if compressedSize == 0 {
		compressionRatio = 1.0
	}

	// Quality score based on how well we met the target
	// Mock achieves reasonable quality (0.5-0.7 range)
	ratioAchievement := compressionRatio / targetRatio
	if ratioAchievement > 1.0 {
		ratioAchievement = 1.0
	}
	qualityScore := 0.4 + (ratioAchievement * 0.3) // Range 0.4-0.7

	return &Result{
		Content:        compressedContent,
		ProcessingTime: time.Since(start),
		QualityScore:   qualityScore,
		Metadata: vectorstore.CompressionMetadata{
			Level:            vectorstore.CompressionLevelSummary,
			Algorithm:        string(algorithm),
			OriginalSize:     originalSize,
			CompressedSize:   compressedSize,
			CompressionRatio: compressionRatio,
			CompressedAt:     &start,
		},
	}, nil
}

// simulateAbstractiveSummary creates a mock summary that simulates what an LLM would produce
func (m *MockAbstractiveCompressor) simulateAbstractiveSummary(content string, targetRatio float64) string {
	// Split into sentences
	sentences := strings.Split(content, ".")
	if len(sentences) == 0 {
		return content
	}

	// Calculate target length with some buffer to avoid over-compression
	// Add 10% buffer to ensure we don't compress too aggressively
	targetLength := int(float64(len(content)) / (targetRatio * 0.9))
	if targetLength < 50 {
		targetLength = 50 // Minimum meaningful summary
	}

	var summary strings.Builder
	currentLength := 0

	// Select sentences until we reach target length
	// Prioritize earlier sentences (simulates extracting key information)
	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if trimmed == "" {
			continue
		}

		// Add period back
		if !strings.HasSuffix(trimmed, ".") && !strings.HasSuffix(trimmed, "!") && !strings.HasSuffix(trimmed, "?") {
			trimmed += "."
		}

		// Check if adding this sentence would exceed target
		sentenceLen := len(trimmed) + 1 // +1 for space
		if currentLength+sentenceLen > targetLength && currentLength > 0 {
			break
		}

		if summary.Len() > 0 {
			summary.WriteString(" ")
		}
		summary.WriteString(trimmed)
		currentLength += sentenceLen
	}

	result := summary.String()
	if result == "" {
		// If no sentences were selected, take first N characters
		if len(content) > targetLength {
			result = content[:targetLength]
		} else {
			result = content
		}
	}

	// Don't apply filler removal for mock to keep lengths more predictable
	// Real LLM would condense but also rephrase, which maintains similar lengths

	return result
}

// GetCapabilities returns the capabilities of this mock compressor
func (m *MockAbstractiveCompressor) GetCapabilities(ctx context.Context) Capabilities {
	return Capabilities{
		SupportedAlgorithms: []Algorithm{AlgorithmAbstractive},
		MaxContentLength:    50000, // Match real abstractive compressor
		SupportsTargetRatio: true,
		QualityScoreRange: struct {
			Min float64
			Max float64
		}{
			Min: 0.4,
			Max: 0.7,
		},
	}
}
