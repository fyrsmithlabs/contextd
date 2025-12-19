package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// mockTroubleshootStore is a mock implementation for troubleshoot.VectorStore.
type mockTroubleshootStore struct{}

func (m *mockTroubleshootStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) error {
	return nil
}

func (m *mockTroubleshootStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return []vectorstore.SearchResult{}, nil
}

// mockVectorStore is a mock implementation for vectorstore.Store.
type mockVectorStore struct{}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error) {
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	return ids, nil
}

func (m *mockVectorStore) Search(ctx context.Context, query string, k int) ([]vectorstore.SearchResult, error) {
	return []vectorstore.SearchResult{}, nil
}

func (m *mockVectorStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return []vectorstore.SearchResult{}, nil
}

func (m *mockVectorStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return []vectorstore.SearchResult{}, nil
}

func (m *mockVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return nil
}

func (m *mockVectorStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	return nil
}

func (m *mockVectorStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	return nil
}

func (m *mockVectorStore) DeleteCollection(ctx context.Context, collectionName string) error {
	return nil
}

func (m *mockVectorStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	return true, nil
}

func (m *mockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *mockVectorStore) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	return &vectorstore.CollectionInfo{Name: collectionName, PointCount: 0, VectorSize: 384}, nil
}

func (m *mockVectorStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error) {
	return []vectorstore.SearchResult{}, nil
}

func (m *mockVectorStore) Close() error {
	return nil
}

func (m *mockVectorStore) SetIsolationMode(mode vectorstore.IsolationMode) {
	// No-op for mock
}

func (m *mockVectorStore) IsolationMode() vectorstore.IsolationMode {
	return vectorstore.NewNoIsolation()
}

func TestNewServer(t *testing.T) {
	logger := zap.NewNop()

	// Create mock services
	troubleshootStore := &mockTroubleshootStore{}
	vectorStore := &mockVectorStore{}

	checkpointSvc, err := checkpoint.NewServiceWithStore(checkpoint.DefaultServiceConfig(), vectorStore, logger)
	require.NoError(t, err)

	remediationSvc, err := remediation.NewService(remediation.DefaultServiceConfig(), vectorStore, logger)
	require.NoError(t, err)

	repositorySvc := repository.NewService(vectorStore)
	troubleshootSvc, err := troubleshoot.NewService(troubleshootStore, logger, nil)
	require.NoError(t, err)
	reasoningbankSvc, err := reasoningbank.NewService(vectorStore, logger)
	require.NoError(t, err)
	scrubber := secrets.MustNew(secrets.DefaultConfig())

	// Test server creation
	// Note: foldingSvc is optional, so we pass nil for most tests
	t.Run("successful creation", func(t *testing.T) {
		cfg := &Config{
			Name:    "test-server",
			Version: "1.0.0",
			Logger:  logger,
		}

		server, err := NewServer(cfg, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, nil, scrubber)
		require.NoError(t, err)
		require.NotNil(t, server)
		require.NotNil(t, server.mcp)
		require.Equal(t, "test-server", cfg.Name)

		// Clean up
		require.NoError(t, server.Close())
	})

	t.Run("nil config uses defaults", func(t *testing.T) {
		server, err := NewServer(nil, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, nil, scrubber)
		require.NoError(t, err)
		require.NotNil(t, server)

		// Clean up
		require.NoError(t, server.Close())
	})

	t.Run("missing checkpoint service", func(t *testing.T) {
		cfg := DefaultConfig()
		_, err := NewServer(cfg, nil, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, nil, scrubber)
		require.Error(t, err)
		require.Contains(t, err.Error(), "checkpoint service is required")
	})

	t.Run("missing remediation service", func(t *testing.T) {
		cfg := DefaultConfig()
		_, err := NewServer(cfg, checkpointSvc, nil, repositorySvc, troubleshootSvc, reasoningbankSvc, nil, scrubber)
		require.Error(t, err)
		require.Contains(t, err.Error(), "remediation service is required")
	})

	t.Run("missing repository service", func(t *testing.T) {
		cfg := DefaultConfig()
		_, err := NewServer(cfg, checkpointSvc, remediationSvc, nil, troubleshootSvc, reasoningbankSvc, nil, scrubber)
		require.Error(t, err)
		require.Contains(t, err.Error(), "repository service is required")
	})

	t.Run("missing troubleshoot service", func(t *testing.T) {
		cfg := DefaultConfig()
		_, err := NewServer(cfg, checkpointSvc, remediationSvc, repositorySvc, nil, reasoningbankSvc, nil, scrubber)
		require.Error(t, err)
		require.Contains(t, err.Error(), "troubleshoot service is required")
	})

	t.Run("missing reasoningbank service", func(t *testing.T) {
		cfg := DefaultConfig()
		_, err := NewServer(cfg, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, nil, nil, scrubber)
		require.Error(t, err)
		require.Contains(t, err.Error(), "reasoningbank service is required")
	})

	t.Run("missing scrubber", func(t *testing.T) {
		cfg := DefaultConfig()
		_, err := NewServer(cfg, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "scrubber is required")
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)
	require.Equal(t, "contextd-v2", cfg.Name)
	require.Equal(t, "1.0.0", cfg.Version)
	require.NotNil(t, cfg.Logger)
}

func TestServerClose(t *testing.T) {
	logger := zap.NewNop()

	// Create mock services
	troubleshootStore := &mockTroubleshootStore{}
	vectorStore := &mockVectorStore{}

	checkpointSvc, err := checkpoint.NewServiceWithStore(checkpoint.DefaultServiceConfig(), vectorStore, logger)
	require.NoError(t, err)

	remediationSvc, err := remediation.NewService(remediation.DefaultServiceConfig(), vectorStore, logger)
	require.NoError(t, err)

	repositorySvc := repository.NewService(vectorStore)
	troubleshootSvc, err := troubleshoot.NewService(troubleshootStore, logger, nil)
	require.NoError(t, err)
	reasoningbankSvc, err := reasoningbank.NewService(vectorStore, logger)
	require.NoError(t, err)
	scrubber := secrets.MustNew(secrets.DefaultConfig())

	server, err := NewServer(nil, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, nil, scrubber)
	require.NoError(t, err)

	// Close should succeed
	err = server.Close()
	require.NoError(t, err)

	// Second close should also succeed (idempotent)
	err = server.Close()
	require.NoError(t, err)
}
