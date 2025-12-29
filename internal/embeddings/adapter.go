// Package embeddings provides embedding generation via multiple providers.
package embeddings

import (
	"context"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// RemediationEmbedder adapts Provider to implement remediation.Embedder interface.
type RemediationEmbedder struct {
	embedder  vectorstore.Embedder
	dimension int
}

// NewRemediationEmbedder creates an adapter for remediation service.
// Dimension should match the embedding model (e.g., 384 for bge-small-en-v1.5).
func NewRemediationEmbedder(embedder vectorstore.Embedder, dimension int) *RemediationEmbedder {
	return &RemediationEmbedder{
		embedder:  embedder,
		dimension: dimension,
	}
}

// Embed generates an embedding vector for the given text.
func (e *RemediationEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return e.embedder.EmbedQuery(ctx, text)
}

// EmbedBatch generates embeddings for multiple texts.
func (e *RemediationEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.embedder.EmbedDocuments(ctx, texts)
}

// Dimension returns the dimensionality of the embedding vectors.
func (e *RemediationEmbedder) Dimension() int {
	return e.dimension
}
