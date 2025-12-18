package vectorstore_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// chromemTestEmbedder returns normalized vectors for testing.
type chromemTestEmbedder struct {
	vectorSize int
}

func (e *chromemTestEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = e.makeEmbedding(text)
	}
	return embeddings, nil
}

func (e *chromemTestEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.makeEmbedding(text), nil
}

// makeEmbedding creates a normalized embedding based on text hash.
func (e *chromemTestEmbedder) makeEmbedding(text string) []float32 {
	embedding := make([]float32, e.vectorSize)
	// Create deterministic embedding based on text
	hash := 0
	for _, c := range text {
		hash = (hash*31 + int(c)) % 1000
	}
	// Fill with normalized values
	var sumSq float32
	for i := range embedding {
		embedding[i] = float32((hash+i)%100) / 100.0
		sumSq += embedding[i] * embedding[i]
	}
	// Normalize to unit vector (chromem requires normalized vectors)
	if sumSq > 0 {
		norm := float32(1.0) / sqrt32(sumSq)
		for i := range embedding {
			embedding[i] *= norm
		}
	}
	return embedding
}

func sqrt32(x float32) float32 {
	if x <= 0 {
		return 0
	}
	// Newton's method for square root
	z := x / 2
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func newTestChromemStore(t *testing.T) (*vectorstore.ChromemStore, string) {
	t.Helper()

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "chromem_test_*")
	require.NoError(t, err)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false, // Faster for tests
		DefaultCollection: "test_collection",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(), // Disable isolation for general tests
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	store, err := vectorstore.NewChromemStore(config, embedder, logger)
	require.NoError(t, err)

	return store, tmpDir
}

func TestChromemConfig_ApplyDefaults(t *testing.T) {
	config := vectorstore.ChromemConfig{}
	config.ApplyDefaults()

	assert.Equal(t, "~/.config/contextd/vectorstore", config.Path)
	assert.Equal(t, "contextd_default", config.DefaultCollection)
	assert.Equal(t, 384, config.VectorSize)
}

func TestChromemConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    vectorstore.ChromemConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: vectorstore.ChromemConfig{
				Path:              "/tmp/test",
				DefaultCollection: "test",
				VectorSize:        384,
			},
			wantError: false,
		},
		{
			name: "zero vector size",
			config: vectorstore.ChromemConfig{
				Path:              "/tmp/test",
				DefaultCollection: "test",
				VectorSize:        0,
			},
			wantError: true,
		},
		{
			name: "negative vector size",
			config: vectorstore.ChromemConfig{
				Path:              "/tmp/test",
				DefaultCollection: "test",
				VectorSize:        -1,
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

func TestNewChromemStore(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	assert.NotNil(t, store)
}

func TestNewChromemStore_ExpandsHomePath(t *testing.T) {
	// Test that ~ is expanded
	config := vectorstore.ChromemConfig{
		Path:              "~/.config/contextd/test_vectorstore",
		DefaultCollection: "test",
		VectorSize:        384,
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	store, err := vectorstore.NewChromemStore(config, embedder, logger)
	require.NoError(t, err)
	defer store.Close()

	// Clean up
	home, _ := os.UserHomeDir()
	os.RemoveAll(filepath.Join(home, ".config/contextd/test_vectorstore"))
}

func TestChromemStore_AddDocuments(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	docs := []vectorstore.Document{
		{ID: "doc1", Content: "First document about Go programming"},
		{ID: "doc2", Content: "Second document about vector databases"},
	}

	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Equal(t, "doc1", ids[0])
	assert.Equal(t, "doc2", ids[1])
}

func TestChromemStore_AddDocuments_EmptyReturnsError(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	_, err := store.AddDocuments(ctx, []vectorstore.Document{})
	assert.Error(t, err)
}

func TestChromemStore_AddDocuments_WithMetadata(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	docs := []vectorstore.Document{
		{
			ID:      "doc1",
			Content: "Document with metadata",
			Metadata: map[string]interface{}{
				"owner":   "alice",
				"project": "contextd",
				"count":   42,
			},
		},
	}

	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
}

func TestChromemStore_Search(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Add documents
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Go programming language tutorial"},
		{ID: "doc2", Content: "Python machine learning guide"},
		{ID: "doc3", Content: "Go concurrency patterns"},
	}
	_, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search
	results, err := store.Search(ctx, "Go programming", 2)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 2)
}

func TestChromemStore_SearchInCollection(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create collection and add docs
	err := store.CreateCollection(ctx, "custom_collection", 384)
	require.NoError(t, err)

	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Custom collection document", Collection: "custom_collection"},
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search in specific collection
	results, err := store.SearchInCollection(ctx, "custom_collection", "document", 5, nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestChromemStore_SearchInCollection_NotFound(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	_, err := store.SearchInCollection(ctx, "nonexistent", "query", 5, nil)
	assert.ErrorIs(t, err, vectorstore.ErrCollectionNotFound)
}

func TestChromemStore_SearchWithFilters(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Add documents with metadata
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Alice's document", Metadata: map[string]interface{}{"owner": "alice"}},
		{ID: "doc2", Content: "Bob's document", Metadata: map[string]interface{}{"owner": "bob"}},
	}
	_, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search with filter
	results, err := store.SearchWithFilters(ctx, "document", 10, map[string]interface{}{"owner": "alice"})
	require.NoError(t, err)
	// Should only return Alice's document
	for _, r := range results {
		if owner, ok := r.Metadata["owner"]; ok {
			assert.Equal(t, "alice", owner)
		}
	}
}

func TestChromemStore_DeleteDocuments(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Add documents
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Document to delete"},
		{ID: "doc2", Content: "Document to keep"},
	}
	_, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Delete one document
	err = store.DeleteDocuments(ctx, []string{"doc1"})
	require.NoError(t, err)

	// Search should not find deleted document
	results, err := store.Search(ctx, "delete", 10)
	require.NoError(t, err)
	for _, r := range results {
		assert.NotEqual(t, "doc1", r.ID)
	}
}

func TestChromemStore_CreateCollection(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	err := store.CreateCollection(ctx, "new_collection", 384)
	require.NoError(t, err)

	// Verify collection exists
	exists, err := store.CollectionExists(ctx, "new_collection")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestChromemStore_CreateCollection_InvalidName(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Invalid collection names should fail
	err := store.CreateCollection(ctx, "Invalid-Name", 384)
	assert.Error(t, err)

	err = store.CreateCollection(ctx, "", 384)
	assert.Error(t, err)
}

func TestChromemStore_CreateCollection_ZeroUsesDefault(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Passing 0 should use the store's configured vector size (384 from test embedder)
	err := store.CreateCollection(ctx, "zero_size", 0)
	require.NoError(t, err)

	// Verify collection exists
	exists, err := store.CollectionExists(ctx, "zero_size")
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify collection info shows correct dimension
	info, err := store.GetCollectionInfo(ctx, "zero_size")
	require.NoError(t, err)
	assert.Equal(t, 384, info.VectorSize)
}

func TestChromemStore_CreateCollection_WrongVectorSize(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Vector size mismatch should fail
	err := store.CreateCollection(ctx, "wrong_size", 768)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match configured size")
}

func TestChromemStore_DeleteCollection(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create and then delete
	err := store.CreateCollection(ctx, "to_delete", 384)
	require.NoError(t, err)

	err = store.DeleteCollection(ctx, "to_delete")
	require.NoError(t, err)

	// Verify deleted
	exists, err := store.CollectionExists(ctx, "to_delete")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestChromemStore_CollectionExists(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Non-existent collection
	exists, err := store.CollectionExists(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)

	// Create collection
	err = store.CreateCollection(ctx, "exists_test", 384)
	require.NoError(t, err)

	// Should exist now
	exists, err = store.CollectionExists(ctx, "exists_test")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestChromemStore_ListCollections(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create some collections
	err := store.CreateCollection(ctx, "collection_a", 384)
	require.NoError(t, err)
	err = store.CreateCollection(ctx, "collection_b", 384)
	require.NoError(t, err)

	// List collections
	collections, err := store.ListCollections(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(collections), 2)
	assert.Contains(t, collections, "collection_a")
	assert.Contains(t, collections, "collection_b")
}

func TestChromemStore_GetCollectionInfo(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create collection and add docs
	err := store.CreateCollection(ctx, "info_test", 384)
	require.NoError(t, err)

	docs := []vectorstore.Document{
		{ID: "doc1", Content: "First document", Collection: "info_test"},
		{ID: "doc2", Content: "Second document", Collection: "info_test"},
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Get info
	info, err := store.GetCollectionInfo(ctx, "info_test")
	require.NoError(t, err)
	assert.Equal(t, "info_test", info.Name)
	assert.Equal(t, 384, info.VectorSize)
	assert.Equal(t, 2, info.PointCount)
}

func TestChromemStore_GetCollectionInfo_NotFound(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	_, err := store.GetCollectionInfo(ctx, "nonexistent")
	assert.ErrorIs(t, err, vectorstore.ErrCollectionNotFound)
}

func TestChromemStore_ExactSearch(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create collection and add docs
	err := store.CreateCollection(ctx, "exact_test", 384)
	require.NoError(t, err)

	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Exact search test document", Collection: "exact_test"},
	}
	_, err = store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// ExactSearch should work same as SearchInCollection for chromem
	results, err := store.ExactSearch(ctx, "exact_test", "search test", 5)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestChromemStore_Close(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)

	err := store.Close()
	assert.NoError(t, err)
}

func TestChromemStore_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_persist_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "persist_test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(), // Disable isolation for test
	}

	// Create store and add documents
	store1, err := vectorstore.NewChromemStore(config, embedder, logger)
	require.NoError(t, err)

	docs := []vectorstore.Document{
		{ID: "persist_doc", Content: "This document should persist"},
	}
	_, err = store1.AddDocuments(ctx, docs)
	require.NoError(t, err)
	store1.Close()

	// Create new store with same path - data should persist
	store2, err := vectorstore.NewChromemStore(config, embedder, logger)
	require.NoError(t, err)
	defer store2.Close()

	// Search should find the persisted document
	results, err := store2.Search(ctx, "persist", 5)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)

	found := false
	for _, r := range results {
		if r.ID == "persist_doc" {
			found = true
			break
		}
	}
	assert.True(t, found, "persisted document should be found after reopening store")
}

// TestChromemStore_ImplementsStoreInterface verifies interface compliance at compile time.
func TestChromemStore_ImplementsStoreInterface(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	// This is a compile-time check - if it compiles, the interface is satisfied
	var _ vectorstore.Store = store
}

// TestNewChromemStore_NilEmbedder verifies that nil embedder returns error.
func TestNewChromemStore_NilEmbedder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_nil_embedder_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		DefaultCollection: "test",
		VectorSize:        384,
	}

	_, err = vectorstore.NewChromemStore(config, nil, zap.NewNop())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedder is required")
}

