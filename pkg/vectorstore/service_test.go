package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

func TestNewService(t *testing.T) {
	// Create a test embedder
	llm, err := openai.New(
		openai.WithBaseURL("http://localhost:8080/v1"),
		openai.WithModel("BAAI/bge-small-en-v1.5"),
		openai.WithToken("placeholder"),
	)
	require.NoError(t, err)

	embedder, err := embeddings.NewEmbedder(llm)
	require.NoError(t, err)

	tests := []struct {
		name       string
		config     Config
		wantErr    bool
		errMessage string
	}{
		{
			name: "valid Qdrant configuration",
			config: Config{
				URL:            "http://localhost:6333",
				CollectionName: "test_collection",
				Embedder:       embedder,
			},
			wantErr: false,
		},
		{
			name: "empty URL",
			config: Config{
				URL:            "",
				CollectionName: "test",
			},
			wantErr:    true,
			errMessage: "URL required",
		},
		{
			name: "empty collection name",
			config: Config{
				URL:            "http://localhost:6333",
				CollectionName: "",
			},
			wantErr:    true,
			errMessage: "collection name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		collectionName string
		want           Config
	}{
		{
			name: "default configuration",
			envVars: map[string]string{
				"QDRANT_URL": "",
			},
			collectionName: "test_collection",
			want: Config{
				URL:            "http://localhost:6333",
				CollectionName: "test_collection",
			},
		},
		{
			name: "custom URL",
			envVars: map[string]string{
				"QDRANT_URL": "http://custom:9090",
			},
			collectionName: "custom_collection",
			want: Config{
				URL:            "http://custom:9090",
				CollectionName: "custom_collection",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				if v != "" {
					os.Setenv(k, v)
					defer os.Unsetenv(k)
				}
			}

			got := ConfigFromEnv(tt.collectionName)
			assert.Equal(t, tt.want.URL, got.URL)
			assert.Equal(t, tt.want.CollectionName, got.CollectionName)
		})
	}
}

func TestService_AddDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	ctx := context.Background()

	// Create collection in Qdrant (384 dimensions for TEI BAAI/bge-small-en-v1.5)
	createQdrantCollection(t, "test_add_docs", 384)

	// Create embedder
	embedSvc, err := createTestEmbedder()
	require.NoError(t, err)

	config := Config{
		URL:            getQdrantURL(),
		CollectionName: "test_add_docs",
		Embedder:       embedSvc,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	// Test: Add documents
	docs := []Document{
		{
			ID:       "doc1",
			Content:  "first document content",
			Metadata: map[string]interface{}{"owner": "alice", "file": "main.go"},
		},
		{
			ID:       "doc2",
			Content:  "second document content",
			Metadata: map[string]interface{}{"owner": "alice", "file": "server.go"},
		},
	}

	err = service.AddDocuments(ctx, docs)
	require.NoError(t, err)

	t.Logf("Successfully added %d documents", len(docs))
}

