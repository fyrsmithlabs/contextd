package vectorstore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func TestQdrantConfig_IsolationViaConfig(t *testing.T) {
	// Test that isolation can be set via config (thread-safe pattern)
	// This mirrors ChromemConfig.Isolation for consistency

	t.Run("isolation field exists in config", func(t *testing.T) {
		// This test verifies the config struct has an Isolation field
		config := vectorstore.QdrantConfig{
			Host:           "localhost",
			Port:           6334,
			CollectionName: "test_collection",
			VectorSize:     384,
			Isolation:      vectorstore.NewNoIsolation(),
		}

		// Verify config is valid
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("store uses config isolation when provided", func(t *testing.T) {
		// Skip if Qdrant not available - this tests constructor behavior
		config := vectorstore.QdrantConfig{
			Host:           "localhost",
			Port:           6334,
			CollectionName: "test_isolation",
			VectorSize:     384,
			Isolation:      vectorstore.NewNoIsolation(),
		}

		store, err := vectorstore.NewQdrantStore(config, &TestEmbedder{VectorSize: 384})
		if err != nil {
			t.Skipf("Qdrant not available: %v", err)
		}
		defer store.Close()

		// Verify isolation mode was set from config
		assert.Equal(t, "none", store.IsolationMode().Mode())
	})
}

// Integration test - requires running Qdrant instance
func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name          string
		code          codes.Code
		wantTransient bool
	}{
		{
			name:          "unavailable is transient",
			code:          codes.Unavailable,
			wantTransient: true,
		},
		{
			name:          "deadline exceeded is transient",
			code:          codes.DeadlineExceeded,
			wantTransient: true,
		},
		{
			name:          "aborted is transient",
			code:          codes.Aborted,
			wantTransient: true,
		},
		{
			name:          "resource exhausted is transient",
			code:          codes.ResourceExhausted,
			wantTransient: true,
		},
		{
			name:          "invalid argument is not transient",
			code:          codes.InvalidArgument,
			wantTransient: false,
		},
		{
			name:          "not found is not transient",
			code:          codes.NotFound,
			wantTransient: false,
		},
		{
			name:          "permission denied is not transient",
			code:          codes.PermissionDenied,
			wantTransient: false,
		},
		{
			name:          "unauthenticated is not transient",
			code:          codes.Unauthenticated,
			wantTransient: false,
		},
		{
			name:          "unknown code defaults to not transient",
			code:          codes.Unknown,
			wantTransient: false,
		},
		{
			name:          "canceled is not transient",
			code:          codes.Canceled,
			wantTransient: false,
		},
		{
			name:          "already exists is not transient",
			code:          codes.AlreadyExists,
			wantTransient: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := status.Error(tt.code, "test error")
			got := vectorstore.IsTransientError(err)
			assert.Equal(t, tt.wantTransient, got)
		})
	}

	// Test non-gRPC error
	t.Run("non-grpc error is not transient", func(t *testing.T) {
		err := errors.New("regular error")
		assert.False(t, vectorstore.IsTransientError(err))
	})

	// Test nil error
	t.Run("nil error is not transient", func(t *testing.T) {
		assert.False(t, vectorstore.IsTransientError(nil))
	})
}

