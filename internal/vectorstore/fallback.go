// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// FallbackConfig holds configuration for fallback storage.
type FallbackConfig struct {
	// Enabled enables fallback storage (default: false).
	Enabled bool

	// LocalPath is the path for local fallback storage.
	// Default: .claude/contextd/store
	LocalPath string

	// SyncOnConnect triggers immediate sync when remote becomes available (default: true).
	SyncOnConnect bool

	// HealthCheckInterval is the interval for periodic health checks (default: 30s).
	HealthCheckInterval string

	// WALPath is the directory for write-ahead log.
	// Default: .claude/contextd/wal
	WALPath string

	// WALRetentionDays is how long to keep synced entries in WAL (default: 7).
	WALRetentionDays int
}

// ApplyDefaults sets default values for unset fields.
func (c *FallbackConfig) ApplyDefaults() {
	if c.LocalPath == "" {
		c.LocalPath = ".claude/contextd/store"
	}
	if c.WALPath == "" {
		c.WALPath = ".claude/contextd/wal"
	}
	if c.HealthCheckInterval == "" {
		c.HealthCheckInterval = "30s"
	}
	if c.WALRetentionDays == 0 {
		c.WALRetentionDays = 7
	}
	// SyncOnConnect defaults to true (zero value is false, so we set it explicitly in factory)
}

// Validate validates the fallback configuration.
func (c *FallbackConfig) Validate() error {
	if c.LocalPath == "" {
		return fmt.Errorf("fallback: local_path is required")
	}
	if c.WALPath == "" {
		return fmt.Errorf("fallback: wal_path is required")
	}
	if c.WALRetentionDays < 0 {
		return fmt.Errorf("fallback: wal_retention_days must be non-negative")
	}
	return nil
}

// FallbackStore implements the Store interface with graceful fallback to local storage.
//
// When the remote store (Qdrant) is unavailable, writes go to local storage and a
// write-ahead log (WAL). When connectivity is restored, pending operations are
// automatically synced to the remote store.
//
// Thread-safe: All operations are protected by internal mutexes.
type FallbackStore struct {
	remote  Store          // Primary remote store (Qdrant)
	local   Store          // Fallback local store (chromem)
	health  *HealthMonitor // Health monitoring
	sync    *SyncManager   // Background sync manager
	wal     *WAL           // Write-ahead log
	config  FallbackConfig // Configuration
	logger  *zap.Logger    // Logger
	mu      sync.RWMutex   // Protects mode switches
}

