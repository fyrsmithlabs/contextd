// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
	"sync"

	"github.com/fyrsmithlabs/contextd/internal/registry"
	"go.uber.org/zap"
)

// StoreProvider manages chromem.DB instances per scope path.
//
// This enables database-per-project isolation where:
//   - Each project gets its own chromem.DB at a unique filesystem path
//   - Collection names are simple ("checkpoints", "memories") not prefixed
//   - Physical filesystem isolation prevents data leakage
//
// Path hierarchy:
//   - Project (free): {basePath}/{tenant}/{project}/
//   - Project (paid): {basePath}/{tenant}/{team}/{project}/
//   - Team shared: {basePath}/{tenant}/{team}/
//   - Org shared: {basePath}/{tenant}/
//
// SECURITY NOTE: LOCAL DEVELOPMENT ONLY
//
// The current implementation does NOT include authorization checks.
// Any caller can request any tenant/team/project store without verification.
// This is acceptable for:
//   - Local development (single-user, localhost)
//   - CLI tools (trusted environment)
//   - Testing environments
//
// For multi-tenant production deployments, you MUST add:
//   - Session-based authentication
//   - Tenant membership verification before granting store access
//   - Audit logging for all store access
//
// TODO: Implement AuthorizedStoreProvider wrapper for production use.
type StoreProvider interface {
	// GetProjectStore returns a store for project-level collections.
	// Path: {basePath}/{tenant}/{project}/ (free tier)
	// Path: {basePath}/{tenant}/{team}/{project}/ (paid tier)
	GetProjectStore(ctx context.Context, tenant, team, project string) (Store, error)

	// GetTeamStore returns a store for team-level shared collections.
	// Path: {basePath}/{tenant}/{team}/
	GetTeamStore(ctx context.Context, tenant, team string) (Store, error)

	// GetOrgStore returns a store for org-level shared collections.
	// Path: {basePath}/{tenant}/
	GetOrgStore(ctx context.Context, tenant string) (Store, error)

	// Close closes all managed stores.
	Close() error
}

// ChromemStoreProvider implements StoreProvider using chromem-go.
type ChromemStoreProvider struct {
	registry   *registry.Registry
	embedder   Embedder
	logger     *zap.Logger
	compress   bool
	vectorSize int

	mu     sync.RWMutex            // protects stores map
	stores map[string]*ChromemStore // path -> *ChromemStore
}

// ProviderConfig holds configuration for ChromemStoreProvider.
type ProviderConfig struct {
	// BasePath is the root directory for all vectorstore data.
	// Default: ~/.config/contextd/vectorstore
	BasePath string

	// Compress enables gzip compression for stored data.
	Compress bool

	// VectorSize is the expected embedding dimension.
	// Default: 384 (for FastEmbed bge-small-en-v1.5)
	VectorSize int
}

// ApplyDefaults sets default values for unset fields.
func (c *ProviderConfig) ApplyDefaults() {
	if c.VectorSize == 0 {
		c.VectorSize = 384
	}
}

// NewChromemStoreProvider creates a new StoreProvider backed by chromem-go.
func NewChromemStoreProvider(config ProviderConfig, embedder Embedder, logger *zap.Logger) (*ChromemStoreProvider, error) {
	if embedder == nil {
		return nil, fmt.Errorf("%w: embedder is required", ErrInvalidConfig)
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	config.ApplyDefaults()

	// Create registry
	reg, err := registry.NewRegistry(config.BasePath)
	if err != nil {
		return nil, fmt.Errorf("creating registry: %w", err)
	}

	return &ChromemStoreProvider{
		registry:   reg,
		embedder:   embedder,
		logger:     logger,
		compress:   config.Compress,
		vectorSize: config.VectorSize,
		stores:     make(map[string]*ChromemStore),
	}, nil
}

// GetProjectStore returns a store scoped to a specific project.
func (p *ChromemStoreProvider) GetProjectStore(ctx context.Context, tenant, team, project string) (Store, error) {
	// Ensure project exists (auto-registers tenant/team/project if needed)
	if err := p.registry.EnsureProjectExists(tenant, team, project); err != nil {
		return nil, fmt.Errorf("ensuring project exists: %w", err)
	}

	// Get filesystem path
	path, err := p.registry.GetProjectPath(tenant, team, project)
	if err != nil {
		return nil, fmt.Errorf("getting project path: %w", err)
	}

	return p.getOrCreateStore(path)
}

// GetTeamStore returns a store scoped to a team (for shared collections).
func (p *ChromemStoreProvider) GetTeamStore(ctx context.Context, tenant, team string) (Store, error) {
	// Ensure tenant and team exist
	if _, err := p.registry.RegisterTenant(tenant); err != nil {
		return nil, fmt.Errorf("registering tenant: %w", err)
	}
	if _, err := p.registry.RegisterTeam(tenant, team); err != nil {
		return nil, fmt.Errorf("registering team: %w", err)
	}

	// Get filesystem path
	path, err := p.registry.GetTeamPath(tenant, team)
	if err != nil {
		return nil, fmt.Errorf("getting team path: %w", err)
	}

	return p.getOrCreateStore(path)
}

// GetOrgStore returns a store scoped to an org (for org-level shared collections).
func (p *ChromemStoreProvider) GetOrgStore(ctx context.Context, tenant string) (Store, error) {
	// Ensure tenant exists
	if _, err := p.registry.RegisterTenant(tenant); err != nil {
		return nil, fmt.Errorf("registering tenant: %w", err)
	}

	// Get filesystem path
	path, err := p.registry.GetOrgPath(tenant)
	if err != nil {
		return nil, fmt.Errorf("getting org path: %w", err)
	}

	return p.getOrCreateStore(path)
}

// getOrCreateStore returns a cached store or creates a new one.
func (p *ChromemStoreProvider) getOrCreateStore(path string) (Store, error) {
	// Fast path: check cache with read lock
	p.mu.RLock()
	if store, ok := p.stores[path]; ok {
		p.mu.RUnlock()
		return store, nil
	}
	p.mu.RUnlock()

	// Slow path: acquire write lock and create store
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if store, ok := p.stores[path]; ok {
		return store, nil
	}

	// Create new store at path
	config := ChromemConfig{
		Path:              path,
		Compress:          p.compress,
		DefaultCollection: "default",
		VectorSize:        p.vectorSize,
	}

	store, err := NewChromemStore(config, p.embedder, p.logger)
	if err != nil {
		return nil, fmt.Errorf("creating store at %s: %w", path, err)
	}

	p.stores[path] = store

	p.logger.Info("created project store",
		zap.String("path", path),
		zap.Int("vector_size", p.vectorSize),
	)

	return store, nil
}

// Close closes all managed stores.
func (p *ChromemStoreProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for path, store := range p.stores {
		if err := store.Close(); err != nil {
			p.logger.Error("failed to close store",
				zap.String("path", path),
				zap.Error(err),
			)
			lastErr = err
		}
	}
	// Clear the map after closing
	p.stores = make(map[string]*ChromemStore)
	return lastErr
}

// Registry returns the underlying registry for direct access if needed.
func (p *ChromemStoreProvider) Registry() *registry.Registry {
	return p.registry
}

// Ensure ChromemStoreProvider implements StoreProvider.
var _ StoreProvider = (*ChromemStoreProvider)(nil)
