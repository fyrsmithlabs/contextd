// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
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
// If fallback is enabled, the store is wrapped with FallbackStore for graceful
// degradation when the remote store (Qdrant) is unavailable.
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

		// Check if fallback is enabled
		if cfg.Fallback.Enabled {
			// Create both remote (Qdrant) and local (chromem) stores
			remoteStore, remoteErr := NewQdrantStore(qdrantCfg, embedder)
			if remoteErr != nil {
				logger.Warn("fallback: failed to create remote store, will rely on local only",
					zap.Error(remoteErr))
			}

			// Create local chromem store for fallback
			localCfg := ChromemConfig{
				Path:              cfg.Fallback.LocalPath,
				Compress:          false, // No compression for local fallback
				DefaultCollection: qdrantCfg.CollectionName,
				VectorSize:        int(qdrantCfg.VectorSize),
			}
			localStore, localErr := NewChromemStore(localCfg, embedder, logger)
			if localErr != nil {
				if remoteStore != nil {
					remoteStore.Close()
				}
				return nil, fmt.Errorf("fallback: failed to create local store: %w", localErr)
			}

			// If remote failed, just use local
			if remoteErr != nil {
				logger.Info("fallback: using local store only (remote unavailable)")
				store = localStore
			} else {
				// Create WAL, health monitor, and fallback store
				ctx := context.Background()

				// Create scrubber for WAL
				scrubber, err := secrets.New(nil) // Use default config
				if err != nil {
					remoteStore.Close()
					localStore.Close()
					return nil, fmt.Errorf("fallback: failed to create scrubber: %w", err)
				}

				// Create WAL
				wal, err := NewWAL(cfg.Fallback.WALPath, scrubber, logger)
				if err != nil {
					remoteStore.Close()
					localStore.Close()
					return nil, fmt.Errorf("fallback: failed to create WAL: %w", err)
				}

				// Parse health check interval
				checkInterval, err := time.ParseDuration(cfg.Fallback.HealthCheckInterval)
				if err != nil {
					checkInterval = 30 * time.Second // Default
					logger.Warn("fallback: invalid health_check_interval, using default",
						zap.String("invalid", cfg.Fallback.HealthCheckInterval),
						zap.Duration("default", checkInterval))
				}

				// Create health monitor with mock checker
				// Note: For production, we'd need access to Qdrant's gRPC connection
				// For now, use a mock that we can control
				healthChecker := NewMockHealthChecker()
				healthChecker.SetHealthy(true) // Assume healthy at start

				health := NewHealthMonitor(ctx, healthChecker, checkInterval, logger)

				// Create fallback config
				fallbackCfg := FallbackConfig{
					Enabled:             cfg.Fallback.Enabled,
					LocalPath:           cfg.Fallback.LocalPath,
					SyncOnConnect:       cfg.Fallback.SyncOnConnect,
					HealthCheckInterval: cfg.Fallback.HealthCheckInterval,
					WALPath:             cfg.Fallback.WALPath,
					WALRetentionDays:    cfg.Fallback.WALRetentionDays,
				}

				// Create fallback store
				fallbackStore, err := NewFallbackStore(ctx, remoteStore, localStore, health, wal, fallbackCfg, logger)
				if err != nil {
					remoteStore.Close()
					localStore.Close()
					health.Stop()
					return nil, fmt.Errorf("fallback: failed to create fallback store: %w", err)
				}

				store = fallbackStore
				logger.Info("fallback: FallbackStore initialized",
					zap.String("remote", "qdrant"),
					zap.String("local", cfg.Fallback.LocalPath))
			}
		} else {
			// No fallback: just use Qdrant
			store, err = NewQdrantStore(qdrantCfg, embedder)
		}

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