// TestNewChromemStore_NilLogger verifies that nil logger uses no-op logger (doesn't panic).
func TestNewChromemStore_NilLogger(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_nil_logger_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		DefaultCollection: "test",
		VectorSize:        384,
		Isolation:         vectorstore.NewNoIsolation(), // Disable isolation for test
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}

	// Should not panic with nil logger
	store, err := vectorstore.NewChromemStore(config, embedder, nil)
	require.NoError(t, err)
	defer store.Close()

	// Should be able to use the store without panicking
	ctx := context.Background()
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Test document"},
	}
	_, err = store.AddDocuments(ctx, docs)
	assert.NoError(t, err)
}

// TestChromemStore_AddDocuments_MixedCollections verifies batch rejects mixed collections.
func TestChromemStore_AddDocuments_MixedCollections(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Documents with different collections should fail
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "First document", Collection: "collection_a"},
		{ID: "doc2", Content: "Second document", Collection: "collection_b"},
	}

	_, err := store.AddDocuments(ctx, docs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all documents must target the same collection")
}

// TestChromemStore_AddDocuments_SameCollection verifies batch accepts same collection.
func TestChromemStore_AddDocuments_SameCollection(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// All documents with same collection should succeed
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "First document", Collection: "same_collection"},
		{ID: "doc2", Content: "Second document", Collection: "same_collection"},
	}

	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
}

// TestChromemStore_AddDocuments_MixedEmptyCollection verifies empty collection uses default.
func TestChromemStore_AddDocuments_MixedEmptyCollection(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Mix of empty and explicit collection - empty should use first doc's collection
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "First document", Collection: "explicit_collection"},
		{ID: "doc2", Content: "Second document", Collection: ""}, // Should be OK - uses default
	}

	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
}

