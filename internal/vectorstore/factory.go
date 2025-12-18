// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"go.uber.org/zap"
)

// StoreOption configures a Store after creation.
type StoreOption func(store Store)

// WithIsolation sets the isolation mode for a store.
// Use NewPayloadIsolation() for multi-tenant payload filtering,
// NewFilesystemIsolation() for database-per-project isolation,
// or NewNoIsolation() for testing only.
func WithIsolation(mode IsolationMode) StoreOption {
	return func(store Store) {
		store.SetIsolationMode(mode)
	}
}

// NewStore creates a new Store based on the configuration.
//
// This factory function examines the VectorStoreConfig.Provider field and
// creates the appropriate store implementation:
//   - "chromem" (default): Creates an embedded ChromemStore (no external deps)
//   - "qdrant": Creates a QdrantStore (requires external Qdrant server)
//
// Tenant Isolation:
//
// By default, stores use PayloadIsolation mode for fail-closed security.
// All operations require tenant context (TenantInfo in ctx) or return ErrMissingTenant.
// To disable isolation for testing, use the WithIsolation option:
//
//	store, err := vectorstore.NewStore(cfg, embedder, logger,
//	    vectorstore.WithIsolation(vectorstore.NewNoIsolation()))  // Testing only!
//
// The chromem provider is recommended for most users as it requires no setup:
//
//	brew install contextd  # Just works!
//
// Example usage:
//
//	cfg := config.Load()
//	store, err := vectorstore.NewStore(cfg, embedder, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
func NewStore(cfg *config.Config, embedder Embedder, logger *zap.Logger, opts ...StoreOption) (Store, error) {
	var store Store
	var err error

	switch cfg.VectorStore.Provider {
	case "chromem", "":
		// Default: chromem (embedded, zero external dependencies)
		chromemCfg := ChromemConfig{
			Path:              cfg.VectorStore.Chromem.Path,
			Compress:          cfg.VectorStore.Chromem.Compress,
			DefaultCollection: cfg.VectorStore.Chromem.DefaultCollection,
			VectorSize:        cfg.VectorStore.Chromem.VectorSize,
		}
		store, err = NewChromemStore(chromemCfg, embedder, logger)

	case "qdrant":
		// Qdrant: requires external Qdrant server
		qdrantCfg := QdrantConfig{
			Host:           cfg.Qdrant.Host,
			Port:           cfg.Qdrant.Port,
			CollectionName: cfg.Qdrant.CollectionName,
			VectorSize:     cfg.Qdrant.VectorSize,
		}
		store, err = NewQdrantStore(qdrantCfg, embedder)

	default:
		return nil, fmt.Errorf("unsupported vectorstore provider: %s (supported: chromem, qdrant)", cfg.VectorStore.Provider)
	}

	if err != nil {
		return nil, err
	}

	// Apply options (e.g., isolation mode)
	for _, opt := range opts {
		opt(store)
	}

	return store, nil
}

// NewStoreFromProvider creates a store directly from provider name and specific config.
// This is useful when you need more control over configuration.
func NewStoreFromProvider(provider string, chromemCfg *ChromemConfig, qdrantCfg *QdrantConfig, embedder Embedder, logger *zap.Logger, opts ...StoreOption) (Store, error) {
	var store Store
	var err error

	switch provider {
	case "chromem", "":
		if chromemCfg == nil {
			return nil, fmt.Errorf("chromem config required for chromem provider")
		}
		store, err = NewChromemStore(*chromemCfg, embedder, logger)

	case "qdrant":
		if qdrantCfg == nil {
			return nil, fmt.Errorf("qdrant config required for qdrant provider")
		}
		store, err = NewQdrantStore(*qdrantCfg, embedder)

	default:
		return nil, fmt.Errorf("unsupported vectorstore provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}

	// Apply options (e.g., isolation mode)
	for _, opt := range opts {
		opt(store)
	}

	return store, nil
}
