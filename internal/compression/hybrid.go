package compression

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// RoutingStrategy defines how content should be compressed
type RoutingStrategy string

const (
	// RoutingStrategyExtractive routes to extractive compression
	RoutingStrategyExtractive RoutingStrategy = "extractive"
	// RoutingStrategyAbstractive routes to abstractive compression
	RoutingStrategyAbstractive RoutingStrategy = "abstractive"
	// RoutingStrategyMixed uses both approaches for mixed content
	RoutingStrategyMixed RoutingStrategy = "mixed"
)

// HybridCompressor combines extractive and abstractive approaches
// with intelligent routing based on content type
type HybridCompressor struct {
	extractive  *ExtractiveCompressor
	abstractive Compressor // Use interface for dependency injection
	config      Config
}

// NewHybridCompressor creates a new hybrid compressor
func NewHybridCompressor(config Config) *HybridCompressor {
	return &HybridCompressor{
		extractive:  NewExtractiveCompressor(config),
		abstractive: NewAbstractiveCompressor(config),
		config:      config,
	}
}

// NewHybridCompressorWithAbstractive creates a hybrid compressor with injected abstractive compressor
// This allows for testing with mock implementations
func NewHybridCompressorWithAbstractive(config Config, abstractive Compressor) *HybridCompressor {
	return &HybridCompressor{
		extractive:  NewExtractiveCompressor(config),
		abstractive: abstractive,
		config:      config,
	}
}

// Compress implements the Compressor interface using a hybrid approach with content-aware routing
func (c *HybridCompressor) Compress(ctx context.Context, content string, algorithm Algorithm, targetRatio float64) (*Result, error) {
	start := time.Now()

	// Detect content type for smart routing
	contentType := c.detectContentType(content)

	// Determine routing strategy based on content type
	routing := c.determineRoutingStrategy(contentType)

	var result *Result
	var err error

	switch routing {
	case RoutingStrategyExtractive:
		// Use extractive for code content
		result, err = c.extractive.Compress(ctx, content, AlgorithmExtractive, targetRatio)
		if err != nil {
			return nil, fmt.Errorf("extractive compression failed: %w", err)
		}

	case RoutingStrategyAbstractive:
		// Use abstractive for documentation
		result, err = c.abstractive.Compress(ctx, content, AlgorithmAbstractive, targetRatio)
		if err != nil {
			return nil, fmt.Errorf("abstractive compression failed: %w", err)
		}

	case RoutingStrategyMixed:
		// Split mixed content and compress each section appropriately
		result, err = c.compressMixedContent(ctx, content, targetRatio)
		if err != nil {
			return nil, fmt.Errorf("mixed content compression failed: %w", err)
		}

	default:
		// Fallback to chained approach (original implementation)
		result, err = c.compressChained(ctx, content, targetRatio)
		if err != nil {
			return nil, fmt.Errorf("chained compression failed: %w", err)
		}
	}

	// Update metadata to reflect hybrid algorithm
	result.Metadata.Algorithm = string(AlgorithmHybrid)
	// Hybrid compression always produces summary-level output
	result.Metadata.Level = vectorstore.CompressionLevelSummary
	result.ProcessingTime = time.Since(start)

	return result, nil
}

// detectContentType identifies the primary content type
func (c *HybridCompressor) detectContentType(content string) ContentType {
	// Reuse the extractive compressor's detection logic
	return c.extractive.detectContentType(content)
}

// determineRoutingStrategy decides which compression approach to use
func (c *HybridCompressor) determineRoutingStrategy(contentType ContentType) RoutingStrategy {
	switch contentType {
	case ContentTypeCode:
		return RoutingStrategyExtractive // Code benefits from extractive (preserves structure)
	case ContentTypeMarkdown:
		return RoutingStrategyAbstractive // Docs benefit from abstractive (semantic summarization)
	case ContentTypeMixed:
		return RoutingStrategyMixed // Mixed content needs hybrid approach
	case ContentTypeConversation:
		return RoutingStrategyExtractive // Conversations preserve turn structure
	default:
		return RoutingStrategyExtractive // Default to extractive for plain text
	}
}

