package compression

import (
	"math"
	"strings"
	"unicode"
)

// QualityMetrics holds metrics for evaluating compression quality
type QualityMetrics struct {
	OriginalSize   int
	CompressedSize int
	TargetRatio    float64
}

// NewQualityMetrics creates a new quality metrics calculator
func NewQualityMetrics(originalSize, compressedSize int, targetRatio float64) *QualityMetrics {
	return &QualityMetrics{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		TargetRatio:    targetRatio,
	}
}

// CompressionRatioScore calculates score based on compression ratio vs target
// Returns 1.0 if target is met or exceeded, penalizes if below target
func (m *QualityMetrics) CompressionRatioScore() float64 {
	if m.CompressedSize == 0 || m.OriginalSize == 0 {
		return 0.0
	}

	actualRatio := float64(m.OriginalSize) / float64(m.CompressedSize)

	// Perfect score if target is met or exceeded
	if actualRatio >= m.TargetRatio {
		return 1.0
	}

	// Penalize proportionally if below target
	return actualRatio / m.TargetRatio
}

// InformationRetentionScore measures how well keywords/concepts are preserved
func (m *QualityMetrics) InformationRetentionScore(original, compressed string) float64 {
	retention := m.KeywordRetentionRate(original, compressed)

	// Apply exponential curve to reward high retention
	return math.Pow(retention, 0.8)
}

// KeywordRetentionRate calculates the percentage of important keywords retained
func (m *QualityMetrics) KeywordRetentionRate(original, compressed string) float64 {
	originalKeywords := extractKeywords(original)
	compressedKeywords := extractKeywords(compressed)

	if len(originalKeywords) == 0 {
		return 0.0
	}

	// Count how many original keywords appear in compressed
	retained := 0
	for keyword := range originalKeywords {
		if compressedKeywords[keyword] {
			retained++
		}
	}

	return float64(retained) / float64(len(originalKeywords))
}

// SemanticSimilarityScore measures meaning preservation using word overlap
func (m *QualityMetrics) SemanticSimilarityScore(original, compressed string) float64 {
	originalWords := extractWords(original)
	compressedWords := extractWords(compressed)

	if len(originalWords) == 0 || len(compressedWords) == 0 {
		return 0.0
	}

	// Calculate Jaccard similarity (intersection / union)
	intersection := 0
	union := make(map[string]bool)

	for word := range originalWords {
		union[word] = true
		if compressedWords[word] {
			intersection++
		}
	}

	for word := range compressedWords {
		union[word] = true
	}

	if len(union) == 0 {
		return 0.0
	}

	return float64(intersection) / float64(len(union))
}

// ReadabilityScore measures the readability of the compressed text
func (m *QualityMetrics) ReadabilityScore(text string) float64 {
	sentences := splitSentences(text)
	words := extractWords(text)

	if len(words) == 0 {
		return 0.0
	}

	// Base score on sentence structure
	sentenceScore := 0.0
	if len(sentences) > 0 {
		// Reward proper sentence structure
		avgWordsPerSentence := float64(len(words)) / float64(len(sentences))

		// Optimal: 10-20 words per sentence
		if avgWordsPerSentence >= 10 && avgWordsPerSentence <= 20 {
			sentenceScore = 1.0
		} else if avgWordsPerSentence >= 5 && avgWordsPerSentence < 10 {
			sentenceScore = 0.8
		} else if avgWordsPerSentence >= 3 {
			sentenceScore = 0.6
		} else {
			sentenceScore = 0.4
		}
	} else {
		// No sentences (fragments only)
		sentenceScore = 0.3
	}

	// Check for sentence-ending punctuation
	punctuationScore := 0.0
	if strings.ContainsAny(text, ".!?") {
		punctuationScore = 0.2
	}

	return math.Min(sentenceScore+punctuationScore, 1.0)
}

// CompositeScore calculates weighted average of all quality metrics
func (m *QualityMetrics) CompositeScore(original, compressed string) float64 {
	compressionScore := m.CompressionRatioScore()
	retentionScore := m.InformationRetentionScore(original, compressed)
	similarityScore := m.SemanticSimilarityScore(original, compressed)
	readabilityScore := m.ReadabilityScore(compressed)

	// Weighted average
	weights := map[string]float64{
		"compression": 0.25,
		"retention":   0.30,
		"similarity":  0.30,
		"readability": 0.15,
	}

	composite := compressionScore*weights["compression"] +
		retentionScore*weights["retention"] +
		similarityScore*weights["similarity"] +
		readabilityScore*weights["readability"]

	return composite
}

