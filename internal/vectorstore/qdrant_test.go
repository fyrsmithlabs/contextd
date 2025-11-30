package vectorstore_test

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbedder is a simple embedder that returns fixed-size zero vectors.
type mockEmbedder struct {
	vectorSize int
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, m.vectorSize)
		// Simple hash-based embedding for testing
		for j := range embeddings[i] {
			embeddings[i][j] = float32((i + j) % 10) / 10.0
		}
	}
	return embeddings, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	vectors, err := m.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

func TestValidateCollectionName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid org collection",
			input:     "org_memories",
			wantError: false,
		},
		{
			name:      "valid team collection",
			input:     "platform_memories",
			wantError: false,
		},
		{
			name:      "valid project collection",
			input:     "platform_contextd_memories",
			wantError: false,
		},
		{
			name:      "empty name",
			input:     "",
			wantError: true,
		},
		{
			name:      "uppercase letters",
			input:     "Org_Memories",
			wantError: true,
		},
		{
			name:      "special characters",
			input:     "org-memories",
			wantError: true,
		},
		{
			name:      "too long",
			input:     "a123456789012345678901234567890123456789012345678901234567890123456789",
			wantError: true,
		},
		{
			name:      "path traversal attempt",
			input:     "../memories",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vectorstore.ValidateCollectionName(tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQdrantConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    vectorstore.QdrantConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: vectorstore.QdrantConfig{
				Host:           "localhost",
				Port:           6334,
				CollectionName: "test_collection",
				VectorSize:     384,
			},
			wantError: false,
		},
		{
			name: "missing host",
			config: vectorstore.QdrantConfig{
				Port:           6334,
				CollectionName: "test_collection",
				VectorSize:     384,
			},
			wantError: true,
		},
		{
			name: "invalid port",
			config: vectorstore.QdrantConfig{
				Host:           "localhost",
				Port:           0,
				CollectionName: "test_collection",
				VectorSize:     384,
			},
			wantError: true,
		},
		{
			name: "missing collection name",
			config: vectorstore.QdrantConfig{
				Host:       "localhost",
				Port:       6334,
				VectorSize: 384,
			},
			wantError: true,
		},
		{
			name: "missing vector size",
			config: vectorstore.QdrantConfig{
				Host:           "localhost",
				Port:           6334,
				CollectionName: "test_collection",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQdrantConfig_ApplyDefaults(t *testing.T) {
	config := vectorstore.QdrantConfig{}
	config.ApplyDefaults()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1000000000, int(config.RetryBackoff)) // 1 second in nanoseconds
	assert.Equal(t, 50*1024*1024, config.MaxMessageSize)
	assert.Equal(t, 5, config.CircuitBreakerThreshold)
	assert.Equal(t, qdrant.Distance_Cosine, config.Distance)
}

// Integration test - requires running Qdrant instance
func TestQdrantStore_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires Qdrant running on localhost:6334
	// Skip if not available
	ctx := context.Background()

	config := vectorstore.QdrantConfig{
		Host:           "localhost",
		Port:           6334,
		CollectionName: "test_integration",
		VectorSize:     10,
		UseTLS:         false,
	}

	embedder := &mockEmbedder{vectorSize: 10}

	store, err := vectorstore.NewQdrantStore(config, embedder)
	if err != nil {
		t.Skipf("Qdrant not available: %v", err)
	}
	defer store.Close()

	// Clean up collection if it exists
	exists, _ := store.CollectionExists(ctx, config.CollectionName)
	if exists {
		_ = store.DeleteCollection(ctx, config.CollectionName)
	}

	// Create collection
	err = store.CreateCollection(ctx, config.CollectionName, 10)
	require.NoError(t, err)

	// Verify collection exists
	exists, err = store.CollectionExists(ctx, config.CollectionName)
	require.NoError(t, err)
	assert.True(t, exists)

	// Add documents
	docs := []vectorstore.Document{
		{
			ID:      "doc1",
			Content: "test document one",
			Metadata: map[string]interface{}{
				"owner": "alice",
			},
		},
		{
			ID:      "doc2",
			Content: "test document two",
			Metadata: map[string]interface{}{
				"owner": "bob",
			},
		},
	}

	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 2)

	// Search
	results, err := store.Search(ctx, "test query", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Search with filters
	results, err = store.SearchWithFilters(ctx, "test query", 10, map[string]interface{}{
		"owner": "alice",
	})
	require.NoError(t, err)
	// Should only return alice's document

	// Get collection info
	info, err := store.GetCollectionInfo(ctx, config.CollectionName)
	require.NoError(t, err)
	assert.Equal(t, config.CollectionName, info.Name)
	assert.Equal(t, 2, info.PointCount)

	// List collections
	collections, err := store.ListCollections(ctx)
	require.NoError(t, err)
	assert.Contains(t, collections, config.CollectionName)

	// Delete documents
	err = store.DeleteDocuments(ctx, []string{"doc1"})
	require.NoError(t, err)

	// Clean up
	err = store.DeleteCollection(ctx, config.CollectionName)
	require.NoError(t, err)

	// Verify deletion
	exists, err = store.CollectionExists(ctx, config.CollectionName)
	require.NoError(t, err)
	assert.False(t, exists)
}