// TestChromemStore_CreateCollection_Duplicate verifies ErrCollectionExists on duplicate.
func TestChromemStore_CreateCollection_Duplicate(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create collection first time
	err := store.CreateCollection(ctx, "duplicate_test", 384)
	require.NoError(t, err)

	// Create same collection again should return ErrCollectionExists
	err = store.CreateCollection(ctx, "duplicate_test", 384)
	assert.ErrorIs(t, err, vectorstore.ErrCollectionExists)
}

// TestChromemStore_SearchInCollection_InvalidK verifies k validation.
func TestChromemStore_SearchInCollection_InvalidK(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create collection with docs first
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Test document"},
	}
	_, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// k=0 should fail
	_, err = store.SearchInCollection(ctx, "test_collection", "query", 0, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "k must be positive")

	// k=-1 should fail
	_, err = store.SearchInCollection(ctx, "test_collection", "query", -1, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "k must be positive")
}

// TestChromemStore_SearchInCollection_EmptyQuery verifies empty query validation.
func TestChromemStore_SearchInCollection_EmptyQuery(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create collection with docs first
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Test document"},
	}
	_, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Empty query should fail
	_, err = store.SearchInCollection(ctx, "test_collection", "", 5, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query cannot be empty")
}

// TestChromemStore_SearchInCollection_EmptyCollection verifies search on empty collection.
func TestChromemStore_SearchInCollection_EmptyCollection(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Create empty collection
	err := store.CreateCollection(ctx, "empty_collection", 384)
	require.NoError(t, err)

	// Search should return empty results, not error
	results, err := store.SearchInCollection(ctx, "empty_collection", "query", 5, nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestChromemStore_SearchInCollection_AfterAutoCreate reproduces bug #19:
// repository_search fails with "collection not found" after repository_index.
// The issue: AddDocuments auto-creates collection via getOrCreateCollection,
// but SearchInCollection uses GetCollection which may not find it.
func TestChromemStore_SearchInCollection_AfterAutoCreate(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	ctx := context.Background()

	// Add documents to a NEW collection (auto-creates via getOrCreateCollection)
	// NOTE: We deliberately do NOT call CreateCollection first - this simulates
	// the repository_index flow where AddDocuments auto-creates the collection
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "First document about Go programming", Collection: "auto_created_collection"},
		{ID: "doc2", Content: "Second document about testing", Collection: "auto_created_collection"},
	}

	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 2)

	// SearchInCollection should find the auto-created collection
	results, err := store.SearchInCollection(ctx, "auto_created_collection", "Go programming", 5, nil)
	require.NoError(t, err, "SearchInCollection should find collection created by AddDocuments")
	assert.NotEmpty(t, results, "Search should return results from the auto-created collection")
}

// =============================================================================
// Tenant Isolation Tests
// =============================================================================

// newTestChromemStoreWithIsolation creates a test store with specific isolation mode.
func newTestChromemStoreWithIsolation(t *testing.T, isolation vectorstore.IsolationMode) (*vectorstore.ChromemStore, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "chromem_isolation_test_*")
	require.NoError(t, err)

	config := vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "test_collection",
		VectorSize:        384,
		Isolation:         isolation,
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}
	store, err := vectorstore.NewChromemStore(config, embedder, zap.NewNop())
	require.NoError(t, err)

	return store, tmpDir
}

// TestChromemStore_PayloadIsolation_AddDocuments verifies tenant metadata injection.
func TestChromemStore_PayloadIsolation_AddDocuments(t *testing.T) {
	store, tmpDir := newTestChromemStoreWithIsolation(t, vectorstore.NewPayloadIsolation())
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	// Create context with tenant info
	tenant := &vectorstore.TenantInfo{
		TenantID:  "org-123",
		TeamID:    "team-1",
		ProjectID: "proj-1",
	}
	ctx := vectorstore.ContextWithTenant(context.Background(), tenant)

	// Add document
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Test document with tenant isolation"},
	}
	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 1)

	// Verify metadata was injected
	assert.Equal(t, "org-123", docs[0].Metadata["tenant_id"])
	assert.Equal(t, "team-1", docs[0].Metadata["team_id"])
	assert.Equal(t, "proj-1", docs[0].Metadata["project_id"])
}