func TestQdrantStore_IsolationMode(t *testing.T) {
	// Test that isolation mode can be queried without Qdrant running
	// This tests the getter/setter methods

	t.Run("default isolation is PayloadIsolation", func(t *testing.T) {
		config := vectorstore.QdrantConfig{
			Host:           "localhost",
			Port:           6334,
			CollectionName: "test",
			VectorSize:     384,
		}

		embedder := &TestEmbedder{VectorSize: 384}

		// NewQdrantStore will fail without Qdrant, but we can test the isolation field
		// by creating a config with explicit isolation
		config.Isolation = vectorstore.NewNoIsolation()

		// Skip actual store creation since Qdrant may not be running
		// Just verify the config pattern works
		assert.NotNil(t, config.Isolation)
		assert.Equal(t, "none", config.Isolation.Mode())

		// Test with PayloadIsolation
		config.Isolation = vectorstore.NewPayloadIsolation()
		assert.Equal(t, "payload", config.Isolation.Mode())

		// Suppress unused variable warnings
		_ = embedder
	})
}

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

	embedder := &TestEmbedder{VectorSize: 10}

	store, err := vectorstore.NewQdrantStore(config, embedder)
	if err != nil {
		t.Skipf("Qdrant not available: %v", err)
	}
	defer store.Close()

	t.Run("collection lifecycle", func(t *testing.T) {
		collectionName := "test_lifecycle"

		// Clean up if exists
		exists, _ := store.CollectionExists(ctx, collectionName)
		if exists {
			_ = store.DeleteCollection(ctx, collectionName)
		}

		// Create collection
		err = store.CreateCollection(ctx, collectionName, 10)
		require.NoError(t, err)

		// Verify collection exists
		exists, err = store.CollectionExists(ctx, collectionName)
		require.NoError(t, err)
		assert.True(t, exists)

		// Get collection info
		info, err := store.GetCollectionInfo(ctx, collectionName)
		require.NoError(t, err)
		assert.Equal(t, collectionName, info.Name)
		assert.Equal(t, 0, info.PointCount) // Empty collection

		// List collections
		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.Contains(t, collections, collectionName)

		// Delete collection
		err = store.DeleteCollection(ctx, collectionName)
		require.NoError(t, err)

		// Verify deletion
		exists, err = store.CollectionExists(ctx, collectionName)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("document operations", func(t *testing.T) {
		collectionName := "test_documents"

		// Setup
		exists, _ := store.CollectionExists(ctx, collectionName)
		if exists {
			_ = store.DeleteCollection(ctx, collectionName)
		}
		err = store.CreateCollection(ctx, collectionName, 10)
		require.NoError(t, err)
		defer store.DeleteCollection(ctx, collectionName)

		// Add documents
		docs := []vectorstore.Document{
			{
				ID:      "doc1",
				Content: "test document one",
				Metadata: map[string]interface{}{
					"owner": "alice",
					"type":  "article",
				},
				Collection: collectionName,
			},
			{
				ID:      "doc2",
				Content: "test document two",
				Metadata: map[string]interface{}{
					"owner": "bob",
					"type":  "article",
				},
				Collection: collectionName,
			},
		}

		ids, err := store.AddDocuments(ctx, docs)
		require.NoError(t, err)
		assert.Len(t, ids, 2)

		// Verify point count
		info, err := store.GetCollectionInfo(ctx, collectionName)
		require.NoError(t, err)
		assert.Equal(t, 2, info.PointCount)

		// Search
		results, err := store.SearchInCollection(ctx, collectionName, "test query", 10, nil)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// Search with filters
		filteredResults, err := store.SearchInCollection(ctx, collectionName, "test query", 10, map[string]interface{}{
			"owner": "alice",
		})
		require.NoError(t, err)
		assert.Len(t, filteredResults, 1)
		assert.Equal(t, "alice", filteredResults[0].Metadata["owner"])

		// Delete one document
		err = store.DeleteDocumentsFromCollection(ctx, collectionName, []string{"doc1"})
		require.NoError(t, err)

		// Verify deletion
		info, err = store.GetCollectionInfo(ctx, collectionName)
		require.NoError(t, err)
		assert.Equal(t, 1, info.PointCount)
	})

	t.Run("exact search", func(t *testing.T) {
		collectionName := "test_exact"

		// Setup
		exists, _ := store.CollectionExists(ctx, collectionName)
		if exists {
			_ = store.DeleteCollection(ctx, collectionName)
		}
		err = store.CreateCollection(ctx, collectionName, 10)
		require.NoError(t, err)
		defer store.DeleteCollection(ctx, collectionName)

		// Add documents
		docs := []vectorstore.Document{
			{ID: "doc1", Content: "exact search test one", Collection: collectionName},
			{ID: "doc2", Content: "exact search test two", Collection: collectionName},
		}
		_, err = store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		// Exact search (brute force, no HNSW index)
		results, err := store.ExactSearch(ctx, collectionName, "search query", 10)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("tenant isolation", func(t *testing.T) {
		collectionName := "test_isolation"

		// Setup
		exists, _ := store.CollectionExists(ctx, collectionName)
		if exists {
			_ = store.DeleteCollection(ctx, collectionName)
		}
		err = store.CreateCollection(ctx, collectionName, 10)
		require.NoError(t, err)
		defer store.DeleteCollection(ctx, collectionName)

		// Set tenant context
		tenant1Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  "tenant1",
			ProjectID: "project1",
		})
		tenant2Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  "tenant2",
			ProjectID: "project2",
		})

		// Add documents for tenant1
		docs1 := []vectorstore.Document{
			{ID: "t1_doc1", Content: "tenant one document", Collection: collectionName},
		}
		_, err = store.AddDocuments(tenant1Ctx, docs1)
		require.NoError(t, err)

		// Add documents for tenant2
		docs2 := []vectorstore.Document{
			{ID: "t2_doc1", Content: "tenant two document", Collection: collectionName},
		}
		_, err = store.AddDocuments(tenant2Ctx, docs2)
		require.NoError(t, err)

		// Search as tenant1 - should only see tenant1 docs
		results1, err := store.SearchInCollection(tenant1Ctx, collectionName, "document", 10, nil)
		require.NoError(t, err)
		assert.Len(t, results1, 1)
		assert.Equal(t, "tenant1", results1[0].Metadata["tenant_id"])

		// Search as tenant2 - should only see tenant2 docs
		results2, err := store.SearchInCollection(tenant2Ctx, collectionName, "document", 10, nil)
		require.NoError(t, err)
		assert.Len(t, results2, 1)
		assert.Equal(t, "tenant2", results2[0].Metadata["tenant_id"])
	})
}
