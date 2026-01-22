package integration

import (
	"os"
	"strconv"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createTestEmbedder creates a test embedder for integration tests.
func createTestEmbedder(t *testing.T) vectorstore.Embedder {
	cfg := embeddings.ProviderConfig{
		Provider: "fastembed",
		Model:    "BAAI/bge-small-en-v1.5",
		CacheDir: t.TempDir(),
	}

	provider, err := embeddings.NewProvider(cfg)
	require.NoError(t, err, "Should create test embedder")

	return provider
}

// createTestVectorStore creates a test vector store and returns cleanup function.
// Uses chromem (embedded) by default for integration tests.
func createTestVectorStore(t *testing.T) (vectorstore.Store, func()) {
	tmpDir := t.TempDir()

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		DefaultCollection: "test-collection",
		VectorSize:        384, // BAAI/bge-small-en-v1.5
		Isolation:         vectorstore.NewPayloadIsolation(),
	}

	embedder := createTestEmbedder(t)
	logger := zap.NewNop()

	store, err := vectorstore.NewChromemStore(config, embedder, logger)
	require.NoError(t, err, "Should create test vector store")

	cleanup := func() {
		if store != nil {
			store.Close()
		}
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

// createTestQdrantStore creates a test Qdrant store (requires Qdrant container).
// Only use in containerized tests where Qdrant is available.
func createTestQdrantStore(t *testing.T) (vectorstore.Store, func()) {
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		host = "localhost"
	}

	portStr := os.Getenv("QDRANT_PORT")
	if portStr == "" {
		portStr = "6334"
	}
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "Should parse QDRANT_PORT as integer")

	config := vectorstore.QdrantConfig{
		Host:           host,
		Port:           port,
		CollectionName: "test-collection",
		VectorSize:     384,
		Isolation:      vectorstore.NewPayloadIsolation(),
	}

	embedder := createTestEmbedder(t)

	store, err := vectorstore.NewQdrantStore(config, embedder)
	require.NoError(t, err, "Should create test Qdrant store")

	cleanup := func() {
		if store != nil {
			store.Close()
		}
	}

	return store, cleanup
}

// getTestStoreProvider returns the vector store provider for integration tests.
// Checks VECTOR_STORE environment variable:
// - "qdrant" = Use Qdrant (requires container)
// - "chromem" or empty = Use chromem (default)
func getTestStoreProvider(t *testing.T) (vectorstore.Store, func()) {
	provider := os.Getenv("VECTOR_STORE")
	if provider == "" {
		provider = "chromem"
	}

	switch provider {
	case "qdrant":
		return createTestQdrantStore(t)
	case "chromem":
		return createTestVectorStore(t)
	default:
		t.Fatalf("Unknown vector store provider: %s", provider)
		return nil, nil
	}
}

// createTestEmbeddings creates a test embeddings provider for integration tests.
func createTestEmbeddings(t *testing.T) embeddings.Provider {
	cfg := embeddings.ProviderConfig{
		Provider: "fastembed",
		Model:    "BAAI/bge-small-en-v1.5",
		CacheDir: t.TempDir(),
	}

	provider, err := embeddings.NewProvider(cfg)
	require.NoError(t, err, "Should create embeddings provider")

	return provider
}
