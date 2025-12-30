package vectorstore_test

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/require"
)

// TestEmbedder is a mock embedder for testing vectorstore implementations.
// It generates deterministic embeddings based on input text for reproducible tests.
type TestEmbedder struct {
	VectorSize int
}

// EmbedDocuments generates mock embeddings for multiple texts.
// Embeddings are deterministic based on text content for test reproducibility.
func (e *TestEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = e.makeEmbedding(texts[i], i)
	}
	return embeddings, nil
}

// EmbedQuery generates a mock embedding for a single query text.
func (e *TestEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.makeEmbedding(text, 0), nil
}

// makeEmbedding creates a deterministic embedding for testing.
// Uses text hash and index to generate reproducible but unique vectors.
func (e *TestEmbedder) makeEmbedding(text string, index int) []float32 {
	embedding := make([]float32, e.VectorSize)
	for j := range embedding {
		// Simple hash-based embedding for testing
		embedding[j] = float32((len(text)+j+index)%10) / 10.0
	}
	return embedding
}

// setupQdrantCollection creates a test collection, cleaning up any existing
// collection first, and registers cleanup to delete it when the test completes.
func setupQdrantCollection(t *testing.T, ctx context.Context, store *vectorstore.QdrantStore, name string, vectorSize int) {
	t.Helper()

	// Clean up if exists
	exists, _ := store.CollectionExists(ctx, name)
	if exists {
		_ = store.DeleteCollection(ctx, name)
	}

	// Create collection
	err := store.CreateCollection(ctx, name, vectorSize)
	require.NoError(t, err)

	// Register cleanup
	t.Cleanup(func() {
		_ = store.DeleteCollection(ctx, name)
	})
}
