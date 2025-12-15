package remediation

import "context"

// Embedder generates embeddings for text.
// This interface will be implemented by the memory/embeddings package.
// TODO: Move this to a shared package once internal/embeddings is ported.
type Embedder interface {
	// Embed generates an embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the dimensionality of the embedding vectors.
	Dimension() int
}