// QualityThresholds defines minimum acceptable quality scores
type QualityThresholds struct {
	MinCompressionRatio     float64
	MinInformationRetention float64
	MinSemanticSimilarity   float64
	MinReadability          float64
	MinCompositeScore       float64
}

// QualityGate enforces quality thresholds
type QualityGate struct {
	Thresholds QualityThresholds
}

// NewQualityGate creates a new quality gate with specified thresholds
func NewQualityGate(thresholds QualityThresholds) *QualityGate {
	return &QualityGate{
		Thresholds: thresholds,
	}
}

// QualityGateResult contains the result of quality gate evaluation
type QualityGateResult struct {
	Pass                      bool
	FailureReason             string
	CompressionRatioScore     float64
	InformationRetentionScore float64
	SemanticSimilarityScore   float64
	ReadabilityScore          float64
	CompositeScore            float64
}

// Evaluate checks if quality metrics meet all thresholds
func (g *QualityGate) Evaluate(metrics *QualityMetrics, original, compressed string) *QualityGateResult {
	result := &QualityGateResult{
		Pass: true,
	}

	// Calculate all scores
	result.CompressionRatioScore = metrics.CompressionRatioScore()
	result.InformationRetentionScore = metrics.InformationRetentionScore(original, compressed)
	result.SemanticSimilarityScore = metrics.SemanticSimilarityScore(original, compressed)
	result.ReadabilityScore = metrics.ReadabilityScore(compressed)
	result.CompositeScore = metrics.CompositeScore(original, compressed)

	// Check each threshold
	if g.Thresholds.MinCompressionRatio > 0 && result.CompressionRatioScore < g.Thresholds.MinCompressionRatio {
		result.Pass = false
		result.FailureReason = "compression_ratio below threshold"
		return result
	}

	if g.Thresholds.MinInformationRetention > 0 && result.InformationRetentionScore < g.Thresholds.MinInformationRetention {
		result.Pass = false
		result.FailureReason = "information_retention below threshold"
		return result
	}

	if g.Thresholds.MinSemanticSimilarity > 0 && result.SemanticSimilarityScore < g.Thresholds.MinSemanticSimilarity {
		result.Pass = false
		result.FailureReason = "semantic_similarity below threshold"
		return result
	}

	if g.Thresholds.MinReadability > 0 && result.ReadabilityScore < g.Thresholds.MinReadability {
		result.Pass = false
		result.FailureReason = "readability below threshold"
		return result
	}

	if g.Thresholds.MinCompositeScore > 0 && result.CompositeScore < g.Thresholds.MinCompositeScore {
		result.Pass = false
		result.FailureReason = "composite_score below threshold"
		return result
	}

	return result
}

// Helper functions

// extractKeywords extracts important keywords (filtering common words)
func extractKeywords(text string) map[string]bool {
	words := extractWords(text)
	stopWords := getStopWords()

	keywords := make(map[string]bool)
	for word := range words {
		if !stopWords[word] && len(word) > 3 {
			keywords[word] = true
		}
	}

	return keywords
}

// extractWords extracts all words from text
func extractWords(text string) map[string]bool {
	words := make(map[string]bool)

	var current strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			words[current.String()] = true
			current.Reset()
		}
	}

	if current.Len() > 0 {
		words[current.String()] = true
	}

	return words
}

// splitSentences splits text into sentences
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)

		if r == '.' || r == '!' || r == '?' {
			sentence := strings.TrimSpace(current.String())
			// Minimum length of 10 to filter out fragments like "e.g." or "i.e."
			// while preserving legitimate short sentences
			if len(sentence) > 10 {
				sentences = append(sentences, sentence)
				current.Reset()
			}
		}
	}

	return sentences
}

// getStopWords returns common words to filter out
func getStopWords() map[string]bool {
	return map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"should": true, "could": true, "may": true, "might": true, "must": true,
		"can": true, "this": true, "that": true, "these": true, "those": true,
		"it": true, "its": true, "as": true, "which": true, "who": true,
		"when": true, "where": true, "why": true, "how": true,
	}
}
