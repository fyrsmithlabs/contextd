// Package reranker provides document re-ranking functionality for improving search quality.
package reranker

import (
	"context"
)

// Document represents a searchable document with metadata and scores.
type Document struct {
	ID      string  // Unique identifier for the document
	Content string  // Text content to be re-ranked
	Score   float32 // Original similarity score from search
}

// ScoredDocument represents a document with re-ranking scores.
type ScoredDocument struct {
	Document
	RerankerScore float32 // Score from re-ranker (0.0-1.0)
	OriginalRank  int     // Original rank position in results (0-indexed)
}

// Reranker provides an interface for document re-ranking algorithms.
type Reranker interface {
	// Rerank re-ranks documents based on query relevance.
	// Takes a query string, list of documents, and desired top K results.
	// Returns re-ranked documents sorted by RerankerScore in descending order,
	// limited to topK results.
	//
	// The caller is responsible for ensuring ctx is not nil.
	Rerank(ctx context.Context, query string, docs []Document, topK int) ([]ScoredDocument, error)

	// Close closes the reranker and releases any resources.
	// Should be called when the reranker is no longer needed.
	Close() error
}