// compressMixedContent handles content with both code and documentation
func (c *HybridCompressor) compressMixedContent(ctx context.Context, content string, targetRatio float64) (*Result, error) {
	start := time.Now()

	// Split into code and non-code sections
	sections := c.splitIntoSections(content)

	var compressedSections []string
	totalOriginalSize := 0
	totalCompressedSize := 0
	qualityScores := []float64{}

	for _, section := range sections {
		totalOriginalSize += len(section.Content)

		// Determine compression method for this section
		var sectionResult *Result
		var err error

		if section.IsCode {
			// Use extractive for code sections
			sectionResult, err = c.extractive.Compress(ctx, section.Content, AlgorithmExtractive, targetRatio)
		} else {
			// Use abstractive for documentation sections
			sectionResult, err = c.abstractive.Compress(ctx, section.Content, AlgorithmAbstractive, targetRatio)
		}

		if err != nil {
			// If compression fails, keep original section
			compressedSections = append(compressedSections, section.Content)
			totalCompressedSize += len(section.Content)
			qualityScores = append(qualityScores, 1.0) // Original content has perfect quality
			continue
		}

		compressedSections = append(compressedSections, sectionResult.Content)
		totalCompressedSize += len(sectionResult.Content)
		qualityScores = append(qualityScores, sectionResult.QualityScore)
	}

	// Join compressed sections
	finalContent := strings.Join(compressedSections, "\n\n")
	if len(finalContent) < totalCompressedSize {
		totalCompressedSize = len(finalContent)
	}

	// Calculate average quality score
	avgQualityScore := 0.0
	if len(qualityScores) > 0 {
		for _, score := range qualityScores {
			avgQualityScore += score
		}
		avgQualityScore /= float64(len(qualityScores))
	}

	// Calculate compression ratio
	compressionRatio := float64(totalOriginalSize) / float64(totalCompressedSize)
	if totalCompressedSize == 0 {
		compressionRatio = 1.0
	}

	return &Result{
		Content:        finalContent,
		ProcessingTime: time.Since(start),
		QualityScore:   avgQualityScore,
		Metadata: vectorstore.CompressionMetadata{
			Level:            vectorstore.CompressionLevelSummary,
			Algorithm:        string(AlgorithmHybrid),
			OriginalSize:     totalOriginalSize,
			CompressedSize:   totalCompressedSize,
			CompressionRatio: compressionRatio,
			CompressedAt:     &start,
		},
	}, nil
}

// ContentSection represents a section of content with metadata
type ContentSection struct {
	Content string
	IsCode  bool
}

// splitIntoSections splits mixed content into code and non-code sections
func (c *HybridCompressor) splitIntoSections(content string) []ContentSection {
	var sections []ContentSection
	var currentSection strings.Builder
	var inCodeBlock bool
	var isCodeSection bool

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Check for code block markers
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			// Save current section if it has content
			if currentSection.Len() > 0 {
				sections = append(sections, ContentSection{
					Content: currentSection.String(),
					IsCode:  isCodeSection,
				})
				currentSection.Reset()
			}

			// Toggle code block state
			inCodeBlock = !inCodeBlock
			isCodeSection = inCodeBlock

			// Include the code fence in the code section
			currentSection.WriteString(line)
			currentSection.WriteString("\n")
			continue
		}

		currentSection.WriteString(line)
		currentSection.WriteString("\n")
	}

	// Add final section
	if currentSection.Len() > 0 {
		sections = append(sections, ContentSection{
			Content: currentSection.String(),
			IsCode:  isCodeSection,
		})
	}

	// If no sections were created, treat entire content as one section
	if len(sections) == 0 {
		contentType := c.detectContentType(content)
		sections = append(sections, ContentSection{
			Content: content,
			IsCode:  contentType == ContentTypeCode,
		})
	}

	return sections
}

// compressChained implements the original chained compression approach (fallback)
func (c *HybridCompressor) compressChained(ctx context.Context, content string, targetRatio float64) (*Result, error) {
	// First apply extractive compression to reduce content
	extractiveResult, err := c.extractive.Compress(ctx, content, AlgorithmExtractive, targetRatio*1.5) // Less aggressive
	if err != nil {
		return nil, err
	}

	// Then apply abstractive compression to the extractive result
	abstractiveResult, err := c.abstractive.Compress(ctx, extractiveResult.Content, AlgorithmAbstractive, targetRatio)
	if err != nil {
		return nil, err
	}

	// Combine results
	finalContent := abstractiveResult.Content
	processingTime := time.Since(*extractiveResult.Metadata.CompressedAt)

	// Calculate combined metrics
	originalSize := len(content)
	compressedSize := len(finalContent)
	compressionRatio := float64(originalSize) / float64(compressedSize)
	if compressedSize == 0 {
		compressionRatio = 1.0
	}

	// Calculate comprehensive quality metrics for the final result
	qualityMetrics := NewQualityMetrics(originalSize, compressedSize, targetRatio)
	qualityScore := qualityMetrics.CompositeScore(content, finalContent)

	timestamp := time.Now()
	return &Result{
		Content:        finalContent,
		ProcessingTime: processingTime,
		QualityScore:   qualityScore,
		Metadata: vectorstore.CompressionMetadata{
			Level:            vectorstore.CompressionLevelSummary,
			Algorithm:        string(AlgorithmHybrid),
			OriginalSize:     originalSize,
			CompressedSize:   compressedSize,
			CompressionRatio: compressionRatio,
			CompressedAt:     &timestamp,
		},
	}, nil
}

// GetCapabilities returns the capabilities of this compressor
func (c *HybridCompressor) GetCapabilities(ctx context.Context) Capabilities {
	return Capabilities{
		SupportedAlgorithms: []Algorithm{AlgorithmHybrid},
		MaxContentLength:    75000, // 75KB (balance between extractive and abstractive)
		SupportsTargetRatio: true,
		QualityScoreRange: struct {
			Min float64
			Max float64
		}{
			Min: 0.4,
			Max: 0.95,
		},
	}
}
