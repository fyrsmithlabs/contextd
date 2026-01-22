package vectorstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestMetadataHealthChecker_AllHealthy(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create two healthy collections
	for i, hash := range []string{"healthy1", "healthy2"} {
		collPath := filepath.Join(path, hash)
		require.NoError(t, os.MkdirAll(collPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(collPath, "00000000.gob"), []byte("metadata"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(collPath, "abcd"+string(rune('0'+i))+".gob"), []byte("document"), 0644))
	}

	checker := NewMetadataHealthChecker(path, logger)
	health, err := checker.Check(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, health.Total)
	assert.Equal(t, 2, health.HealthyCount)
	assert.Equal(t, 0, health.CorruptCount)
	assert.True(t, health.IsHealthy())
	assert.Equal(t, "healthy", health.Status())
	assert.Len(t, health.Healthy, 2)
	assert.Empty(t, health.Corrupt)
}

func TestMetadataHealthChecker_CorruptCollection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create healthy collection
	healthyPath := filepath.Join(path, "healthy1")
	require.NoError(t, os.MkdirAll(healthyPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "00000000.gob"), []byte("metadata"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "doc1.gob"), []byte("document"), 0644))

	// Create corrupt collection (documents but no metadata)
	corruptPath := filepath.Join(path, "corrupt1")
	require.NoError(t, os.MkdirAll(corruptPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(corruptPath, "doc2.gob"), []byte("document"), 0644))

	checker := NewMetadataHealthChecker(path, logger)
	health, err := checker.Check(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, health.Total)
	assert.Equal(t, 1, health.HealthyCount)
	assert.Equal(t, 1, health.CorruptCount)
	assert.False(t, health.IsHealthy())
	assert.Equal(t, "degraded", health.Status())
	assert.Contains(t, health.Healthy, "healthy1")
	assert.Contains(t, health.Corrupt, "corrupt1")
	assert.Contains(t, health.Details["corrupt1"], "corrupt: 1 documents, no metadata")
}

func TestMetadataHealthChecker_EmptyCollection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create empty collection (no metadata, no documents)
	emptyPath := filepath.Join(path, "empty1")
	require.NoError(t, os.MkdirAll(emptyPath, 0755))

	checker := NewMetadataHealthChecker(path, logger)
	health, err := checker.Check(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, health.Total)
	assert.Equal(t, 0, health.HealthyCount)
	assert.Equal(t, 0, health.CorruptCount)
	assert.True(t, health.IsHealthy()) // Empty is not corrupt
	assert.Len(t, health.Empty, 1)
	assert.Contains(t, health.Empty, "empty1")
}

func TestMetadataHealthChecker_MixedState(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create healthy collection
	healthyPath := filepath.Join(path, "healthy1")
	require.NoError(t, os.MkdirAll(healthyPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "00000000.gob"), []byte("metadata"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "doc1.gob"), []byte("doc"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(healthyPath, "doc2.gob"), []byte("doc"), 0644))

	// Create corrupt collection
	corruptPath := filepath.Join(path, "corrupt1")
	require.NoError(t, os.MkdirAll(corruptPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(corruptPath, "doc3.gob"), []byte("doc"), 0644))

	// Create empty collection
	emptyPath := filepath.Join(path, "empty1")
	require.NoError(t, os.MkdirAll(emptyPath, 0755))

	// Create hidden directory (should be ignored)
	hiddenPath := filepath.Join(path, ".quarantine")
	require.NoError(t, os.MkdirAll(hiddenPath, 0755))

	checker := NewMetadataHealthChecker(path, logger)
	health, err := checker.Check(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 3, health.Total) // Should not count .quarantine
	assert.Equal(t, 1, health.HealthyCount)
	assert.Equal(t, 1, health.CorruptCount)
	assert.False(t, health.IsHealthy())
	assert.Equal(t, "degraded", health.Status())
	assert.Len(t, health.Healthy, 1)
	assert.Len(t, health.Corrupt, 1)
	assert.Len(t, health.Empty, 1)
}

func TestMetadataHealthChecker_SkipsFiles(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := t.TempDir()

	// Create a healthy collection
	collPath := filepath.Join(path, "healthy1")
	require.NoError(t, os.MkdirAll(collPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collPath, "00000000.gob"), []byte("metadata"), 0644))

	// Create a file in the root (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(path, "somefile.txt"), []byte("data"), 0644))

	checker := NewMetadataHealthChecker(path, logger)
	health, err := checker.Check(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, health.Total) // Should only count the directory
	assert.Equal(t, 1, health.HealthyCount)
}
