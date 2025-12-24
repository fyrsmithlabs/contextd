package compression

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// ExtractiveCompressor implements extractive summarization using sentence scoring
type ExtractiveCompressor struct {
	config Config
}

// NewExtractiveCompressor creates a new extractive compressor
func NewExtractiveCompressor(config Config) *ExtractiveCompressor {
	return &ExtractiveCompressor{
		config: config,
	}
}

// Compress implements the Compressor interface using extractive summarization
func (c *ExtractiveCompressor) Compress(ctx context.Context, content string, algorithm Algorithm, targetRatio float64) (*Result, error) {
	start := time.Now()

	// Split content into sentences
	sentences := c.splitIntoSentences(content)
	if len(sentences) == 0 {
		return &Result{
			Content:        content,
			ProcessingTime: time.Since(start),
			QualityScore:   1.0,
			Metadata: vectorstore.CompressionMetadata{
				Level:            vectorstore.CompressionLevelFolded,
				Algorithm:        string(algorithm),
				OriginalSize:     len(content),
				CompressedSize:   len(content),
				CompressionRatio: 1.0,
				CompressedAt:     &start,
			},
		}, nil
	}

	// Score sentences
	scores := c.scoreSentences(sentences)

	// Select top sentences based on target ratio
	targetLength := int(float64(len(content)) / targetRatio)
	selectedSentences := c.selectSentences(sentences, scores, targetLength)

	// Join selected sentences
	compressedContent := strings.Join(selectedSentences, " ")

	// Calculate metrics
	originalSize := len(content)
	compressedSize := len(compressedContent)
	compressionRatio := float64(originalSize) / float64(compressedSize)

	// Calculate comprehensive quality metrics
	qualityMetrics := NewQualityMetrics(originalSize, compressedSize, targetRatio)
	qualityScore := qualityMetrics.CompositeScore(content, compressedContent)

	return &Result{
		Content:        compressedContent,
		ProcessingTime: time.Since(start),
		QualityScore:   qualityScore,
		Metadata: vectorstore.CompressionMetadata{
			Level:            vectorstore.CompressionLevelFolded,
			Algorithm:        string(algorithm),
			OriginalSize:     originalSize,
			CompressedSize:   compressedSize,
			CompressionRatio: compressionRatio,
			CompressedAt:     &start,
		},
	}, nil
}

// GetCapabilities returns the capabilities of this compressor
func (c *ExtractiveCompressor) GetCapabilities(ctx context.Context) Capabilities {
	return Capabilities{
		SupportedAlgorithms: []Algorithm{AlgorithmExtractive},
		MaxContentLength:    100000, // 100KB
		SupportsTargetRatio: true,
		QualityScoreRange: struct {
			Min float64
			Max float64
		}{
			Min: 0.0,
			Max: 1.0,
		},
	}
}

// splitIntoSentences splits text into sentences
func (c *ExtractiveCompressor) splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)

		// Simple sentence boundary detection
		if r == '.' || r == '!' || r == '?' {
			// Look ahead for potential sentence end
			sentence := strings.TrimSpace(current.String())
			if len(sentence) > 10 { // Minimum sentence length
				sentences = append(sentences, sentence)
				current.Reset()
			}
		}
	}

	// Add remaining content as a sentence
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if len(sentence) > 0 {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// scoreSentences assigns importance scores to sentences
func (c *ExtractiveCompressor) scoreSentences(sentences []string) []float64 {
	scores := make([]float64, len(sentences))

	// Calculate word frequency for TF-IDF like scoring
	wordFreq := c.calculateWordFrequency(sentences)

	for i, sentence := range sentences {
		score := 0.0

		// Position bonus (earlier sentences are more important)
		positionBonus := 1.0 / (float64(i) + 1.0)
		score += positionBonus * 0.3

		// Length bonus (medium-length sentences are preferred)
		words := strings.Fields(sentence)
		lengthScore := math.Min(float64(len(words))/20.0, 1.0) // Peak at 20 words
		if len(words) > 20 {
			lengthScore = 1.0 - (float64(len(words))-20.0)/50.0 // Decline after 20
			lengthScore = math.Max(lengthScore, 0.1)
		}
		score += lengthScore * 0.4

		// Word frequency score (TF-IDF like)
		freqScore := 0.0
		for _, word := range words {
			word = strings.ToLower(strings.TrimFunc(word, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsNumber(r)
			}))
			if freq, exists := wordFreq[word]; exists && freq > 1 {
				freqScore += 1.0 / float64(freq) // Inverse frequency
			}
		}
		if len(words) > 0 {
			freqScore /= float64(len(words))
		}
		score += freqScore * 0.3

		scores[i] = score
	}

	return scores
}

// calculateWordFrequency calculates word frequencies across all sentences
func (c *ExtractiveCompressor) calculateWordFrequency(sentences []string) map[string]int {
	freq := make(map[string]int)

	for _, sentence := range sentences {
		words := strings.Fields(sentence)
		for _, word := range words {
			word = strings.ToLower(strings.TrimFunc(word, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsNumber(r)
			}))
			if len(word) > 2 { // Ignore very short words
				freq[word]++
			}
		}
	}

	return freq
}

// selectSentences selects the best sentences to include in the summary
func (c *ExtractiveCompressor) selectSentences(sentences []string, scores []float64, targetLength int) []string {
	type sentenceScore struct {
		index int
		score float64
	}

	// Create score slice
	scoreSlice := make([]sentenceScore, len(sentences))
	for i, score := range scores {
		scoreSlice[i] = sentenceScore{index: i, score: score}
	}

	// Sort by score (descending)
	sort.Slice(scoreSlice, func(i, j int) bool {
		return scoreSlice[i].score > scoreSlice[j].score
	})

	// Select sentences until target length is reached
	var selected []string
	currentLength := 0

	for _, ss := range scoreSlice {
		sentence := sentences[ss.index]
		if currentLength+len(sentence) <= targetLength {
			selected = append(selected, sentence)
			currentLength += len(sentence)

			// Add space between sentences
			if currentLength < targetLength {
				currentLength++
			}
		}
		// If it doesn't fit, continue to check next sentences
	}

	// Edge case: if no sentences were selected (targetLength too small),
	// select the highest-scoring sentence to ensure non-empty output
	if len(selected) == 0 && len(sentences) > 0 {
		selected = append(selected, sentences[scoreSlice[0].index])
	}

	// Sort selected sentences by original order to maintain coherence
	sort.Slice(selected, func(i, j int) bool {
		// Find original indices
		var idxI, idxJ int
		for _, ss := range scoreSlice {
			if sentences[ss.index] == selected[i] {
				idxI = ss.index
			}
			if sentences[ss.index] == selected[j] {
				idxJ = ss.index
			}
		}
		return idxI < idxJ
	})

	return selected
}