// TestChromemStore_PayloadIsolation_FailsClosed verifies fail-closed behavior.
func TestChromemStore_PayloadIsolation_FailsClosed(t *testing.T) {
	store, tmpDir := newTestChromemStoreWithIsolation(t, vectorstore.NewPayloadIsolation())
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	// Try to add document WITHOUT tenant context - should fail
	ctx := context.Background()
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Test document"},
	}
	_, err := store.AddDocuments(ctx, docs)
	require.Error(t, err)
	assert.ErrorIs(t, err, vectorstore.ErrMissingTenant)
}

// TestChromemStore_PayloadIsolation_SearchFailsClosed verifies search fails without tenant.
func TestChromemStore_PayloadIsolation_SearchFailsClosed(t *testing.T) {
	store, tmpDir := newTestChromemStoreWithIsolation(t, vectorstore.NewPayloadIsolation())
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	// First add some data WITH tenant context
	tenant := &vectorstore.TenantInfo{TenantID: "org-123"}
	tenantCtx := vectorstore.ContextWithTenant(context.Background(), tenant)
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Test document"},
	}
	_, err := store.AddDocuments(tenantCtx, docs)
	require.NoError(t, err)

	// Search WITHOUT tenant context - should fail
	_, err = store.SearchInCollection(context.Background(), "test_collection", "test", 5, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, vectorstore.ErrMissingTenant)
}

// TestChromemStore_PayloadIsolation_SearchInjectsTenantFilter verifies filter injection.
func TestChromemStore_PayloadIsolation_SearchInjectsTenantFilter(t *testing.T) {
	store, tmpDir := newTestChromemStoreWithIsolation(t, vectorstore.NewPayloadIsolation())
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	// Add documents for two tenants with proper tenant context
	tenant1 := &vectorstore.TenantInfo{TenantID: "org-1"}
	tenant1Ctx := vectorstore.ContextWithTenant(context.Background(), tenant1)
	tenant1Docs := []vectorstore.Document{
		{ID: "t1-doc1", Content: "Tenant 1 document alpha"},
		{ID: "t1-doc2", Content: "Tenant 1 document beta"},
	}

	tenant2 := &vectorstore.TenantInfo{TenantID: "org-2"}
	tenant2Ctx := vectorstore.ContextWithTenant(context.Background(), tenant2)
	tenant2Docs := []vectorstore.Document{
		{ID: "t2-doc1", Content: "Tenant 2 document alpha"},
	}

	_, err := store.AddDocuments(tenant1Ctx, tenant1Docs)
	require.NoError(t, err)
	_, err = store.AddDocuments(tenant2Ctx, tenant2Docs)
	require.NoError(t, err)

	// Search as tenant 1 - should only find tenant 1 documents
	results, err := store.SearchInCollection(tenant1Ctx, "test_collection", "document", 10, nil)
	require.NoError(t, err)

	// Should only find tenant 1 documents
	for _, r := range results {
		tenantID, ok := r.Metadata["tenant_id"]
		if ok {
			assert.Equal(t, "org-1", tenantID, "Should only find tenant 1 documents")
		}
	}
}

// TestChromemStore_NoIsolation_AllowsEverything verifies no isolation mode.
func TestChromemStore_NoIsolation_AllowsEverything(t *testing.T) {
	store, tmpDir := newTestChromemStore(t)
	defer os.RemoveAll(tmpDir)
	defer store.Close()

	// Default is NoIsolation
	ctx := context.Background()

	// Add document without tenant context - should succeed
	docs := []vectorstore.Document{
		{ID: "doc1", Content: "Test document"},
	}
	ids, err := store.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 1)

	// Search without tenant context - should succeed
	results, err := store.SearchInCollection(ctx, "test_collection", "test", 5, nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// TestChromemStore_IsolationViaConfig verifies isolation is set at construction via config.
func TestChromemStore_IsolationViaConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chromem_isolation_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	t.Run("uses isolation from config not hardcoded default", func(t *testing.T) {
		// Set NoIsolation in config - if constructor ignores config, it will use PayloadIsolation
		config := vectorstore.ChromemConfig{
			Path:              tmpDir,
			DefaultCollection: "test",
			VectorSize:        384,
			Isolation:         vectorstore.NewNoIsolation(),
		}

		store, err := vectorstore.NewChromemStore(config, embedder, logger)
		require.NoError(t, err)
		defer store.Close()

		// Isolation should match config (none), NOT the hardcoded default (payload)
		assert.Equal(t, "none", store.IsolationMode().Mode())
	})
}
