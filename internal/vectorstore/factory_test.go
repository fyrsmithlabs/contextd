package vectorstore_test

import (
	"os"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewStore_Chromem(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "factory_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		VectorStore: config.VectorStoreConfig{
			Provider: "chromem",
			Chromem: config.ChromemConfig{
				Path:              tmpDir,
				Compress:          false,
				DefaultCollection: "test",
				VectorSize:        384,
			},
		},
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	store, err := vectorstore.NewStore(cfg, embedder, logger)
	require.NoError(t, err)
	defer store.Close()

	// Verify it's a working store
	assert.NotNil(t, store)
}

func TestNewStore_ChromemDefault(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "factory_default_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		VectorStore: config.VectorStoreConfig{
			Provider: "", // Empty should default to chromem
			Chromem: config.ChromemConfig{
				Path:              tmpDir,
				Compress:          false,
				DefaultCollection: "test",
				VectorSize:        384,
			},
		},
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	store, err := vectorstore.NewStore(cfg, embedder, logger)
	require.NoError(t, err)
	defer store.Close()

	assert.NotNil(t, store)
}

func TestNewStore_InvalidProvider(t *testing.T) {
	cfg := &config.Config{
		VectorStore: config.VectorStoreConfig{
			Provider: "invalid_provider",
		},
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	_, err := vectorstore.NewStore(cfg, embedder, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported vectorstore provider")
}

func TestNewStoreFromProvider_Chromem(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "factory_provider_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	chromemCfg := &vectorstore.ChromemConfig{
		Path:              tmpDir,
		Compress:          false,
		DefaultCollection: "test",
		VectorSize:        384,
	}

	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	store, err := vectorstore.NewStoreFromProvider("chromem", chromemCfg, nil, embedder, logger)
	require.NoError(t, err)
	defer store.Close()

	assert.NotNil(t, store)
}

func TestNewStoreFromProvider_MissingConfig(t *testing.T) {
	embedder := &chromemTestEmbedder{vectorSize: 384}
	logger := zap.NewNop()

	// Chromem without config should fail
	_, err := vectorstore.NewStoreFromProvider("chromem", nil, nil, embedder, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chromem config required")

	// Qdrant without config should fail
	_, err = vectorstore.NewStoreFromProvider("qdrant", nil, nil, embedder, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "qdrant config required")
}