func TestService_Search(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	ctx := context.Background()

	// Create collection in Qdrant (384 dimensions for TEI BAAI/bge-small-en-v1.5)
	createQdrantCollection(t, "test_search", 384)

	// Create embedder
	embedSvc, err := createTestEmbedder()
	require.NoError(t, err)

	config := Config{
		URL:            getQdrantURL(),
		CollectionName: "test_search",
		Embedder:       embedSvc,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	// Add test documents
	docs := []Document{
		{
			ID:       "doc1",
			Content:  "golang programming language",
			Metadata: map[string]interface{}{"owner": "alice", "topic": "programming"},
		},
		{
			ID:       "doc2",
			Content:  "python data science",
			Metadata: map[string]interface{}{"owner": "bob", "topic": "data"},
		},
	}
	require.NoError(t, service.AddDocuments(ctx, docs))

	// Test: Search
	results, err := service.Search(ctx, "golang", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	t.Logf("Found %d results", len(results))
	for i, r := range results {
		t.Logf("Result %d: ID=%s, Score=%.4f", i+1, r.ID, r.Score)
	}
}

func TestService_SearchWithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	ctx := context.Background()

	// Create collection in Qdrant (384 dimensions for TEI BAAI/bge-small-en-v1.5)
	createQdrantCollection(t, "test_search_filters", 384)

	// Create embedder
	embedSvc, err := createTestEmbedder()
	require.NoError(t, err)

	config := Config{
		URL:            getQdrantURL(),
		CollectionName: "test_search_filters",
		Embedder:       embedSvc,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	// Add test documents
	docs := []Document{
		{
			ID:       "doc1",
			Content:  "test content for alice",
			Metadata: map[string]interface{}{"owner": "alice", "project": "contextd"},
		},
		{
			ID:       "doc2",
			Content:  "test content for bob",
			Metadata: map[string]interface{}{"owner": "bob", "project": "other"},
		},
	}
	require.NoError(t, service.AddDocuments(ctx, docs))

	// Test: Search with owner filter
	// Note: Qdrant expects filters in boolean query format
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "owner",
				"match": map[string]interface{}{
					"value": "alice",
				},
			},
		},
	}
	results, err := service.SearchWithFilters(ctx, "test content", 5, filters)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Verify all results belong to alice
	for _, r := range results {
		owner, ok := r.Metadata["owner"]
		require.True(t, ok, "owner metadata should exist")
		assert.Equal(t, "alice", owner, "all results should belong to alice")
	}

	t.Logf("Found %d filtered results for alice", len(results))
}

func TestService_DeleteDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup
	ctx := context.Background()

	// Create collection in Qdrant (384 dimensions for TEI BAAI/bge-small-en-v1.5)
	createQdrantCollection(t, "test_delete", 384)

	// Create embedder
	embedSvc, err := createTestEmbedder()
	require.NoError(t, err)

	config := Config{
		URL:            getQdrantURL(),
		CollectionName: "test_delete",
		Embedder:       embedSvc,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	// Add test document
	docs := []Document{
		{
			ID:       "doc_to_delete",
			Content:  "temporary content",
			Metadata: map[string]interface{}{"owner": "alice"},
		},
	}
	require.NoError(t, service.AddDocuments(ctx, docs))

	// Test: Delete document
	err = service.DeleteDocuments(ctx, []string{"doc_to_delete"})
	require.NoError(t, err)

	t.Log("Successfully deleted document")
}

// Helper function to get Qdrant URL from environment
func getQdrantURL() string {
	url := os.Getenv("QDRANT_URL")
	if url == "" {
		url = "http://localhost:6333"
	}
	return url
}

// createQdrantCollection creates a collection in Qdrant via REST API.
// This is needed because langchaingo's Qdrant store doesn't auto-create collections.
func createQdrantCollection(t *testing.T, collectionName string, vectorSize int) {
	t.Helper()

	qdrantURL := getQdrantURL()

	// Collection creation payload
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	// Create collection
	url := fmt.Sprintf("%s/collections/%s", qdrantURL, collectionName)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 200 OK or 409 Conflict (already exists) are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		t.Fatalf("failed to create collection: HTTP %d", resp.StatusCode)
	}

	// Register cleanup to delete collection after test
	t.Cleanup(func() {
		deleteQdrantCollection(t, collectionName)
	})
}

// deleteQdrantCollection deletes a collection from Qdrant via REST API.
func deleteQdrantCollection(t *testing.T, collectionName string) {
	t.Helper()

	qdrantURL := getQdrantURL()
	url := fmt.Sprintf("%s/collections/%s", qdrantURL, collectionName)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Logf("failed to create delete request: %v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("failed to delete collection: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Logf("unexpected status when deleting collection: HTTP %d", resp.StatusCode)
	}
}

// Helper function to create test embedder
func createTestEmbedder() (*embeddings.EmbedderImpl, error) {
	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}

	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "BAAI/bge-small-en-v1.5"
	}

	llm, err := openai.New(
		openai.WithBaseURL(baseURL),
		openai.WithModel(model),
		openai.WithToken("placeholder"),
	)
	if err != nil {
		return nil, err
	}

	return embeddings.NewEmbedder(llm)
}