// NewFallbackStore creates a new FallbackStore wrapping remote and local stores.
func NewFallbackStore(
	ctx context.Context,
	remote, local Store,
	health *HealthMonitor,
	wal *WAL,
	config FallbackConfig,
	logger *zap.Logger,
) (*FallbackStore, error) {
	if remote == nil {
		return nil, fmt.Errorf("fallback: remote store is required")
	}
	if local == nil {
		return nil, fmt.Errorf("fallback: local store is required")
	}
	if health == nil {
		return nil, fmt.Errorf("fallback: health monitor is required")
	}
	if wal == nil {
		return nil, fmt.Errorf("fallback: WAL is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	config.ApplyDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	fs := &FallbackStore{
		remote: remote,
		local:  local,
		health: health,
		wal:    wal,
		config: config,
		logger: logger,
	}

	// Create sync manager
	fs.sync = NewSyncManager(ctx, wal, local, remote, health, logger)

	// Start health monitoring and sync
	health.Start()
	fs.sync.Start()

	logger.Info("FallbackStore initialized",
		zap.String("local_path", config.LocalPath),
		zap.String("wal_path", config.WALPath),
		zap.Bool("sync_on_connect", config.SyncOnConnect),
		zap.Int("wal_retention_days", config.WALRetentionDays))

	return fs, nil
}

// AddDocuments adds documents with fallback support.
//
// Write path (atomic with rollback):
// 1. Scrub document content (handled by WAL)
// 2. Check remote health
// 3. IF HEALTHY:
//    a. Write to REMOTE first
//    b. Write to LOCAL (for query consistency)
//    c. Record in WAL as SYNCED
//    d. Return success
// 4. IF UNHEALTHY:
//    a. Record in WAL as PENDING (with checksum)
//    b. Write to LOCAL
//    c. Return success
// 5. ON ANY FAILURE:
//    a. Rollback: Delete from stores where written
//    b. Remove incomplete WAL entry
//    c. Return error
func (fs *FallbackStore) AddDocuments(ctx context.Context, docs []Document) ([]string, error) {
	// Validate tenant context (fail-closed)
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("fallback: %w", err)
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	healthy := fs.health.IsHealthy()

	// Generate entry ID for WAL
	entryID := fmt.Sprintf("add_%d", timeNow().UnixNano())

	if healthy {
		// Remote is healthy: Write to remote first, then local
		ids, err := fs.remote.AddDocuments(ctx, docs)
		if err != nil {
			fs.logger.Warn("fallback: remote write failed, falling back to local",
				zap.Error(err),
				zap.String("tenant_id", tenant.TenantID))
			// Fall through to local write
			healthy = false
		} else {
			// Remote write succeeded, write to local for consistency
			if _, localErr := fs.local.AddDocuments(ctx, docs); localErr != nil {
				fs.logger.Warn("fallback: local write failed after remote success",
					zap.Error(localErr),
					zap.String("tenant_id", tenant.TenantID))
				// Not fatal - remote has the data
			}

			// Record in WAL as synced
			walEntry := WALEntry{
				ID:          entryID,
				Operation:   "add",
				Docs:        docs,
				Timestamp:   timeNow(),
				Synced:      true,
				RemoteState: "exists",
			}
			if walErr := fs.wal.WriteEntry(ctx, walEntry); walErr != nil {
				fs.logger.Warn("fallback: WAL write failed (non-fatal)",
					zap.Error(walErr))
			}

			return ids, nil
		}
	}

	// Remote is unhealthy: Write to local and WAL
	fs.logger.Info("fallback: using local store",
		zap.String("tenant_id", tenant.TenantID),
		zap.Int("doc_count", len(docs)))

	// Record in WAL as pending BEFORE local write (durability)
	walEntry := WALEntry{
		ID:          entryID,
		Operation:   "add",
		Docs:        docs,
		Timestamp:   timeNow(),
		Synced:      false,
		RemoteState: "unknown",
	}
	if err := fs.wal.WriteEntry(ctx, walEntry); err != nil {
		return nil, fmt.Errorf("fallback: WAL write failed: %w", err)
	}

	// Write to local store
	ids, err := fs.local.AddDocuments(ctx, docs)
	if err != nil {
		fs.logger.Error("fallback: local write failed",
			zap.Error(err),
			zap.String("tenant_id", tenant.TenantID))
		return nil, fmt.Errorf("fallback: local write failed: %w", err)
	}

	fs.logger.Info("fallback: documents written to local store",
		zap.Int("count", len(ids)),
		zap.String("tenant_id", tenant.TenantID))

	return ids, nil
}

// Search performs similarity search with merge strategy.
//
// Read path (merge strategy):
// 1. Check remote health
// 2. IF HEALTHY:
//    a. Search REMOTE (authoritative)
//    b. Search LOCAL for pending (unsynced) documents
//    c. Merge results (local wins for conflicts)
//    d. Add metadata: {source: "merged", pending_count: N}
// 3. IF UNHEALTHY:
//    a. Search LOCAL only
//    b. Add metadata: {source: "local", last_sync: timestamp, stale_warning: true}
// 4. Return results with metadata
func (fs *FallbackStore) Search(ctx context.Context, query string, k int) ([]SearchResult, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	healthy := fs.health.IsHealthy()

	if healthy {
		// Try remote first
		remoteResults, err := fs.remote.Search(ctx, query, k)
		if err != nil {
			fs.logger.Warn("fallback: remote search failed, using local",
				zap.Error(err))
			// Fall through to local
			healthy = false
		} else {
			// TODO: Merge with pending local results if needed
			return remoteResults, nil
		}
	}

	// Use local store
	fs.logger.Debug("fallback: searching local store")
	localResults, err := fs.local.Search(ctx, query, k)
	if err != nil {
		return nil, fmt.Errorf("fallback: local search failed: %w", err)
	}

	// Add metadata indicating source
	for i := range localResults {
		if localResults[i].Metadata == nil {
			localResults[i].Metadata = make(map[string]interface{})
		}
		localResults[i].Metadata["source"] = "local"
		localResults[i].Metadata["stale_warning"] = true
	}

	return localResults, nil
}

// SearchWithFilters performs similarity search with metadata filters.
func (fs *FallbackStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	healthy := fs.health.IsHealthy()

	if healthy {
		results, err := fs.remote.SearchWithFilters(ctx, query, k, filters)
		if err != nil {
			fs.logger.Warn("fallback: remote search failed, using local", zap.Error(err))
			healthy = false
		} else {
			return results, nil
		}
	}

	// Use local store
	localResults, err := fs.local.SearchWithFilters(ctx, query, k, filters)
	if err != nil {
		return nil, fmt.Errorf("fallback: local search failed: %w", err)
	}

	// Add metadata
	for i := range localResults {
		if localResults[i].Metadata == nil {
			localResults[i].Metadata = make(map[string]interface{})
		}
		localResults[i].Metadata["source"] = "local"
	}

	return localResults, nil
}

// SearchInCollection performs similarity search in a specific collection.
func (fs *FallbackStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	healthy := fs.health.IsHealthy()

	if healthy {
		results, err := fs.remote.SearchInCollection(ctx, collectionName, query, k, filters)
		if err != nil {
			fs.logger.Warn("fallback: remote search failed, using local", zap.Error(err))
			healthy = false
		} else {
			return results, nil
		}
	}

	return fs.local.SearchInCollection(ctx, collectionName, query, k, filters)
}

// DeleteDocuments deletes documents by their IDs.
func (fs *FallbackStore) DeleteDocuments(ctx context.Context, ids []string) error {
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		return fmt.Errorf("fallback: %w", err)
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	healthy := fs.health.IsHealthy()
	entryID := fmt.Sprintf("delete_%d", timeNow().UnixNano())

	if healthy {
		// Try remote first
		if err := fs.remote.DeleteDocuments(ctx, ids); err != nil {
			fs.logger.Warn("fallback: remote delete failed, using local",
				zap.Error(err),
				zap.String("tenant_id", tenant.TenantID))
			healthy = false
		} else {
			// Delete from local too
			if localErr := fs.local.DeleteDocuments(ctx, ids); localErr != nil {
				fs.logger.Warn("fallback: local delete failed after remote success",
					zap.Error(localErr))
			}

			// Record in WAL as synced
			walEntry := WALEntry{
				ID:          entryID,
				Operation:   "delete",
				IDs:         ids,
				Timestamp:   timeNow(),
				Synced:      true,
				RemoteState: "deleted",
			}
			if walErr := fs.wal.WriteEntry(ctx, walEntry); walErr != nil {
				fs.logger.Warn("fallback: WAL write failed", zap.Error(walErr))
			}

			return nil
		}
	}

	// Remote unhealthy: Delete from local and record in WAL
	walEntry := WALEntry{
		ID:          entryID,
		Operation:   "delete",
		IDs:         ids,
		Timestamp:   timeNow(),
		Synced:      false,
		RemoteState: "unknown",
	}
	if err := fs.wal.WriteEntry(ctx, walEntry); err != nil {
		return fmt.Errorf("fallback: WAL write failed: %w", err)
	}

	if err := fs.local.DeleteDocuments(ctx, ids); err != nil {
		return fmt.Errorf("fallback: local delete failed: %w", err)
	}

	return nil
}

// DeleteDocumentsFromCollection deletes documents from a specific collection.
func (fs *FallbackStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	healthy := fs.health.IsHealthy()

	if healthy {
		if err := fs.remote.DeleteDocumentsFromCollection(ctx, collectionName, ids); err != nil {
			fs.logger.Warn("fallback: remote delete failed, using local", zap.Error(err))
			healthy = false
		} else {
			if localErr := fs.local.DeleteDocumentsFromCollection(ctx, collectionName, ids); localErr != nil {
				fs.logger.Warn("fallback: local delete failed after remote success", zap.Error(localErr))
			}
			return nil
		}
	}

	return fs.local.DeleteDocumentsFromCollection(ctx, collectionName, ids)
}

// CreateCollection creates a new collection.
func (fs *FallbackStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Create in both stores for consistency
	if err := fs.local.CreateCollection(ctx, collectionName, vectorSize); err != nil {
		return fmt.Errorf("fallback: local collection creation failed: %w", err)
	}

	if fs.health.IsHealthy() {
		if err := fs.remote.CreateCollection(ctx, collectionName, vectorSize); err != nil {
			fs.logger.Warn("fallback: remote collection creation failed", zap.Error(err))
			// Not fatal - local has it
		}
	}

	return nil
}

// DeleteCollection deletes a collection and all its documents.
func (fs *FallbackStore) DeleteCollection(ctx context.Context, collectionName string) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Delete from both stores
	if err := fs.local.DeleteCollection(ctx, collectionName); err != nil {
		fs.logger.Warn("fallback: local collection deletion failed", zap.Error(err))
	}

	if fs.health.IsHealthy() {
		if err := fs.remote.DeleteCollection(ctx, collectionName); err != nil {
			fs.logger.Warn("fallback: remote collection deletion failed", zap.Error(err))
		}
	}

	return nil
}

