package embeddings

import (
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// TestEmbedderInterface verifies that Service implements vectorstore.Embedder.
// This will fail to compile if the interface is not satisfied.
func TestEmbedderInterface(t *testing.T) {
	var _ vectorstore.Embedder = (*Service)(nil)
	t.Log("Service correctly implements vectorstore.Embedder interface")
}
