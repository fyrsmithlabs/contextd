package vectorstore

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestFallbackStore_AddDocuments_RemoteHealthy tests adding documents when remote is healthy.
func TestFallbackStore_AddDocuments_RemoteHealthy(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create mock embedder
	embedder := &MockEmbedder{
		embedding: make([]float32, 384),
	}

	// Create remote and local stores
	remoteCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_remote",
		VectorSize:        384,
	}
	remote, err := NewChromemStore(remoteCfg, embedder, logger)
	require.NoError(t, err)
	defer remote.Close()

	localCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_local",
		VectorSize:        384,
	}
	local, err := NewChromemStore(localCfg, embedder, logger)
	require.NoError(t, err)
	defer local.Close()

	// Create health monitor (healthy)
	healthChecker := NewMockHealthChecker()
	healthChecker.SetHealthy(true)
	health := NewHealthMonitor(ctx, healthChecker, 30*time.Second, logger)

	// Create WAL
	scrubber := secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), &scrubber, logger)
	require.NoError(t, err)

	// Create fallback config
	config := FallbackConfig{
		Enabled:             true,
		LocalPath:           localCfg.Path,
		SyncOnConnect:       true,
		HealthCheckInterval: "30s",
		WALPath:             t.TempDir(),
		WALRetentionDays:    7,
	}

	// Create fallback store
	fs, err := NewFallbackStore(ctx, remote, local, health, wal, config, logger)
	require.NoError(t, err)
	defer fs.Close()

	// Add tenant context
	tenant := &TenantInfo{
		TenantID:  "test-tenant",
		ProjectID: "test-project",
	}
	ctx = ContextWithTenant(ctx, tenant)

	// Add documents
	docs := []Document{
		{ID: "doc1", Content: "test content 1", Metadata: map[string]interface{}{"key": "value1"}},
		{ID: "doc2", Content: "test content 2", Metadata: map[string]interface{}{"key": "value2"}},
	}

	ids, err := fs.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 2)

	// Verify documents are in remote store
	remoteResults, err := remote.Search(ctx, "test", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(remoteResults), 2)
}

// TestFallbackStore_AddDocuments_RemoteUnhealthy tests adding documents when remote is unhealthy.
func TestFallbackStore_AddDocuments_RemoteUnhealthy(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create mock embedder
	embedder := &MockEmbedder{
		embedding: make([]float32, 384),
	}

	// Create remote and local stores
	remoteCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_remote",
		VectorSize:        384,
	}
	remote, err := NewChromemStore(remoteCfg, embedder, logger)
	require.NoError(t, err)
	defer remote.Close()

	localCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_local",
		VectorSize:        384,
	}
	local, err := NewChromemStore(localCfg, embedder, logger)
	require.NoError(t, err)
	defer local.Close()

	// Create health monitor (unhealthy)
	healthChecker := NewMockHealthChecker()
	healthChecker.SetHealthy(false)
	health := NewHealthMonitor(ctx, healthChecker, 30*time.Second, logger)

	// Create WAL
	scrubber := secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), &scrubber, logger)
	require.NoError(t, err)

	// Create fallback config
	config := FallbackConfig{
		Enabled:             true,
		LocalPath:           localCfg.Path,
		SyncOnConnect:       true,
		HealthCheckInterval: "30s",
		WALPath:             t.TempDir(),
		WALRetentionDays:    7,
	}

	// Create fallback store
	fs, err := NewFallbackStore(ctx, remote, local, health, wal, config, logger)
	require.NoError(t, err)
	defer fs.Close()

	// Add tenant context
	tenant := &TenantInfo{
		TenantID:  "test-tenant",
		ProjectID: "test-project",
	}
	ctx = ContextWithTenant(ctx, tenant)

	// Add documents
	docs := []Document{
		{ID: "doc3", Content: "test content 3", Metadata: map[string]interface{}{"key": "value3"}},
		{ID: "doc4", Content: "test content 4", Metadata: map[string]interface{}{"key": "value4"}},
	}

	ids, err := fs.AddDocuments(ctx, docs)
	require.NoError(t, err)
	assert.Len(t, ids, 2)

	// Verify documents are in local store (not remote)
	localResults, err := local.Search(ctx, "test", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(localResults), 2)

	// Verify WAL has pending entries
	pending := wal.PendingEntries()
	assert.Greater(t, len(pending), 0)
}