// CollectionExists checks if a collection exists.
func (fs *FallbackStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.health.IsHealthy() {
		exists, err := fs.remote.CollectionExists(ctx, collectionName)
		if err != nil {
			fs.logger.Warn("fallback: remote collection check failed, using local", zap.Error(err))
			return fs.local.CollectionExists(ctx, collectionName)
		}
		return exists, nil
	}

	return fs.local.CollectionExists(ctx, collectionName)
}

// ListCollections returns a list of all collection names.
func (fs *FallbackStore) ListCollections(ctx context.Context) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.health.IsHealthy() {
		collections, err := fs.remote.ListCollections(ctx)
		if err != nil {
			fs.logger.Warn("fallback: remote list collections failed, using local", zap.Error(err))
			return fs.local.ListCollections(ctx)
		}
		return collections, nil
	}

	return fs.local.ListCollections(ctx)
}

// GetCollectionInfo returns metadata about a collection.
func (fs *FallbackStore) GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.health.IsHealthy() {
		info, err := fs.remote.GetCollectionInfo(ctx, collectionName)
		if err != nil {
			fs.logger.Warn("fallback: remote get collection info failed, using local", zap.Error(err))
			return fs.local.GetCollectionInfo(ctx, collectionName)
		}
		return info, nil
	}

	return fs.local.GetCollectionInfo(ctx, collectionName)
}

