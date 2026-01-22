package vectorstore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createTestChromemStore creates a chromem store for testing with a unique temporary directory.
// The name parameter is used as a prefix for the default collection name.
func createTestChromemStore(t *testing.T, name string) (*ChromemStore, *MockEmbedder) {
	t.Helper()

	embedder := &MockEmbedder{
		embedding: make([]float32, 384),
	}

	config := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_" + name,
		VectorSize:        384,
	}

	store, err := NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)

	// Register cleanup
	t.Cleanup(func() {
		store.Close()
	})

	return store, embedder
}