// TestFallbackStore_Search_RemoteHealthy tests searching when remote is healthy.
func TestFallbackStore_Search_RemoteHealthy(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create mock embedder
	embedder := &MockEmbedder{
		embedding: make([]float32, 384),
	}

	// Create remote and local stores
	remoteCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_remote",
		VectorSize:        384,
	}
	remote, err := NewChromemStore(remoteCfg, embedder, logger)
	require.NoError(t, err)
	defer remote.Close()

	localCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_local",
		VectorSize:        384,
	}
	local, err := NewChromemStore(localCfg, embedder, logger)
	require.NoError(t, err)
	defer local.Close()

	// Create health monitor (healthy)
	healthChecker := NewMockHealthChecker()
	healthChecker.SetHealthy(true)
	health := NewHealthMonitor(ctx, healthChecker, 30*time.Second, logger)

	// Create WAL
	scrubber := secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), &scrubber, logger)
	require.NoError(t, err)

	// Create fallback config
	config := FallbackConfig{
		Enabled:             true,
		LocalPath:           localCfg.Path,
		SyncOnConnect:       true,
		HealthCheckInterval: "30s",
		WALPath:             t.TempDir(),
		WALRetentionDays:    7,
	}

	// Create fallback store
	fs, err := NewFallbackStore(ctx, remote, local, health, wal, config, logger)
	require.NoError(t, err)
	defer fs.Close()

	// Add tenant context
	tenant := &TenantInfo{
		TenantID:  "test-tenant",
		ProjectID: "test-project",
	}
	ctx = ContextWithTenant(ctx, tenant)

	// Add documents to remote
	docs := []Document{
		{ID: "doc5", Content: "search test content", Metadata: map[string]interface{}{"key": "value5"}},
	}
	_, err = remote.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search via fallback store (should hit remote)
	results, err := fs.Search(ctx, "search", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

// TestFallbackStore_Search_RemoteUnhealthy tests searching when remote is unhealthy.
func TestFallbackStore_Search_RemoteUnhealthy(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create mock embedder
	embedder := &MockEmbedder{
		embedding: make([]float32, 384),
	}

	// Create remote and local stores
	remoteCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_remote",
		VectorSize:        384,
	}
	remote, err := NewChromemStore(remoteCfg, embedder, logger)
	require.NoError(t, err)
	defer remote.Close()

	localCfg := ChromemConfig{
		Path:              t.TempDir(),
		DefaultCollection: "test_local",
		VectorSize:        384,
	}
	local, err := NewChromemStore(localCfg, embedder, logger)
	require.NoError(t, err)
	defer local.Close()

	// Create health monitor (unhealthy)
	healthChecker := NewMockHealthChecker()
	healthChecker.SetHealthy(false)
	health := NewHealthMonitor(ctx, healthChecker, 30*time.Second, logger)

	// Create WAL
	scrubber := secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), &scrubber, logger)
	require.NoError(t, err)

	// Create fallback config
	config := FallbackConfig{
		Enabled:             true,
		LocalPath:           localCfg.Path,
		SyncOnConnect:       true,
		HealthCheckInterval: "30s",
		WALPath:             t.TempDir(),
		WALRetentionDays:    7,
	}

	// Create fallback store
	fs, err := NewFallbackStore(ctx, remote, local, health, wal, config, logger)
	require.NoError(t, err)
	defer fs.Close()

	// Add tenant context
	tenant := &TenantInfo{
		TenantID:  "test-tenant",
		ProjectID: "test-project",
	}
	ctx = ContextWithTenant(ctx, tenant)

	// Add documents to local store
	docs := []Document{
		{ID: "doc6", Content: "local search test", Metadata: map[string]interface{}{"key": "value6"}},
	}
	_, err = local.AddDocuments(ctx, docs)
	require.NoError(t, err)

	// Search via fallback store (should hit local)
	results, err := fs.Search(ctx, "local", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)

	// Verify metadata indicates local source
	if len(results) > 0 {
		assert.Equal(t, "local", results[0].Metadata["source"])
		assert.Equal(t, true, results[0].Metadata["stale_warning"])
	}
}

// MockEmbedder for testing
type MockEmbedder struct {
	embedding []float32
	err       error
}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	results := make([][]float32, len(texts))
	for i := range texts {
		results[i] = m.embedding
	}
	return results, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embedding, nil
}