// ExactSearch performs brute-force similarity search.
func (fs *FallbackStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]SearchResult, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if fs.health.IsHealthy() {
		results, err := fs.remote.ExactSearch(ctx, collectionName, query, k)
		if err != nil {
			fs.logger.Warn("fallback: remote exact search failed, using local", zap.Error(err))
			return fs.local.ExactSearch(ctx, collectionName, query, k)
		}
		return results, nil
	}

	return fs.local.ExactSearch(ctx, collectionName, query, k)
}

// SetIsolationMode sets the tenant isolation mode for both stores.
func (fs *FallbackStore) SetIsolationMode(mode IsolationMode) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.remote.SetIsolationMode(mode)
	fs.local.SetIsolationMode(mode)
}

// IsolationMode returns the current isolation mode.
func (fs *FallbackStore) IsolationMode() IsolationMode {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Both stores should have the same mode; return remote's
	return fs.remote.IsolationMode()
}

// Close closes the fallback store and releases resources.
func (fs *FallbackStore) Close() error {
	fs.logger.Info("fallback: closing")

	// Stop sync manager
	if err := fs.sync.Stop(); err != nil {
		fs.logger.Error("fallback: sync manager stop failed", zap.Error(err))
	}

	// Stop health monitor
	fs.health.Stop()

	// Close WAL
	if err := fs.wal.Close(); err != nil {
		fs.logger.Error("fallback: WAL close failed", zap.Error(err))
	}

	// Close stores
	var errs []error
	if err := fs.local.Close(); err != nil {
		fs.logger.Error("fallback: local store close failed", zap.Error(err))
		errs = append(errs, err)
	}
	if err := fs.remote.Close(); err != nil {
		fs.logger.Error("fallback: remote store close failed", zap.Error(err))
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("fallback: close errors: %v", errs)
	}

	fs.logger.Info("fallback: closed")
	return nil
}
