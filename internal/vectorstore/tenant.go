// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"errors"
	"sync"
)

// Tenant isolation error types - fail closed security model.
var (
	// ErrMissingTenant is returned when tenant info is missing from context
	// and cannot be resolved from defaults.
	ErrMissingTenant = errors.New("tenant info missing from context")

	// ErrInvalidTenant is returned when tenant identifier is invalid.
	ErrInvalidTenant = errors.New("invalid tenant identifier")

	// ErrMissingProject is returned when a tenant is otherwise resolvable but
	// has no ProjectID. ProjectID is the floor of contextd's isolation model:
	// even solo-dev defaults must produce a non-empty project to avoid
	// cross-repo bleed.
	ErrMissingProject = errors.New("project_id missing from tenant context")
)

// defaultResolverMu guards defaultResolver.
var defaultResolverMu sync.RWMutex

// defaultResolver is an optional hook used by TenantFromContext to derive a
// TenantInfo when none is present on the context. Registered by callers that
// want softer single-tenant defaults (e.g. cmd/contextd, the MCP server).
//
// Returning nil is allowed and yields ErrMissingTenant. Returning a TenantInfo
// without a ProjectID yields ErrMissingProject.
var defaultResolver func() *TenantInfo

// SetDefaultTenantResolver installs the package-level default tenant resolver.
// Pass nil to unregister (e.g. in tests).
//
// This indirection keeps vectorstore decoupled from internal/tenant while
// still letting the rest of the binary opt in to derived defaults.
func SetDefaultTenantResolver(fn func() *TenantInfo) {
	defaultResolverMu.Lock()
	defer defaultResolverMu.Unlock()
	defaultResolver = fn
}

// getDefaultResolver returns the currently installed default resolver, or nil.
func getDefaultResolver() func() *TenantInfo {
	defaultResolverMu.RLock()
	defer defaultResolverMu.RUnlock()
	return defaultResolver
}

// tenantContextKey is the context key for TenantInfo.
type tenantContextKey struct{}

// TenantInfo holds tenant context for filtering and isolation.
//
// Multi-tenancy hierarchy:
//   - TenantID: Organization or user identifier. Required when explicitly
//     constructing TenantInfo; defaults to $USER / git user / "local" when
//     derived from the package default resolver.
//   - TeamID (optional): Team scope within tenant. No default.
//   - ProjectID: Project scope. The floor of contextd's isolation model -
//     enforced by TenantFromContext via ErrMissingProject when absent.
//
// Security: All fields are validated before use in queries.
type TenantInfo struct {
	// TenantID is the organization/user identifier.
	TenantID string

	// TeamID is the team identifier (optional).
	TeamID string

	// ProjectID is the project identifier (the isolation floor).
	ProjectID string
}

// Validate checks that required fields are present and valid.
//
// TenantID must be non-empty. ProjectID is not enforced here so that callers
// constructing TenantInfo by hand (e.g. legacy SaaS code paths that scope by
// tenant only) keep working; the ProjectID floor is enforced where defaults
// are resolved - see TenantFromContext.
func (t *TenantInfo) Validate() error {
	if t.TenantID == "" {
		return ErrInvalidTenant
	}
	return nil
}

// ContextWithTenant adds TenantInfo to a context.
//
// The TenantInfo is defensively copied by value before being stored. This
// guarantees tenant-context immutability: once a context is created, later
// mutation of the caller's *TenantInfo cannot change the tenant scope observed
// by in-flight operations. This is a security guarantee, not just a convenience
// — without the copy a caller (or a racing goroutine) could re-scope documents
// and queries to a different tenant mid-flight.
func ContextWithTenant(ctx context.Context, tenant *TenantInfo) context.Context {
	if tenant == nil {
		return context.WithValue(ctx, tenantContextKey{}, (*TenantInfo)(nil))
	}
	// Copy by value so external mutation of the caller's struct is invisible
	// to anything reading from this context.
	cp := *tenant
	return context.WithValue(ctx, tenantContextKey{}, &cp)
}

// TenantFromContext extracts TenantInfo from a context.
//
// Resolution order:
//  1. Explicit TenantInfo on the context wins (caller-provided IDs always
//     take precedence).
//  2. If no TenantInfo is present and a default resolver has been registered
//     via SetDefaultTenantResolver, derive a TenantInfo from defaults.
//  3. If derivation produces an empty TenantID, return ErrMissingTenant.
//  4. If derivation produces an empty ProjectID, return ErrMissingProject -
//     ProjectID is the floor of isolation.
//
// When no resolver is registered, this function preserves its historical
// fail-closed contract and returns ErrMissingTenant for an empty context.
func TenantFromContext(ctx context.Context) (*TenantInfo, error) {
	val := ctx.Value(tenantContextKey{})
	if val != nil {
		tenant, ok := val.(*TenantInfo)
		if ok && tenant != nil {
			return tenant, nil
		}
	}

	// No tenant in context - try the registered default resolver.
	resolver := getDefaultResolver()
	if resolver == nil {
		return nil, ErrMissingTenant
	}
	derived := resolver()
	if derived == nil {
		return nil, ErrMissingTenant
	}
	if derived.TenantID == "" {
		return nil, ErrMissingTenant
	}
	if derived.ProjectID == "" {
		return nil, ErrMissingProject
	}
	return derived, nil
}

// MustTenantFromContext extracts TenantInfo from context or panics.
// Use only when tenant presence is guaranteed by middleware.
func MustTenantFromContext(ctx context.Context) *TenantInfo {
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		panic("tenant info required but missing from context")
	}
	return tenant
}

// HasTenant checks if TenantInfo is present in context without error.
func HasTenant(ctx context.Context) bool {
	_, err := TenantFromContext(ctx)
	return err == nil
}

// TenantMetadata returns tenant info as a metadata map for document storage.
func (t *TenantInfo) TenantMetadata() map[string]interface{} {
	meta := map[string]interface{}{
		"tenant_id": t.TenantID,
	}
	if t.TeamID != "" {
		meta["team_id"] = t.TeamID
	}
	if t.ProjectID != "" {
		meta["project_id"] = t.ProjectID
	}
	return meta
}

// TenantFilter returns filter conditions for queries.
// Returns conditions that match this tenant's scope.
func (t *TenantInfo) TenantFilter() map[string]interface{} {
	filter := map[string]interface{}{
		"tenant_id": t.TenantID,
	}
	// Only add team/project filters if specified
	if t.TeamID != "" {
		filter["team_id"] = t.TeamID
	}
	if t.ProjectID != "" {
		filter["project_id"] = t.ProjectID
	}
	return filter
}
