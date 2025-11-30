package compression

import (
	"context"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// Algorithm represents a compression algorithm
type Algorithm string

const (
	// AlgorithmExtractive uses extractive summarization (sentence selection)
	AlgorithmExtractive Algorithm = "extractive"
	// AlgorithmAbstractive uses abstractive summarization (content generation)
	AlgorithmAbstractive Algorithm = "abstractive"
	// AlgorithmHybrid combines extractive and abstractive approaches
	AlgorithmHybrid Algorithm = "hybrid"
)

// Compressor defines the interface for content compression
type Compressor interface {
	// Compress compresses the given content using the specified algorithm
	Compress(ctx context.Context, content string, algorithm Algorithm, targetRatio float64) (*Result, error)

	// GetCapabilities returns the capabilities of this compressor
	GetCapabilities(ctx context.Context) Capabilities
}

// Result represents the result of a compression operation
type Result struct {
	// Compressed content
	Content string

	// Compression metadata
	Metadata vectorstore.CompressionMetadata

	// Processing time
	ProcessingTime time.Duration

	// Quality score (0.0 to 1.0, higher is better)
	QualityScore float64
}

// Capabilities describes what a compressor can do
type Capabilities struct {
	// Supported algorithms
	SupportedAlgorithms []Algorithm

	// Maximum content length supported
	MaxContentLength int

	// Whether it supports target compression ratios
	SupportsTargetRatio bool

	// Quality score range
	QualityScoreRange struct {
		Min float64
		Max float64
	}
}

// Config holds configuration for compression operations
type Config struct {
	// Default algorithm to use
	DefaultAlgorithm Algorithm

	// Target compression ratio (original/compressed)
	TargetRatio float64

	// Quality threshold (minimum acceptable quality score)
	QualityThreshold float64

	// Maximum processing time per compression
	MaxProcessingTime time.Duration

	// Anthropic API key for abstractive compression
	AnthropicAPIKey string
}
