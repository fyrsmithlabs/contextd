// Package embeddings provides embedding generation via TEI.
package embeddings

import (
	"context"
)

// RemediationEmbedder adapts Service to implement remediation.Embedder interface.
type RemediationEmbedder struct {
	svc       *Service
	dimension int
}

// NewRemediationEmbedder creates an adapter for remediation service.
// Dimension should match the embedding model (e.g., 384 for bge-small-en-v1.5).
func NewRemediationEmbedder(svc *Service, dimension int) *RemediationEmbedder {
	return &RemediationEmbedder{
		svc:       svc,
		dimension: dimension,
	}
}

// Embed generates an embedding vector for the given text.
func (e *RemediationEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return e.svc.EmbedQuery(ctx, text)
}

// EmbedBatch generates embeddings for multiple texts.
func (e *RemediationEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return e.svc.EmbedDocuments(ctx, texts)
}

// Dimension returns the dimensionality of the embedding vectors.
func (e *RemediationEmbedder) Dimension() int {
	return e.dimension
}
