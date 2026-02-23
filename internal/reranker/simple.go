package reranker

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// ErrNilContext is returned when a nil context is passed to Rerank.
var ErrNilContext = errors.New("context cannot be nil")

// SimpleReranker implements a simple TF-IDF based reranking algorithm.
// It calculates term overlap between the query and documents, then combines
// the original score with the overlap score to produce a final ranking.
type SimpleReranker struct{}

// NewSimpleReranker creates a new SimpleReranker instance.
func NewSimpleReranker() *SimpleReranker {
	return &SimpleReranker{}
}

// Rerank re-ranks documents using a simple TF-IDF approach.
// The algorithm:
// 1. Tokenizes the query into lowercased terms
// 2. For each document, calculates term overlap with the query
// 3. Combines original score (50% weight) with overlap score (50% weight)
// 4. Sorts by combined score and returns top K results
func (r *SimpleReranker) Rerank(ctx context.Context, query string, docs []Document, topK int) ([]ScoredDocument, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if topK <= 0 {
		topK = len(docs)
	}
	if len(docs) == 0 {
		return []ScoredDocument{}, nil
	}

	// Tokenize query into lowercase terms
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		// If query has no tokens, fall back to original ranking
		return fallbackRank(docs, topK), nil
	}

	// Calculate scores for each document
	type scoredDoc struct {
		doc               ScoredDocument
		rerankerScore     float32
		combinedScore     float32
		overlapPercentage float32
	}

	scoredDocs := make([]scoredDoc, len(docs))
	for i, doc := range docs {
		// Calculate term overlap between query and document
		docTokens := tokenize(doc.Content)
		overlap := calculateTermOverlap(queryTokens, docTokens)

		// Combine scores: 50% original + 50% overlap
		// This preserves some reliance on semantic similarity (original score)
		// while boosting documents with high term overlap
		const originalWeight = 0.5
		const overlapWeight = 0.5
		combinedScore := float32(originalWeight)*doc.Score + float32(overlapWeight)*overlap

		scoredDocs[i] = scoredDoc{
			doc: ScoredDocument{
				Document: Document{
					ID:      doc.ID,
					Content: doc.Content,
					Score:   doc.Score,
				},
				RerankerScore: overlap,
				OriginalRank:  i,
			},
			rerankerScore:     overlap,
			combinedScore:     combinedScore,
			overlapPercentage: overlap,
		}
	}

	// Sort by combined score (descending)
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].combinedScore > scoredDocs[j].combinedScore
	})

	// Extract top K results
	limit := topK
	if limit > len(scoredDocs) {
		limit = len(scoredDocs)
	}

	result := make([]ScoredDocument, limit)
	for i := 0; i < limit; i++ {
		result[i] = scoredDocs[i].doc
	}

	return result, nil
}

// Close closes the reranker. SimpleReranker has no resources to clean up.
func (r *SimpleReranker) Close() error {
	return nil
}

// tokenize splits text into lowercase terms, filtering out common stopwords.
func tokenize(text string) []string {
	// Convert to lowercase and split on whitespace/punctuation
	text = strings.ToLower(text)
	// Simple tokenization: split on whitespace and basic punctuation
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !isAlphanumeric(r)
	})

	// Filter stopwords
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !isStopword(token) && len(token) > 2 {
			filtered = append(filtered, token)
		}
	}
	return filtered
}

// isAlphanumeric returns true if the rune is alphanumeric or underscore.
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_'
}

// isStopword returns true if the token is a common English stopword.
func isStopword(token string) bool {
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "can": true, "this": true,
		"that": true, "these": true, "those": true, "i": true, "you": true, "he": true,
		"she": true, "it": true, "we": true, "they": true, "what": true, "which": true,
		"who": true, "when": true, "where": true, "why": true, "how": true,
	}
	return stopwords[token]
}

// calculateTermOverlap calculates the ratio of query terms found in document tokens.
// Returns a score between 0.0 and 1.0 representing term overlap percentage.
func calculateTermOverlap(queryTokens, docTokens []string) float32 {
	if len(queryTokens) == 0 {
		return 0.0
	}

	// Build a set of document tokens for efficient lookup
	docTokenSet := make(map[string]bool)
	for _, token := range docTokens {
		docTokenSet[token] = true
	}

	// Count how many unique query tokens appear in document
	// Use a counted map to avoid counting duplicate query tokens multiple times
	matchCount := 0
	counted := make(map[string]bool)
	for _, queryToken := range queryTokens {
		if docTokenSet[queryToken] && !counted[queryToken] {
			matchCount++
			counted[queryToken] = true
		}
	}

	// Return overlap as percentage (0.0 - 1.0)
	return float32(matchCount) / float32(len(queryTokens))
}

// fallbackRank returns documents ranked by original score when reranking cannot proceed.
func fallbackRank(docs []Document, topK int) []ScoredDocument {
	// Sort by original score (descending)
	sorted := make([]Document, len(docs))
	copy(sorted, docs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	// Extract top K
	limit := topK
	if limit > len(sorted) {
		limit = len(sorted)
	}

	result := make([]ScoredDocument, limit)
	for i := 0; i < limit; i++ {
		result[i] = ScoredDocument{
			Document:      sorted[i],
			RerankerScore: sorted[i].Score, // Use original score as fallback
			OriginalRank:  i,
		}
	}
	return result
}
